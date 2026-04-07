-- +goose Up
ALTER TABLE builds          ADD COLUMN uploaded_by TEXT NOT NULL DEFAULT '';
ALTER TABLE upload_sessions ADD COLUMN uploaded_by TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN on older versions; left intentionally blank.
-- On PostgreSQL: ALTER TABLE builds DROP COLUMN uploaded_by;
--                ALTER TABLE upload_sessions DROP COLUMN uploaded_by;
