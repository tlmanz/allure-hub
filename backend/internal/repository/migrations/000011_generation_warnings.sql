-- +goose Up
ALTER TABLE builds ADD COLUMN generation_warnings TEXT NOT NULL DEFAULT '[]';

-- +goose Down
