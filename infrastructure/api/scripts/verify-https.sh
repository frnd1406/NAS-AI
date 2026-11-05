#!/bin/bash

# HTTPS Verification Script
# Überprüft ob HTTPS korrekt konfiguriert ist

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}=== HTTPS Verification Tool ===${NC}\n"

# Configuration
API_URL="${1:-https://api.your-domain.com}"
HEALTH_ENDPOINT="$API_URL/health"

echo -e "${BLUE}Testing URL:${NC} $API_URL\n"

# Test 1: Check if HTTPS is accessible
echo -e "${YELLOW}[1/6]${NC} Testing HTTPS connectivity..."
if curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$HEALTH_ENDPOINT" > /tmp/http_code 2>/dev/null; then
    HTTP_CODE=$(cat /tmp/http_code)
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "  ${GREEN}✓${NC} HTTPS connection successful (HTTP $HTTP_CODE)"
    else
        echo -e "  ${YELLOW}⚠${NC} HTTPS accessible but returned HTTP $HTTP_CODE"
    fi
else
    echo -e "  ${RED}✗${NC} Cannot connect to $HEALTH_ENDPOINT"
    echo -e "  Check if:"
    echo -e "    - Cloudflare Tunnel is running"
    echo -e "    - API is running on localhost:8080"
    echo -e "    - DNS is configured correctly"
    exit 1
fi

# Test 2: Verify SSL Certificate
echo -e "\n${YELLOW}[2/6]${NC} Verifying SSL Certificate..."
DOMAIN=$(echo "$API_URL" | sed 's|https://||' | sed 's|/.*||')
CERT_INFO=$(echo | openssl s_client -servername "$DOMAIN" -connect "$DOMAIN:443" 2>/dev/null | openssl x509 -noout -issuer -subject -dates 2>/dev/null || echo "")

if [ -n "$CERT_INFO" ]; then
    echo -e "  ${GREEN}✓${NC} Valid SSL Certificate found"
    echo "$CERT_INFO" | while IFS= read -r line; do
        echo -e "    ${BLUE}→${NC} $line"
    done
else
    echo -e "  ${YELLOW}⚠${NC} Could not verify certificate (might be Cloudflare proxy)"
fi

# Test 3: Check TLS Version
echo -e "\n${YELLOW}[3/6]${NC} Checking TLS Version..."
TLS_VERSION=$(curl -sI --tlsv1.2 "$HEALTH_ENDPOINT" -o /dev/null -w "%{ssl_version}" 2>/dev/null || echo "unknown")
if [[ "$TLS_VERSION" =~ TLS.*1\.[23] ]]; then
    echo -e "  ${GREEN}✓${NC} Using $TLS_VERSION (secure)"
else
    echo -e "  ${YELLOW}⚠${NC} TLS Version: $TLS_VERSION"
fi

# Test 4: Verify HTTPS Redirect (HTTP → HTTPS)
echo -e "\n${YELLOW}[4/6]${NC} Testing HTTP to HTTPS Redirect..."
HTTP_URL="${API_URL/https:/http:}"
REDIRECT_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$HTTP_URL/health" 2>/dev/null || echo "000")

if [ "$REDIRECT_CODE" = "301" ] || [ "$REDIRECT_CODE" = "308" ]; then
    echo -e "  ${GREEN}✓${NC} HTTP redirects to HTTPS (HTTP $REDIRECT_CODE)"
elif [ "$REDIRECT_CODE" = "000" ]; then
    echo -e "  ${BLUE}ℹ${NC} HTTP not accessible (Cloudflare Tunnel only serves HTTPS)"
else
    echo -e "  ${YELLOW}⚠${NC} HTTP returned $REDIRECT_CODE (no redirect)"
fi

# Test 5: Check Security Headers
echo -e "\n${YELLOW}[5/6]${NC} Checking Security Headers..."
HEADERS=$(curl -sI "$HEALTH_ENDPOINT" 2>/dev/null)

