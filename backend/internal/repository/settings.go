package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// SystemSettingsRepo implements domain.SystemSettingsRepository using SQL.
type SystemSettingsRepo struct{ db *DB }

func NewSystemSettingsRepo(db *DB) *SystemSettingsRepo { return &SystemSettingsRepo{db} }

// Get returns the value for key, or ("", nil) when the key is absent.
func (r *SystemSettingsRepo) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT value FROM system_settings WHERE key = ? LIMIT 1`),
		key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("repository: get system setting %q: %w", key, err)
	}
	return value, nil
}

// Set upserts key with value, updating updated_at.
func (r *SystemSettingsRepo) Set(ctx context.Context, key, value string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`INSERT INTO system_settings (key, value, updated_at)
		          VALUES (?, ?, ?)
		          ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`),
		key, value, now,
	)
	if err != nil {
		return fmt.Errorf("repository: set system setting %q: %w", key, err)
	}
	return nil
}

// compile-time interface check
var _ domain.SystemSettingsRepository = (*SystemSettingsRepo)(nil)
