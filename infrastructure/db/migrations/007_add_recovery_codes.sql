-- Migration: Add recovery codes (backup codes for WebAuthn/2FA)
-- Date: 2026-07-31
-- Purpose: One-time recovery codes so a user who loses their authenticator can
--          still complete the second factor. Only bcrypt hashes are stored.
-- Note: The backend also self-provisions this table via EnsureTable() at
--       startup; this migration keeps managed/prod databases consistent.

BEGIN;

CREATE TABLE IF NOT EXISTS recovery_codes (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    code_hash  TEXT NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recovery_codes_user_id ON recovery_codes(user_id);

-- Log migration
INSERT INTO schema_migrations (version, description, applied_at)
VALUES ('007', 'Add recovery codes (2FA backup codes)', NOW())
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- Rollback script (if needed):
-- BEGIN;
-- DROP INDEX IF EXISTS idx_recovery_codes_user_id;
-- DROP TABLE IF EXISTS recovery_codes;
-- DELETE FROM schema_migrations WHERE version = '007';
-- COMMIT;
