-- Migration 002: File moderation and appeal workflow

ALTER TABLE files
    ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(16) NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS moderation_reason TEXT,
    ADD COLUMN IF NOT EXISTS moderated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_files_moderation_status ON files(moderation_status);

CREATE TABLE IF NOT EXISTS file_appeals (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id       UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status        VARCHAR(16) NOT NULL DEFAULT 'pending',
    reason        TEXT NOT NULL,
    admin_comment TEXT,
    reviewed_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_file_appeals_file_id ON file_appeals(file_id);
CREATE INDEX IF NOT EXISTS idx_file_appeals_user_id ON file_appeals(user_id);
CREATE INDEX IF NOT EXISTS idx_file_appeals_status ON file_appeals(status);
CREATE INDEX IF NOT EXISTS idx_file_appeals_created_at ON file_appeals(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_file_appeals_pending_unique ON file_appeals(file_id) WHERE status = 'pending';
