-- +goose Up
ALTER TABLE environments ADD COLUMN icon TEXT NOT NULL DEFAULT 'deployed_code';

-- +goose Down
