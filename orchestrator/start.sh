#!/bin/bash
set -e

# Orchestrator startup script

# Load environment
API_URL=${API_URL:-http://localhost:8080}

echo "Starting Orchestrator..."
echo "  API URL: $API_URL"

# Build
go build -o bin/orchestrator orchestrator_loop.go

# Run
export API_URL=$API_URL
./bin/orchestrator
