package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// APIKeyRepo implements domain.APIKeyRepository using SQL.
type APIKeyRepo struct{ db *DB }

func NewAPIKeyRepo(db *DB) *APIKeyRepo { return &APIKeyRepo{db} }

func (r *APIKeyRepo) Create(ctx context.Context, k *domain.APIKey) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`INSERT INTO api_keys (id, name, created_by, role, key_hash, created_at, expires_at, is_active)
		          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		k.ID, k.Name, k.CreatedBy, k.Role, k.KeyHash,
		k.CreatedAt.UTC().Format(time.RFC3339),
		nullTimePtr(k.ExpiresAt),
		boolToInt(k.IsActive),
	)
	if err != nil {
		return fmt.Errorf("repository: create api key: %w", err)
	}
	return nil
}

// GetByHash looks up an active key by its SHA-256 hash. Returns nil if not found.
func (r *APIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	row := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, name, created_by, role, key_hash, last_used_at, created_at, expires_at, is_active
		         FROM api_keys WHERE key_hash = ? LIMIT 1`),
		keyHash,
	)
	return scanAPIKey(row)
}

// Search returns keys whose name or created_by match query (case-insensitive
// substring), ordered by created_at descending, with limit/offset pagination.
// An empty query matches all keys.
func (r *APIKeyRepo) Search(ctx context.Context, query string, limit, offset int) ([]*domain.APIKey, error) {
	like := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT id, name, created_by, role, key_hash, last_used_at, created_at, expires_at, is_active
		         FROM api_keys
		         WHERE LOWER(name) LIKE LOWER(?) OR LOWER(created_by) LIKE LOWER(?)
		         ORDER BY created_at DESC LIMIT ? OFFSET ?`),
		like, like, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: search api keys: %w", err)
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		k, err := scanAPIKeyRow(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// CountSearch returns the total number of keys matching query.
func (r *APIKeyRepo) CountSearch(ctx context.Context, query string) (int, error) {
	like := "%" + query + "%"
	var n int
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT COUNT(*) FROM api_keys
		         WHERE LOWER(name) LIKE LOWER(?) OR LOWER(created_by) LIKE LOWER(?)`),
		like, like,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("repository: count api keys: %w", err)
	}
	return n, nil
}

// List returns all keys ordered by created_at descending.
func (r *APIKeyRepo) List(ctx context.Context) ([]*domain.APIKey, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, created_by, role, key_hash, last_used_at, created_at, expires_at, is_active
		 FROM api_keys ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list api keys: %w", err)
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		k, err := scanAPIKeyRow(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// UpdateLastUsed sets last_used_at to now for the given key ID.
func (r *APIKeyRepo) UpdateLastUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`UPDATE api_keys SET last_used_at = ? WHERE id = ?`),
		time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("repository: update api key last_used_at: %w", err)
	}
	return nil
}

// Revoke sets is_active = 0 (soft delete) for the given key ID.
func (r *APIKeyRepo) Revoke(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`UPDATE api_keys SET is_active = 0 WHERE id = ?`),
		id,
	)
	if err != nil {
		return fmt.Errorf("repository: revoke api key: %w", err)
	}
	return nil
}

// Delete permanently removes a key.
func (r *APIKeyRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`DELETE FROM api_keys WHERE id = ?`),
		id,
	)
	if err != nil {
		return fmt.Errorf("repository: delete api key: %w", err)
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func scanAPIKey(row *sql.Row) (*domain.APIKey, error) {
	var k domain.APIKey
	var lastUsedAt, createdAt, expiresAt sql.NullString
	var isActive int
	err := row.Scan(&k.ID, &k.Name, &k.CreatedBy, &k.Role, &k.KeyHash,
		&lastUsedAt, &createdAt, &expiresAt, &isActive)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repository: scan api key: %w", err)
	}
	return finishAPIKeyScan(&k, lastUsedAt, createdAt, expiresAt, isActive)
}

func scanAPIKeyRow(rows *sql.Rows) (*domain.APIKey, error) {
	var k domain.APIKey
	var lastUsedAt, createdAt, expiresAt sql.NullString
	var isActive int
	err := rows.Scan(&k.ID, &k.Name, &k.CreatedBy, &k.Role, &k.KeyHash,
		&lastUsedAt, &createdAt, &expiresAt, &isActive)
	if err != nil {
		return nil, fmt.Errorf("repository: scan api key row: %w", err)
	}
	return finishAPIKeyScan(&k, lastUsedAt, createdAt, expiresAt, isActive)
}

func finishAPIKeyScan(k *domain.APIKey, lastUsedAt, createdAt, expiresAt sql.NullString, isActive int) (*domain.APIKey, error) {
	var err error
	if k.CreatedAt, err = parseTimestamp(createdAt.String); err != nil {
		return nil, err
	}
	if lastUsedAt.Valid && lastUsedAt.String != "" {
		t, err := parseTimestamp(lastUsedAt.String)
		if err != nil {
			return nil, err
		}
		k.LastUsedAt = &t
	}
	if expiresAt.Valid && expiresAt.String != "" {
		t, err := parseTimestamp(expiresAt.String)
		if err != nil {
			return nil, err
		}
		k.ExpiresAt = &t
	}
	k.IsActive = isActive == 1
	return k, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullTimePtr(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(time.RFC3339), Valid: true}
}
