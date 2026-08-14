# Email Testing Guide

Diese Anleitung zeigt, wie du die Email-Funktionalität der NAS.AI API testest.

## Übersicht

Die API sendet automatisch Emails bei folgenden Events:
1. **Registrierung** - Verification Email mit 24h Token
2. **Email Verifizierung** - Welcome Email nach erfolgreicher Verifizierung
3. **Password Reset** - Reset Link mit 1h Token

## Voraussetzungen

### 1. Domain bei Resend verifizieren

Die Domain `your-domain.com` ist bereits konfiguriert mit folgenden DNS Records:

```bash
# DKIM Record (TXT)
resend._domainkey.your-domain.com
Value: p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDVL1VXo+NbND+1ORSZZntLheQnKdXsJ0K9bK2xqLx2SEw8VhGyI/UpwK4zhJDsFkiGQrVMvC6/57IzxRGDKVy+8Iy38JDKbnWpCeyu06CieplpcB2v8TwdhxhDMq+vP22/48yO4TGsiCFsKFymzENcwN7Pq7smPh5qpYkdLLGh7wIDAQAB

# SPF Record (TXT)
send.your-domain.com
Value: v=spf1 include:amazonses.com ~all

# SPF Record (MX)
send.your-domain.com
Value: feedback-smtp.us-east-1.amazonses.com
Priority: 10
```

### 2. Server starten

```bash
# Mit allen Email-Env-Vars
export JWT_SECRET="$(openssl rand -base64 32)"
export PORT=8080
export CORS_ORIGINS="http://localhost:5173,https://your-domain.com"
export RATE_LIMIT_PER_MIN=100
export RESEND_API_KEY="re_REPLACE_WITH_YOUR_RESEND_KEY"
export EMAIL_FROM="NAS.AI <noreply@your-domain.com>"
export FRONTEND_URL="https://your-domain.com"
export CLOUDFLARE_API_TOKEN="GjKJMQiS998conpswEJhOwQ5b-fKSGjVmmFsofJf"

./bin/api
```

## Test 1: Verification Email bei Registrierung

### Request

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "deine.email@example.com",
    "password": "SecurePassword123"
  }'
