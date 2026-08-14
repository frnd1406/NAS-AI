# NAS Password Manager Documentation

**Created:** 2025-11-17
**Location:** `/srv/password-manager/`
**Status:** ✅ Active

---

## 🔐 OVERVIEW

Sicherer Password Manager für kritische Secrets (Vault Keys, API Keys, etc.) mit GPG AES-256 Verschlüsselung.

**Security Features:**
- ✅ AES-256 Encryption (GPG)
- ✅ Master Password (auto-generated, 256-bit)
- ✅ Encrypted at rest
- ✅ Secure file permissions (600)
- ✅ Automatic temp file shredding

---

## 📁 FILE STRUCTURE

```
/srv/password-manager/
├── pwmanager.sh              # Main CLI tool
├── .master_password          # Master password (600 permissions, root only)
└── vault/
    ├── secrets.gpg           # Encrypted secrets vault
    └── secrets_backup_*.gpg  # Backups (created on demand)
```

**Symlink:** `/usr/local/bin/pwmanager` → `/srv/password-manager/pwmanager.sh`

---

## 🚀 QUICK START

### List all stored secrets:
```bash
sudo pwmanager list
```

### Retrieve a secret:
```bash
sudo pwmanager get vault_unseal_key_1
```

### Add a new secret:
```bash
sudo pwmanager add my_secret_name
# You'll be prompted to enter the value
```

### Create a backup:
```bash
sudo pwmanager backup
```

---

## 📚 COMMAND REFERENCE

| Command | Description | Example |
|---------|-------------|---------|
| `init` | Initialize vault (first time) | `sudo pwmanager init` |
| `add <name>` | Add new secret | `sudo pwmanager add api_key` |
| `get <name>` | Retrieve secret | `sudo pwmanager get vault_root_token` |
| `list` | List all secret names | `sudo pwmanager list` |
| `export` | Export all secrets (unencrypted JSON) | `sudo pwmanager export` |
| `import <file>` | Import secrets from JSON file | `sudo pwmanager import secrets.json` |
| `backup` | Create encrypted backup | `sudo pwmanager backup` |

---

## 🔑 CURRENTLY STORED SECRETS

**Vault Keys (6):**
- `vault_root_token` - Vault root access token
- `vault_unseal_key_1` - Unseal key 1/5
- `vault_unseal_key_2` - Unseal key 2/5
- `vault_unseal_key_3` - Unseal key 3/5 (required minimum)
- `vault_unseal_key_4` - Unseal key 4/5
- `vault_unseal_key_5` - Unseal key 5/5

**Note:** Vault requires **3 of 5** unseal keys to unseal after restart.

---

## 🔓 MASTER PASSWORD

**Location:** `/srv/password-manager/.master_password`

**Access:**
```bash
# View master password (root only)
sudo cat /srv/password-manager/.master_password

# Copy to clipboard (if xclip installed)
sudo cat /srv/password-manager/.master_password | xclip -selection clipboard
```

**⚠️ IMPORTANT:**
- Master password is auto-generated (256-bit random)
- File has 600 permissions (root read/write only)
- Required to decrypt any secrets from the vault
- **Backup this file offsite** (e.g., USB drive, paper backup)

---

## 💾 BACKUP & RECOVERY

### Create Backup:
```bash
sudo pwmanager backup
```

Creates: `/srv/password-manager/vault/secrets_backup_YYYYMMDD_HHMMSS.gpg`

### Manual Backup (offsite):
```bash
# Copy encrypted vault
sudo cp /srv/password-manager/vault/secrets.gpg /path/to/backup/

# Copy master password
sudo cp /srv/password-manager/.master_password /path/to/backup/

# Or export all secrets to JSON (unencrypted - handle with care!)
sudo pwmanager export
```

### Recovery:
```bash
# 1. Restore files
sudo cp backup/secrets.gpg /srv/password-manager/vault/
sudo cp backup/.master_password /srv/password-manager/

# 2. Set permissions
sudo chmod 600 /srv/password-manager/.master_password
sudo chmod 600 /srv/password-manager/vault/secrets.gpg

# 3. Test
sudo pwmanager list
```

---

## 🛡️ SECURITY BEST PRACTICES

### DO:
- ✅ Keep master password file backed up offsite
- ✅ Create regular backups (`pwmanager backup`)
- ✅ Use `sudo` for all pwmanager commands
- ✅ Verify file permissions (600) after recovery
- ✅ Use strong, unique secrets when adding new entries

