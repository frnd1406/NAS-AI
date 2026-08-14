-- NAS.AI Database Schema
-- Phase 2: Authentication & Users

-- Enable pgcrypto for gen_random_uuid
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',  -- RBAC: user or admin
    email_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT username_min_length CHECK (length(username) >= 3),
    CONSTRAINT email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'))  -- Valid roles only
);

-- Indexes for fast lookups
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users(email_verified);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);  -- RBAC index

-- Refresh tokens table (optional - for tracking)
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked BOOLEAN DEFAULT FALSE,

    -- Index for fast lookups
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for users table
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert a test user (password: "testpassword123" hashed with bcrypt cost 12)
-- Password hash: $2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyE3H.6K0s3i
INSERT INTO users (username, email, password_hash)
VALUES (
    'testuser',
    'test@example.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyE3H.6K0s3i'
) ON CONFLICT DO NOTHING;

-- Log initialization
DO $$
BEGIN
    RAISE NOTICE 'Database schema initialized successfully';
    RAISE NOTICE 'Test user created: test@example.com / testpassword123';
END $$;

-- Monitoring samples (CPU/RAM telemetry)
CREATE TABLE IF NOT EXISTS monitoring_samples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    cpu_percent NUMERIC(5,2) NOT NULL,
    ram_percent NUMERIC(5,2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_monitoring_samples_created_at ON monitoring_samples(created_at DESC);

-- System metrics (agent-reported)
CREATE TABLE IF NOT EXISTS system_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id TEXT NOT NULL,
    cpu_usage NUMERIC(5,2) NOT NULL,
    ram_usage NUMERIC(5,2) NOT NULL,
    disk_usage NUMERIC(5,2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_metrics_created_at ON system_metrics(created_at DESC);

-- System alerts (Phase 3)
CREATE TABLE IF NOT EXISTS system_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    is_resolved BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ai_analysis TEXT
);

CREATE INDEX IF NOT EXISTS idx_system_alerts_open ON system_alerts(is_resolved, created_at DESC);

-- Phase 6: ensure ai_analysis column exists
ALTER TABLE system_alerts ADD COLUMN IF NOT EXISTS ai_analysis TEXT;

-- Phase 2.1: Vector-DB Integration (1024D)
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS file_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id VARCHAR(255) NOT NULL,
    chunk_index INT NOT NULL DEFAULT 0,
    content TEXT NOT NULL,
    embedding vector(1024),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(file_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_file_embeddings_file_id ON file_embeddings(file_id);
CREATE INDEX IF NOT EXISTS idx_file_embeddings_created_at ON file_embeddings(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_file_embeddings_embedding_hnsw
    ON file_embeddings USING hnsw (embedding vector_cosine_ops);
