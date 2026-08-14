#!/usr/bin/env bash
set -euo pipefail

# CI gate: ensure JWT_SECRET is present and >= 32 chars.
if [[ -z "${JWT_SECRET:-}" ]]; then
  echo "ERROR: JWT_SECRET is not set"
  exit 1
fi

len=${#JWT_SECRET}

if (( len < 32 )); then
  echo "ERROR: JWT_SECRET too short (${len} chars). Must be >= 32."
  exit 1
fi

echo "JWT_SECRET length OK (${len} chars)"
