package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// CleanupRunRepo implements domain.CleanupRunRepository using SQL.
type CleanupRunRepo struct{ db *DB }

func NewCleanupRunRepo(db *DB) *CleanupRunRepo { return &CleanupRunRepo{db} }

const maxCleanupRuns = 5

// Save inserts a new cleanup run record and prunes the table to maxCleanupRuns rows.
func (r *CleanupRunRepo) Save(ctx context.Context, run *domain.CleanupRun) error {
	dryRunInt := 0
	if run.DryRun {
		dryRunInt = 1
	}
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`INSERT INTO cleanup_runs
		          (id, started_at, finished_at, status, deleted_count, skipped_count, dry_run, error_message)
		          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		run.ID,
		run.StartedAt.UTC().Format(time.RFC3339),
		run.FinishedAt.UTC().Format(time.RFC3339),
		run.Status,
		run.DeletedCount,
		run.SkippedCount,
		dryRunInt,
		run.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("repository: save cleanup run: %w", err)
	}

	// Keep only the most recent maxCleanupRuns rows.
	_, err = r.db.ExecContext(ctx,
		r.db.Ph(`DELETE FROM cleanup_runs WHERE id NOT IN (
		          SELECT id FROM cleanup_runs ORDER BY started_at DESC LIMIT ?
		         )`),
		maxCleanupRuns,
	)
	if err != nil {
		return fmt.Errorf("repository: prune cleanup runs: %w", err)
	}
	return nil
}

// ListRecent returns the most recent cleanup runs ordered by started_at DESC.
func (r *CleanupRunRepo) ListRecent(ctx context.Context, limit int) ([]*domain.CleanupRun, error) {
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT id, started_at, finished_at, status, deleted_count, skipped_count, dry_run, error_message
		          FROM cleanup_runs ORDER BY started_at DESC LIMIT ?`),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list recent cleanup runs: %w", err)
	}
	defer rows.Close()

	var runs []*domain.CleanupRun
	for rows.Next() {
		var run domain.CleanupRun
		var startedAt, finishedAt string
		var dryRunInt int
		if err := rows.Scan(
			&run.ID, &startedAt, &finishedAt,
			&run.Status, &run.DeletedCount, &run.SkippedCount,
			&dryRunInt, &run.ErrorMessage,
		); err != nil {
			return nil, fmt.Errorf("repository: scan cleanup run: %w", err)
		}
		run.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		run.FinishedAt, _ = time.Parse(time.RFC3339, finishedAt)
		run.DryRun = dryRunInt != 0
		runs = append(runs, &run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate cleanup runs: %w", err)
	}
	return runs, nil
}

// compile-time interface check
var _ domain.CleanupRunRepository = (*CleanupRunRepo)(nil)
