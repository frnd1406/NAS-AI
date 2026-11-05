#!/bin/bash

# Start NAS API with environment variables
# This script loads .env and starts the API

set -e

cd /home/user/Agent/infrastructure/api

# Load environment variables
if [ -f .env ]; then
    echo "✓ Loading environment from .env"
    set -a
    source .env
    set +a
else
    echo "✗ .env file not found!"
    echo "  Run: ./scripts/generate-secrets.sh"
    exit 1
fi

# Check if API binary exists
if [ ! -f bin/api ]; then
    echo "✗ API binary not found at bin/api"
    echo "  Build it first: go build -o bin/api src/main.go"
    exit 1
fi

# Start API
echo "Starting NAS API..."
echo "  Port: $PORT"
echo "  Environment: $ENV"
echo "  CORS Origins: $CORS_ORIGINS"
echo ""

./bin/api
