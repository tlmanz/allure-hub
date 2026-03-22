-- Create environments table
CREATE TABLE IF NOT EXISTS environments (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- Seed a default environment so existing projects remain valid.
-- Uses standard SQL compatible with both SQLite (3.24+) and PostgreSQL.
INSERT INTO environments (id, name, created_at)
VALUES ('default', 'Default', CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- Add environment_id to projects (defaults to 'default' for all existing rows)
ALTER TABLE projects
ADD COLUMN environment_id TEXT NOT NULL DEFAULT 'default'
REFERENCES environments(id) ON DELETE CASCADE;
