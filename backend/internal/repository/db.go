// Package repository contains SQL implementations of the domain repository interfaces.
package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	// SQLite driver — pure Go, no CGO required.
	_ "modernc.org/sqlite"

	// PostgreSQL driver — registers as "pgx".
	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB wraps *sql.DB and exposes the placeholder rewriter used by all repos.
type DB struct {
	*sql.DB
	driver string
}

// PoolConfig holds connection pool settings for the database.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Open opens the metadata database, runs all pending migrations, and returns a DB.
// driver must be "sqlite" or "postgres".
func Open(driver, dsn string, pool PoolConfig, log *zap.Logger) (*DB, error) {
	var driverName string
	switch driver {
	case "sqlite":
		driverName = "sqlite"
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		if !strings.Contains(dsn, "_fk=") {
			dsn += sep + "_fk=on&_journal=WAL&_busy_timeout=5000"
		}
	case "postgres":
		driverName = "pgx"
	default:
		return nil, fmt.Errorf("repository: unsupported driver %q (want sqlite or postgres)", driver)
	}

	raw, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("repository: open %s: %w", driver, err)
	}
	if err := raw.Ping(); err != nil {
		raw.Close()
		return nil, fmt.Errorf("repository: ping %s: %w", driver, err)
	}

	// Configure connection pool.
	if driver == "sqlite" {
		// SQLite allows only one writer; a single conn avoids SQLITE_BUSY under
		// concurrent requests. WAL mode still allows concurrent reads on this conn.
		raw.SetMaxOpenConns(1)
	} else {
		raw.SetMaxOpenConns(pool.MaxOpenConns)
	}
	raw.SetMaxIdleConns(pool.MaxIdleConns)
	raw.SetConnMaxLifetime(pool.ConnMaxLifetime)
	raw.SetConnMaxIdleTime(pool.ConnMaxIdleTime)

	db := &DB{DB: raw, driver: driver}
	if err := migrate(db, log); err != nil {
		raw.Close()
		return nil, fmt.Errorf("repository: migrate: %w", err)
	}
	return db, nil
}

// InList returns a comma-separated string of n '?' placeholders suitable for
// use inside a SQL IN clause, already rewritten for the current driver.
// Returns an empty string when n == 0 (callers must guard against empty IN lists).
func (db *DB) InList(n int) string {
	if n == 0 {
		return ""
	}
	return "?" + strings.Repeat(",?", n-1)
}

// parseTimestamp parses a timestamp string written by either Go (RFC3339) or
// SQLite's CURRENT_TIMESTAMP / datetime() function ("2006-01-02 15:04:05").
// Existing rows in the database may use either format.
func parseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// PostgreSQL timestamp with timezone offset (with and without fractional seconds).
	if t, err := time.Parse("2006-01-02 15:04:05.999999999-07", s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05-07", s); err == nil {
		return t.UTC(), nil
	}
	// SQLite default datetime format — no T separator, no timezone (UTC implied).
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("repository: unrecognised timestamp %q", s)
}

// Ph rewrites ? placeholders to $1, $2, … for PostgreSQL.
func (db *DB) Ph(query string) string {
	if db.driver != "postgres" {
		return query
	}
	n := 0
	var b strings.Builder
	for _, ch := range query {
		if ch == '?' {
			n++
			fmt.Fprintf(&b, "$%d", n)
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}
