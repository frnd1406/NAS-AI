-- Migration: Add email verification fields to users table
-- Date: 2025-11-23
-- Purpose: Support email verification flow for user registration

-- Add email_verified column (default FALSE for existing users)
ALTER TABLE users
ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT FALSE;

-- Add verified_at column (NULL until email is verified)
ALTER TABLE users
ADD COLUMN IF NOT EXISTS verified_at TIMESTAMP WITH TIME ZONE;

-- Create index on email_verified for faster queries
CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users(email_verified);

-- Log migration
DO $$
BEGIN
    RAISE NOTICE 'Migration 001: Email verification columns added to users table';
END $$;
