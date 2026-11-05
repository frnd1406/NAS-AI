#!/bin/bash

# Stop all NAS API services

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}=== Stopping NAS API Stack ===${NC}\n"

# Stop API (find and kill process)
echo -e "${YELLOW}[1/3]${NC} Stopping API..."
API_PID=$(pgrep -f "bin/api" || echo "")
if [ -n "$API_PID" ]; then
    kill $API_PID 2>/dev/null && echo -e "  ${GREEN}✓${NC} API stopped (PID: $API_PID)" || echo -e "  ${YELLOW}⚠${NC} Could not stop API"
else
    echo -e "  ${YELLOW}⚠${NC} API not running"
fi

# Stop PostgreSQL
echo -e "\n${YELLOW}[2/3]${NC} Stopping PostgreSQL..."
if docker ps | grep -q nas-api-postgres; then
    docker stop nas-api-postgres > /dev/null 2>&1
    echo -e "  ${GREEN}✓${NC} PostgreSQL stopped"
else
    echo -e "  ${YELLOW}⚠${NC} PostgreSQL not running"
fi

# Stop Redis
echo -e "\n${YELLOW}[3/3]${NC} Stopping Redis..."
if docker ps | grep -q nas-api-redis; then
    docker stop nas-api-redis > /dev/null 2>&1
    echo -e "  ${GREEN}✓${NC} Redis stopped"
else
    echo -e "  ${YELLOW}⚠${NC} Redis not running"
fi

echo -e "\n${GREEN}✓ All services stopped${NC}\n"
