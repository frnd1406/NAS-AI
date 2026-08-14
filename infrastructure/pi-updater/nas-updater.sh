#!/usr/bin/env bash
# NAS Homelab host updater - genau drei Aktionen: status|upgrade|reboot. Laeuft als root.
# Kein Zugriff auf Vault/Daten. Wird ueber gehaerteten SSH-Kanal (FIDO2-Touch-Key) getriggert.
set -uo pipefail
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
LOG=/var/log/nas-updater.log
logline(){ echo "$(date -u '+%F %T')Z $*" >> "$LOG" 2>/dev/null || true; }

reboot_due(){
  local runbase instbase newest
  runbase=$(uname -r | grep -oE '^[0-9]+\.[0-9]+\.[0-9]+')
  instbase=$(dpkg-query -W -f='${Version}\n' 'linux-image-rpi-*' 2>/dev/null \
             | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | sort -V | tail -1)
  if [ -f /var/run/reboot-required ]; then echo yes; return; fi
  if [ -n "$instbase" ] && [ -n "$runbase" ] && [ "$instbase" != "$runbase" ]; then
    newest=$(printf '%s\n%s\n' "$instbase" "$runbase" | sort -V | tail -1)
    [ "$newest" = "$instbase" ] && { echo yes; return; }
  fi
  echo no
}

case "${1:-}" in
  status)
    apt-get update -qq 2>/dev/null
    total=$(apt list --upgradable 2>/dev/null | grep -c 'upgradable'); total=${total:-0}
    sec=$(apt list --upgradable 2>/dev/null | grep -ci 'security'); sec=${sec:-0}
    kupd=$(apt list --upgradable 2>/dev/null | grep -c 'linux-image'); kupd=${kupd:-0}
    rb=$(reboot_due)
    action=no
    if [ "$total" -gt 0 ] || [ "$rb" = yes ]; then action=yes; fi
    echo "HOST=$(hostname)"
    echo "TOTAL_UPGRADABLE=$total"
    echo "SECURITY=$sec"
    echo "KERNEL_RUNNING=$(uname -r)"
    echo "KERNEL_UPDATE_PENDING=$([ "$kupd" -gt 0 ] && echo yes || echo no)"
    echo "REBOOT_DUE=$rb"
    echo "ACTION_NEEDED=$action"
    ;;
  upgrade)
    logline "UPGRADE start"
    export DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a
    apt-get update -qq 2>/dev/null
    if apt-get -y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold full-upgrade; then
      logline "UPGRADE ok"; echo "UPGRADE_OK"
    else
      logline "UPGRADE failed"; echo "UPGRADE_FAILED"; exit 1
    fi
    echo "REBOOT_DUE=$(reboot_due)"
    ;;
  docker)
    COMPOSE_DIR=/home/nasserver/nasai/infrastructure
    if ! command -v docker >/dev/null 2>&1 || [ ! -f "$COMPOSE_DIR/docker-compose.pi.yml" ]; then
      echo "DOCKER_SKIP: kein Docker/Compose auf diesem Host"; exit 0
    fi
    logline "DOCKER pull+recreate start"
    cd "$COMPOSE_DIR" || { echo "DOCKER_FAILED: cd"; exit 1; }
    # Nur externe Images (postgres+redis); die lokal gebaute 'api' bleibt unangetastet.
    if docker compose -f docker-compose.pi.yml --env-file .env.pi pull postgres redis \
       && docker compose -f docker-compose.pi.yml --env-file .env.pi up -d postgres redis; then
      docker image prune -f >/dev/null 2>&1 || true
      logline "DOCKER ok"; echo "DOCKER_OK"
    else
      logline "DOCKER failed"; echo "DOCKER_FAILED"; exit 1
    fi
    ;;
  reboot)
    logline "REBOOT requested"; echo "REBOOTING"
    ( sleep 2; systemctl reboot ) >/dev/null 2>&1 &
    ;;
  *)
    echo "usage: nas-updater.sh {status|upgrade|docker|reboot}" >&2; exit 2 ;;
esac
