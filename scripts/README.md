# NAS.AI Scripts - Übersicht

Alle relevanten CLI-Funktionen sind jetzt im monolithischen `nas-cli.sh` gebündelt. Einzelne Altskripte wurden entfernt. Dieses README beschreibt die aktuellen Tools.

## 🚀 Haupt-CLI: `scripts/nas-cli.sh`

### Features (Menü)
- 🔍 API Health Check (single)
- 📡 API Monitoring Loop
- 🧪 Endpoint Tests (inkl. Auth-Tests, wenn Tokens gesetzt sind)
- 📚 API Docs generieren
- 💾 Git Savepoint (add/commit/push)
- 📜 Docker Logs (optional `--no-log-prefix`)
- ✅ API Health Check (erweitert, mehr Endpoints)
- 🐳 Docker Clean Rebuild
- 🚀 Deploy Prod (mit Smart-Waits für DB/API)
- 🔐 Login & Tokens setzen (fragt E-Mail/Passwort ab, setzt JWT/CSRF)

### Nutzung
```bash
cd NasServer
./scripts/nas-cli.sh
```

Optional: Basis-URL überschreiben
```bash
API_URL=https://dein-host ./scripts/nas-cli.sh
```

### Login & Tokens (für Auth-Tests)
- Im Menü `L` wählen, E-Mail/Passwort eingeben.
- Tokens werden gesetzt (`JWT_TOKEN`, `CSRF_TOKEN`) und in derselben Session für Endpoint-Tests genutzt.

### Farben & Status
- 200/erwartete Codes: grün ✅ mit kurzer Erklärung
- Abweichungen: rot ❌ mit Status-Erklärung und Response-Auszug

## 🔗 Weitere Skripte

### `scripts/add-api-endpoint.sh`
Interaktiver Generator für neue API-Endpunkte (Handler-Gerüst + Hinweise zur Routen-Registrierung).

### `scripts/scp-send.sh`
SCP-Helper für Datei-Transfers.
```bash
./scripts/scp-send.sh <local_file> [user] [host] [remote_path]
# Default: user=user, host=192.168.1.181, remote=/home/user/
```

### Quick Reference
- API Docs: `API_ENDPOINTS.md` (kann über `nas-cli` regeneriert werden)
- Schnellreferenz: `scripts/API_QUICK_REFERENCE.txt` (statisch)

## 🗑️ Entfernte/ersetzte Skripte
Die folgenden Altskripte wurden in `nas-cli.sh` integriert und gelöscht:
- `api-health-monitor.sh`, `api-health-check.sh`
- `test-api-endpoints.sh`
- `generate-api-docs.sh`
- `git_savepoint.sh`
- `deploy-prod.sh`
- `docker-rebuild.sh`

Nutze stattdessen das Menü von `nas-cli.sh`.

# 5. Production deployen
cd NasServer/infrastructure
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d api

# 6. Endpoint testen
cd NasServer/scripts
./test-api-endpoints.sh

# 7. Dokumentation aktualisieren
./generate-api-docs.sh
```

---

### Production-Probleme debuggen

```bash
# 1. Container-Status prüfen
docker compose --env-file NasServer/infrastructure/.env.prod \
  -f NasServer/infrastructure/docker-compose.prod.yml ps

# 2. API-Logs ansehen
docker compose --env-file NasServer/infrastructure/.env.prod \
  -f NasServer/infrastructure/docker-compose.prod.yml logs --tail=50 api

# 3. Endpoints testen
VERBOSE=true ./scripts/test-api-endpoints.sh

# 4. Neustart mit Validierung
./infrastructure/scripts/restart-prod.sh
```

---

### API vollständig neu deployen

```bash
# Mit Datenbank-Reset
./scripts/deploy-prod.sh
# Antworte mit 'y' bei "Datenbank KOMPLETT löschen?"

# Ohne Datenbank-Reset (Update Mode)
./scripts/deploy-prod.sh
# Antworte mit 'n' bei "Datenbank KOMPLETT löschen?"
```

---

## 🛠️ Weitere nützliche Befehle

### API Image neu bauen
```bash
cd NasServer/infrastructure/api
docker build --no-cache -t nas-api:1.0.0 .
```

### WebUI Image neu bauen
```bash
cd NasServer/infrastructure
docker build \
  --build-arg VITE_API_BASE_URL="https://api.example.com" \
  -t nas-webui:1.0.0 \
  webui
```

### Container-Logs live ansehen
```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml logs -f api
```

### Backup-Permissions fixen
```bash
sudo chmod 777 /var/lib/docker/volumes/infrastructure_nas_backups/_data
sudo chmod 777 /var/lib/docker/volumes/infrastructure_nas_data/_data
```

### Alle Container stoppen und Volumes löschen
```bash
cd NasServer/infrastructure
docker compose --env-file .env.prod -f docker-compose.prod.yml down -v
```

---

## 📊 Script Overview

| Script | Zweck | Interaktiv | Dauer |
|--------|-------|------------|-------|
| `deploy-prod.sh` | Full Production Deployment | Ja | 2-5 Min |
| `restart-prod.sh` | Quick Restart | Nein | 10-20 Sek |
| `test-api-endpoints.sh` | API Testing | Nein | 5-10 Sek |
| `add-api-endpoint.sh` | Endpoint Generator | Ja | 1-2 Min |
| `generate-api-docs.sh` | Documentation | Nein | < 1 Sek |

---

## 🚨 Troubleshooting

### "Permission Denied" bei Backups
```bash
sudo chmod 777 /var/lib/docker/volumes/infrastructure_nas_backups/_data
```

### API gibt 404 für neue Endpoints
```bash
# API wurde nicht neu gebaut!
cd NasServer/infrastructure/api
docker build --no-cache -t nas-api:1.0.0 .
docker compose --env-file ../infrastructure/.env.prod -f ../infrastructure/docker-compose.prod.yml up -d api
```

### Container startet nicht
```bash
# Logs ansehen
docker compose --env-file .env.prod -f docker-compose.prod.yml logs api

# Health Check
docker compose --env-file .env.prod -f docker-compose.prod.yml exec api wget -q --spider http://localhost:8080/health
```

### WebUI zeigt alte Version
```bash
# Browser-Cache leeren oder Hard Reload:
# Chrome/Firefox: Ctrl+Shift+R
# Safari: Cmd+Option+R
```

---

## 📝 Best Practices

1. **Immer testen nach Änderungen:**
   ```bash
   ./scripts/test-api-endpoints.sh
   ```

2. **Dokumentation aktualisieren:**
   ```bash
   ./scripts/generate-api-docs.sh
   ```

3. **Logs bei Problemen:**
   ```bash
   docker compose -f docker-compose.prod.yml logs --tail=100 api
   ```

4. **Backups vor großen Änderungen:**
   ```bash
   # Via WebUI oder API
   curl -X POST https://example.com/api/v1/backups \
     -H "Authorization: Bearer $JWT_TOKEN" \
     -H "X-CSRF-Token: $CSRF_TOKEN"
   ```

---

## 🔗 Weiterführende Links

- API Dokumentation: `NasServer/API_ENDPOINTS.md`
- Quick Reference: `NasServer/scripts/API_QUICK_REFERENCE.txt`
- Docker Compose Config: `NasServer/infrastructure/docker-compose.prod.yml`
- API Source: `NasServer/infrastructure/api/`
- WebUI Source: `NasServer/infrastructure/webui/`

---

**Erstellt:** 2025-11-27
**Version:** 1.0.0