# HSTS
if echo "$HEADERS" | grep -qi "strict-transport-security"; then
    HSTS_VALUE=$(echo "$HEADERS" | grep -i "strict-transport-security" | cut -d: -f2- | xargs)
    echo -e "  ${GREEN}✓${NC} HSTS: $HSTS_VALUE"
else
    echo -e "  ${YELLOW}⚠${NC} HSTS header not found"
fi

# X-Content-Type-Options
if echo "$HEADERS" | grep -qi "x-content-type-options"; then
    echo -e "  ${GREEN}✓${NC} X-Content-Type-Options: $(echo "$HEADERS" | grep -i "x-content-type-options" | cut -d: -f2- | xargs)"
else
    echo -e "  ${YELLOW}⚠${NC} X-Content-Type-Options not set"
fi

# X-Frame-Options
if echo "$HEADERS" | grep -qi "x-frame-options"; then
    echo -e "  ${GREEN}✓${NC} X-Frame-Options: $(echo "$HEADERS" | grep -i "x-frame-options" | cut -d: -f2- | xargs)"
else
    echo -e "  ${YELLOW}⚠${NC} X-Frame-Options not set"
fi

# Cloudflare specific
if echo "$HEADERS" | grep -qi "cf-ray"; then
    CF_RAY=$(echo "$HEADERS" | grep -i "cf-ray" | cut -d: -f2- | xargs)
    echo -e "  ${GREEN}✓${NC} Cloudflare Protection Active (CF-Ray: $CF_RAY)"
else
    echo -e "  ${BLUE}ℹ${NC} Not using Cloudflare proxy"
fi

# Test 6: API Health Check Response
echo -e "\n${YELLOW}[6/6]${NC} Testing API Health Response..."
HEALTH_RESPONSE=$(curl -s "$HEALTH_ENDPOINT" 2>/dev/null)

if echo "$HEALTH_RESPONSE" | jq -e '.status == "ok"' > /dev/null 2>&1; then
    echo -e "  ${GREEN}✓${NC} API Health Check: OK"
    echo -e "    ${BLUE}Response:${NC} $(echo "$HEALTH_RESPONSE" | jq -c .)"
elif [ -n "$HEALTH_RESPONSE" ]; then
    echo -e "  ${YELLOW}⚠${NC} API responded but health check format unexpected"
    echo -e "    ${BLUE}Response:${NC} $HEALTH_RESPONSE"
else
    echo -e "  ${RED}✗${NC} No response from API"
fi

# Summary
echo -e "\n${GREEN}=== Verification Summary ===${NC}\n"

# SSL Labs Test Link
echo -e "${BLUE}For detailed SSL analysis:${NC}"
echo -e "  https://www.ssllabs.com/ssltest/analyze.html?d=${API_URL#https://}"

# Security Headers Test Link
echo -e "\n${BLUE}For security headers analysis:${NC}"
echo -e "  https://securityheaders.com/?q=${API_URL}"

# Cloudflare Tunnel Status
echo -e "\n${BLUE}Check Cloudflare Tunnel status:${NC}"
echo -e "  systemctl status cloudflared"

# Check if tunnel is running
if systemctl is-active --quiet cloudflared 2>/dev/null; then
    echo -e "  ${GREEN}✓${NC} Cloudflare Tunnel service is running"
else
    echo -e "  ${YELLOW}⚠${NC} Cloudflare Tunnel service status unknown"
fi

# Check if API is running
if curl -s -o /dev/null --max-time 2 http://localhost:8080/health 2>/dev/null; then
    echo -e "  ${GREEN}✓${NC} API is running on localhost:8080"
else
    echo -e "  ${YELLOW}⚠${NC} Cannot reach API on localhost:8080"
fi

echo -e "\n${GREEN}✓ HTTPS Verification Complete!${NC}\n"