```

### Erwartete Response

```json
{
  "user": {
    "id": "uuid-hier",
    "username": "testuser",
    "email": "deine.email@example.com",
    "email_verified": false,
    "created_at": "2025-11-21T20:20:57Z",
    "updated_at": "2025-11-21T20:20:57Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "csrf_token": "s5ZPAxlMAOXt6LB70nxH..."
}
```

### Server Logs (Erfolg)

```json
{"level":"info","msg":"User created successfully","user_id":"uuid-hier"}
{"level":"info","msg":"User registered successfully","email":"deine.email@example.com"}
{"level":"info","msg":"Verification email sent successfully","email_id":"resend-id","to":"deine.email@example.com"}
```

### Email Inhalt

Die Email enthält:
- **Subject**: "Verify your NAS.AI email address"
- **From**: NAS.AI <noreply@your-domain.com>
- **Button**: "Verify Email Address"
- **Link**: `https://your-domain.com/verify-email?token=XXXXXX`
- **Expiry**: 24 Stunden

## Test 2: Email Verifizierung

### 1. Token aus Redis holen

```bash
docker exec nas-api-redis redis-cli KEYS "verify:*"
```

Output:
```
verify:Y3oRFVsqf4LVvowCK8HfoyzmWrVrYVsB_nz8-4AvF8M=
```

### 2. Email verifizieren

```bash
curl -X POST http://localhost:8080/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{
    "token": "Y3oRFVsqf4LVvowCK8HfoyzmWrVrYVsB_nz8-4AvF8M="
  }'
```

### Response

```json
{
  "message": "Email verified successfully"
}
```

### Server Logs

```json
{"level":"info","msg":"Verification token validated","user_id":"uuid"}
{"level":"info","msg":"User email verified successfully","user_id":"uuid"}
{"level":"info","msg":"Welcome email sent successfully","email_id":"resend-id-2","to":"deine.email@example.com"}
```

### Welcome Email Inhalt

- **Subject**: "Welcome to NAS.AI!"
- **Features List**: Upload files, secure storage, fast access, security
- **Design**: Colored header with 🎉 emoji

## Test 3: Password Reset Flow

### 1. Password Reset anfordern

```bash
curl -X POST http://localhost:8080/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{
    "email": "deine.email@example.com"
  }'
```

### Response (immer gleich - no user enumeration)

```json
{
  "message": "If the email exists, a password reset link has been sent"
}
```

### 2. Reset Token aus Redis holen

```bash
docker exec nas-api-redis redis-cli KEYS "reset:*"
```

### 3. Password zurücksetzen

```bash
curl -X POST http://localhost:8080/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{
    "token": "vFjHAq10CeZ-e31N3RwnYOwBT2KERnFrA0mlRqsdkfs=",
    "new_password": "NewSecurePassword456"
  }'
```

### Password Reset Email Inhalt

- **Subject**: "Reset your NAS.AI password"
- **Button**: Red "Reset Password" button
- **Warning Box**: Yellow warning if not requested
- **Link**: `https://your-domain.com/reset-password?token=XXXXX`
- **Expiry**: 1 Stunde

## Troubleshooting

### Domain nicht verifiziert

**Error**:
```json
{"error":"The your-domain.com domain is not verified"}
```

**Lösung**: Domain Status prüfen
```bash
curl -X GET https://api.resend.com/domains/daf3d9fb-dcc1-4080-8860-0e6c38826262 \
  -H "Authorization: Bearer re_REPLACE_WITH_YOUR_RESEND_KEY" | jq '.status'
```

Status sollte `"verified"` sein.

### DNS Records prüfen

```bash
# DKIM Record
dig TXT resend._domainkey.your-domain.com

# SPF TXT Record
dig TXT send.your-domain.com

# SPF MX Record
dig MX send.your-domain.com
```

### Email kommt nicht an

1. **Spam Ordner prüfen**
2. **Server Logs prüfen** auf `email_id` - wenn vorhanden, wurde Email verschickt
3. **Resend Dashboard** prüfen: https://resend.com/emails
4. **Email Delivery Status** prüfen:
   ```bash
   curl -X GET https://api.resend.com/emails/EMAIL_ID_HIER \
     -H "Authorization: Bearer re_REPLACE_WITH_YOUR_RESEND_KEY"
   ```

## Production Checklist

- [ ] Domain bei Resend verifiziert
- [ ] DNS Records in Cloudflare eingetragen
- [ ] `EMAIL_FROM` korrekt gesetzt
- [ ] `FRONTEND_URL` auf production domain
- [ ] `RESEND_API_KEY` als Secret gespeichert
- [ ] Email Templates getestet
- [ ] Spam Score geprüft (https://www.mail-tester.com)

## API Endpoints

### Alle Email-bezogenen Endpoints

```bash
# Public
POST /auth/register              # Sendet Verification Email
POST /auth/verify-email          # Verifiziert Email, sendet Welcome Email
POST /auth/forgot-password       # Sendet Password Reset Email
POST /auth/reset-password        # Setzt Password zurück

# Protected (requires JWT)
POST /auth/resend-verification   # Sendet Verification Email erneut
```

## Email Templates Location

Die Email Templates befinden sich in:
```
src/services/email_service.go
```

Funktionen:
- `renderVerificationHTML()` / `renderVerificationText()`
- `renderPasswordResetHTML()` / `renderPasswordResetText()`
- `renderWelcomeHTML()` / `renderWelcomeText()`

## Monitoring

### Email Versand überwachen

```bash
# Alle Verification Tokens
docker exec nas-api-redis redis-cli KEYS "verify:*"

# Alle Reset Tokens
docker exec nas-api-redis redis-cli KEYS "reset:*"

# Token TTL prüfen
docker exec nas-api-redis redis-cli TTL "verify:TOKEN_HIER"
```

### Server Logs filtern

```bash
# Alle Email-Events
tail -f logs/api.log | grep "email"

# Nur erfolgreiche Emails
tail -f logs/api.log | grep "email sent successfully"

# Nur Fehler
tail -f logs/api.log | grep "Failed to send"
```

## Resend API Referenz

### Domain Status prüfen
```bash
curl https://api.resend.com/domains/DOMAIN_ID \
  -H "Authorization: Bearer API_KEY"
```

### Email Status prüfen
```bash
curl https://api.resend.com/emails/EMAIL_ID \
  -H "Authorization: Bearer API_KEY"
```

### Alle Domains auflisten
```bash
curl https://api.resend.com/domains \
  -H "Authorization: Bearer API_KEY"
```

## Cloudflare DNS via API

### DNS Records auflisten
```bash
curl "https://api.cloudflare.com/client/v4/zones/4c7c2dbd3698e0e1eb74edf48736a2bd/dns_records" \
  -H "Authorization: Bearer CLOUDFLARE_TOKEN" | jq
```

### DNS Record hinzufügen
```bash
curl -X POST "https://api.cloudflare.com/client/v4/zones/ZONE_ID/dns_records" \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "TXT",
    "name": "subdomain",
    "content": "value",
    "ttl": 1
  }'
```

## Best Practices

1. **Immer TTL=1 (Auto)** für DNS Records bei Cloudflare
2. **Tokens sind Single-Use** - nach Verwendung aus Redis gelöscht
3. **Email async senden** - blockiert nicht die Response
4. **Fehler nicht an User zeigen** - nur loggen
5. **Plain Text Fallback** immer mitliefern
6. **DMARC Policy** für bessere Deliverability empfohlen

## Security Notes

- Tokens sind **32-byte random** (crypto/rand)
- Tokens sind **Base64 URL encoded**
- Verification Token: **24h TTL**
- Reset Token: **1h TTL**
- Alle Tokens sind **single-use** (Redis delete nach Validierung)
- **No user enumeration** bei forgot-password (immer 200 OK)

## Support

- Resend Docs: https://resend.com/docs
- Resend Status: https://status.resend.com
- Cloudflare DNS Docs: https://developers.cloudflare.com/api/operations/dns-records-for-a-zone-list-dns-records