### DON'T:
- ❌ Share master password over network/email
- ❌ Store master password in plaintext elsewhere
- ❌ Run pwmanager without sudo (won't have vault access)
- ❌ Export secrets unnecessarily (`pwmanager export` creates unencrypted file)
- ❌ Commit `.master_password` or `secrets.gpg` to git

---

## 🔧 ADVANCED USAGE

### Direct GPG Access (manual):
```bash
# Decrypt vault (requires master password input)
gpg --decrypt /srv/password-manager/vault/secrets.gpg

# With stored master password
MASTER_PW=$(sudo cat /srv/password-manager/.master_password)
gpg --quiet --batch --yes --passphrase "$MASTER_PW" --decrypt /srv/password-manager/vault/secrets.gpg | jq '.'
```

### Get specific key programmatically:
```bash
#!/bin/bash
# get_vault_key.sh - Retrieves a specific Vault key

KEY_NAME="$1"
if [ -z "$KEY_NAME" ]; then
    echo "Usage: $0 <key_name>"
    exit 1
fi

MASTER_PW=$(sudo cat /srv/password-manager/.master_password)
gpg --quiet --batch --yes --passphrase "$MASTER_PW" \
    --decrypt /srv/password-manager/vault/secrets.gpg | \
    jq -r ".${KEY_NAME}.value"
```

### Vault Unseal Helper Script:
```bash
#!/bin/bash
# vault_unseal.sh - Unseal Vault using stored keys

MASTER_PW=$(sudo cat /srv/password-manager/.master_password)
VAULT_DATA=$(gpg --quiet --batch --yes --passphrase "$MASTER_PW" --decrypt /srv/password-manager/vault/secrets.gpg)

for i in 1 2 3; do
    KEY=$(echo "$VAULT_DATA" | jq -r ".vault_unseal_key_${i}.value")
    curl -s -X POST -d "{\"key\":\"$KEY\"}" https://192.168.x.x:8200/v1/sys/unseal | jq '.sealed'
done
```

---

## 📝 JSON FORMAT

Secrets are stored in JSON format:

```json
{
  "secret_name": {
    "value": "secret_value_here",
    "created": "2025-11-17T19:54:00Z"
  },
  "another_secret": {
    "value": "another_value",
    "created": "2025-11-17T20:00:00Z"
  }
}
```

---

## 🔄 INTEGRATION EXAMPLES

### Use in systemd service:
```bash
[Service]
Environment="SECRET_VALUE=$(sudo pwmanager get my_secret)"
ExecStart=/usr/bin/my-service
```

### Use in shell script:
```bash
#!/bin/bash
API_KEY=$(sudo pwmanager get resend_api_key)
curl -H "Authorization: Bearer $API_KEY" https://api.example.com
```

---

## ❓ TROUBLESHOOTING

### "Permission denied" error:
```bash
# Fix vault directory permissions
sudo chmod 700 /srv/password-manager/vault
sudo chmod 600 /srv/password-manager/vault/secrets.gpg
```

### "Vault not found" error:
```bash
# Reinitialize (if no data exists)
sudo pwmanager init

# Or restore from backup
sudo cp /path/to/backup/secrets.gpg /srv/password-manager/vault/
```

### "GPG decryption failed":
```bash
# Verify master password file exists
ls -lh /srv/password-manager/.master_password

# Test manual decryption
sudo gpg --decrypt /srv/password-manager/vault/secrets.gpg
```

---

## 📊 STATUS CHECK

### Verify password manager health:
```bash
# Check files exist
ls -lh /srv/password-manager/
ls -lh /srv/password-manager/vault/

# Test retrieval
sudo pwmanager list

# Verify master password
sudo cat /srv/password-manager/.master_password | wc -c
# Should output: 45 (44 chars + newline)
```

---

## 🎯 MAINTENANCE

### Regular Tasks:
- **Weekly:** Create backup (`sudo pwmanager backup`)
- **Monthly:** Test recovery from backup
- **Annually:** Review stored secrets, remove unused

### Backup Cleanup:
```bash
# Keep last 10 backups, remove older
cd /srv/password-manager/vault
ls -t secrets_backup_*.gpg | tail -n +11 | xargs rm
```

---

## 📌 QUICK REFERENCE CARD

```bash
# Most common commands:
sudo pwmanager list                    # List all secrets
sudo pwmanager get vault_root_token    # Get root token
sudo pwmanager get vault_unseal_key_1  # Get unseal key 1
sudo pwmanager backup                  # Create backup

# Master password:
sudo cat /srv/password-manager/.master_password
```

---

## 🔗 RELATED DOCUMENTATION

- **Vault Documentation:** `vault/`
- **Security Handbook:** `docs/security/SECURITY_HANDBOOK.md`

---

**Last Updated:** 2025-11-21
**Maintained by:** SystemSetupAgent
**Status:** ✅ Production Ready