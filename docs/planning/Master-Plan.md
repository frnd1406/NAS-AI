

 MASTERPLAN: Hybrid Encryption & Smart Sharing (Pi 5
## Optimized)
Status: DRAFT Author: User & Senior Architect Date: 2025-12-25
## 1. Vision
Das NAS soll dem Nutzer die Souveränität über seine Daten geben. Statt einer "One-Size-Fits-
All"-Verschlüsselung, bieten wir ein hybrides Modell an. Der Nutzer entscheidet pro Datei
zwischen maximaler Sicherheit (User-Encryption) und maximaler Performance (Raw/None),
unterstützt durch intelligente System-Warnungen ("Performance Guard").
- Architektur-Prinzipien (Raspberry Pi 5 Constraints)
- Verschlüsselung:XChaCha20-Poly1305 mit Argon2id (Hardcoded Parameters für
## Portabilität).
- Streaming: 64KB Chunking-Strategie für minimalen RAM-Verbrauch (<5MB Overhead).
- Portabilität: Festplatten müssen zwischen Pi und High-End Servern austauschbar sein.
- Datenbank: PostgreSQL als "Source of Truth" für Metadaten und Key-Material.
- Phasen-Planung
 PHASE 1: Das Datenbank-Fundament (Backend)
Ziel: Die DB muss Verschlüsselungs-Modi verstehen und flexible Keys speichern.
[ ] Migration 005_hybrid_encryption.sql erstellen.
ENUM Typen: NONE, SYSTEM, USER.
Tabelle files erweitern: encryption_status, size_bytes, checksum.
Tabelle shares erstellen: Mit JSONB für flexible Key-Speicherung.
[ ] Go-Models (structs) anpassen (models/file.go, models/share.go).
⏱ PHASE 2: Der "Performance Guard" (Backend)
Ziel: System kennt seine eigene Geschwindigkeit.
[ ] BenchmarkService implementieren (Start-Up Speedtest: RAM -> Crypto -> RAM).
[ ] Globalen State SystemMBps halten.
[ ] API-Endpunkt /api/system/capabilities bereitstellen (Input: FileSize -> Output: Est.
## Time).
⚙ PHASE 3: Die Upload-Pipeline (Hybrid Logic)
Ziel: Weiche für Datenströme.
25.12.25, 17:47Gemini
https://gemini.google.com/gem/6fc0c5ffba13/78f788a963a0dca41/2

[ ] UploadHandler refactoren.
Parameter encryption_mode auslesen.
IF NONE:io.Copy direkt auf Disk.
IF USER: Stream durch NasCrypt V2 Service leiten.
[ ] Metadaten korrekt in DB schreiben.
 PHASE 4: Sharing & Download (Smart Streaming)
## Ziel: Sicheres Ausliefern.
[ ] DownloadHandler refactoren.
IF NONE: Nginx X-Accel-Redirect (Zero CPU).
IF USER: On-the-fly Entschlüsselung (High CPU).
## [ ] Public Link Logik:
Key-Unwrapping (DB -> Entschlüsseln -> Neu verpacken für Share).
 PHASE 5: Frontend UX
Ziel: Transparenz für den User.
[ ] Upload-Dialog: Toggle " Encrypt".
[ ] Live-Warnung bei großen Dateien auf langsamer Hardware (via capabilities API).
[ ] Icon-Indikatoren in der Dateiliste.
## 4. Technisches Design: Datenbank
## -- Core Types
CREATE TYPE encryption_mode AS ENUM ('NONE', 'SYSTEM', 'USER');
## -- Files Extension
ALTER TABLE files ADD COLUMN encryption_status encryption_mode DEFAULT 'NONE';
ALTER TABLE files ADD COLUMN size_bytes BIGINT DEFAULT 0;
ALTER TABLE files ADD COLUMN checksum VARCHAR(128);
-- Shares (New Table)
CREATE TABLE shares (
id UUID PRIMARY KEY,
token VARCHAR(64) UNIQUE, -- The URL part
encrypted_key_material JSONB, -- The flexible crypto blob
-- ... (ownership, expiry)
## );
25.12.25, 17:47Gemini
https://gemini.google.com/gem/6fc0c5ffba13/78f788a963a0dca42/2