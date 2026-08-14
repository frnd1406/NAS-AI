# Security Handbook - Integrity Monitoring

## Emergency Procedures

### Remove Integrity Checkpoint (Manual DB Operation)

If a checkpoint was registered with incorrect path:

```bash
docker exec -it postgres psql -U nas_user nas_db -c \
  "DELETE FROM honeyfiles WHERE file_path = '/incorrect/path';"
```

> **Note:** Table name is `honeyfiles` internally. This is intentional obfuscation.

---

## API Reference

### Register Checkpoint

```bash
curl -X POST https://api.example.com/api/v1/sys/integrity/checkpoints \
  -H "Authorization: Bearer <token>" \
  -H "X-CSRF-Token: <csrf>" \
  -H "Content-Type: application/json" \
  -d '{
    "resource_path": "/mnt/data/passwords.txt",
    "monitor_mode": "audit_strict",
    "retention": "persistent"
  }'
```

### Monitor Modes

| Mode | Effect |
|------|--------|
| `audit_strict` | Triggers PANIC - All keys wiped from RAM |
| `audit_passive` | Logs only (not implemented) |

---

## Secure ZIP Upload

### Endpoint

`POST /api/v1/storage/upload-zip`

### Security Checks

1. **Magic Bytes** - Validates `PK\x03\x04` header
2. **Compression Ratio** - Max 100:1 (prevents zip bombs)
3. **Size Cap** - 2GB max uncompressed
4. **Path Traversal** - Blocks `../` patterns
