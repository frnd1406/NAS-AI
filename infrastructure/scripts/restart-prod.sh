#!/usr/bin/env bash
set -euo pipefail

# Farben für Output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT}/docker-compose.prod.yml"
ENV_FILE="${ROOT}/.env.prod"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo -e "${RED}ERROR: Env file not found at ${ENV_FILE}. Aborting.${NC}"
  exit 1
fi

echo -e "${YELLOW}🔄 Restarting production stack with ${COMPOSE_FILE}...${NC}"
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" down
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" up -d

echo -e "${YELLOW}⏳ Waiting for containers to start (5s)...${NC}"
sleep 5

# Validierung kritischer Container
echo -e "${YELLOW}🏥 Validating critical containers...${NC}"
CRITICAL_CONTAINERS=("postgres" "api" "webui")
FAILED_CONTAINERS=()

check_container() {
    local container_name=$1
    local status=$(docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" ps -q "$container_name" 2>/dev/null)

    if [ -z "$status" ]; then
        return 1 # Container existiert nicht
    fi

    local running=$(docker inspect -f '{{.State.Running}}' $(docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" ps -q "$container_name") 2>/dev/null)
    if [ "$running" != "true" ]; then
        return 1 # Container läuft nicht
    fi

    return 0 # Container läuft
}

for container in "${CRITICAL_CONTAINERS[@]}"; do
    if check_container "$container"; then
        echo -e "${GREEN}✅ $container${NC}"
    else
        echo -e "${RED}❌ $container - MISSING OR NOT RUNNING!${NC}"
        FAILED_CONTAINERS+=("$container")
    fi
done

if [ ${#FAILED_CONTAINERS[@]} -gt 0 ]; then
    echo -e "${RED}❌ ERROR: Critical containers missing or not running!${NC}"
    echo -e "${RED}   Affected containers: ${FAILED_CONTAINERS[*]}${NC}"
    echo -e "${YELLOW}📋 Current container status:${NC}"
    docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" ps
    echo -e "${YELLOW}📋 Logs of failed containers:${NC}"
    for container in "${FAILED_CONTAINERS[@]}"; do
        echo -e "${YELLOW}--- Logs for $container ---${NC}"
        docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" logs --tail=20 "$container" 2>/dev/null || echo "No logs available"
    done
    exit 1
fi

echo -e "${GREEN}✅ All critical containers are running!${NC}"
echo ""
echo -e "${YELLOW}📊 Current status:${NC}"
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" ps
