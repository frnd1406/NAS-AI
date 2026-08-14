# Architecture & Development Guide

Technische Referenz für **NAS.AI** — ein sicheres, selbst-gehostetes NAS mit Zero-Knowledge-Verschlüsselung und einem gehärteten Go-Backend.

**Version:** 2.1.1 · **Betriebsmodus:** Backend-only (API + PostgreSQL + Redis); weitere Dienste optional.

---

## Überblick

NAS.AI ist als Microservice-Architektur mit Docker Compose aufgebaut. Kern ist eine Go-REST-API, die Authentifizierung, Datei-Operationen und die Zero-Knowledge-Verschlüsselung kapselt. Optionale Komponenten (WebUI, AI-Wissensmodul, Orchestrator) sind entkoppelt und können einzeln zugeschaltet werden.

---

## Dienste

| Dienst | Port | Zweck | Status |
|---|---|---|---|
| `api` | 8080 | Go REST-API (Kern) | aktiv |
| `postgres` | 5432 | Primärdatenbank (`pgvector`) | aktiv |
| `redis` | 6379 | Cache, Sessions, Rate-Limiting | aktiv |
| `webui` | 80 | React-Dashboard | optional |
| `ai_knowledge_agent` | 5000 | Embeddings / RAG (Python) | **optional, derzeit deaktiviert** |
| `orchestrator` | 9000 | Health-Monitoring, Metriken | optional |

**Aktuelle Deployment-Realität:** Auf dem Zielgerät (Raspberry Pi 5) laufen bewusst nur `postgres`, `redis` und `api`. Die AI-Routen sind in der API nicht registriert (Modul deaktiviert); der Handler-Code bleibt für eine spätere Reaktivierung erhalten.

---

## Backend-Architektur (Go)

Geschichtete Architektur mit klarer Trennung der Verantwortlichkeiten:

```
Handler-Layer  (handlers/)   → HTTP: Validierung, Request/Response
      ↓
Service-Layer  (services/)   → Geschäftslogik, Verschlüsselung, Datei-Ops
      ↓
Repository     (repository/) → Datenbankzugriff, Queries
      ↓
PostgreSQL + Redis
```

- **Handlers** (`handlers/*.go`) — HTTP-Verarbeitung, Validierung, Response-Formatierung
- **Services** (`services/*.go`) — Geschäftslogik, Verschlüsselung, Datei-Operationen, E-Mail
- **Repository** (`repository/*.go`) — Datenbank-Operationen, SQL
- **Middleware** (`middleware/*.go`) — Auth, CSRF, Rate-Limiting, Vault-Guard

---

## Sicherheits-Architektur

### Zero-Knowledge-Verschlüsselung
- **Cipher:** XChaCha20-Poly1305 (AEAD, `golang.org/x/crypto/chacha20poly1305`)
- **KDF:** Argon2id (`golang.org/x/crypto/argon2`)
- **Schlüsselhierarchie:** Master-Passwort → KEK → DEK → Datei-/DB-/Backup-Schlüssel
- **Zustand:** System startet **LOCKED**; Entsperren per Master-Passwort. Der DEK liegt ausschließlich im RAM (`services/security/encryption_service.go`) und muss nach jedem Neustart neu entsperrt werden.
- **Datenklasse:** Trennung öffentlich vs. Vault; Vault-Daten sind für Automatisierung/Agenten grundsätzlich tabu.

### Transport & Betrieb
- **Private Root-CA** + serverseitiges Zertifikat; produktives TLS (`ENV=production`). Clients validieren gegen die eingebettete CA.
- **API-Container entprivilegiert:** läuft als `nobody`, `no-new-privileges`, `cap_drop: ALL`.
- **Netzwerk:** Binding nur auf LAN-IPv4 (`BIND_ADDR`), kein Port-Forwarding, mDNS-Advertising im LAN.
- **Auth:** JWT (Access 15 min / Refresh 7 Tage), CSRF (Double-Submit-Cookie), Rate-Limiting (Redis).
- **Path-Traversal-Schutz:** alle Datei-Operationen validiert.
- **Host-Härtung:** Firewall-Whitelist, SSH key-only, automatische Sicherheitsupdates. Patch-Steuerung siehe [`docs/SPEICHER-ARCHITEKTUR.md`](docs/SPEICHER-ARCHITEKTUR.md) §12.

