-- +goose Up
CREATE TABLE IF NOT EXISTS system_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cleanup_runs (
    id            TEXT    PRIMARY KEY,
    started_at    TEXT    NOT NULL,
    finished_at   TEXT    NOT NULL,
    status        TEXT    NOT NULL,
    deleted_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    dry_run       INTEGER NOT NULL DEFAULT 0,
    error_message TEXT    NOT NULL DEFAULT ''
);

-- Seed cleanup worker defaults.
INSERT INTO system_settings (key, value, updated_at) VALUES ('retention_days',          '90',    CURRENT_TIMESTAMP) ON CONFLICT (key) DO NOTHING;
INSERT INTO system_settings (key, value, updated_at) VALUES ('cleanup_interval_hours',  '6',     CURRENT_TIMESTAMP) ON CONFLICT (key) DO NOTHING;
INSERT INTO system_settings (key, value, updated_at) VALUES ('cleanup_dry_run',         'false', CURRENT_TIMESTAMP) ON CONFLICT (key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS cleanup_runs;
DROP TABLE IF EXISTS system_settings;
