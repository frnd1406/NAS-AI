#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

# --- 1. KONFIGURATION & STYLING ---

# Farben & Icons
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Pfade (Context Aware)
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INFRA_DIR="$BASE_DIR/infrastructure"
ENV_FILE="$INFRA_DIR/.env.prod"
COMPOSE_FILE="$INFRA_DIR/docker-compose.prod.yml"

# Error Handling Trap
trap 'echo -e "\n${RED}💥 FATAL ERROR an Zeile $LINENO. Das Skript bricht ab.${NC}"; exit 1' ERR

# --- 2. DOPAMINE & VISUALS ---

function show_header() {
    clear
    echo -e "${CYAN}"
    cat << "EOF"
  _   _    _    ____      _    ___ 
 | \ | |  / \  / ___|    / \  |_ _|
 |  \| | / _ \ \___ \   / _ \  | | 
 | |\  |/ ___ \ ___) | / ___ \ | | 
 |_| \_/_/   \_\____/ /_/   \_\___|
                                   
EOF
    echo -e "${BLUE}>> COMMAND & CONTROL CENTER V2.0${NC}"
    echo -e "${YELLOW}User: $USER | System: NAS.AI | Status: ${GREEN}ONLINE${NC}"
    echo "==================================================="
}

function hype_loader() {
    tput civis # Cursor verstecken
    echo -e "${GREEN}"
    echo -ne "SYSTEM STARTUP: [${NC}"
    for i in {1..40}; do
        # Zufällige Farbe für den Glitch-Effekt
        R_COL=$((RANDOM % 6 + 31))
        echo -ne "\e[1;${R_COL}m#\e[0m"
        sleep 0.02
    done
    echo -e "${GREEN}] 100%${NC}"
    
    # Fake Checks für das "Feeling"
    local CHECKS=("🔒 Encrypting Flux Capacitors..." "🧠 Waking up AI Agents..." "📡 Scanning Subnet..." "💉 Injecting Coffee...")
    for msg in "${CHECKS[@]}"; do
        echo -e "${GREEN}✓ $msg${NC}"
        sleep 0.1
    done
    sleep 0.5
    tput cnorm # Cursor zeigen
}

# --- 3. CORE LOGIC ---

function check_preflight() {
    # Check: Docker Running?
    if ! docker info > /dev/null 2>&1; then
        echo -e "${RED}❌ ERROR: Docker läuft nicht! Bitte starten.${NC}"
        exit 1
    fi
    # Check: Config Files
    if [ ! -f "$ENV_FILE" ]; then
        echo -e "${RED}❌ ERROR: .env.prod fehlt unter $INFRA_DIR${NC}"
        exit 1
    fi
}

function wait_for_enter() {
    echo -e "\n${YELLOW}>> Drücke ENTER für Menü...${NC}"
    read
}

# --- 4. MODULE (The "Functions") ---

# MODUL: LOGS (Smart Filter)
function smart_logs() {
    local service=$1
    echo -e "${MAGENTA}🕵️  Starte Smart-Logs für: ${service:-ALLES} (CTRL+C zum Beenden)${NC}"
    echo "---------------------------------------------------"
    
    # Hier passiert die Magie: sed entfernt Klammern, awk analysiert HTTP Codes
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" logs -f $service 2>&1 | \
    sed -u 's/\[//g; s/\]//g' | \
    grep --line-buffered -v "beenden" | \
    awk '
    {
        # Zeile ausgeben (Standard)
        print $0; 
        
        # Forensics Logic: HTTP Codes erklären
        if ($0 ~ / 401 /) print "\t\033[0;31m🔒 AUTH FAIL: Falscher Token oder Login-Versuch\033[0m";
        if ($0 ~ / 403 /) print "\t\033[0;31m🚫 FORBIDDEN: Zugriff verweigert (RBAC)\033[0m";
        if ($0 ~ / 404 /) print "\t\033[1;33m🔍 NOT FOUND: Scanner sucht Lücken?\033[0m";
        if ($0 ~ / 500 /) print "\t\033[0;31m💥 SERVER ERROR: Check Backend Code!\033[0m";
        if ($0 ~ / 502 /) print "\t\033[0;31m🔌 BAD GATEWAY: Container down?\033[0m";
        if ($0 ~ /panic:/) print "\t\033[0;31m🔥 GO PANIC: Kritischer Absturz!\033[0m";
    }'
}

