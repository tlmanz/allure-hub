package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// ProjectRepo implements domain.ProjectRepository using SQL.
type ProjectRepo struct{ db *DB }

func NewProjectRepo(db *DB) *ProjectRepo { return &ProjectRepo{db} }

func (r *ProjectRepo) Create(ctx context.Context, p *domain.Project) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`INSERT INTO projects (id, environment_id, name, created_at) VALUES (?, ?, ?, ?)`),
		p.ID, p.EnvironmentID, p.Name, p.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("repository: create project: %w", err)
	}
	return nil
}

func (r *ProjectRepo) Get(ctx context.Context, envID, id string) (*domain.Project, error) {
	row := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, environment_id, name, created_at FROM projects WHERE environment_id = ? AND id = ?`),
		envID, id,
	)
	p, err := scanProject(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrProjectNotFound
	}
	return p, err
}

func (r *ProjectRepo) List(ctx context.Context, envID string) ([]*domain.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT id, environment_id, name, created_at FROM projects WHERE environment_id = ? ORDER BY created_at DESC`),
		envID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list projects: %w", err)
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepo) Delete(ctx context.Context, envID, id string) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`DELETE FROM projects WHERE environment_id = ? AND id = ?`),
		envID, id,
	)
	if err != nil {
		return fmt.Errorf("repository: delete project: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProject(s scanner) (*domain.Project, error) {
	var p domain.Project
	var createdAt string
	if err := s.Scan(&p.ID, &p.EnvironmentID, &p.Name, &createdAt); err != nil {
		return nil, err
	}
	t, err := parseTimestamp(createdAt)
	if err != nil {
		return nil, err
	}
	p.CreatedAt = t
	return &p, nil
}
