#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT}/docker-compose.prod.yml"
ENV_FILE="${ROOT}/.env.prod"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "ERROR: Env file not found at ${ENV_FILE}. Aborting."
  exit 1
fi

services="${*:-api webui postgres redis monitoring analysis-agent pentester-agent}"

echo "Tailing logs for: ${services}"
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" logs -f --tail=100 ${services}
