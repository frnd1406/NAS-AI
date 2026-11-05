# 🚀 NAS.AI Server Installation Guide

## Voraussetzungen

- Docker & Docker Compose installiert
- Root- oder sudo-Zugriff
- Domain mit DNS-Einträgen (für Produktion)
- Min. 4GB RAM, 20GB Festplatte

## 📦 Benötigte Dateien

### Minimale Installation:
```
infrastructure/
├── .env.prod                    ← Erstellen (siehe Schritt 1)
├── docker-compose.prod.yml
├── api/
├── ai_knowledge_agent/
├── monitoring/
├── analysis/
└── pentester/
```

## 🔧 Installations-Schritte

### Schritt 1: Environment-Datei erstellen

```bash
# In infrastructure/ Verzeichnis
cp .env.prod.template .env.prod
```

### Schritt 2: Secrets generieren

```bash
# PostgreSQL Passwort generieren
echo "POSTGRES_PASSWORD=$(openssl rand -base64 32)"

# JWT Secret generieren
echo "JWT_SECRET=$(openssl rand -base64 64)"

# Monitoring Token generieren
echo "MONITORING_TOKEN=$(openssl rand -base64 32)"
```

**Diese Werte in .env.prod eintragen!**

### Schritt 3: Domain-Konfiguration anpassen

In `.env.prod` bearbeiten:
```bash
DOMAIN=ihre-domain.com
API_DOMAIN=api.ihre-domain.com
CORS_ORIGINS=https://ihre-domain.com,https://api.ihre-domain.com
FRONTEND_URL=https://ihre-domain.com
EMAIL_FROM=noreply@ihre-domain.com
```

### Schritt 4: System starten

```bash
# Alle Images bauen
docker compose -f docker-compose.prod.yml build

# Services starten
docker compose -f docker-compose.prod.yml up -d

# Logs überprüfen
docker compose -f docker-compose.prod.yml logs -f
```

### Schritt 5: Gesundheits-Check

```bash
# API Health Check
curl http://localhost:8080/health

# AI Agent Health Check
curl http://localhost:5000/health

# Sollte jeweils {"status": "ok"} zurückgeben
```

## 🔍 Service-Übersicht

| Service | Port | Beschreibung |
|---------|------|--------------|
| api | 8080 | Backend API (intern) |
| ai-knowledge-agent | 5000 | Semantic Search |
| postgres | 5432 | Datenbank (intern) |
| redis | 6379 | Cache (intern) |
| monitoring | - | System-Metriken |
| analysis | - | Log-Analyse |
| pentester | - | Security-Tests |

## 🔐 Sicherheits-Checkliste

- [ ] Alle `CHANGE_ME` Werte in .env.prod ersetzt
- [ ] Starke Passwörter verwendet (min. 32 Zeichen)
- [ ] .env.prod Dateiberechtigungen: `chmod 600 .env.prod`
- [ ] .env.prod NICHT in Git committen
- [ ] Firewall konfiguriert (nur Port 8080/443 öffentlich)
- [ ] SSL-Zertifikate installiert (Let's Encrypt empfohlen)

## 🆘 Troubleshooting

### Container startet nicht:
```bash
# Detaillierte Logs anzeigen
docker compose -f docker-compose.prod.yml logs <service-name>

# Container-Status prüfen
docker compose -f docker-compose.prod.yml ps
```

### Datenbank-Verbindung fehlgeschlagen:
```bash
# PostgreSQL-Logs prüfen
docker logs nas-api-postgres

# Passwort in .env.prod überprüfen
grep POSTGRES_PASSWORD .env.prod
```

### AI Agent lädt Model nicht:
```bash
# AI Agent Logs prüfen
docker logs nas-ai-knowledge-agent

# Mehr Speicher zuweisen (in docker-compose.prod.yml):
# memory: 8g  # statt 4g
```

## 📚 Nächste Schritte

1. **Nginx Reverse Proxy** einrichten (für HTTPS)
2. **Backup-Strategie** konfigurieren
3. **Monitoring** einrichten (Prometheus/Grafana)
4. **Ersten Admin-User** erstellen

## 🔄 Updates durchführen

```bash
# Code aktualisieren (Git Pull)
git pull origin main

# Services neu bauen
docker compose -f docker-compose.prod.yml build

# Rolling Update (ohne Downtime)
docker compose -f docker-compose.prod.yml up -d --no-deps --build <service-name>
```

## 📞 Support

- GitHub Issues: https://github.com/your-repo/issues
- Dokumentation: https://docs.your-domain.com
