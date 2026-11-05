#!/usr/bin/env bash
# db-backup.sh - automatisches Postgres-Backup (pg_dump) mit Aufbewahrung.
# Laeuft als nasserver per Cron. Nutzt die Container-eigenen DB-Credentials.
set -euo pipefail

BACKUP_DIR="/mnt/nvme/db-backups"
RETENTION=14              # so viele juengste Dumps behalten
CONTAINER="nas-postgres"

mkdir -p "$BACKUP_DIR"
TS=$(date +%Y%m%d-%H%M%S)
OUT="$BACKUP_DIR/nas_db_${TS}.sql.gz"

# Dump erzeugen (Container liefert POSTGRES_USER/DB selbst -> keine geratenen Creds)
if docker exec "$CONTAINER" sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' 2>/dev/null | gzip > "$OUT"; then
  # Leeres/kaputtes Dump abfangen (min. 200 Bytes)
  SZ=$(stat -c%s "$OUT" 2>/dev/null || echo 0)
  if [ "$SZ" -lt 200 ]; then
    echo "$(date '+%F %T') FEHLER: Dump zu klein ($SZ B) -> geloescht" >&2
    rm -f "$OUT"; exit 1
  fi
  echo "$(date '+%F %T') OK  $OUT  ($(du -h "$OUT" | cut -f1))"
else
  echo "$(date '+%F %T') FEHLER: pg_dump fehlgeschlagen" >&2
  rm -f "$OUT"; exit 1
fi

# Aufbewahrung: aeltere als die $RETENTION juengsten loeschen
ls -1t "$BACKUP_DIR"/nas_db_*.sql.gz 2>/dev/null | tail -n +$((RETENTION+1)) | xargs -r rm -f
COUNT=$(ls -1 "$BACKUP_DIR"/nas_db_*.sql.gz 2>/dev/null | wc -l)
echo "$(date '+%F %T') Aufbewahrung: $COUNT Dumps im Ordner (max $RETENTION)"
