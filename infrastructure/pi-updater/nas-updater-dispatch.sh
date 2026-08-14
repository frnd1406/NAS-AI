#!/usr/bin/env bash
# Forced-command dispatcher fuer den nas-updater FIDO2-Key.
# Erlaubt AUSSCHLIESSLICH status|upgrade|reboot (kein Shell-Zugang), egal was der Client sendet.
set -uo pipefail
cmd="${SSH_ORIGINAL_COMMAND:-}"
case "$cmd" in
  status|upgrade|docker|reboot)
    exec sudo /usr/local/sbin/nas-updater.sh "$cmd"
    ;;
  *)
    echo "denied: nur 'status', 'upgrade', 'docker' oder 'reboot' erlaubt" >&2
    exit 2
    ;;
esac
