-- =================================================================
-- Migration 003: Enable pgvector Extension for AI Embeddings
-- =================================================================
-- Purpose: Enable vector operations for semantic search and AI features
-- Date: 2025-11-29
-- Phase: 2.1 - Vector-DB Integration
-- =================================================================

BEGIN;

-- Create pgvector extension for embedding storage and similarity search
CREATE EXTENSION IF NOT EXISTS vector;

-- Log successful installation
DO $$
BEGIN
  RAISE NOTICE 'pgvector extension enabled successfully';
  RAISE NOTICE 'Vector type is now available for embedding storage';
  RAISE NOTICE 'Distance functions available: <-> (L2), <#> (inner product), <=> (cosine)';
END $$;

COMMIT;

-- Verification query (run after migration)
-- SELECT * FROM pg_extension WHERE extname = 'vector';
