# NAS.AI Developer Guide

**Version:** 2.2  
**Updated:** 2026-01-01

## ⚠️ WICHTIG: Architectural Rules

Dies sind die unverhandelbaren Regeln für die Entwicklung in diesem Repository.

### SRP (Single Responsibility Principle)
Handler (`src/handlers/`) sind **nur** für HTTP-Parsing, Validierung und Response-Konstruktion zuständig. Jegliche Business-Logik **muss** in Services (`src/services/`) ausgelagert werden.

### No External Calls in Handlers
Rufe niemals externe APIs (AI, S3, externe Services) direkt im Handler auf. Nutze immer einen Service Wrapper (z.B. `AIAgentService`).

### Security First
*   **Path Traversal**: Alle Pfade müssen **im Service** validiert und gesäubert werden, bevor auf das Dateisystem zugegriffen wird.
*   **Zip Bombs**: Archiventpackung muss Größen- und Raten-Limits durchsetzen (siehe `ArchiveService`).

### Testing
Services müssen so designed sein, dass sie zu 100% unit-testbar sind. Injiziere Abhängigkeiten (Dependency Injection), um Mocking zu ermöglichen.

---

## 1. Setup & Umgebung

### Voraussetzungen
- Go 1.22+
- Node.js 20+
- Docker & Docker Compose

### Quick Start

```bash
# 1. Repository klonen
git clone <repo-url>
cd f1406

# 2. Environment konfigurieren
cp infrastructure/.env.example infrastructure/.env.prod
# Secrets in .env.prod eintragen!

# 3. Dev-Infrastruktur starten
cd infrastructure
docker compose -f docker-compose.dev.yml up -d

# 4. Backend starten
cd api && go run ./src/main.go

# 5. Frontend starten (neues Terminal)
cd webui && npm install && npm run dev
```

## 2. Code-Konventionen

| Bereich | Standard |
|---------|----------|
| **Sprache** | Englisch (Kommentare, Variablen, Commits) |
| **Go** | `gofmt`, Error-Handling, Context-Usage |
| **React** | Functional Components, Hooks |
| **Config** | Keine Hardcoded Werte! `.env` oder Config-Structs |

## 3. Contributing Flow

Kurzfassung für alle Mitwirkenden (Menschen und KI-Agenten): [`CONTRIBUTING.md`](../../CONTRIBUTING.md).

1. Branch erstellen: `feature/<description>` (auch `fix/`, `docs/`, `chore/`, `ci/`, `refactor/`, `test/`, `perf/`, `hotfix/`, `release/`)
2. Implementieren (Tests schreiben)
3. Lokale Tests: `go test ./...` im betroffenen Go-Modul (WebUI ist derzeit pausiert und nicht in der CI)
4. Security Scan: `make security-scan` in `infrastructure/api`
5. Pull Request gegen `main`

### Regeln auf GitHub (automatisch geprüft)

| Regel | Wo definiert |
|-------|--------------|
| Kein direkter Push auf `main`, kein Force-Push, kein Löschen | `.github/rulesets/main.json` |
| Merge nur per PR; Review durch Code Owner (`@frnd1406`) nötig, nur der Admin darf ohne Fremd-Review mergen | `.github/rulesets/main.json`, `.github/CODEOWNERS` |
| Pflicht-Checks: `CI` (gofmt, vet, Tests, Lint neuer Befunde), `Rule check`, `Secrets (gitleaks)`, `Go security (gosec)`, `Dependency review`, `Analyze (…)` | `.github/rulesets/main.json` |
| Labels nach Bereich und Größe, Warnung bei Überschneidung mit anderen offenen PRs | `.github/workflows/pr-hygiene.yml` |
| Inaktive PRs: nach 14 Tagen `stale`, nach 21 Tagen geschlossen | `.github/workflows/stale.yml` |
| Release-Tags `v*` sind unveränderlich | `.github/rulesets/tags.json` |
| Commit-Autor = GitHub-noreply-Adresse, Conventional Commits (Englisch), keine `Co-Authored-By`-Zeilen, Branch-Namen wie oben | `.github/workflows/pr-rules.yml` |
| Repo-Flags (Merge-Optionen, Secret Scanning, Push Protection, Dependabot) | `.github/repo-settings.json` |

Der Workflow `repo-settings.yml` überträgt Flags und Rulesets bei Änderungen auf `main` und wöchentlich auf GitHub. Dafür braucht es das Secret `REPO_ADMIN_TOKEN` (Fine-grained PAT, nur dieses Repo, „Administration: Read and write“).

Lokale Git-Identität für Commits:

```bash
git config user.name  "Felix Freund"
git config user.email "226513012+frnd1406@users.noreply.github.com"
```

## 4. Secrets Management

> ⚠️ **WICHTIG:** Alle Secrets gehören in `.env.prod` oder einen Secret Manager!

```bash
# Secrets NIEMALS im Code!
# Richtig:
JWT_SECRET=${JWT_SECRET}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}

# Falsch:
JWT_SECRET="hardcoded-secret-123"
```

Für API-Tokens (Cloudflare, Resend, etc.):
- Development: `.env.prod`
- Production: Secret Manager / Vault

## 5. Troubleshooting

| Problem | Lösung |
|---------|--------|
| Port Konflikt | `lsof -i :8080` → Prozess killen |
| DB Connection | `docker compose ps` → postgres prüfen |
| Permission Denied | Schreibrechte in `/mnt/data` prüfen |

## 6. Nützliche Commands

```bash
# Alle Services starten
./scripts/nas-cli.sh

# Logs anzeigen
docker compose -f docker-compose.prod.yml logs -f api

# API testen
curl http://localhost:8080/health
```