-- Migration 010: JWT logout invalidation via token_version
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INT NOT NULL DEFAULT 0;
