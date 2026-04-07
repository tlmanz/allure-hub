-- +goose Up
ALTER TABLE upload_sessions ADD COLUMN failed_at_phase TEXT NOT NULL DEFAULT '';

-- +goose Down
