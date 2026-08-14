

## Context:
infrastructure/api/src/handlers/file_content.go
infrastructure/api/src/services/secure_ai_feeder.go
infrastructure/api/src/services/encryption_service.go
infrastructure/api/src/models/settings.go
AUFTRAG: Phase 3 (REVISED) - "Smart Defaults & User Freedom"
Du bist Senior Backend Architect. Wir bauen die Upload- und AI-Logik. Wichtigste
Anforderung: Das System hat intelligente Defaults ("Policies"), aber der User hat IMMER das
letzte Wort ("Override"). Das Produkt muss auf High-End Servern und Raspberry Pis skalieren.
TEIL 1: Hybrid Upload Pipeline (Mit User-Override)
- Update UploadHandler (file_content.go):
Lese den Parameter encryption_mode aus dem Request (Form-Data).
Werte: "AUTO", "FORCE_USER", "FORCE_NONE".
Injeziere den EncryptionPolicyService.
- Service EncryptionPolicyService:
Methode: DetermineMode(filename string, size int64, userOverride string)
EncryptionMode
Logik-Ablauf:
## 1. Check Override:
IF userOverride == "FORCE_USER" -> RETURN USER (Kunde ist König).
IF userOverride == "FORCE_NONE" -> RETURN NONE.
- Check Hardware Limit (Nur im AUTO Modus):
IF size > PolicyMaxEncryptSizeBytes -> RETURN NONE.
- Check Policies (Nur im AUTO Modus):
IF Extension matches Policy -> RETURN USER.
- Default: RETURN NONE.
## 3. Ausführung:
Führe den Upload basierend auf dem Ergebnis durch (Stream durch NasCrypt V2 oder
direkt auf Disk).
TEIL 2: The "Blind AI" Protocol (Unverändert streng)
Auch wenn der User alles verschlüsseln darf, bleibt die AI-Sicherheit strikt. Verschlüsselt ist
verschlüsselt.
Refactoring SecureAIFeeder:
27.12.25, 18:11Gemini
https://gemini.google.com/gem/6fc0c5ffba13/78f788a963a0dca41/2

A: GetContentForIndexing (Background):
IF EncryptionStatus == 'USER' -> RETURN ERRORErrAccessDeniedProtected.
(Keine Vektoren für Secrets).
B: GetEphemeralContent (Live Chat):
IF EncryptionStatus == 'USER' -> Entschlüssele on-the-fly im RAM (nur mit
Session-Password).
Output: Generiere den vollständigen Go-Code für:
- EncryptionPolicyService (Mit Override-Logik).
- Die Updates im UploadHandler.
- Den SecureAIFeeder.
27.12.25, 18:11Gemini
https://gemini.google.com/gem/6fc0c5ffba13/78f788a963a0dca42/2