# MODUL: FORENSICS (Gauner Check)
function forensic_ip_check() {
    echo -e "${RED}🕵️  FORENSIC TARGET ANALYSIS${NC}"
    read -p ">> IP-Adresse eingeben: " TARGET_IP
    
    if [[ -z "$TARGET_IP" ]]; then echo "Abbruch."; return; fi
    
    echo -e "${YELLOW}>> Scanne $TARGET_IP...${NC}"
    echo "---------------------------------------------------"
    # curl auf ip-api.com (mit Timeout)
    curl -s -m 10 "http://ip-api.com/json/$TARGET_IP" | grep -E '"country"|"city"|"isp"|"org"|"as"|"query"' | tr -d '{}"\'
    echo "---------------------------------------------------"
    wait_for_enter
}

# MODUL: DEPLOYMENT
function deploy_full() {
    echo -e "${YELLOW}🚀 Starte Full Production Deployment...${NC}"
    echo -e "${CYAN}>> Building custom images...${NC}"
    # Build erst, dann Pull für externe Images, dann Up
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" build
    echo -e "${CYAN}>> Pulling external images (falls vorhanden)...${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" pull --ignore-pull-failures || true
    echo -e "${CYAN}>> Starting all services...${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --remove-orphans
    echo -e "${GREEN}✅ Deployment abgeschlossen.${NC}"
    wait_for_enter
}

# MODUL: CLEAN RESTART (No Cache, Force Recreate)
function clean_restart() {
    echo -e "${RED}🔥 CLEAN RESTART - Keine Caches, Alles Neu!${NC}"
    echo -e "${YELLOW}>> Dies baut alle Images ohne Cache neu und startet Container frisch.${NC}"
    echo ""
    
    # Step 1: Stop all containers
    echo -e "${CYAN}[1/4] Stoppe alle Container...${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" down --remove-orphans
    
    # Step 2: Build without cache
    echo -e "${CYAN}[2/4] Baue Images OHNE Cache (kann dauern)...${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" build --no-cache
    
    # Step 3: Pull latest external images
    echo -e "${CYAN}[3/4] Hole neueste externe Images...${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" pull --ignore-pull-failures || true
    
    # Step 4: Force recreate all containers
    echo -e "${CYAN}[4/4] Starte alle Container NEU (force-recreate)...${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --force-recreate --remove-orphans
    
    echo -e "${GREEN}✅ CLEAN RESTART abgeschlossen!${NC}"
    echo -e "${CYAN}>> Alle Services laufen mit frischen Images.${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps
    wait_for_enter
}

# MODUL: QUICK REBUILD (With Cache, Force Recreate)
function quick_rebuild() {
    echo -e "${YELLOW}⚡ QUICK REBUILD - Mit Cache, aber Force Recreate${NC}"
    echo ""
    
    echo -e "${CYAN}[1/2] Baue Images (mit Cache)...${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" build
    
    echo -e "${CYAN}[2/2] Starte alle Container NEU...${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --force-recreate --remove-orphans
    
    echo -e "${GREEN}✅ Quick Rebuild abgeschlossen!${NC}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps
    wait_for_enter
}

