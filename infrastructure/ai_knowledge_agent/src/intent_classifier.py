"""
Intent Classifier for Unified AI Query System.

Uses Llama to classify user queries into:
- type: "search" (file lookup) or "question" (needs AI answer)
- count_hint: "exact_match" | "few" | "many" 
- refined_query: Optimized search query
- filters: Extracted metadata filters (year, file_type, etc.)
"""

import json
import logging
import os
import re
import requests

logger = logging.getLogger("ai_knowledge_agent")

OLLAMA_URL = os.getenv("OLLAMA_URL", "http://host.docker.internal:11434")

# Three dedicated models:
# 1. Embedding model (mxbai-embed-large) - for database vectors
# 2. Classifier model (llama3.2:1b) - ONLY for intent classification (fast!)
# 3. LLM model (llama3.2) - for RAG answer generation
CLASSIFIER_MODEL = os.getenv("CLASSIFIER_MODEL", "llama3.2:1b")
LLM_MODEL = os.getenv("LLM_MODEL", "llama3.2")

# Limit mapping based on count_hint
LIMIT_MAP = {
    "exact_match": 3,
    "few": 10,
    "many": 50
}

# Default fallback intent
DEFAULT_INTENT = {
    "type": "search",
    "count_hint": "few",
    "refined_query": None,
    "filters": {},
    "confidence": 0.5
}


def classify_intent(query: str) -> dict:
    """
    Analyze user input and classify the intent.
    
    Returns:
        {
            "type": "search" | "question",
            "count_hint": "exact_match" | "few" | "many",
            "refined_query": str,
            "filters": { "year": int|None, "file_type": str|None },
            "limit": int,
            "confidence": float
        }
    """
    if not query or not query.strip():
        return {**DEFAULT_INTENT, "refined_query": query, "limit": LIMIT_MAP["few"]}
    
    # Try fast heuristic classification first
    heuristic_result = _heuristic_classify(query)
    if heuristic_result["confidence"] >= 0.9:
        logger.info("Intent classified via heuristic: %s", heuristic_result)
        return heuristic_result
    
    # Use Llama for complex cases
    try:
        llama_result = _llama_classify(query)
        logger.info("Intent classified via Llama: %s", llama_result)
        return llama_result
    except Exception as e:
        logger.warning("Llama classification failed, using heuristic: %s", e)
        return heuristic_result


def _heuristic_classify(query: str) -> dict:
    """
    Fast rule-based classification for obvious cases.
    High confidence for clear patterns, low for ambiguous.
    """
    query_lower = query.lower().strip()
    
    result = {
        "type": "search",
        "count_hint": "few",
        "refined_query": query,
        "filters": {},
        "confidence": 0.7
    }
    
    # PRIORITY 1: Question mark at end → definitely a question
    if query.strip().endswith("?"):
        result["type"] = "question"
        result["confidence"] = 0.98  # Highest priority
        # Add limit and return early for questions with ?
        result["limit"] = LIMIT_MAP.get(result["count_hint"], 10)
        return result
    
    # PRIORITY 2: Question words at start (Was, Wer, Wie, etc.)
    question_word_patterns = [
        r"^(was|wer|wie|warum|wann|wo|welche|welcher|welches)\b",
        r"^(erkläre|erklär|beschreibe|fasse|zusammenfassung|nenne|liste|zeige)\b",
        r"^(can you|could you|what|how|why|when|where|who|which)\b",
        r"^(bitte|kannst du|könntest du|sag mir|zeig mir)\b"
    ]
    
    for pattern in question_word_patterns:
        if re.search(pattern, query_lower):
            result["type"] = "question"
            result["confidence"] = 0.92
            break
    
    # PRIORITY 3: Complex sentences (with , or .) that look like requests
    # Sentences with multiple clauses are usually questions/requests, not keyword searches
    if result["type"] == "search":  # Only if not already classified as question
        has_comma = "," in query
        has_period = "." in query and not query.strip().endswith(".")  # Period in middle
        word_count = len(query.split())
        
        if (has_comma or has_period) and word_count > 5:
            # Complex sentence - likely a request/question
            result["type"] = "question"
            result["confidence"] = 0.91  # Above LLM threshold
    
    # Bulk patterns → count_hint: many
    bulk_patterns = [
        r"^alle\b",
        r"^sämtliche\b", 
        r"^jede\b",
        r"^all\b",
        r"^every\b",
        r"^list(e)?\b"
    ]
    
    for pattern in bulk_patterns:
        if re.search(pattern, query_lower):
            result["count_hint"] = "many"
            result["confidence"] = max(result["confidence"], 0.8)
            break
    
    # Specific file patterns → count_hint: exact_match
    specific_patterns = [
        r"^(die|das|der)\s+(datei|dokument|rechnung|vertrag)\b",
        r"^(file|document)\s+\w+",
        r"\.(pdf|txt|doc|xlsx?)$"
    ]
    
    for pattern in specific_patterns:
        if re.search(pattern, query_lower):
            result["count_hint"] = "exact_match"
            result["confidence"] = max(result["confidence"], 0.8)
            break
    
    # Extract year filter
    year_match = re.search(r"\b(20\d{2})\b", query)
    if year_match:
        result["filters"]["year"] = int(year_match.group(1))
    
    # Extract file type filter
    type_match = re.search(r"\.(pdf|txt|doc|docx|xlsx?|csv|json|md)\b", query_lower)
    if type_match:
        result["filters"]["file_type"] = type_match.group(1)
    
    # Add limit based on count_hint
    result["limit"] = LIMIT_MAP.get(result["count_hint"], 10)
    
    return result


