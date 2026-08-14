# Service Orchestrator

Überwacht die Gesundheit und Verfügbarkeit der NAS AI Dienste.

## 🎯 Purpose

Gewährleistung der Systemstabilität durch kontinuierliches Monitoring der Core-Container (API, Datenbank, AI-Agent).

## ⚙️ Mechanism

*   **Polling**: Führt regelmäßige Checks via HTTP oder TCP durch.
*   **Self-Healing**: Startet Dienste bei Fehlfunktion automatisch neu oder alarmiert.

## 📝 Configuration

Die Konfiguration erfolgt über `registry.json`.

```json
{
  "services": [
    { "name": "api", "url": "http://api:8080/health", "critical": true },
    { "name": "db", "check": "tcp:5432", "critical": true }
  ]
}
```
