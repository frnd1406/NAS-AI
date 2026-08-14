# NAS.AI Infrastructure

**Version:** 2.1.1

---

## Services

| Service | Port | Technology | Status |
|---------|------|------------|--------|
| **postgres** | 5432 | pgvector/pg16 | ✅ |
| **redis** | 6379 | Redis 7 Alpine | ✅ |
| **api** | 8080 | Go 1.22 | ✅ |
| **webui** | 80 | Vite + TailwindCSS | ✅ |
| **monitoring** | - | Go Metrics Agent | ✅ |
| **analysis-agent** | - | Go Analytics | ✅ |
| **pentester-agent** | - | Go Security | ✅ |
| **ai-knowledge-agent** | 5000 | Python ML | ✅ |

---

## Quick Start

```bash
# Production
docker compose -f docker-compose.prod.yml up -d

# Development
docker compose -f docker-compose.dev.yml up -d
```

---

## Structure

```
infrastructure/
├── docker-compose.prod.yml   # Production stack
├── docker-compose.dev.yml    # Development stack
├── .env.prod                 # Production secrets
│
├── api/                      # Go Backend (Port 8080)
├── webui/                    # Vite Frontend (Port 80)
├── db/                       # PostgreSQL migrations
├── ai_knowledge_agent/       # Python ML Service (Port 5000)
├── monitoring/               # Metrics Agent
├── analysis/                 # Analytics Agent
├── pentester/                # Security Agent
└── scripts/                  # Deployment scripts
```

---

## Environment Variables

Create `.env.prod` with:

```bash
POSTGRES_PASSWORD=your_secure_password
JWT_SECRET=your_jwt_secret
MONITORING_TOKEN=your_monitoring_token
CORS_ORIGINS=https://your-domain.com
FRONTEND_URL=https://your-domain.com
# Encrypted Storage (optional)
ENCRYPTED_STORAGE_PATH=/media/<file-name>
```

---

## Deployment

```bash
# Full rebuild
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d

# Restart single service
docker compose -f docker-compose.prod.yml restart api

# View logs
docker compose -f docker-compose.prod.yml logs -f api
```

---

## Health Checks

| Service | Endpoint |
|---------|----------|
| API | `http://localhost:8080/health` |
| AI Agent | `http://localhost:5000/health` |
| Orchestrator | `http://localhost:9000/health` |

---

**Maintained by:** NAS.AI Team
