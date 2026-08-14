-- =================================================================
-- Migration 005: Hybrid Encryption and Smart Sharing
-- =================================================================
-- Purpose: Add encryption support and file sharing capabilities
-- Date: 2025-12-25
-- Phase: 1 - Database Foundation for Hybrid Encryption
-- Spec: Master-Plan.md Section 4 (Technisches Design: Datenbank)
-- =================================================================

BEGIN;

-- =================================================================
-- 1. ENUM TYPES
-- =================================================================

-- Encryption mode determines who holds the decryption key
-- NONE   = Raw storage, no encryption (max performance)
-- SYSTEM = Server-side encryption with system key (backup recoverable)  
-- USER   = Client/User-side encryption (max security, zero-knowledge)
CREATE TYPE encryption_mode AS ENUM ('NONE', 'SYSTEM', 'USER');

-- Share type determines how a file is shared
-- LINK          = Public/password-protected share link
-- INTERNAL_USER = Share with another registered user
CREATE TYPE share_type AS ENUM ('LINK', 'INTERNAL_USER');

-- =================================================================
-- 2. FILES TABLE (Core File Metadata)
-- =================================================================

CREATE TABLE IF NOT EXISTS files (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Ownership
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- File identity
    filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(127) DEFAULT 'application/octet-stream',
    
    -- Storage location (relative path from storage root)
    storage_path TEXT NOT NULL,
    
    -- Size and integrity
    size_bytes BIGINT NOT NULL DEFAULT 0,
    checksum VARCHAR(128),  -- SHA-256 or BLAKE3 hash
    
    -- Encryption metadata
    encryption_status encryption_mode NOT NULL DEFAULT 'NONE',
    
    -- Cryptographic material for USER/SYSTEM encrypted files
    -- Contains: { nonce, salt, argon2_params, wrapped_key } as needed
    encryption_metadata JSONB,
    
    -- Audit timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Soft delete support
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

-- Performance index for encryption heuristics 
-- Query pattern: Find large files by encryption status for re-encryption jobs
CREATE INDEX IF NOT EXISTS idx_files_encryption_size 
    ON files(encryption_status, size_bytes) 
    WHERE deleted_at IS NULL;

-- Owner lookup (common query pattern)
CREATE INDEX IF NOT EXISTS idx_files_owner_id 
    ON files(owner_id) 
    WHERE deleted_at IS NULL;

-- Storage path lookup for deduplication/integrity checks
CREATE INDEX IF NOT EXISTS idx_files_storage_path 
    ON files(storage_path);

-- =================================================================
-- 3. SHARES TABLE (File Sharing)
-- =================================================================

CREATE TABLE IF NOT EXISTS shares (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- What is being shared
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    
    -- Who created the share
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Share type
    share_type share_type NOT NULL DEFAULT 'LINK',
    
    -- For LINK shares: unique URL token
    -- Example: /share/aB3xY9kL2mN
    token VARCHAR(64) UNIQUE,
    
    -- For INTERNAL_USER shares: target user
    shared_with_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    
    -- Cryptographic key material for re-wrapped encryption
    -- For USER-encrypted files, stores the re-encrypted key for this share
    -- Format: { "algorithm": "...", "wrapped_key": "...", "salt": "..." }
    encrypted_key_material JSONB,
    
    -- Password protection for LINK shares
    -- Argon2id hash of optional password
    password_hash VARCHAR(128),
    
    -- Access control
    permissions VARCHAR(20) NOT NULL DEFAULT 'read',  -- 'read', 'write', 'admin'
    
    -- Expiration
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Usage tracking
    access_count INT DEFAULT 0,
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT shares_type_check CHECK (
        (share_type = 'LINK' AND token IS NOT NULL) OR
        (share_type = 'INTERNAL_USER' AND shared_with_user_id IS NOT NULL)
    ),
    CONSTRAINT shares_permissions_check CHECK (
        permissions IN ('read', 'write', 'admin')
    )
);

-- Fast token lookup for public share access
CREATE UNIQUE INDEX IF NOT EXISTS idx_shares_token 
    ON shares(token) 
    WHERE token IS NOT NULL;

-- Find all shares for a file
CREATE INDEX IF NOT EXISTS idx_shares_file_id 
    ON shares(file_id);

-- Find shares by target user (for "Shared with me" view)
CREATE INDEX IF NOT EXISTS idx_shares_shared_with 
    ON shares(shared_with_user_id) 
    WHERE shared_with_user_id IS NOT NULL;

-- Cleanup expired shares (cron job pattern)
CREATE INDEX IF NOT EXISTS idx_shares_expires_at 
    ON shares(expires_at) 
    WHERE expires_at IS NOT NULL;

-- =================================================================
-- 4. TRIGGERS
-- =================================================================

-- Auto-update updated_at on files modification
CREATE OR REPLACE FUNCTION update_files_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_files_updated_at
    BEFORE UPDATE ON files
    FOR EACH ROW
    EXECUTE FUNCTION update_files_updated_at();

-- =================================================================
-- 5. LOGGING
-- =================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 005: Hybrid Encryption and Sharing tables created';
    RAISE NOTICE 'Created types: encryption_mode, share_type';
    RAISE NOTICE 'Created tables: files, shares';
    RAISE NOTICE 'Created performance indexes for encryption heuristics';
END $$;

COMMIT;

-- =================================================================
-- ROLLBACK SCRIPT (Emergency Use)
-- =================================================================
-- BEGIN;
-- DROP TRIGGER IF EXISTS trigger_files_updated_at ON files;
-- DROP FUNCTION IF EXISTS update_files_updated_at();
-- DROP TABLE IF EXISTS shares;
-- DROP TABLE IF EXISTS files;
-- DROP TYPE IF EXISTS share_type;
-- DROP TYPE IF EXISTS encryption_mode;
-- COMMIT;
