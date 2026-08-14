# AI Knowledge Agent

**Phase:** 2.2 - AI Core Infrastructure  
**Port:** 5000 (API) | **Status:** ✅ Production

---

## Purpose

Semantic embedding generation and vector search for NAS.AI:
- **Embedding API** (`/embed`, `/embed_query`) for text vectorization
- **File Processing** (`/process`) with PostgreSQL/pgvector storage
- **Corpus Generator** for test document creation

---

## API Endpoints

### Health Check
```
GET /health → {"status": "ok", "model_loaded": true, "db_ok": true}
```

### Generate Embedding (Standardized)
```bash
POST /embed
Content-Type: application/json

{"text": "Your text here"}

# Response:
{"status": "ok", "data": {"embedding": [...], "dimensions": 384}, "error": null}
```

### Process File (Store in DB)
```bash
POST /process
Content-Type: application/json

{"file_path": "/mnt/data/doc.txt", "file_id": "doc_001", "mime_type": "text/plain"}

# Response:
{"status": "success", "file_id": "doc_001", "content_length": 1234, "embedding_dim": 384}
```

### Query Embedding (Legacy)
```bash
POST /embed_query
{"text": "search query"}
# Response: {"embedding": [...]}
```

---

## Model

| Property | Value |
|----------|-------|
| **Model** | sentence-transformers/all-MiniLM-L6-v2 |
| **Dimensions** | 384 |
| **Max Sequence** | 256 tokens |
| **Size** | ~90 MB |

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PGHOST` | postgres | PostgreSQL host |
| `PGPORT` | 5432 | PostgreSQL port |
| `PGDATABASE` | nas_db | Database name |
| `PGUSER` | nas_user | Database user |
| `PGPASSWORD` | - | Database password |

---

## Corpus Generator

Generate test documents for semantic search validation:

```bash
# Setup
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# Generate 50 documents
python generate_corpus.py --count 50 --output ./output

# With noise injection (OCR simulation)
python generate_corpus.py --count 50 --noise 0.5 --output ./output
```

### Document Types
- **Invoices** (PDF/HTML/TXT) - German business invoices
- **Tech Logs** - Server/application logs
- **Emails** - Business correspondence

### Output
```
output/
├── docs/
│   ├── rechnung_re_2023_1234.pdf
│   └── log_web-3_20231015.txt
└── ground_truth.json
```

---

## Docker

```bash
# Build
docker build -t nas-ai-knowledge-agent:1.0.0 .

# Run standalone
docker run -p 5000:5000 \
  -e PGHOST=postgres \
  -e PGPASSWORD=secret \
  nas-ai-knowledge-agent:1.0.0
```

---

## Architecture

```
src/main.py           # Flask API (embed, process, health)
├── init_db_pool()    # Connection pooling + DDL at startup
├── /embed            # Standardized embedding endpoint
├── /process          # File → embedding → pgvector
└── /health           # Readiness probe

generators/           # Corpus data generators
renderers/            # PDF/HTML/TXT output
templates/            # Jinja2 templates
noise.py              # OCR-style corruption
```

---

**Updated:** 2025-12-04
