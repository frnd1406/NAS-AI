# Changelog

## v2.2.0
- UPS monitoring: new `GET /api/v1/system/hardware/ups` endpoint reading NUT over TCP (read-only), exposing online / on-battery / low-battery state.
- Power alert banner and live UPS status card in the dashboard: on-battery warning, low-battery escalation, and an all-clear once mains power returns.

## v2.1.1
- Production TLS via a private Root CA (replaces self-signed / trust-on-first-use).
- Hardened API container: runs unprivileged (`nobody`, `cap_drop: ALL`, `no-new-privileges`).
- API bound to the LAN interface only (public IPv6 listener dropped).
- Removed the full host-root bind mount from the API container.
- Zero-knowledge vault: XChaCha20-Poly1305 (AEAD) with Argon2id key derivation, DEK kept in RAM only.
- AI knowledge module made optional (public routes disabled by default).
- Automated PostgreSQL backup scheduling (daily, with retention).

## v1.0.0
- Initial Release: Core Storage Engine, JWT Auth System, Postgres Persistence.
- Automated Backup Scheduling with retention and configurable destination.
- UI Overhaul (Nebula) for files, backups, and alerts.
