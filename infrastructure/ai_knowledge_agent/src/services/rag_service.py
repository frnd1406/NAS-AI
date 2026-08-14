import os
import logging
import requests
import json
import time
import threading
import redis as redis_lib

from flask import current_app

try:
    from src.intent_classifier import classify_intent, get_limit_for_intent, CLASSIFIER_MODEL
except ImportError:
    # Adjust import if running from different context
    from intent_classifier import classify_intent, get_limit_for_intent, CLASSIFIER_MODEL

logger = logging.getLogger("ai_knowledge_agent")

class RAGService:
    def __init__(self, db):
        self.db = db
        
        # Configuration
        self.ollama_url = os.getenv("OLLAMA_URL", "http://host.docker.internal:11434")
        self.embedding_model = os.getenv("EMBEDDING_MODEL", "mxbai-embed-large")
        self.llm_model = os.getenv("LLM_MODEL", "llama3.2")
        self.auto_index_enabled = os.getenv("AUTO_INDEX_ENABLED", "false").lower() == "true"
        self.worker_enabled = os.getenv("WORKER_ENABLED", "true").lower() == "true"
        
        # Redis Configuration
        self.redis_url = os.getenv("REDIS_URL", "redis://redis:6379")
        self.job_stream = "ai:jobs"
        self.consumer_group = "ai-workers"
        self.result_key_prefix = "ai:results:"
        
        self.model_loaded = False
        self.redis_client = None

    # === Ollama Integration ===

    def get_ollama_embedding(self, text: str) -> list:
        """Get embedding from Ollama model."""
        try:
            response = requests.post(
                f"{self.ollama_url}/api/embeddings",
                json={"model": self.embedding_model, "prompt": text},
                timeout=300
            )
            response.raise_for_status()
            return response.json()["embedding"]
        except Exception as e:
            logger.error("Ollama embedding failed: %s", e)
            raise

    def get_llama_response(self, prompt: str, documents: list) -> dict:
        """Get intelligent response from Llama 3.2."""
        system_prompt = """Du bist ein intelligenter KI-Assistent für ein NAS-Dokumentensystem.

DEINE AUFGABE:
1. Analysiere die bereitgestellten Dokumente und entscheide SELBST, welche relevant sind
2. Zitiere NUR die Dokumente, die wirklich zur Frage passen (0-5, je nach Relevanz)
3. Gib eine DIREKTE Antwort wenn die Information vorhanden ist (z.B. "Der Server kostet 149,99€")
4. Wenn keine passenden Dokumente existieren, sag das ehrlich

ANTWORT-FORMAT (strikt einhalten):
---
RELEVANTE QUELLEN: [Liste der wirklich relevanten Dateinamen, oder "Keine"]
KONFIDENZ: [HOCH/MITTEL/NIEDRIG]
ANTWORT: [Deine direkte Antwort mit Quellenverweisen wie [Dok1]]
---

BEISPIELE:
- Frage "Was kostet der Server?" → Wenn Rechnung 149,99€ zeigt: "Der Server kostet 149,99€ [rechnung_xyz.txt]"
- Frage nach etwas Unbekanntem → "Dazu habe ich keine Informationen in den Dokumenten gefunden."
- Mehrere relevante Docs → "Basierend auf [dok1] und [dok2]: ..."

Antworte IMMER auf Deutsch. Sei präzise und direkt."""

        # Build context with document info
        context_parts = []
        for i, doc in enumerate(documents, 1):
            sim_percent = int(doc['similarity'] * 100)
            context_parts.append(f"[Dokument {i}: {doc['file_id']} (Ähnlichkeit: {sim_percent}%)]\n{doc['content'][:2000]}")
        
        context = "\n\n---\n\n".join(context_parts)

        full_prompt = f"""Hier sind {len(documents)} Dokumente aus der Datenbank:

{context}

---

FRAGE DES BENUTZERS: {prompt}

Analysiere die Dokumente und antworte im vorgegebenen Format:"""

        try:
            response = requests.post(
                f"{self.ollama_url}/api/generate",
                json={
                    "model": self.llm_model,
                    "prompt": full_prompt,
                    "system": system_prompt,
                    "stream": False,
                    "options": {"temperature": 0.2, "num_predict": 800}
                },
                timeout=600
            )
            response.raise_for_status()
            raw_response = response.json()["response"]
            
            # Parse the structured response
            return self._parse_llama_response(raw_response, documents)
            
        except Exception as e:
            logger.error("Llama generation failed: %s", e)
            raise

    def _parse_llama_response(self, raw: str, documents: list) -> dict:
        """Parse Llama's structured response into components."""
        result = {
            "answer": raw,
            "cited_sources": [],
            "confidence": "MITTEL",
            "raw_response": raw
        }
        
        try:
            # Extract RELEVANTE QUELLEN
            if "RELEVANTE QUELLEN:" in raw:
                sources_line = raw.split("RELEVANTE QUELLEN:")[1].split("\n")[0].strip()
                if sources_line.lower() != "keine" and sources_line != "[]":
                    # Find matching documents
                    for doc in documents:
                        if doc['file_id'] in sources_line or doc['file_id'].replace('.txt', '') in sources_line:
                            result["cited_sources"].append({
                                "file_id": doc['file_id'],
                                "file_path": doc['file_path'],
                                "similarity": doc['similarity']
                            })
            
            # Extract KONFIDENZ
            if "KONFIDENZ:" in raw:
                conf_line = raw.split("KONFIDENZ:")[1].split("\n")[0].strip().upper()
                if conf_line in ["HOCH", "MITTEL", "NIEDRIG"]:
                    result["confidence"] = conf_line
            
            # Extract ANTWORT
            if "ANTWORT:" in raw:
                answer_part = raw.split("ANTWORT:")[1].strip()
                if "---" in answer_part:
                    answer_part = answer_part.split("---")[0].strip()
                result["answer"] = answer_part
                
        except Exception as e:
            logger.warning("Could not parse structured response: %s", e)
        
        return result

    def check_ollama_health(self):
        """Check if Ollama is available and models are loaded."""
        try:
            response = requests.get(f"{self.ollama_url}/api/tags", timeout=5)
            if response.status_code == 200:
                models = [m["name"] for m in response.json().get("models", [])]
                embed_ok = any(self.embedding_model in m for m in models)
                llm_ok = any(self.llm_model in m for m in models)
                self.model_loaded = embed_ok and llm_ok
                if self.model_loaded:
                    logger.info("Ollama ready: %s + %s", self.embedding_model, self.llm_model)
                else:
                    logger.warning("Missing models. Available: %s", models)
                return self.model_loaded
        except Exception as e:
            logger.error("Ollama health check failed: %s", e)
            self.model_loaded = False
        return False

    def prewarm_models(self):
        """Pre-warm Ollama models."""
        logger.info("Pre-warming Ollama models...")
        try:
            # Embedding model
            logger.info("Loading embedding model: %s", self.embedding_model)
            _ = self.get_ollama_embedding("Warmup test")
            
            # Classifier model
            logger.info("Loading classifier model: %s", CLASSIFIER_MODEL)
            requests.post(f"{self.ollama_url}/api/generate", json={"model": CLASSIFIER_MODEL, "prompt": "Hi", "stream": False, "options": {"num_predict": 1}}, timeout=120)
            
            # LLM model
            logger.info("Loading LLM model: %s", self.llm_model)
            requests.post(f"{self.ollama_url}/api/generate", json={"model": self.llm_model, "prompt": "Hi", "stream": False, "options": {"num_predict": 1}}, timeout=300)
            
            self.model_loaded = True
            logger.info("🚀 Models pre-warmed!")
            return True
        except Exception as e:
            logger.error("Model pre-warming failed: %s", e)
            self.model_loaded = False
            return False

    # === File Processing ===

    def process_file(self, content: str, mime_type: str, file_id: str, file_path: str = None):
        """Process and index a file content."""
        if not self.model_loaded:
             raise Exception("Ollama not loaded")
        
        logger.info("Generating embedding for %s (%d chars)", file_id, len(content))
        embedding = self.get_ollama_embedding(content[:8000])
        
        metadata = {
            "file_path": file_path or "memory",
            "mime_type": mime_type,
            "content_length": len(content)
        }
        
        self.db.save_embedding(file_id, content, embedding, metadata)
        logger.info("File indexed: %s", file_id)

    def scan_and_index(self, data_dir="/mnt/data"):
        """Scan directory and index new files."""
        if not self.model_loaded or not self.db.db_pool:
            return 0
        
        if not os.path.exists(data_dir):
            return 0
            
        TEXT_EXTENSIONS = {'.txt', '.md', '.json', '.csv', '.log', '.xml', '.html', '.py', '.js', '.go', '.sh'}
        indexed_count = 0
        skipped_count = 0
        
        existing_files = self.db.get_existing_files()
        
        for root, dirs, files in os.walk(data_dir):
            dirs[:] = [d for d in dirs if not d.startswith('.') and d != '.trash']
            
            for filename in files:
                if filename.startswith('.'): continue
                file_ext = os.path.splitext(filename)[1].lower()
                if file_ext not in TEXT_EXTENSIONS: continue
                if filename in existing_files:
                    skipped_count += 1
                    continue
                
                try:
                    file_path = os.path.join(root, filename)
                    with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                        content = f.read()
                    
                    if not content.strip(): continue
                    
                    self.process_file(content, "text/plain", filename, file_path)
                    indexed_count += 1
                except Exception as e:
                    logger.error("Failed to index %s: %s", filename, e)
        
        return indexed_count

    # === Search & Query Logic ===

    def unified_query(self, query: str):
        """Handle unified query (Search OR RAG)."""
        if not self.model_loaded:
             raise Exception("Ollama not loaded")

        # 1. Intent Classification
        intent = classify_intent(query)
        search_query = intent.get("refined_query") or query
        limit = intent["limit"]
        
        # 2. Get Embedding & Search
        query_embedding = self.get_ollama_embedding(search_query)
        docs = self.db.search_similar(query_embedding, limit)
        
        # 3. Route Result
        if intent["type"] == "question":
            if not docs:
                return {
                    "mode": "answer", "intent": intent, "answer": "Keine relevanten Dokumente gefunden.",
                    "sources": [], "confidence": "NIEDRIG", "query": query
                }
                
            llama_result = self.get_llama_response(query, docs)
            return {
                "mode": "answer", "intent": intent, "answer": llama_result["answer"],
                "sources": llama_result["cited_sources"], "confidence": llama_result["confidence"],
                "all_candidates": len(docs), "query": query, "model": self.llm_model
            }
        else:
            files = []
            for doc in docs:
                files.append({
                    "file_id": doc["file_id"],
                    "file_path": doc["file_path"],
                    "content": doc["content"][:500],
                    "similarity": doc["similarity"]
                })
            return {
                "mode": "search", "intent": intent, "files": files,
                "total_found": len(files), "query": query
            }

    # === Background Tasks & Redis Worker ===

    def background_health_check(self):
        """Periodic health check and auto-indexer."""
        while True:
            time.sleep(30)
            try:
                # Check Ollama
                if not self.check_ollama_health():
                    self.prewarm_models()
                
                # Auto-index
                if self.auto_index_enabled and self.model_loaded:
                    new_files = self.scan_and_index()
                    if new_files > 0:
                        logger.info("🆕 Auto-indexed %d new files", new_files)
                        
            except Exception as e:
                logger.warning("Background health check error: %s", e)

    def start_background_threads(self):
        t = threading.Thread(target=self.background_health_check, daemon=True)
        t.start()
        logger.info("Background health check thread started")
        
        if self.worker_enabled:
            self.start_redis_worker()

    def init_redis(self):
        try:
            self.redis_client = redis_lib.from_url(self.redis_url, decode_responses=True)
            self.redis_client.ping()
            logger.info("✅ Redis connection established")
            return True
        except Exception as e:
            logger.warning("Redis connection failed: %s", e)
            self.redis_client = None
            return False

    def ensure_consumer_group(self):
        if not self.redis_client: return
        try:
            self.redis_client.xgroup_create(self.job_stream, self.consumer_group, id="0", mkstream=True)
        except redis_lib.ResponseError as e:
            if "BUSYGROUP" not in str(e):
                logger.error("Failed to create consumer group: %s", e)

    def worker_loop(self):
        if not self.redis_client: return
        consumer_name = f"worker-{os.getpid()}"
        logger.info("🚀 Starting Redis worker: %s", consumer_name)
        
        while True:
            try:
                messages = self.redis_client.xreadgroup(
                    self.consumer_group, consumer_name, {self.job_stream: ">"}, count=1, block=5000
                )
                if not messages: continue
                
                for stream, entries in messages:
                    for entry_id, data in entries:
                        self.process_redis_job(entry_id, data)
                        
            except Exception as e:
                logger.error("Worker error: %s", e)
                time.sleep(1)

    def process_redis_job(self, entry_id, data):
        job_id = data.get("job_id")
        query = data.get("query", "")
        if not job_id:
             self.redis_client.xack(self.job_stream, self.consumer_group, entry_id)
             return

        # Update status
        self.redis_client.setex(f"{self.result_key_prefix}{job_id}", 3600, json.dumps({"job_id": job_id, "status": "processing"}))
        
        try:
            result = self.unified_query(query)
            result["job_id"] = job_id
            result["status"] = "completed"
            result["completed_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        except Exception as e:
            result = {"job_id": job_id, "status": "failed", "error": str(e)}
        
        self.redis_client.setex(f"{self.result_key_prefix}{job_id}", 3600, json.dumps(result))
        self.redis_client.xack(self.job_stream, self.consumer_group, entry_id)
        logger.info("✅ Job %s completed", job_id)

    def start_redis_worker(self):
        if not self.init_redis(): return
        self.ensure_consumer_group()
        t = threading.Thread(target=self.worker_loop, daemon=True)
        t.start()
