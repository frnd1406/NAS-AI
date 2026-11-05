#!/usr/bin/env bash
# smb-monitor.sh - report UFW blocks on port 445 and failed SMB logins to NAS-AI.
#
# Reads only NEW journal entries (cursor files in $STATE_DIR), counts events per
# source IP and POSTs them to $API_URL (POST /api/v1/system/events).
# The monitoring token is read from $MONITORING_TOKEN_FILE only; it is never
# printed, logged or passed as a command-line argument.
# Disabled unless SMB_MONITOR_ENABLED=true (see /etc/default/smb-monitor).
set -euo pipefail
umask 077

SMB_MONITOR_ENABLED="${SMB_MONITOR_ENABLED:-false}"
API_URL="${API_URL:-http://192.168.178.81:8080/api/v1/system/events}"
MONITORING_TOKEN_FILE="${MONITORING_TOKEN_FILE:-}"
CACERT="${CACERT:-}"   # optional: pinned server cert / private root CA for https API_URL
STATE_DIR="${STATE_DIR:-/var/lib/smb-monitor}"
WINDOW_SECONDS="${WINDOW_SECONDS:-30}"
SOURCE_NAME="${SOURCE_NAME:-$(hostname -s | tr -c 'A-Za-z0-9._\n-' '_')}"

log() { logger -t smb-monitor -- "$*" 2>/dev/null || echo "smb-monitor: $*" >&2; }

case "${SMB_MONITOR_ENABLED,,}" in
  true|1|yes|on) ;;
  *) exit 0 ;;   # disabled (default)
esac

if [[ -z "$MONITORING_TOKEN_FILE" || ! -r "$MONITORING_TOKEN_FILE" ]]; then
  log "MONITORING_TOKEN_FILE not set or not readable; nothing sent"
  exit 1
fi
mkdir -p "$STATE_DIR"

# Token: accept a raw token file or a KEY=VALUE env file (MONITORING_TOKEN=...).
read_token() {
  local t
  if grep -q '^MONITORING_TOKEN=' "$MONITORING_TOKEN_FILE"; then
    t="$(grep -m1 '^MONITORING_TOKEN=' "$MONITORING_TOKEN_FILE" | cut -d= -f2-)"
  else
    t="$(head -n1 "$MONITORING_TOKEN_FILE")"
  fi
  t="${t%\"}"; t="${t#\"}"; t="${t%\'}"; t="${t#\'}"
  printf '%s' "$t"
}
TOKEN="$(read_token)"
[[ -n "$TOKEN" ]] || { log "token file is empty"; exit 1; }

# Read new journal lines for one stream. Echo lines; store the new cursor in
# "$cursor_file.new" so it is only committed after a successful send.
read_journal() {
  local cursor_file="$1"; shift
  local out args=("$@")
  if [[ -s "$cursor_file" ]]; then
    args+=(--after-cursor="$(cat "$cursor_file")")
  else
    args+=(--since="-${WINDOW_SECONDS}s")
  fi
  out="$(journalctl "${args[@]}" --no-pager --show-cursor -o cat 2>/dev/null || true)"
  grep -o '^-- cursor: .*' <<<"$out" | head -n1 | sed 's/^-- cursor: //' >"$cursor_file.new" || true
  grep -v '^-- cursor: ' <<<"$out" || true
}

commit_cursor() {
  [[ -s "$1.new" ]] && mv -f "$1.new" "$1" || rm -f "$1.new"
}

send() {  # type ip count
  local body code
  body="$(printf '{"source":"%s","type":"%s","src_ip":"%s","count":%d,"window_seconds":%d}' \
          "$SOURCE_NAME" "$1" "$2" "$3" "$WINDOW_SECONDS")"
  # Header goes through a process-substitution fd, so the token is not in argv.
  local ca=(); [[ -n "$CACERT" ]] && ca=(--cacert "$CACERT")
  code="$(curl -sS "${ca[@]}" -o /dev/null -w '%{http_code}' --max-time 10 \
          -H @<(printf 'X-Monitoring-Token: %s\n' "$TOKEN") \
          -H 'Content-Type: application/json' -d "$body" "$API_URL" 2>/dev/null || echo 000)"
  if [[ "$code" != "201" ]]; then
    log "send failed type=$1 ip=$2 http=$code"
    return 1
  fi
}

rc=0
report() {  # type, counts on stdin as "<n> <ip>"
  local type="$1" n ip
  while read -r n ip; do
    [[ -n "${ip:-}" ]] || continue
    send "$type" "$ip" "$n" || rc=1
  done
}

# 1) UFW blocks on 445 (kernel log)
kc="$STATE_DIR/cursor"
read_journal "$kc" -k | grep -E 'UFW (LIMIT )?BLOCK' | grep -E 'DPT=445( |$)' \
  | grep -oE ' SRC=[0-9a-fA-F.:]+' | sed 's/ SRC=//' | sort | uniq -c | awk '{print $1, $2}' \
  | report ufw_block || true

# 2) smbd authentication failures
sc="$STATE_DIR/cursor.smbd"
read_journal "$sc" -u smbd -u smb | grep -E 'NT_STATUS_(WRONG_PASSWORD|LOGON_FAILURE|NO_SUCH_USER|ACCOUNT_(LOCKED_OUT|DISABLED))' \
  | grep -oE 'ipv[46]:[0-9a-fA-F.:]+:[0-9]+' | sed -E 's/^ipv[46]://; s/:[0-9]+$//' \
  | sort | uniq -c | awk '{print $1, $2}' | report smb_auth_fail || true

if [[ $rc -eq 0 ]]; then
  commit_cursor "$kc"; commit_cursor "$sc"
else
  rm -f "$kc.new" "$sc.new"   # keep old cursors: retry the same entries next run
fi
exit $rc
