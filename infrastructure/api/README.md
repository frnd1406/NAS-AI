# NAS AI Backend API



## 🏛 Core Philosophy

**"Thin Handlers, Fat Services"**

*   **Handlers**: Zuständig für HTTP-Parsing, Validierung und Response-Mapping.
*   **Services**: Enthalten die eigentliche Logik, Sicherheits-Checks und Datenverarbeitung.

## 📦 Service Catalog

*   **[ArchiveService](./src/services/archive_service.go)**
    *   Sicherer Upload und Download von Archiven.
    *   Schutz gegen Zip-Slip und Zip-Bomb Angriffe.

*   **[ContentDeliveryService](./src/services/content_delivery_service.go)**
    *   Streaming von Multimedia-Inhalten.
    *   Behandelt Range-Requests (auch für verschlüsselte Dateien).
    *   On-the-fly Entschlüsselung.

*   **[AIAgentService](./src/services/ai_agent_service.go)**
    *   Gateway zum Python-basierten AI Knowledge Agent.

*   **[EncryptionService](./src/services/encryption_service.go)**
    *   Verwaltet Master-Keys und Datei-Verschlüsselung (XChaCha20-Poly1305).

## 🛠 Development

### Setup

```bash
# Kopiere Beispiel-Konfiguration
cp .env.example .env
```

### Commands

```bash
make run      # Startet den Server lokal
make test     # Führt Unittests aus
make swagger  # Aktualisiert API-Dokumentation
```

## Host events (UFW / SMB)

`POST /api/v1/system/events` lets host-side monitors report security events. It is protected like `/system/metrics` by the `X-Monitoring-Token` header. Payload: `{"source","type","src_ip","count","window_seconds"}` with `type` in `ufw_block | smb_auth_fail` (IP is parsed, `count` 1..100000, `window_seconds` 1..3600). Each event becomes a `system_alerts` row (`ufw_block` = WARNING, `smb_auth_fail` with count >= 3 = HIGH); the same source/type/IP within 5 minutes is merged into one alert (never downgraded). Every create/update is published to the Redis channel `system_alerts`. Clients read alerts with `GET /api/v1/system/alerts` (JWT).
