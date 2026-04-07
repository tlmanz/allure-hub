-- +goose Up
ALTER TABLE api_keys ADD COLUMN auto_create_env_project INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite does not support DROP COLUMN before 3.35; leave the column in place on rollback.
