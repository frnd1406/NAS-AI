#!/bin/sh
set -e

# Ownership der gemounteten Volumes fuer den App-User (nobody:nogroup)
chown -R nobody:nogroup /mnt/data /mnt/backups 2>/dev/null || true
# Dev-TLS-Verzeichnis muss fuer nobody les-/schreibbar sein (Cert-Persistenz/TOFU)
chown -R nobody:nogroup /app/dev-certs 2>/dev/null || true

# root nur fuer den chown oben; die API laeuft danach ENTPRIVILEGIERT als nobody:nogroup
exec su-exec nobody:nogroup /app/api
