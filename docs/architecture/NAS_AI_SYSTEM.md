# NAS.AI – System Architecture & Infrastructure

**Version:** 2.0  
**Phase:** 2.2 - AI Core Infrastructure  
**Updated:** 2025-12-04

---

## Table of Contents
1. [System Overview](#1-system-overview)
2. [Service Architecture](#2-service-architecture)
3. [Data & Control Flows](#3-data--control-flows)
4. [Security & Governance](#4-security--governance)
5. [Filesystem Layout](#5-filesystem-layout)
6. [API Contracts](#6-api-contracts)
7. [AI Knowledge Layer](#7-ai-knowledge-layer)
8. [Testing & QA](#8-testing--qa)
9. [Incident Response](#9-incident-response)
10. [References](#10-references)

---

## 1. System Overview

```
                             ┌────────────────────────┐
                             │   Users & Clients      │
                             │  WebUI / Mobile / CLI  │
                             └──────────┬─────────────┘
                                        │ HTTPS (JWT)
                                        ▼
┌────────────────────────────────────────────────────────────────────┐
│                        Experience Tier                             │
│ ┌─────────────┐   ┌───────────────┐   ┌─────────────────────────┐  │
│ │ Vite/React  │<->│ API Gateway   │<->│ WebSocket Events        │  │
│ │ TailwindCSS │   │ (Go :8080)    │   │ Toast/Push Notifications│  │
│ └─────────────┘   └───────────────┘   └─────────────────────────┘  │
└───────────┬────────────────────────────────────────────────────────┘
            │ REST/JSON
            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Service Tier                                    │
│ ┌─────────────┐        ┌────────────────┐           ┌────────────────┐  │
│ │ API Service │ <----> │ PostgreSQL     │ <------>  │ Redis Cache    │  │
│ │ (Go)        │        │ + pgvector     │           │                │  │
│ └─────┬───────┘        └────────────────┘           └────────────────┘  │
│       │                                                                 │
│ ┌─────▼────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────┐       │
│ │ Orchestrator │  │ Monitoring   │  │ Analysis     │  │Pentester│       │
│ │ (Go :9000)   │  │ Agent        │  │ Agent        │  │Agent    │       │
│ └──────────────┘  └──────────────┘  └──────────────┘  └─────────┘       │
│       │                                                                 │
│ ┌─────▼────────────────────────────────────────────────────────┐        │
│ │              AI Knowledge Agent (Python :5000)               │        │
│ │         sentence-transformers • pgvector • embeddings        │        │
│ └──────────────────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────────────┘
            │
            ▼
┌───────────────────┐      ┌──────────────────────┐
│ Storage Layer     │      │ Volumes              │
│ /mnt/data         │      │ postgres_data        │
│ /mnt/backups      │      │ redis_data           │
└───────────────────┘      └──────────────────────┘
```

---

## 2. Service Architecture

### Active Services

| Service | Port | Technology | Purpose |
|---------|------|------------|---------|
| **postgres** | 5432 | pgvector/pg16 | Primary DB + vector storage |
| **redis** | 6379 | Redis 7 | Session cache, rate limiting |
| **api** | 8080 | Go 1.22 | REST API, auth, files |
| **webui** | 80 | Vite + TailwindCSS | Frontend UI |
| **orchestrator** | 9000 | Go | Health monitoring, metrics |
| **monitoring** | - | Go | System metrics collector |
| **analysis-agent** | - | Go | Alert analysis |
| **pentester-agent** | - | Go | Security scanning |
| **ai-knowledge-agent** | 5000 | Python 3.11 | Embeddings, semantic search |

### Docker Compose

```bash
# Production
docker compose -f infrastructure/docker-compose.prod.yml up -d

# Development
docker compose -f infrastructure/docker-compose.dev.yml up -d
```

---

## 3. Data & Control Flows

```
Users → WebUI → API Gateway (:8080)
                    │ JWT/CSRF
                    ▼
               Auth Service ──► Redis (sessions) + Postgres (users)
                    │
                    ├─► File Service ──► /mnt/data (validatePath, quota)
                    │
                    ├─► Backup Scheduler ──► /mnt/backups
                    │
                    └─► AI Search ──► AI Agent (:5000) ──► pgvector
```

### Event Flow

```
Service → Metrics → Orchestrator → Prometheus
              │
              └──► Alerts → WebUI Toast Center
```

---

## 4. Security & Governance

### Authentication
- **JWT Access Tokens** - Short-lived (15min)
- **Refresh Tokens** - Long-lived (7 days)
- **CSRF Protection** - Double-submit cookie
- **Rate Limiting** - 100 req/min per user

### Security Headers
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Content-Security-Policy: default-src 'self'
Strict-Transport-Security: max-age=31536000
```

### Audit Logging
- All auth events → `system_logs` table
- Failed attempts → Rate limit + alert
- Admin actions → Audit trail

---

## 5. Filesystem Layout

```
f1406/
├── infrastructure/           # Docker services
│   ├── api/                  # Go backend
│   ├── webui/                # Vite frontend
│   ├── ai_knowledge_agent/   # Python ML service
│   ├── monitoring/           # Metrics agent
│   ├── analysis/             # Analytics agent
│   ├── pentester/            # Security agent
│   ├── db/                   # PostgreSQL migrations
│   └── docker-compose.*.yml  # Stack definitions
│
├── orchestrator/             # Health monitoring (Go)
├── scripts/                  # CLI tools (nas-cli.sh)
├── docs/                     # Blueprints, policies
├── contrib/                  # Trivy templates
└── .archive/                 # Historical reports
```

### Docker Volumes
```
postgres_data    → /var/lib/postgresql/data
redis_data       → /data
nas_data         → /mnt/data
nas_backups      → /mnt/backups
```

---

## 6. API Contracts

### Core Endpoints

| Module | Endpoints | Auth |
|--------|-----------|------|
| **Health** | `GET /health` | Public |
| **Auth** | `POST /auth/login`, `/register`, `/refresh`, `/logout` | Public |
| **Files** | `GET/POST/PUT/DELETE /files/*` | JWT |
| **Backups** | `GET/POST /backups/*` | JWT + Admin |
| **System** | `GET /system/metrics`, `/stats` | JWT + Admin |
| **AI** | `POST /embed`, `/process` (port 5000) | Internal |

### Response Schema
```json
{
  "status": "ok|error",
  "data": { ... },
  "error": null | "Error message"
}
```

---

## 7. AI Knowledge Layer

### Components
- **Model:** sentence-transformers/all-MiniLM-L6-v2 (384 dims)
- **Storage:** PostgreSQL + pgvector extension
- **Service:** Python Flask on port 5000

### Endpoints
```bash
# Generate embedding
POST /embed {"text": "..."}
→ {"status":"ok", "data":{"embedding":[...], "dimensions":384}}

# Process file
POST /process {"file_path":"...", "file_id":"...", "mime_type":"..."}
→ {"status":"success", "embedding_dim":384}

# Health check
GET /health
→ {"status":"ok", "model_loaded":true, "db_ok":true}
```

### Corpus Generator
```bash
cd infrastructure/ai_knowledge_agent
python generate_corpus.py --count 50 --noise 0.3 --output ./output
```

---

## 8. Testing & QA

### Test Pyramid

| Level | Tool | Scope |
|-------|------|-------|
| Unit | Go test, Vitest | Functions, components |
| Integration | Cypress | Module interactions |
| E2E | Playwright | Full user flows |
| API | curl, Postman | Endpoint contracts |
| Security | Trivy, gosec | Vulnerability scans |

### CLI Testing
```bash
./scripts/nas-cli.sh
# → Menu: API Tests, Forensics, Deployment
```

---

## 9. Incident Response

```
1. DETECT   → Orchestrator health check fails
2. CONTAIN  → Service marked unhealthy
3. COLLECT  → docker logs, metrics snapshot
4. NOTIFY   → Alert to monitoring dashboard
5. FIX      → Restart, rollback, or hotfix
6. VERIFY   → Health check passes
7. DOCUMENT → Update status logs
```

### Health Endpoints
```bash
curl http://localhost:8080/health  # API
curl http://localhost:5000/health  # AI Agent
curl http://localhost:9000/health  # Orchestrator
```

---

## 10. References

| Document | Purpose |
|----------|---------|
| `README.md` | Quick start guide |
| `infrastructure/README.md` | Service documentation |
| `docs/blueprints/` | WebUI design specs |
| `CVE_CHECKLIST.md` | Security vulnerabilities |
| `API_ENDPOINTS_COMPREHENSIVE.md` | Full API reference |
| `scripts/README.md` | CLI tool documentation |

---

**Maintained by:** NAS.AI Team  
**License:** See LICENSE
