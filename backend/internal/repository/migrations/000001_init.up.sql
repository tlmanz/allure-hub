CREATE TABLE IF NOT EXISTS environments (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL
);

INSERT INTO environments (id, name, created_at)
VALUES ('default', 'Default', CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS projects (
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    id             TEXT NOT NULL,
    name           TEXT NOT NULL,
    created_at     TEXT NOT NULL,
    PRIMARY KEY (environment_id, id)
);

CREATE TABLE IF NOT EXISTS builds (
    id         TEXT PRIMARY KEY,
    env_id     TEXT NOT NULL,
    project_id TEXT NOT NULL,
    build_id   TEXT NOT NULL,
    created_at TEXT NOT NULL,
    report_url TEXT NOT NULL DEFAULT '',
    total      INTEGER NOT NULL DEFAULT 0,
    passed     INTEGER NOT NULL DEFAULT 0,
    failed     INTEGER NOT NULL DEFAULT 0,
    skipped    INTEGER NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT '',
    UNIQUE(env_id, project_id, build_id),
    FOREIGN KEY (env_id, project_id) REFERENCES projects(environment_id, id) ON DELETE CASCADE
);
