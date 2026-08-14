#!/usr/bin/env bash
# NAS.AI - Installer fuer Raspberry Pi (ARM64, Single-Node).
# Auf dem Pi im Ordner infrastructure/ ausfuehren:  ./install-pi.sh
#
# Erwartet: SSD unter $SSD_PATH (schnell -> DB/Redis/Chroma/Cert),
#           RAID unter $RAID_PATH (gross -> Dateien/Backups).
# Erzeugt einmalig .env mit frischen Secrets, baut nativ (arm64) und startet.
set -euo pipefail
cd "$(dirname "$0")"

SSD_PATH="${SSD_PATH:-/mnt/ssd}"
RAID_PATH="${RAID_PATH:-/mnt/raid}"
LAN_HOST="${LAN_HOST:-nas.local}"
COMPOSE="docker-compose.pi.yml"
ENV_FILE=".env.pi"

echo "== NAS.AI Pi-Installer =="
echo "   SSD_PATH=$SSD_PATH   RAID_PATH=$RAID_PATH   LAN_HOST=$LAN_HOST"

# 1) Voraussetzungen ---------------------------------------------------------
command -v docker >/dev/null 2>&1 || { echo "FEHLER: Docker fehlt."; exit 1; }
docker compose version >/dev/null 2>&1 || { echo "FEHLER: docker compose (v2) fehlt."; exit 1; }
command -v openssl >/dev/null 2>&1 || { echo "FEHLER: openssl fehlt (fuer Secrets)."; exit 1; }

arch="$(uname -m)"
[ "$arch" = "aarch64" ] || [ "$arch" = "arm64" ] || echo "WARN: Architektur $arch ist nicht arm64 - Images muessen dazu passen."

# 2) Verzeichnisse auf SSD/RAID ---------------------------------------------
for d in "$SSD_PATH/postgres" "$SSD_PATH/redis" "$SSD_PATH/chroma" "$SSD_PATH/dev-tls" \
         "$RAID_PATH/data" "$RAID_PATH/backups"; do
  mkdir -p "$d"
done
mountpoint -q "$SSD_PATH"  || echo "WARN: $SSD_PATH ist kein Mountpoint (liegt evtl. auf der SD-Karte)."
mountpoint -q "$RAID_PATH" || echo "WARN: $RAID_PATH ist kein Mountpoint (RAID nicht gemountet?)."

# 3) Secrets (einmalig erzeugen) --------------------------------------------
if [ ! -f "$ENV_FILE" ]; then
  echo "Erzeuge $ENV_FILE mit frischen Secrets ..."
  umask 077
  {
    echo "SSD_PATH=$SSD_PATH"
    echo "RAID_PATH=$RAID_PATH"
    echo "LAN_HOST=$LAN_HOST"
    echo "DB_PASSWORD=$(openssl rand -hex 24)"
    echo "JWT_SECRET=$(openssl rand -hex 32)"
    echo "MONITORING_TOKEN=$(openssl rand -hex 24)"
    echo "INTERNAL_API_SECRET=$(openssl rand -base64 32)"
  } > "$ENV_FILE"
  chmod 600 "$ENV_FILE"
else
  echo "$ENV_FILE existiert bereits - vorhandene Secrets bleiben."
fi

# 4) Bauen (nativ arm64) + starten ------------------------------------------
echo "Baue Images nativ auf dem Pi (dauert beim ersten Mal) ..."
docker compose -f "$COMPOSE" --env-file "$ENV_FILE" build
echo "Starte Stack ..."
docker compose -f "$COMPOSE" --env-file "$ENV_FILE" up -d

echo
echo "Fertig."
echo "  api  (TLS + mDNS): https://$LAN_HOST:8080"
echo "  webui:             http://$LAN_HOST:3001"
echo "  mDNS-Dienst:       _nasai._tcp  ->  Desktop-App 'Im Netzwerk suchen' findet den Server."
echo
echo "Status:  docker compose -f $COMPOSE --env-file $ENV_FILE ps"
echo "Logs:    docker compose -f $COMPOSE --env-file $ENV_FILE logs -f api"
