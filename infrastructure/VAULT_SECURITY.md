# 🔒 Vault Security Architecture

## Zero-Knowledge Encryption Prinzip

Das NAS.AI System verwendet **Zero-Knowledge Encryption** für maximale Sicherheit:

```
┌─────────────────────────────────────┐
│  User Master-Passwort              │
│  (nur User kennt es!)              │
└──────────────┬──────────────────────┘
               │
         [Argon2id KDF]
               │
               ▼
┌─────────────────────────────────────┐
│  DEK (Data Encryption Key)         │
│  Verschlüsselt im RAM              │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  Dateien verschlüsselt (XChaCha20-Poly1305)│
│  Auf Disk gespeichert              │
└─────────────────────────────────────┘
```

## 🎯 Standard-Verhalten (Maximale Sicherheit)

**Vault ist NICHT persistent:**
- ✅ Vault-Konfiguration in `/tmp/nas-vault-demo`
- ✅ Bei Container-Neustart → Vault ist WEG
- ✅ User muss neu einrichten
- ✅ **Niemand** kann verschlüsselte Dateien ohne Master-Passwort lesen
- ✅ Selbst bei physischem Zugriff auf die Festplatte: Daten bleiben sicher

**Warum ist das gut?**
1. **Zero-Knowledge:** Nur Sie kennen das Master-Passwort
2. **Kein Key-Leak:** Selbst wenn Server gehackt wird, Keys sind weg nach Restart
3. **Compliance:** DSGVO-konform, keine persistenten Schlüssel

## ⚠️ Optional: Vault-Persistenz aktivieren

**NUR wenn Sie die Sicherheitsrisiken verstehen und akzeptieren!**

### Sicherheitsrisiken bei Persistenz:

| Risiko | Beschreibung |
|--------|-------------|
| 🔓 **Physischer Zugriff** | Jemand mit Festplatten-Zugriff kann verschlüsselten DEK + Salt stehlen |
| 🔓 **Container-Kompromittierung** | Bei erfolgreicher Container-Attacke: Keys bleiben auf Disk |
| 🔓 **Backup-Leak** | Backups enthalten verschlüsselte Keys (Brute-Force möglich bei schwachem Passwort) |

### Aktivierung (nur für Convenience):

1. **docker-compose.dev.yml bearbeiten:**

```yaml
services:
  api:
    volumes:
      - nas_data:/mnt/data
      - nas_backups:/mnt/backups
      # WARNUNG: Persistenz aktivieren (Sicherheitsrisiko!)
      - nas_vault:/var/lib/nas/vault  # ← Uncomment diese Zeile
```

2. **Environment auf production setzen:**

```yaml
services:
  api:
    environment:
      ENV: production  # ← Wichtig!
```

3. **Container neu starten:**

```bash
docker-compose down
docker-compose up -d
```

4. **Log-Warnung beachten:**

```
⚠️  Vault persistence enabled: Keys survive restarts (security trade-off)
```

## 🔮 Zukünftige Authentifizierung

**Geplant (Pending):**
- 🔐 **Biometrische Entsperrung** (WebAuthn/Passkey)
  - Fingerabdruck oder Gesichtserkennung
  - Passkey entsperrt DEK im RAM
  - Kein Passwort-Typing nötig
  - Trotzdem Zero-Knowledge!

```
┌─────────────────────────────────────┐
│  Biometrischer Sensor (Touch ID)   │
└──────────────┬──────────────────────┘
               │
         [WebAuthn/Passkey]
               │
               ▼
┌─────────────────────────────────────┐
│  DEK wird entschlüsselt (im RAM)   │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  Dateien verfügbar                 │
└─────────────────────────────────────┘
```

## 📋 Best Practices

### ✅ DO:
- Master-Passwort sicher aufbewahren (Passwort-Manager)
- Regelmäßige Backups der **verschlüsselten** Dateien
- Vault neu einrichten bei Sicherheitsbedenken
- Starkes Passwort verwenden (min. 12 Zeichen, komplex)

### ❌ DON'T:
- Master-Passwort teilen oder aufschreiben
- Vault-Persistenz aktivieren ohne Risiken zu verstehen
- Schwache Passwörter verwenden
- Unverschlüsselte Backups erstellen

## 🔐 Encryption Details

**Algorithm Stack:**
- **Encryption:** XChaCha20-Poly1305 (Extended Nonce AEAD)
- **Key Derivation:** Argon2id (Memory-Hard, GPU-resistent)
- **Salt:** 32 Bytes random (pro Vault)
- **Nonce:** 24 Bytes random (pro Datei)
- **Auth Tag:** 16 Bytes (Integritätsschutz)

> Selected for high performance on non-hardware-accelerated CPUs and extended nonce support (24-byte) to eliminate nonce-reuse risks.

**Sicherheitsparameter:**
```go
Argon2id(
  time=3,        // Iterations
  memory=64MB,   // RAM usage
  threads=4,     // Parallelism
  keylen=32      // 256-bit key
)
```

## 🆘 Vault verloren? Was tun?

**Wenn Container neugestartet und Vault weg:**

1. ✅ **Kein Problem!** Das ist gewolltes Verhalten
2. ✅ Vault neu einrichten mit **demselben** Master-Passwort
3. ✅ Verschlüsselte Dateien bleiben entschlüsselbar
4. ❌ **Passwort vergessen?** → Daten sind **unwiederbringlich verloren**
   - Das ist Zero-Knowledge: Kein Recovery, keine Backdoor
   - Genau wie bei Apple FileVault oder BitLocker

## 📞 Support

Bei Fragen zur Vault-Security:
- GitHub Issues: `infrastructure/issues`
- Dokumentation: `VAULT_SECURITY.md` (diese Datei)

---

**Remember: Zero-Knowledge = Zero Recovery**

Ihr Master-Passwort ist der **einzige** Schlüssel zu Ihren Daten.
Keine Backdoors, keine Master-Keys, keine Recovery-Option.

**Das ist ein Feature, kein Bug!** 🔒
