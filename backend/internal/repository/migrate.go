package repository

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

//go:embed migrations/*.up.sql
var migrationsFS embed.FS

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
)`

func migrate(db *DB, log *zap.Logger) error {
	if _, err := db.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, ".up.sql")

		var count int
		if err := db.QueryRow(db.Ph(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`), version).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if count > 0 {
			continue
		}

		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := db.Exec(
			db.Ph(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`),
			version, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		log.Info("applied migration", zap.String("version", version))
	}
	return nil
}
