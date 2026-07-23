-- Migration 009: cluster-safe upload concurrency leases
CREATE TABLE IF NOT EXISTS upload_chunk_leases (
    id         BIGSERIAL PRIMARY KEY,
    task_id    UUID,
    holder     TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS upload_chunk_leases_expires_idx
    ON upload_chunk_leases (expires_at);
