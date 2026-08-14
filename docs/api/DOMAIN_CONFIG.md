# Domain Configuration

## Domain Setup

**Main Domain:** `your-domain.com`

### Subdomains

- **API:** `api.your-domain.com`
- **Frontend:** `your-domain.com` (root domain)

## Cloudflare Tunnel Configuration

### DNS Records

```
Type: CNAME
Name: api
Content: <TUNNEL-ID>.cfargotunnel.com
Proxy: Proxied (Orange Cloud ☁️)
TTL: Auto
```

### Tunnel Config (`~/.cloudflared/config.yml`)

```yaml
tunnel: nas-api
credentials-file: /home/user/.cloudflared/<TUNNEL-ID>.json

ingress:
  # API Endpoint
  - hostname: api.your-domain.com
    service: http://localhost:8080
    originRequest:
      noTLSVerify: true
      connectTimeout: 30s

  # Frontend (optional)
  - hostname: your-domain.com
    service: http://localhost:5173

  # Catchall (required)
  - service: http_status:404
```

## Environment Variables

### Production

```bash
# Server
export PORT=8080
export ENV=production
export LOG_LEVEL=info

# CORS
export CORS_ORIGINS=https://your-domain.com,https://api.your-domain.com

# Frontend
export FRONTEND_URL=https://your-domain.com

# Email
export EMAIL_FROM="NAS.AI <noreply@your-domain.com>"

# Security
export JWT_SECRET="<your-secure-secret-min-32-chars>"
```

### Development

```bash
# Server
export PORT=8080
export ENV=development
export LOG_LEVEL=debug

# CORS (allows localhost)
export CORS_ORIGINS=http://localhost:5173,http://localhost:3000

# Frontend
export FRONTEND_URL=http://localhost:5173

# Email
export EMAIL_FROM="NAS.AI Dev <noreply@your-domain.com>"
```

## API Endpoints

### Base URLs

- **Production:** `https://api.your-domain.com`
- **Development:** `http://localhost:8080`

### Example Endpoints

```
GET  https://api.your-domain.com/health
POST https://api.your-domain.com/auth/register
POST https://api.your-domain.com/auth/login
GET  https://api.your-domain.com/docs/swagger/index.html
```

## Email Configuration

### Sender Addresses

- **Production:** `noreply@your-domain.com`
- **Support:** `support@your-domain.com`

### Email Templates

All email templates reference:
- Verification links: `https://your-domain.com/verify?token=...`
- Password reset: `https://your-domain.com/reset-password?token=...`

## SSL/TLS

**Managed by:** Cloudflare (automatic)

**Certificate:** Cloudflare Universal SSL
**Encryption Mode:** Full (or Full Strict)
**TLS Version:** 1.2+ (TLS 1.3 enabled)
**HSTS:** Enabled (max-age: 31536000)

## Verification

### Test HTTPS Setup

```bash
# Run verification script
./scripts/verify-https.sh

# Or test specific URL
./scripts/verify-https.sh https://api.your-domain.com
```

### Manual Tests

```bash
# Check API health
curl https://api.your-domain.com/health

# Check SSL certificate
echo | openssl s_client -servername api.your-domain.com -connect api.your-domain.com:443

# Check headers
curl -I https://api.your-domain.com/health
```

## Updated Files

All references to `nas.your-domain.com` have been updated to `your-domain.com`:

- ✅ `src/config/config.go` - Email and Frontend URL defaults
- ✅ `src/main.go` - Swagger documentation
- ✅ `docs/swagger.yaml` - API documentation
- ✅ `docs/swagger.json` - API documentation
- ✅ `docs/docs.go` - Generated swagger docs
- ✅ `scripts/verify-https.sh` - HTTPS verification script
- ✅ `status/APIAgent/phase1/*.md` - Documentation files

## Service Status

### Check Services

```bash
# Cloudflare Tunnel
systemctl status cloudflared

# API Service (if using systemd)
systemctl status nas-api

# Check tunnel logs
journalctl -u cloudflared -f
```

### Restart Services

```bash
# Restart tunnel
sudo systemctl restart cloudflared

# Restart API
sudo systemctl restart nas-api

# Or manually
cd /home/user/Agent/infrastructure/api
./bin/api
```

## Monitoring

### Cloudflare Dashboard

- Analytics: https://dash.cloudflare.com → Analytics
- Tunnel Status: https://dash.cloudflare.com → Zero Trust → Access → Tunnels
- SSL/TLS Settings: https://dash.cloudflare.com → SSL/TLS

### External Tools

- **SSL Test:** https://www.ssllabs.com/ssltest/analyze.html?d=api.your-domain.com
- **Security Headers:** https://securityheaders.com/?q=api.your-domain.com
- **DNS Check:** https://dnschecker.org/#A/api.your-domain.com
