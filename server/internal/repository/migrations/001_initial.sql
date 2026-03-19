-- Migration 001: Initial Schema
-- IdeaSaver Database Schema

-- Users table (synced from Authelia OAuth2)
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(64) UNIQUE NOT NULL,
    display_name  VARCHAR(128),
    email         VARCHAR(256),
    role          VARCHAR(16) DEFAULT 'user',
    storage_quota BIGINT DEFAULT 5368709120,
    storage_used  BIGINT DEFAULT 0,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Files and directories table
CREATE TABLE IF NOT EXISTS files (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    parent_id     UUID REFERENCES files(id) ON DELETE CASCADE,
    name          VARCHAR(512) NOT NULL,
    storage_key   VARCHAR(512),
    is_directory  BOOLEAN DEFAULT FALSE,
    mime_type     VARCHAR(128),
    size          BIGINT DEFAULT 0,
    public_url    VARCHAR(1024),
    thumbnail_key VARCHAR(512),
    deleted_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, parent_id, name)
);

-- Share links table
CREATE TABLE IF NOT EXISTS shares (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    file_id     UUID REFERENCES files(id) ON DELETE CASCADE,
    code        VARCHAR(32) UNIQUE NOT NULL,
    password    VARCHAR(128),
    expires_at  TIMESTAMPTZ,
    view_count  INT DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Upload tasks table (chunked/resumable uploads)
CREATE TABLE IF NOT EXISTS upload_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    filename        VARCHAR(512) NOT NULL,
    total_size      BIGINT NOT NULL,
    uploaded_size   BIGINT DEFAULT 0,
    chunk_size      INT DEFAULT 5242880,
    total_chunks    INT NOT NULL,
    uploaded_chunks INT DEFAULT 0,
    storage_key     VARCHAR(512),
    upload_id       VARCHAR(256),
    status          VARCHAR(16) DEFAULT 'pending',
    target_type     VARCHAR(16) DEFAULT 'file',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Videos table (DogeCloud VCloud)
CREATE TABLE IF NOT EXISTS videos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(512) NOT NULL,
    vid         VARCHAR(128) NOT NULL,
    vcode       VARCHAR(128),
    status      SMALLINT DEFAULT 1,
    play_url    TEXT,
    size        BIGINT DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    action      VARCHAR(64) NOT NULL,
    resource    VARCHAR(64),
    resource_id UUID,
    details     JSONB,
    ip_address  INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
CREATE INDEX IF NOT EXISTS idx_files_parent_id ON files(parent_id);
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files(deleted_at);
CREATE INDEX IF NOT EXISTS idx_videos_user_id ON videos(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_upload_tasks_user_id ON upload_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_upload_tasks_status ON upload_tasks(status);
CREATE INDEX IF NOT EXISTS idx_shares_code ON shares(code);
