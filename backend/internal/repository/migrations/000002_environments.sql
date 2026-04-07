-- +goose Up
-- environments table and projects.environment_id are now part of 000001_init.sql.
-- These statements are kept as no-ops for any database that ran the old 000001.
CREATE TABLE IF NOT EXISTS environments (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL
);

INSERT INTO environments (id, name, created_at)
VALUES ('default', 'Default', CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
