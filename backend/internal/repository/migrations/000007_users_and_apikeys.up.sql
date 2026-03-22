-- Tracks OAuth users who have logged into allure-hub.
-- Created on first login; last_login_at updated on every subsequent login.
CREATE TABLE IF NOT EXISTS tracked_users (
    email          TEXT PRIMARY KEY,
    name           TEXT NOT NULL DEFAULT '',
    avatar_url     TEXT NOT NULL DEFAULT '',
    provider       TEXT NOT NULL DEFAULT '',
    role           TEXT NOT NULL DEFAULT '',
    first_login_at TEXT NOT NULL,
    last_login_at  TEXT NOT NULL
);

-- Long-lived bearer tokens for programmatic access (CI/CD pipelines).
-- Only the SHA-256 hash of the plaintext key is stored.
CREATE TABLE IF NOT EXISTS api_keys (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    created_by   TEXT NOT NULL,
    role         TEXT NOT NULL DEFAULT 'developer',
    key_hash     TEXT NOT NULL UNIQUE,
    last_used_at TEXT,
    created_at   TEXT NOT NULL,
    expires_at   TEXT,
    is_active    INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash     ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_active   ON api_keys(is_active);
