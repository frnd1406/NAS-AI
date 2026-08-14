#!/usr/bin/env bash
#
# gen-selfsigned-cert.sh — generate a self-signed TLS certificate for the API.
#
# The API server is TLS-only. In production, provide real certificate files via
# TLS_CERT_FILE / TLS_KEY_FILE. For lab / on-prem demos you can generate a
# self-signed pair with this script.
#
# Usage:
#   ./scripts/gen-selfsigned-cert.sh [OUTPUT_DIR] [CN]
#
# Then run the API with:
#   TLS_CERT_FILE=<dir>/nas.crt TLS_KEY_FILE=<dir>/nas.key PORT=8443 ./api
#
set -euo pipefail

OUT_DIR="${1:-infrastructure/api/certs}"
CN="${2:-localhost}"
DAYS="${DAYS:-365}"

mkdir -p "$OUT_DIR"

openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
  -nodes \
  -keyout "$OUT_DIR/nas.key" \
  -out "$OUT_DIR/nas.crt" \
  -days "$DAYS" \
  -subj "/CN=$CN" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

chmod 600 "$OUT_DIR/nas.key"

echo "✅ Self-signed certificate written to:"
echo "   TLS_CERT_FILE=$OUT_DIR/nas.crt"
echo "   TLS_KEY_FILE=$OUT_DIR/nas.key"
echo
echo "⚠️  Self-signed certs are for lab/dev only. Use a CA-issued cert in production."