# MODUL: API TESTING
function test_api_endpoint() {
    local method=$1
    local endpoint=$2
    local description=$3
    local data="${4:-}"
    local auth_token="${5:-}"

    echo -ne "${CYAN}Testing: $description${NC} ... "

    local cmd="curl -s -m 10 -X $method"

    if [ -n "$auth_token" ]; then
        cmd="$cmd -H 'Authorization: Bearer $auth_token'"
    fi

    if [ -n "$data" ]; then
        cmd="$cmd -H 'Content-Type: application/json' -d '$data'"
    fi

    cmd="$cmd -w '\n%{http_code}' $endpoint"

    local response=$(eval $cmd 2>&1)
    local http_code=$(echo "$response" | tail -n1)

    case $http_code in
        200|201) echo -e "${GREEN}✓ OK ($http_code)${NC}" ;;
        204) echo -e "${GREEN}✓ OK (No Content)${NC}" ;;
        401) echo -e "${YELLOW}⚠ AUTH Required ($http_code)${NC}" ;;
        403) echo -e "${YELLOW}⚠ Forbidden ($http_code)${NC}" ;;
        404) echo -e "${YELLOW}⚠ Not Found ($http_code)${NC}" ;;
        500) echo -e "${RED}✗ Server Error ($http_code)${NC}" ;;
        502) echo -e "${MAGENTA}⏸ Service in Wartung/Offline ($http_code)${NC}" ;;
        503) echo -e "${MAGENTA}⏸ Service Unavailable ($http_code)${NC}" ;;
        000) echo -e "${RED}✗ Connection Failed${NC}" ;;
        *) echo -e "${YELLOW}? Status: $http_code${NC}" ;;
    esac
}

