#!/bin/bash
set -e

# ============================================================
# NAS.AI Secret Generator (Automated)
# ============================================================

echo "🔐 Generating Secrets..."

# 1. Prepare Secrets Directory
mkdir -p secrets
chmod 700 secrets

# 2. Generate Values
POSTGRES_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=')
JWT_SECRET=$(openssl rand -base64 64 | tr -d '/+=')
MONITORING_TOKEN=$(openssl rand -base64 32 | tr -d '/+=')

# 3. Write Secret Files
echo -n "$POSTGRES_PASSWORD" > secrets/postgres_password
echo -n "$JWT_SECRET" > secrets/jwt_secret
echo -n "$MONITORING_TOKEN" > secrets/monitoring_token
chmod 600 secrets/*

echo "✅ Secret files created in ./secrets/"

# 4. Defaults
DOMAIN="example.com"
API_DOMAIN="api.example.com"
EMAIL_FROM="noreply@example.com"

# 5. Create .env.prod
cat > .env.prod << EOF
# ============================================================
# NAS.AI Production Environment Configuration
# Generated $(date)
# ============================================================

# ============================================================
# DATABASE SECRETS (PostgreSQL)
# ============================================================
POSTGRES_DB=nas_db
POSTGRES_USER=nas_user
# POSTGRES_PASSWORD is set via secret file
DB_HOST=postgres
DB_PORT=5432

# ============================================================
# SECURITY SECRETS
# ============================================================
# JWT_SECRET is set via secret file
# MONITORING_TOKEN is set via secret file

# Domain
DOMAIN=$DOMAIN
API_DOMAIN=$API_DOMAIN
CORS_ORIGINS=https://$DOMAIN,https://$API_DOMAIN
FRONTEND_URL=https://$DOMAIN
EMAIL_FROM=$EMAIL_FROM

# Advanced
LOG_LEVEL=INFO
RATE_LIMIT_PER_MIN=100
BACKUP_SCHEDULE="0 2 * * *"
BACKUP_RETENTION_COUNT=7

# AI Model
AI_MODEL_TYPE=sentence-transformers
AI_MODEL_NAME=sentence-transformers/all-MiniLM-L6-v2
MODEL_LOAD_RETRIES=3
INFERENCE_TIMEOUT=30

# API Configuration
PORT=8080
GIN_MODE=release
EOF

chmod 600 .env.prod
echo "✅ .env.prod created/updated."

