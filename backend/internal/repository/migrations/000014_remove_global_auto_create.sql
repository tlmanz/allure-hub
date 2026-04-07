-- +goose Up
DELETE FROM system_settings WHERE key = 'auto_create_env_project';

-- +goose Down
INSERT INTO system_settings (key, value, updated_at)
VALUES ('auto_create_env_project', 'false', CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;
