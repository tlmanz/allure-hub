-- +goose Up
CREATE TABLE IF NOT EXISTS upload_sessions (
    id              TEXT PRIMARY KEY,
    upload_id       TEXT NOT NULL DEFAULT '',
    build_id        TEXT NOT NULL,
    project_id      TEXT NOT NULL,
    env_id          TEXT NOT NULL,
    file_name       TEXT NOT NULL DEFAULT '',
    total_size      INTEGER NOT NULL DEFAULT 0,
    total_chunks    INTEGER NOT NULL DEFAULT 0,
    received_chunks INTEGER NOT NULL DEFAULT 0,
    phase           TEXT NOT NULL DEFAULT 'uploading',
    error           TEXT NOT NULL DEFAULT '',
    started_at      TEXT NOT NULL,
    completed_at    TEXT,
    report_url      TEXT NOT NULL DEFAULT '',
    passed          INTEGER NOT NULL DEFAULT 0,
    failed          INTEGER NOT NULL DEFAULT 0,
    skipped         INTEGER NOT NULL DEFAULT 0,
    total           INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_upload_sessions_started_at ON upload_sessions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_upload_id  ON upload_sessions(upload_id);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_project_build ON upload_sessions(project_id, build_id);

-- +goose Down
DROP TABLE IF EXISTS upload_sessions;
