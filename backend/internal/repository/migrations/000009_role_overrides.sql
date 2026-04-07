-- +goose Up
CREATE TABLE IF NOT EXISTS role_overrides (
    email       TEXT PRIMARY KEY,
    role        TEXT NOT NULL,
    permissions TEXT NOT NULL DEFAULT '[]'
);

-- +goose Down
DROP TABLE IF EXISTS role_overrides;
