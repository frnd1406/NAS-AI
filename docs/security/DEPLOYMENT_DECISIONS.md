# Deployment decisions and known trade-offs

Deliberate choices in the container setup, with the reasoning and the
conditions under which they must be revisited.

## API container starts as root (`user: root`)

`docker-compose.pi.yml` starts `nas-api` as root. `src/entrypoint.sh` fixes
the ownership of the mounted data directories and then drops to `nobody` via
`su-exec`. Until then the container holds `CHOWN`, `DAC_OVERRIDE`, `SETUID`
and `SETGID`; everything else is dropped (`cap_drop: ALL`, `no-new-privileges`).
The API process itself never runs as root.

It uses `network_mode: host` for mDNS advertising, so the API is bound to the
LAN address only (`BIND_ADDR=${LAN_IP}`) and protected by the host firewall.
Revisit if the data directories can be pre-chowned at provisioning time: then
`user: nobody` and an empty `cap_add` become possible.

## PostgreSQL without TLS (`sslmode=disable`)

PostgreSQL is published on `127.0.0.1` only (pi/prod) and the dev stack uses
the private compose network. The connection never leaves the host. Revisit and
enable `sslmode=verify-full` if the database ever moves to another host or is
published on a non-loopback address.

## Opt-in TLS verification skips

`REDIS_TLS_SKIP_VERIFY` and the internal HTTP client skip (`INTERNAL_TLS_SKIP_VERIFY`)
exist for internal self-signed certificates. Both default to off and must stay
off in production; verify with `grep -r SKIP_VERIFY` on the deployed `.env`.
Prefer trusting the private root CA over enabling them.

## Image versions

Images are pinned to minor versions (`pgvector:0.8.7-pg16`, `redis:7.4-alpine`)
so rebuilds are reproducible. `chromadb/chroma` is still unpinned in
`docker-compose.pi.yml`: pin it to the version that produced the data in
`${SSD_PATH}/chroma` (`docker image inspect chromadb/chroma`) to avoid an
unannounced storage-format upgrade.

### Major upgrades (not done automatically)

- **PostgreSQL 16 → 17/18:** requires `pg_dump`/restore or `pg_upgrade`; the
  data directory is not compatible across majors. Take a backup with
  `infrastructure/db-backup.sh`, restore into a new volume, then switch the image.
- **Python 3.11 → 3.13 (`ai_knowledge_agent`):** supported until Oct 2027.
  Rebuild the image and run `batch_index.py` against a copy first;
  `psycopg2-binary` has no musl wheels and is compiled during the build.
