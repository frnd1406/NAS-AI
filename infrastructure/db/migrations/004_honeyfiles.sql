-- Migration: 004_honeyfiles
-- Description: Adds tables for Honeyfile intrusion detection and forensic events

-- 1. Honeyfiles Table (The Trap Markers)
CREATE TABLE IF NOT EXISTS honeyfiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_path VARCHAR(512) NOT NULL UNIQUE,
    file_type VARCHAR(50) NOT NULL DEFAULT 'general',
    trigger_count INT DEFAULT 0,
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    CONSTRAINT honeyfiles_type_check CHECK (file_type IN ('finance', 'it', 'private', 'general'))
);

-- 2. Honeyfile Events (Forensic Audit Trail)
-- Captures WHO touched the file and HOW
CREATE TABLE IF NOT EXISTS honeyfile_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    honeyfile_id UUID NOT NULL REFERENCES honeyfiles(id) ON DELETE CASCADE,
    triggered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Network Forensics
    ip_address VARCHAR(45) NOT NULL, -- IPv4 or IPv6
    user_agent TEXT,
    
    -- Account Forensics
    user_id UUID REFERENCES users(id),
    
    -- Action Context
    action VARCHAR(50) NOT NULL, -- 'download', 'open', 'list', 'delete'
    metadata JSONB -- Flexible field for extra details (e.g. request headers)
);

-- Index for fast forensic analysis
CREATE INDEX IF NOT EXISTS idx_honeyfile_events_honeyfile_id ON honeyfile_events(honeyfile_id);
CREATE INDEX IF NOT EXISTS idx_honeyfile_events_triggered_at ON honeyfile_events(triggered_at DESC);