def _llama_classify(query: str) -> dict:
    """
    Use Llama for intelligent intent classification.
    """
    system_prompt = """Du bist ein Query-Klassifikator für ein Dokumenten-Suchsystem.

Analysiere die Benutzeranfrage und klassifiziere sie.

ANTWORTE NUR mit diesem exakten JSON-Format, KEINE anderen Zeichen:
{"type":"search","count_hint":"few","refined_query":"optimierte anfrage","filters":{"year":null,"file_type":null}}

REGELN:
- "type": 
  - "search" = User sucht nach Dateien (z.B. "Rechnung Müller", "Server Logs")
  - "question" = User will eine Antwort/Erklärung (z.B. "Was kostet...", "Wie funktioniert...")

- "count_hint":
  - "exact_match" = User sucht 1-3 spezifische Dateien (z.B. "die Rechnung von Müller")
  - "few" = User erwartet 5-10 Ergebnisse (z.B. "Rechnungen 2024")
  - "many" = User will umfassende Liste (z.B. "alle Logs", "sämtliche Berichte")

- "refined_query": Optimiere die Suchanfrage für bessere Ergebnisse
- "filters": Extrahiere Jahr und Dateityp wenn vorhanden

Antworte NUR mit dem JSON, keine Erklärung."""

    prompt = f"Klassifiziere diese Anfrage: \"{query}\""

    try:
        logger.info("Using classifier model: %s", CLASSIFIER_MODEL)
        response = requests.post(
            f"{OLLAMA_URL}/api/generate",
            json={
                "model": CLASSIFIER_MODEL,  # Use dedicated small classification model
                "prompt": prompt,
                "system": system_prompt,
                "stream": False,
                "options": {
                    "temperature": 0.1,
                    "num_predict": 150  # Reduced - only need JSON output
                }
            },
            timeout=15  # Shorter timeout for small model
        )
        response.raise_for_status()
        
        raw_response = response.json().get("response", "").strip()
        
        # Parse JSON from response
        parsed = _parse_llama_json(raw_response, query)
        return parsed
        
    except requests.exceptions.Timeout:
        logger.warning("Llama classification timed out")
        raise
    except Exception as e:
        logger.error("Llama classification error: %s", e)
        raise


def _parse_llama_json(raw: str, original_query: str) -> dict:
    """
    Parse Llama's JSON response with error handling.
    """
    result = {
        "type": "search",
        "count_hint": "few", 
        "refined_query": original_query,
        "filters": {},
        "confidence": 0.8
    }
    
    try:
        # Try to extract JSON from response
        json_match = re.search(r'\{[^{}]*\}', raw)
        if json_match:
            parsed = json.loads(json_match.group())
            
            # Validate and extract fields
            if parsed.get("type") in ["search", "question"]:
                result["type"] = parsed["type"]
            
            if parsed.get("count_hint") in ["exact_match", "few", "many"]:
                result["count_hint"] = parsed["count_hint"]
            
            if parsed.get("refined_query"):
                result["refined_query"] = parsed["refined_query"]
            
            if isinstance(parsed.get("filters"), dict):
                result["filters"] = parsed["filters"]
            
            result["confidence"] = 0.9
            
    except json.JSONDecodeError as e:
        logger.warning("Failed to parse Llama JSON: %s - Raw: %s", e, raw[:200])
    
    # Add limit based on count_hint
    result["limit"] = LIMIT_MAP.get(result["count_hint"], 10)
    
    return result


def get_limit_for_intent(intent: dict) -> int:
    """
    Get the appropriate result limit for an intent.
    """
    count_hint = intent.get("count_hint", "few")
    return LIMIT_MAP.get(count_hint, 10)
