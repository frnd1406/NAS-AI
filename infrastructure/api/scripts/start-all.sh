#!/bin/bash

# Start all services needed for NAS API
# - PostgreSQL
# - Redis
# - Cloudflare Tunnel (if not running)
# - API Server

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== Starting NAS API Stack ===${NC}\n"

# Change to API directory
cd /home/user/Agent/infrastructure/api

# 1. Start PostgreSQL
echo -e "${YELLOW}[1/4]${NC} Starting PostgreSQL..."
if docker ps | grep -q nas-api-postgres; then
    echo -e "  ${GREEN}✓${NC} PostgreSQL already running"
else
    docker start nas-api-postgres > /dev/null 2>&1 || {
        echo -e "  ${RED}✗${NC} Failed to start PostgreSQL"
        echo -e "  Run: docker run -d --name nas-api-postgres ..."
        exit 1
    }
    sleep 2
    if docker ps | grep -q nas-api-postgres; then
        echo -e "  ${GREEN}✓${NC} PostgreSQL started"
    else
        echo -e "  ${RED}✗${NC} PostgreSQL failed to start"
        exit 1
    fi
fi

# 2. Start Redis
echo -e "\n${YELLOW}[2/4]${NC} Starting Redis..."
if docker ps | grep -q nas-api-redis; then
    echo -e "  ${GREEN}✓${NC} Redis already running"
else
    docker start nas-api-redis > /dev/null 2>&1 || {
        echo -e "  ${RED}✗${NC} Failed to start Redis"
        exit 1
    }
    sleep 2
    if docker ps | grep -q nas-api-redis; then
        echo -e "  ${GREEN}✓${NC} Redis started"
    else
        echo -e "  ${RED}✗${NC} Redis failed to start"
        exit 1
    fi
fi

# 3. Check Cloudflare Tunnel
echo -e "\n${YELLOW}[3/4]${NC} Checking Cloudflare Tunnel..."
if systemctl is-active --quiet cloudflared 2>/dev/null; then
    echo -e "  ${GREEN}✓${NC} Cloudflare Tunnel running"
else
    echo -e "  ${YELLOW}⚠${NC} Cloudflare Tunnel not running"
    echo -e "  ${BLUE}Starting tunnel...${NC}"
    sudo systemctl start cloudflared 2>/dev/null || {
        echo -e "  ${YELLOW}⚠${NC} Could not start tunnel (may need manual start)"
    }
fi

# Wait for databases to be ready
echo -e "\n${BLUE}Waiting for databases to be ready...${NC}"
sleep 3

# Check database connectivity
echo -e "${YELLOW}[4/4]${NC} Verifying database connections..."

# Check Postgres
if docker exec nas-api-postgres pg_isready -U nas_user -d nas_db > /dev/null 2>&1; then
    echo -e "  ${GREEN}✓${NC} PostgreSQL: Ready"
else
    echo -e "  ${YELLOW}⚠${NC} PostgreSQL: Not ready yet (may need a few more seconds)"
fi

# Check Redis
if docker exec nas-api-redis redis-cli ping > /dev/null 2>&1; then
    echo -e "  ${GREEN}✓${NC} Redis: Ready"
else
    echo -e "  ${YELLOW}⚠${NC} Redis: Not ready yet"
fi

# Show status
echo -e "\n${GREEN}=== Service Status ===${NC}\n"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "nas-api|NAMES"

echo -e "\n${GREEN}=== Ready to start API ===${NC}\n"
echo -e "${BLUE}To start the API:${NC}"
echo -e "  ./scripts/start-api.sh"
echo -e ""
echo -e "${BLUE}Or run in background:${NC}"
echo -e "  nohup ./scripts/start-api.sh > logs/api.log 2>&1 &"
echo -e ""
echo -e "${BLUE}To stop all services:${NC}"
echo -e "  ./scripts/stop-all.sh"
echo ""
