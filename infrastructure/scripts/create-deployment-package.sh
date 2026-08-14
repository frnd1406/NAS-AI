#!/bin/bash
# ============================================================
# Deployment Package Creator
# ============================================================
# Erstellt ein ZIP-Archiv mit allen benötigten Dateien
# für die Installation auf einem neuen Server
# ============================================================

set -e

# Farben
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}============================================================${NC}"
echo -e "${BLUE}📦 NAS.AI Deployment Package Creator${NC}"
echo -e "${BLUE}============================================================${NC}"
echo ""

# Ins infrastructure Verzeichnis wechseln
cd "$(dirname "$0")/.."
INFRASTRUCTURE_DIR=$(pwd)

# Package Name mit Timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
PACKAGE_NAME="nas-ai-deployment-${TIMESTAMP}"
OUTPUT_DIR="/tmp/${PACKAGE_NAME}"
ARCHIVE_NAME="${PACKAGE_NAME}.tar.gz"

echo -e "${YELLOW}Erstelle Package: ${PACKAGE_NAME}${NC}"
echo ""

# Temporäres Verzeichnis erstellen
mkdir -p "${OUTPUT_DIR}"

echo "📋 Kopiere Dateien..."

# PFLICHT-Dateien kopieren
echo "  ✓ docker-compose.prod.yml"
cp docker-compose.prod.yml "${OUTPUT_DIR}/"

echo "  ✓ .env.prod (mit aktuellen Werten)"
cp .env.prod "${OUTPUT_DIR}/"

echo "  ✓ API Service"
mkdir -p "${OUTPUT_DIR}/api"
rsync -a --exclude='tmp' --exclude='*.log' api/ "${OUTPUT_DIR}/api/"

echo "  ✓ WebUI Service"
mkdir -p "${OUTPUT_DIR}/webui"
rsync -a --exclude='node_modules' --exclude='dist' --exclude='build' webui/ "${OUTPUT_DIR}/webui/"

echo "  ✓ AI Knowledge Agent"
mkdir -p "${OUTPUT_DIR}/ai_knowledge_agent"
rsync -a --exclude='__pycache__' --exclude='*.pyc' --exclude='.pytest_cache' ai_knowledge_agent/ "${OUTPUT_DIR}/ai_knowledge_agent/"

echo "  ✓ Monitoring Agent"
mkdir -p "${OUTPUT_DIR}/monitoring"
rsync -a --exclude='__pycache__' --exclude='*.pyc' monitoring/ "${OUTPUT_DIR}/monitoring/"

echo "  ✓ Analysis Agent"
mkdir -p "${OUTPUT_DIR}/analysis"
rsync -a --exclude='__pycache__' --exclude='*.pyc' analysis/ "${OUTPUT_DIR}/analysis/"

echo "  ✓ Pentester Agent"
mkdir -p "${OUTPUT_DIR}/pentester"
rsync -a --exclude='__pycache__' --exclude='*.pyc' pentester/ "${OUTPUT_DIR}/pentester/"

echo "  ✓ Scripts"
mkdir -p "${OUTPUT_DIR}/scripts"
cp -r scripts/* "${OUTPUT_DIR}/scripts/" 2>/dev/null || true

echo "  ✓ Dokumentation"
cp INSTALLATION.md "${OUTPUT_DIR}/" 2>/dev/null || true
cp DEPLOYMENT-CHECKLIST.md "${OUTPUT_DIR}/" 2>/dev/null || true
cp .env.prod.template "${OUTPUT_DIR}/" 2>/dev/null || true

# README für das Deployment erstellen
cat > "${OUTPUT_DIR}/DEPLOY-README.md" << 'EOFREADME'
# 🚀 NAS.AI Deployment Package

## Schnellstart auf neuem Server

### 1. Archiv entpacken
```bash
tar -xzf nas-ai-deployment-*.tar.gz
cd nas-ai-deployment-*
```

### 2. .env.prod anpassen
```bash
# Wichtig: Prüfe und passe die Werte an!
nano .env.prod

# Mindestens folgendes anpassen:
# - DOMAIN=deine-neue-domain.com
# - CORS_ORIGINS=https://deine-neue-domain.com
# - FRONTEND_URL=https://deine-neue-domain.com
```

### 3. Docker Compose starten
```bash
# Alle Services bauen
docker compose -f docker-compose.prod.yml build

# Services starten
docker compose -f docker-compose.prod.yml up -d

# Logs prüfen
docker compose -f docker-compose.prod.yml logs -f
```

### 4. Health Check
```bash
# API prüfen
curl http://localhost:8080/health

# AI Agent prüfen
curl http://localhost:5000/health
```

## ⚠️ Wichtige Hinweise

1. **Secrets**: Die .env.prod Datei enthält bereits Secrets vom Quellsystem.
   - Für Produktion: NEUE Secrets generieren!
   - Befehl: `./scripts/generate-secrets.sh`

2. **Domain**: Domain-Einstellungen müssen angepasst werden

3. **SSL**: Nginx Reverse Proxy mit Let's Encrypt für HTTPS einrichten

4. **Firewall**: Nur Port 80/443 öffentlich öffnen

## 📊 System-Anforderungen

- CPU: 2+ Cores (empfohlen: 4 Cores)
- RAM: 4 GB Minimum (empfohlen: 8 GB)
- Disk: 20 GB SSD
- OS: Ubuntu 20.04+ / Debian 11+
- Docker: 20.10+
- Docker Compose: 2.0+

## 📞 Support

Siehe INSTALLATION.md für detaillierte Anleitung.
EOFREADME

echo ""
echo -e "${GREEN}✅ Dateien kopiert${NC}"
echo ""

# Archiv erstellen
echo "📦 Erstelle Archiv..."
cd /tmp
tar -czf "${ARCHIVE_NAME}" "${PACKAGE_NAME}/"

# Größe anzeigen
ARCHIVE_SIZE=$(du -h "${ARCHIVE_NAME}" | cut -f1)

echo ""
echo -e "${GREEN}============================================================${NC}"
echo -e "${GREEN}✅ Deployment Package erfolgreich erstellt!${NC}"
echo -e "${GREEN}============================================================${NC}"
echo ""
echo -e "Archiv: ${BLUE}/tmp/${ARCHIVE_NAME}${NC}"
echo -e "Größe:  ${BLUE}${ARCHIVE_SIZE}${NC}"
echo ""
echo -e "${YELLOW}Nächste Schritte:${NC}"
echo "1. Archiv auf Zielserver kopieren:"
echo -e "   ${BLUE}scp /tmp/${ARCHIVE_NAME} user@server:/opt/${NC}"
echo ""
echo "2. Auf Zielserver entpacken:"
echo -e "   ${BLUE}tar -xzf ${ARCHIVE_NAME}${NC}"
echo ""
echo "3. Deployment starten:"
echo -e "   ${BLUE}cd ${PACKAGE_NAME}${NC}"
echo -e "   ${BLUE}docker compose -f docker-compose.prod.yml up -d${NC}"
echo ""

# Cleanup
rm -rf "${OUTPUT_DIR}"
