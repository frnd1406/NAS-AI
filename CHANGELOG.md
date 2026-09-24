# Changelog

## Unreleased
- Security (file storage API): `GET /api/v1/files/content` now requires authentication and is scoped to the caller's home (it was public and read from the shared storage root).
- Security: fixed a race in the storage handlers where concurrent requests from different users could operate on each other's homes.
- Security: smart-download resolves paths inside the caller's home instead of the shared root.
- Security: trash IDs and rename targets can no longer contain path segments (`..` could delete the whole home or move files into another user's home).
- ZIP downloads skip symlinks; `Content-Disposition` file names are encoded (RFC 2231).
- Repaired the failing `handlers/files` and integration tests; added regression tests for each fix.
- Hardened ZIP extraction against archives with forged size headers; partial files are removed on failure.
- Orchestrator: fixed a data race in `ServiceRegistry.List` and replaced stale tests that no longer compiled.
- CI: build/test for all Go modules (WebUI is paused and excluded), golangci-lint for new issues, secret scanning (gitleaks), gosec, govulncheck, dependency review and OpenSSF Scorecard; GitHub Actions are pinned to commit SHAs.
- Repository rules as code: branch/tag rulesets, repository flags and commit rules (author, Conventional Commits, branch names) are defined under `.github/` and applied automatically.
- Multi-contributor workflow: CONTRIBUTING.md, task/bug issue forms, PR hygiene (area and size labels, overlap warnings between open PRs), stale-PR cleanup.
- `make security-scan` runs gitleaks even when gosec reports findings and scans the whole history.
- Security: a leaked Cloudflare API token and sample tokens were purged from the docs and the git history.

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