function test_all_main_api() {
    local BASE_URL="http://localhost:8080"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${CYAN}🧪 TESTING MAIN API (Port 8080)${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

    echo -e "${YELLOW}[PUBLIC ENDPOINTS]${NC}"
    test_api_endpoint "GET" "$BASE_URL/health" "Health Check"
    test_api_endpoint "GET" "$BASE_URL/api/v1/metrics" "Prometheus Metrics"
    test_api_endpoint "GET" "$BASE_URL/api/v1/version" "API Version"

    echo -e "\n${YELLOW}[AUTHENTICATION ENDPOINTS]${NC}"
    test_api_endpoint "POST" "$BASE_URL/api/v1/auth/login" "Login" '{"username":"test","password":"test"}'
    test_api_endpoint "POST" "$BASE_URL/api/v1/auth/register" "Register" '{"username":"test","email":"test@test.com","password":"test123"}'
    test_api_endpoint "POST" "$BASE_URL/api/v1/auth/logout" "Logout"
    test_api_endpoint "POST" "$BASE_URL/api/v1/auth/refresh" "Token Refresh"
    test_api_endpoint "POST" "$BASE_URL/api/v1/auth/forgot-password" "Forgot Password" '{"email":"test@test.com"}'
    test_api_endpoint "POST" "$BASE_URL/api/v1/auth/reset-password" "Reset Password" '{"token":"xxx","password":"newpass"}'
    test_api_endpoint "POST" "$BASE_URL/api/v1/auth/verify-email" "Verify Email" '{"token":"xxx"}'
    test_api_endpoint "POST" "$BASE_URL/api/v1/auth/mfa/enable" "Enable MFA"
    test_api_endpoint "POST" "$BASE_URL/api/v1/auth/mfa/verify" "Verify MFA" '{"code":"123456"}'

    echo -e "\n${YELLOW}[PROTECTED ENDPOINTS - ohne Token]${NC}"
    test_api_endpoint "GET" "$BASE_URL/api/v1/user/profile" "User Profile"
    test_api_endpoint "PUT" "$BASE_URL/api/v1/user/profile" "Update Profile" '{"name":"Test"}'
    test_api_endpoint "GET" "$BASE_URL/api/v1/admin/dashboard" "Admin Dashboard"

    echo -e "\n${YELLOW}[FILE MANAGEMENT]${NC}"
    test_api_endpoint "GET" "$BASE_URL/api/v1/files" "List Files"
    test_api_endpoint "POST" "$BASE_URL/api/v1/files/upload" "Upload File"
    test_api_endpoint "GET" "$BASE_URL/api/v1/files/download/test.txt" "Download File"
    test_api_endpoint "DELETE" "$BASE_URL/api/v1/files/test.txt" "Delete File"
    test_api_endpoint "GET" "$BASE_URL/api/v1/files/trash" "List Trash"
    test_api_endpoint "POST" "$BASE_URL/api/v1/files/trash/restore/test.txt" "Restore from Trash"
    test_api_endpoint "DELETE" "$BASE_URL/api/v1/files/trash/test.txt" "Permanent Delete"
    test_api_endpoint "POST" "$BASE_URL/api/v1/files/trash/empty" "Empty Trash"

    echo -e "\n${YELLOW}[BACKUP MANAGEMENT]${NC}"
    test_api_endpoint "GET" "$BASE_URL/api/v1/backups" "List Backups"
    test_api_endpoint "POST" "$BASE_URL/api/v1/backups/create" "Create Backup"
    test_api_endpoint "POST" "$BASE_URL/api/v1/backups/restore/backup_123" "Restore Backup"
    test_api_endpoint "DELETE" "$BASE_URL/api/v1/backups/backup_123" "Delete Backup"

    echo -e "\n${YELLOW}[SYSTEM MONITORING]${NC}"
    test_api_endpoint "GET" "$BASE_URL/api/v1/system/logs" "System Logs"
    test_api_endpoint "GET" "$BASE_URL/api/v1/system/stats" "System Stats"
    test_api_endpoint "GET" "$BASE_URL/api/v1/system/services" "Service Status"
    test_api_endpoint "POST" "$BASE_URL/api/v1/system/services/api/restart" "Restart Service"

    echo -e "\n${YELLOW}[SETTINGS]${NC}"
    test_api_endpoint "GET" "$BASE_URL/api/v1/settings" "Get Settings"
    test_api_endpoint "PUT" "$BASE_URL/api/v1/settings" "Update Settings" '{"key":"value"}'

    echo -e "\n${MAGENTA}⚠️  HINWEIS: Einige Endpoints können in Wartung sein (502/503)${NC}"
    echo -e "${YELLOW}📖 Details zu allen Endpoints: $BASE_DIR/API_ENDPOINTS_COMPREHENSIVE.md${NC}"
}

function test_all_ai_api() {
    local BASE_URL="http://localhost:5000"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${CYAN}🧠 TESTING AI KNOWLEDGE AGENT (Port 5000)${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

    echo -e "${YELLOW}[AI ENDPOINTS]${NC}"
    test_api_endpoint "GET" "$BASE_URL/health" "AI Health Check"
    test_api_endpoint "POST" "$BASE_URL/api/v1/embed" "Generate Embeddings" '{"text":"Hello World"}'
    test_api_endpoint "POST" "$BASE_URL/api/v1/search" "Semantic Search" '{"query":"test","top_k":5}'

    echo -e "\n${MAGENTA}⚠️  AI Service könnte offline sein wenn nicht deployed (502)${NC}"
}

function test_all_orchestrator_api() {
    local BASE_URL="http://localhost:9000"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${CYAN}🎯 TESTING ORCHESTRATOR (Port 9000)${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

    echo -e "${YELLOW}[ORCHESTRATOR ENDPOINTS]${NC}"
    test_api_endpoint "GET" "$BASE_URL/health" "Orchestrator Health"
    test_api_endpoint "GET" "$BASE_URL/api/v1/services" "List Services"
    test_api_endpoint "POST" "$BASE_URL/api/v1/services/api/restart" "Restart Service"
    test_api_endpoint "GET" "$BASE_URL/api/v1/alerts" "Get Alerts"

    echo -e "\n${MAGENTA}⚠️  Orchestrator könnte offline sein wenn nicht deployed (502)${NC}"
}

function test_planned_endpoints() {
    local BASE_URL="http://localhost:8080"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${CYAN}🚧 TESTING PLANNED/FUTURE ENDPOINTS${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

    echo -e "${YELLOW}[GEPLANTE ENDPOINTS - Erwarten 404/502]${NC}"
    test_api_endpoint "GET" "$BASE_URL/api/v1/analytics/dashboard" "Analytics Dashboard"
    test_api_endpoint "GET" "$BASE_URL/api/v1/notifications" "Notifications"
    test_api_endpoint "GET" "$BASE_URL/api/v1/tasks" "Task Queue Status"
    test_api_endpoint "POST" "$BASE_URL/api/v1/webhooks" "Webhook Management"
    test_api_endpoint "GET" "$BASE_URL/api/v1/audit-log" "Audit Logs"
    test_api_endpoint "GET" "$BASE_URL/api/v1/reports" "Reports"
    test_api_endpoint "POST" "$BASE_URL/api/v1/ai/chat" "AI Chat"
    test_api_endpoint "GET" "$BASE_URL/api/v1/ai/models" "AI Models List"

    echo -e "\n${MAGENTA}⚠️  Diese Endpoints sind noch nicht implementiert (404/502 erwartet)${NC}"
    echo -e "${YELLOW}📖 Siehe Dokumentation für geplante Features${NC}"
}

function test_webui_connectivity() {
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${CYAN}🌐 TESTING WEBUI CONNECTIVITY${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

    test_api_endpoint "GET" "http://localhost:3000" "WebUI Frontend"
    test_api_endpoint "GET" "http://localhost:3000/api/health" "WebUI Backend Health"

    echo -e "\n${YELLOW}Wenn WebUI offline: docker compose restart webui${NC}"
}

function test_ai_embeddings_detailed() {
    local BASE_URL="http://localhost:5000"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${CYAN}🧪 DETAILED AI EMBEDDINGS TEST${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

    echo -e "${YELLOW}Testing verschiedene Texte...${NC}\n"

    local texts=(
        "Hello World"
        "This is a test for NAS.AI system"
        "Künstliche Intelligenz und Machine Learning"
        "Security vulnerability detection"
    )

    for text in "${texts[@]}"; do
        echo -e "${CYAN}Input: \"$text\"${NC}"
        response=$(curl -s -m 15 -X POST "$BASE_URL/api/v1/embed" \
            -H "Content-Type: application/json" \
            -d "{\"text\":\"$text\"}" \
            -w "\n%{http_code}")

        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | head -n-1)

        if [ "$http_code" = "200" ]; then
            echo -e "${GREEN}✓ Embedding generiert${NC}"
            echo "$body" | head -c 100
            echo "..."
        else
            echo -e "${RED}✗ Error: $http_code${NC}"
        fi
        echo ""
    done

    echo -e "${MAGENTA}⚠️  AI Service muss laufen für erfolgreiche Tests${NC}"
}

function test_database_connectivity() {
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${CYAN}💾 TESTING DATABASE CONNECTIVITY${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

    echo -ne "${CYAN}Testing PostgreSQL Connection...${NC} "
    if docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T db psql -U nas_user -d nas_db -c "SELECT 1;" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Connected${NC}"
    else
        echo -e "${RED}✗ Failed${NC}"
    fi

    echo -ne "${CYAN}Testing Redis Connection...${NC} "
    if docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T redis redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Connected${NC}"
    else
        echo -e "${RED}✗ Failed${NC}"
    fi

    echo -ne "${CYAN}Testing pgvector Extension...${NC} "
    if docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T db psql -U nas_user -d nas_db -c "SELECT * FROM pg_extension WHERE extname='vector';" | grep -q vector; then
        echo -e "${GREEN}✓ Installed${NC}"
    else
        echo -e "${RED}✗ Not Found${NC}"
    fi
}

function test_full_system() {
    echo -e "${GREEN}╔═══════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║     FULL SYSTEM TEST - ALL ENDPOINTS         ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════╝${NC}\n"

    test_all_main_api
    echo -e "\n"
    test_all_ai_api
    echo -e "\n"
    test_all_orchestrator_api
    echo -e "\n"
    test_planned_endpoints
    echo -e "\n"
    test_database_connectivity

    echo -e "\n${GREEN}════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}✓ FULL SYSTEM TEST ABGESCHLOSSEN${NC}"
    echo -e "${YELLOW}⚠️  HINWEIS: Einige Services können in Wartung sein (502/503)${NC}"
    echo -e "${YELLOW}📖 Dokumentation: $BASE_DIR/API_ENDPOINTS_COMPREHENSIVE.md${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════${NC}"

    wait_for_enter
}

# --- 5. MENUS (Metasploit Style) ---

function menu_main() {
    while true; do
        show_header
        echo -e "${BLUE}Wähle eine Kategorie:${NC}"
        echo -e "  ${YELLOW}1)${NC} 🚀 Deployment & Core Ops"
        echo -e "  ${YELLOW}2)${NC} 🧠 AI Agents & Intelligence"
        echo -e "  ${YELLOW}3)${NC} 🕵️  Forensics & Logs (Gauner-Jagd)"
        echo -e "  ${YELLOW}4)${NC} 🔧 System Utils & Clean"
        echo -e "  ${YELLOW}5)${NC} 🧪 API Testing Suite"
        echo -e "  ${RED}0) ❌ EXIT${NC}"
        echo ""
        read -p "nas-ai > " choice

        case $choice in
            1) menu_deployment ;;
            2) menu_ai ;;
            3) menu_forensics ;;
            4) menu_utils ;;
            5) menu_testing ;;
            0) echo "Bye Commander."; exit 0 ;;
            *) echo "Ungültig." ;;
        esac
    done
}

