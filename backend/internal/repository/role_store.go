package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tlmanz/authkit"
)

// RoleStore implements authkit.UserRoleStore using SQL.
// It persists per-user role overrides that take precedence over the YAML baseline.
type RoleStore struct{ db *DB }

// NewRoleStore returns a RoleStore backed by db.
func NewRoleStore(db *DB) *RoleStore { return &RoleStore{db} }

// Ensure RoleStore satisfies the interface at compile time.
var _ authkit.UserRoleStore = (*RoleStore)(nil)

// GetOverride returns the stored role override for email, or found=false when
// no override has been set. authkit will fall back to the YAML baseline.
func (s *RoleStore) GetOverride(ctx context.Context, email string) (string, []string, bool, error) {
	var role, permsJSON string
	err := s.db.QueryRowContext(ctx,
		s.db.Ph(`SELECT role, permissions FROM role_overrides WHERE email = ? LIMIT 1`),
		email,
	).Scan(&role, &permsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, fmt.Errorf("repository: get role override: %w", err)
	}
	var perms []string
	if err := json.Unmarshal([]byte(permsJSON), &perms); err != nil {
		return "", nil, false, fmt.Errorf("repository: unmarshal role permissions: %w", err)
	}
	return role, perms, true, nil
}

// SetOverride creates or replaces the role override for email.
// Prefer calling LayeredPolicyProvider.SetOverride - it validates inputs before
// writing here.
func (s *RoleStore) SetOverride(ctx context.Context, email, role string, permissions []string) error {
	permsJSON, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("repository: marshal role permissions: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		s.db.Ph(`INSERT INTO role_overrides (email, role, permissions)
		          VALUES (?, ?, ?)
		          ON CONFLICT(email) DO UPDATE SET role = ?, permissions = ?`),
		email, role, string(permsJSON), role, string(permsJSON),
	)
	if err != nil {
		return fmt.Errorf("repository: set role override: %w", err)
	}
	return nil
}

// DeleteOverride removes the role override for email, reverting that user to
// the YAML baseline on their next login.
func (s *RoleStore) DeleteOverride(ctx context.Context, email string) error {
	_, err := s.db.ExecContext(ctx,
		s.db.Ph(`DELETE FROM role_overrides WHERE email = ?`),
		email,
	)
	if err != nil {
		return fmt.Errorf("repository: delete role override: %w", err)
	}
	return nil
}
