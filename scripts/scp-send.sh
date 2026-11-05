#!/bin/bash
set -euo pipefail

RED=$'\033[0;31m'
GREEN=$'\033[0;32m'
YELLOW=$'\033[1;33m'
BLUE=$'\033[0;34m'
NC=$'\033[0m'

usage() {
  echo "Usage: $0 <local_file> <user> <host> [remote_path]"
  echo "Example: $0 ./build.zip deploy 192.168.0.50 /opt/app"
  exit 1
}

[ $# -lt 1 ] && usage

LOCAL_FILE=$1
USER=${2:-user}
HOST=${3:-192.168.1.181}
REMOTE_PATH=${4:-/home/user/}

if [ ! -f "$LOCAL_FILE" ]; then
  echo "${RED}❌ Local file not found: $LOCAL_FILE${NC}" >&2
  exit 1
fi

if ! command -v scp >/dev/null 2>&1; then
  echo "${RED}❌ scp not found. Install OpenSSH client.${NC}" >&2
  exit 1
fi

TARGET="${USER}@${HOST}:${REMOTE_PATH}"

echo "${BLUE}📤 Sending ${LOCAL_FILE} to ${TARGET}${NC}"
scp -o StrictHostKeyChecking=accept-new "$LOCAL_FILE" "$TARGET"
echo "${GREEN}✅ Transfer complete.${NC}"