function menu_deployment() {
    while true; do
        show_header
        echo -e "${YELLOW}MODULE: DEPLOYMENT${NC}"
        echo ""
        echo -e "  ${CYAN}[FULL RESTART]${NC}"
        echo -e "  1) 🔥 Clean Restart (NO CACHE - Alles Neu)"
        echo -e "  2) ⚡ Quick Rebuild (Mit Cache, Force Recreate)"
        echo ""
        echo -e "  ${CYAN}[SERVICES]${NC}"
        echo -e "  3) 🚀 Full Prod Deployment (Standard)"
        echo -e "  4) 🔄 Restart Backend (API Only)"
        echo -e "  5) 🌐 Restart Frontend (WebUI)"
        echo -e "  6) 🧠 Restart AI Agent"
        echo ""
        echo -e "  ${CYAN}[STATUS]${NC}"
        echo -e "  7) 📊 Service Status anzeigen"
        echo ""
        echo "  0) 🔙 Zurück"
        echo ""
        read -p "nas-ai/deploy > " c
        case $c in
            1) clean_restart ;;
            2) quick_rebuild ;;
            3) deploy_full ;;
            4) docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" restart api; wait_for_enter ;;
            5) docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" build webui && docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --force-recreate webui; wait_for_enter ;;
            6) docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" restart ai-knowledge-agent; wait_for_enter ;;
            7) docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps; wait_for_enter ;;
            0) return ;;
            *) echo "Ungültig." ;;
        esac
    done
}

