package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// TrackedUserRepo implements domain.TrackedUserRepository using SQL.
type TrackedUserRepo struct{ db *DB }

func NewTrackedUserRepo(db *DB) *TrackedUserRepo { return &TrackedUserRepo{db} }

// Upsert inserts a new user or updates name, avatar_url, role, and last_login_at on conflict.
func (r *TrackedUserRepo) Upsert(ctx context.Context, u *domain.TrackedUser) error {
	now := u.LastLoginAt.UTC().Format(time.RFC3339)
	first := u.FirstLoginAt.UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`INSERT INTO tracked_users (email, name, avatar_url, provider, role, first_login_at, last_login_at)
		          VALUES (?, ?, ?, ?, ?, ?, ?)
		          ON CONFLICT(email) DO UPDATE SET
		              name          = excluded.name,
		              avatar_url    = excluded.avatar_url,
		              role          = excluded.role,
		              last_login_at = excluded.last_login_at`),
		u.Email, u.Name, u.AvatarURL, u.Provider, u.Role, first, now,
	)
	if err != nil {
		return fmt.Errorf("repository: upsert tracked user: %w", err)
	}
	return nil
}

// Search returns users whose email or name match query (case-insensitive
// substring), ordered by last_login_at descending, with limit/offset pagination.
// An empty query matches all users.
func (r *TrackedUserRepo) Search(ctx context.Context, query string, limit, offset int) ([]*domain.TrackedUser, error) {
	like := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT email, name, avatar_url, provider, role, first_login_at, last_login_at
		         FROM tracked_users
		         WHERE LOWER(email) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?)
		         ORDER BY last_login_at DESC LIMIT ? OFFSET ?`),
		like, like, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: search tracked users: %w", err)
	}
	defer rows.Close()

	var users []*domain.TrackedUser
	for rows.Next() {
		var u domain.TrackedUser
		var firstLogin, lastLogin string
		if err := rows.Scan(&u.Email, &u.Name, &u.AvatarURL, &u.Provider, &u.Role, &firstLogin, &lastLogin); err != nil {
			return nil, fmt.Errorf("repository: scan tracked user: %w", err)
		}
		var parseErr error
		if u.FirstLoginAt, parseErr = parseTimestamp(firstLogin); parseErr != nil {
			return nil, parseErr
		}
		if u.LastLoginAt, parseErr = parseTimestamp(lastLogin); parseErr != nil {
			return nil, parseErr
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// CountSearch returns the total number of users matching query.
func (r *TrackedUserRepo) CountSearch(ctx context.Context, query string) (int, error) {
	like := "%" + query + "%"
	var n int
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT COUNT(*) FROM tracked_users
		         WHERE LOWER(email) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?)`),
		like, like,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("repository: count tracked users: %w", err)
	}
	return n, nil
}

// List returns all tracked users ordered by last login descending.
func (r *TrackedUserRepo) List(ctx context.Context) ([]*domain.TrackedUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT email, name, avatar_url, provider, role, first_login_at, last_login_at
		 FROM tracked_users ORDER BY last_login_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list tracked users: %w", err)
	}
	defer rows.Close()

	var users []*domain.TrackedUser
	for rows.Next() {
		var u domain.TrackedUser
		var firstLogin, lastLogin string
		if err := rows.Scan(&u.Email, &u.Name, &u.AvatarURL, &u.Provider, &u.Role, &firstLogin, &lastLogin); err != nil {
			return nil, fmt.Errorf("repository: scan tracked user: %w", err)
		}
		if u.FirstLoginAt, err = parseTimestamp(firstLogin); err != nil {
			return nil, err
		}
		if u.LastLoginAt, err = parseTimestamp(lastLogin); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// GetByEmail returns a single tracked user or nil if not found.
func (r *TrackedUserRepo) GetByEmail(ctx context.Context, email string) (*domain.TrackedUser, error) {
	var u domain.TrackedUser
	var firstLogin, lastLogin string
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT email, name, avatar_url, provider, role, first_login_at, last_login_at
		         FROM tracked_users WHERE email = ? LIMIT 1`),
		email,
	).Scan(&u.Email, &u.Name, &u.AvatarURL, &u.Provider, &u.Role, &firstLogin, &lastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get tracked user: %w", err)
	}
	var parseErr error
	if u.FirstLoginAt, parseErr = parseTimestamp(firstLogin); parseErr != nil {
		return nil, parseErr
	}
	if u.LastLoginAt, parseErr = parseTimestamp(lastLogin); parseErr != nil {
		return nil, parseErr
	}
	return &u, nil
}
