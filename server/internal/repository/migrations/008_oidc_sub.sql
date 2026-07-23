-- Migration 003: stable OIDC subject binding (idempotent)
ALTER TABLE users ADD COLUMN IF NOT EXISTS oidc_sub VARCHAR(255);

-- Unique when present so Upsert can ON CONFLICT (oidc_sub)
CREATE UNIQUE INDEX IF NOT EXISTS users_oidc_sub_uidx ON users (oidc_sub)
    WHERE oidc_sub IS NOT NULL AND oidc_sub <> '';