function menu_ai() {
    while true; do
        show_header
        echo -e "${YELLOW}MODULE: AI INTELLIGENCE${NC}"
        echo "  1) 🧠 Logs: Knowledge Agent & Orchestrator (Live)"
        echo "  2) 🧪 Test: Embeddings Endpoint"
        echo "  3) 🔄 Restart AI Cluster"
        echo "  0) 🔙 Zurück"
        echo ""
        read -p "nas-ai/brain > " c
        case $c in
            1) smart_logs "ai-knowledge-agent orchestrator"; wait_for_enter ;;
            2) echo "Führe Test-Script aus..."; "$BASE_DIR/scripts/test-embedding.sh" || echo "Test failed"; wait_for_enter ;;
            3) docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" restart ai-knowledge-agent orchestrator; wait_for_enter ;;
            0) return ;;
        esac
    done
}

function menu_forensics() {
    while true; do
        show_header
        echo -e "${YELLOW}MODULE: FORENSICS & SECURITY${NC}"
        echo "  1) 📡 Live Smart-Logs (API Core)"
        echo "  2) 🕵️  IP-Lookup / Gauner Check"
        echo "  3) 🛡️  Fail2Ban Status (Simuliert)"
        echo "  0) 🔙 Zurück"
        echo ""
        read -p "nas-ai/sec > " c
        case $c in
            1) smart_logs "api"; wait_for_enter ;;
            2) forensic_ip_check ;;
            3) echo "Status OK. 0 Bans currently."; wait_for_enter ;;
            0) return ;;
        esac
    done
}

