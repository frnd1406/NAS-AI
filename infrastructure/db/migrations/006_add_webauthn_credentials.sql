-- Migration: Add WebAuthn/FIDO2 credentials (YubiKey second factor)
-- Date: 2026-07-28
-- Purpose: Store per-user WebAuthn credentials for MFA login.
-- Note: The backend also self-provisions this table via EnsureTable() at
-- startup; this migration keeps managed/prod databases consistent.

BEGIN;

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL,
    credential_id TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL DEFAULT '',
    credential    TEXT NOT NULL,
    sign_count    BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id);

-- Log migration
INSERT INTO schema_migrations (version, description, applied_at)
VALUES ('006', 'Add WebAuthn credentials (MFA)', NOW())
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- Rollback script (if needed):
-- BEGIN;
-- DROP INDEX IF EXISTS idx_webauthn_credentials_user_id;
-- DROP TABLE IF EXISTS webauthn_credentials;
-- DELETE FROM schema_migrations WHERE version = '006';
-- COMMIT;