---

## Storage

Speicherpfade sind konfigurierbar (Rollen statt fester Geräte). Zielbild: Tiers **HOT** (SSD/NVMe), **BULK** (HDD-RAID), **VAULT** (isoliert), **TRANSFER**. Design & Backlog: [`docs/SPEICHER-ARCHITEKTUR.md`](docs/SPEICHER-ARCHITEKTUR.md).

Relevante Env-Variablen: `BACKUP_STORAGE_PATH`, `ENCRYPTED_STORAGE_PATH`, `ALLOWED_STORAGE_ROOTS`.

---

## AI-Wissensmodul (optional, derzeit deaktiviert)

- **Embeddings/RAG:** lokal (kein Cloud-Abruf), Vektorsuche über `pgvector`.
- **Sicheres Einspeisen:** der `SecureAIFeeder` gibt entschlüsselte Inhalte an das Modul, ohne Klartext auf die Platte zu schreiben.
- **Status:** Routen sind aktuell nicht registriert (`handlers/ai/routes.go`); Reaktivierung über die Git-History bzw. ein Feature-Flag.

---

## Entwicklung

### API (Go)
```bash
cd infrastructure/api
make build            # Build
make test             # Tests
make test-coverage    # Coverage (Minimum 80 %)
make security-scan    # gosec + gitleaks
make lint             # Linter
# Run benötigt JWT_SECRET:
export JWT_SECRET=$(openssl rand -base64 32) && make run
```

### WebUI (React + Vite)
```bash
cd infrastructure/webui
npm install && npm run dev      # Dev-Server
npm run build                   # Production-Build
```

### Deploy (Backend-only, Zielgerät)
```bash
cd infrastructure
docker compose -f docker-compose.pi.yml --env-file .env.pi up -d postgres redis api
docker compose -f docker-compose.pi.yml --env-file .env.pi logs -f api
```

---

## Konventionen

### Fehlerbehandlung (Go)
```go
if err != nil {
    logger.WithError(err).Error("Failed to process request")
    c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "Internal server error"})
    return
}
```

### API-Response-Format
```json
{ "status": "ok|error", "data": { }, "error": null }
```

### Auth-Flow
1. Login → JWT Access-Token + Refresh-Token
2. Access-Token im Speicher, Refresh-Token als httpOnly-Cookie
3. Geschützte Routen: `Authorization: Bearer <token>`
4. State-ändernde Requests zusätzlich: `X-CSRF-Token`
5. Refresh-Token erneuert abgelaufene Access-Tokens

---

## Env-Variablen

**Erforderlich (API):** `JWT_SECRET`, `POSTGRES_PASSWORD`, `CSRF_SECRET`

**Optional:** `ENCRYPTED_STORAGE_PATH`, `BACKUP_STORAGE_PATH`, `ALLOWED_STORAGE_ROOTS`, `BIND_ADDR`, `TLS_CERT_FILE`, `TLS_KEY_FILE`, `CORS_ORIGINS`, `FRONTEND_URL`, `AI_SERVICE_URL`

Alle Secrets werden beim Start geladen (fail-fast) und **nie** eingecheckt (siehe `.gitignore` / `.env.template`).

---

## Verzeichnis-Referenz

- `infrastructure/api/` — Go-Backend (Kern)
- `infrastructure/webui/` — React-Frontend (optional)
- `infrastructure/ai_knowledge_agent/` — Python-AI-Modul (optional, deaktiviert)
- `infrastructure/pi-updater/` — YubiKey-gated Host-Patching (Ops)
- `orchestrator/` — Health-Monitoring (optional)
- `docs/` — Architektur-, Security- und Planungsdokumente
- `infrastructure/db/` — PostgreSQL-Migrations & Schema
