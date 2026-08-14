-- Migration: Add user roles to support RBAC
-- Date: 2025-11-29
-- Purpose: Security hardening - implement role-based access control

BEGIN;

-- Add role column to users table with default 'user' role
ALTER TABLE users
ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

-- Add check constraint to ensure only valid roles
ALTER TABLE users
ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'));

-- Create index for faster role lookups
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Make first user (if exists) an admin for initial setup
-- This ensures at least one admin exists
UPDATE users
SET role = 'admin'
WHERE id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1)
AND NOT EXISTS (SELECT 1 FROM users WHERE role = 'admin');

-- Log migration
INSERT INTO schema_migrations (version, description, applied_at)
VALUES ('002', 'Add user roles (RBAC)', NOW())
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- Rollback script (if needed):
-- BEGIN;
-- ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
-- DROP INDEX IF EXISTS idx_users_role;
-- ALTER TABLE users DROP COLUMN IF EXISTS role;
-- DELETE FROM schema_migrations WHERE version = '002';
-- COMMIT;
