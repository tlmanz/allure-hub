package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// EnvironmentRepo implements domain.EnvironmentRepository using SQL.
type EnvironmentRepo struct{ db *DB }

func NewEnvironmentRepo(db *DB) *EnvironmentRepo { return &EnvironmentRepo{db} }

func (r *EnvironmentRepo) Create(ctx context.Context, e *domain.Environment) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`INSERT INTO environments (id, name, icon, created_at) VALUES (?, ?, ?, ?)`),
		e.ID, e.Name, e.Icon, e.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("repository: create environment: %w", err)
	}
	return nil
}

func (r *EnvironmentRepo) Get(ctx context.Context, id string) (*domain.Environment, error) {
	row := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, name, icon, created_at FROM environments WHERE id = ?`), id,
	)
	e, err := scanEnvironment(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrEnvironmentNotFound
	}
	return e, err
}

func (r *EnvironmentRepo) List(ctx context.Context) ([]*domain.Environment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, icon, created_at FROM environments ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list environments: %w", err)
	}
	defer rows.Close()

	var envs []*domain.Environment
	for rows.Next() {
		e, err := scanEnvironment(rows)
		if err != nil {
			return nil, err
		}
		envs = append(envs, e)
	}
	return envs, rows.Err()
}

// CountProjectsBatch returns project counts for all given environment IDs in one
// query, eliminating the N+1 pattern in EnvironmentService.List (M-09).
func (r *EnvironmentRepo) CountProjectsBatch(ctx context.Context, envIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(envIDs))
	if len(envIDs) == 0 {
		return result, nil
	}
	args := make([]any, len(envIDs))
	for i, id := range envIDs {
		args[i] = id
	}
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT environment_id, COUNT(*) FROM projects WHERE environment_id IN (`+r.db.InList(len(envIDs))+`) GROUP BY environment_id`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: batch count projects: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var envID string
		var cnt int
		if err := rows.Scan(&envID, &cnt); err != nil {
			return nil, err
		}
		result[envID] = cnt
	}
	return result, rows.Err()
}


func (r *EnvironmentRepo) Update(ctx context.Context, id, name, icon string) error {
	res, err := r.db.ExecContext(ctx,
		r.db.Ph(`UPDATE environments SET name = ?, icon = ? WHERE id = ?`), name, icon, id,
	)
	if err != nil {
		return fmt.Errorf("repository: update environment: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository: update environment rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrEnvironmentNotFound
	}
	return nil
}

func (r *EnvironmentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.db.Ph(`DELETE FROM environments WHERE id = ?`), id)
	if err != nil {
		return fmt.Errorf("repository: delete environment: %w", err)
	}
	return nil
}

func scanEnvironment(s scanner) (*domain.Environment, error) {
	var e domain.Environment
	var createdAt string
	if err := s.Scan(&e.ID, &e.Name, &e.Icon, &createdAt); err != nil {
		return nil, err
	}
	t, err := parseTimestamp(createdAt)
	if err != nil {
		return nil, err
	}
	e.CreatedAt = t
	return &e, nil
}