function menu_utils() {
    while true; do
        show_header
        echo -e "${YELLOW}MODULE: UTILITIES${NC}"
        echo "  1) 🧹 Docker Prune (Alles bereinigen)"
        echo "  2) 💾 Backup Database Now"
        echo "  3) 📜 Generate API Docs (Swagger)"
        echo "  0) 🔙 Zurück"
        echo ""
        read -p "nas-ai/utils > " c
        case $c in
            1) docker system prune -af; wait_for_enter ;;
            2) echo "Backup läuft..."; docker compose exec db pg_dump -U nas_user nas_db > "$INFRA_DIR/backup_$(date +%F).sql"; echo "Done."; wait_for_enter ;;
            3) echo "Generating Swag..."; wait_for_enter ;; # Hier Befehl einfügen wenn vorhanden
            0) return ;;
        esac
    done
}

function menu_testing() {
    while true; do
        show_header
        echo -e "${YELLOW}MODULE: API TESTING SUITE${NC}"
        echo -e "${MAGENTA}⚠️  HINWEIS: Einige Services können in Wartung sein (502/503)${NC}"
        echo ""
        echo -e "  ${CYAN}[COMPREHENSIVE TESTS]${NC}"
        echo -e "  1) 🚀 Full System Test (Alle Endpoints)"
        echo -e "  2) 🧪 Test Main API (Port 8080)"
        echo -e "  3) 🧠 Test AI Knowledge Agent (Port 5000)"
        echo -e "  4) 🎯 Test Orchestrator (Port 9000)"
        echo ""
        echo -e "  ${CYAN}[SPECIALIZED TESTS]${NC}"
        echo -e "  5) 🚧 Test Planned/Future Endpoints"
        echo -e "  6) 🌐 Test WebUI Connectivity"
        echo -e "  7) 🤖 Test AI Embeddings (Detailed)"
        echo -e "  8) 💾 Test Database Connectivity"
        echo ""
        echo -e "  ${CYAN}[DOCUMENTATION]${NC}"
        echo -e "  9) 📖 Show API Documentation Path"
        echo ""
        echo -e "  0) 🔙 Zurück"
        echo ""
        read -p "nas-ai/testing > " c
        case $c in
            1) test_full_system ;;
            2) test_all_main_api; wait_for_enter ;;
            3) test_all_ai_api; wait_for_enter ;;
            4) test_all_orchestrator_api; wait_for_enter ;;
            5) test_planned_endpoints; wait_for_enter ;;
            6) test_webui_connectivity; wait_for_enter ;;
            7) test_ai_embeddings_detailed; wait_for_enter ;;
            8) test_database_connectivity; wait_for_enter ;;
            9) echo -e "${GREEN}API Dokumentation:${NC}";
               echo "  - Comprehensive: $BASE_DIR/API_ENDPOINTS_COMPREHENSIVE.md";
               echo "  - Summary: $BASE_DIR/API_ENDPOINTS_SUMMARY.md";
               echo "  - Quick Ref: $BASE_DIR/API_ENDPOINTS_QUICK_REFERENCE.txt";
               echo "  - Index: $BASE_DIR/API_DOCUMENTATION_INDEX.md";
               wait_for_enter ;;
            0) return ;;
            *) echo "Ungültig." ;;
        esac
    done
}

# --- 6. EXECUTION ---

check_preflight
hype_loader
menu_main