package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// BuildRepo implements domain.BuildRepository using SQL.
type BuildRepo struct{ db *DB }

func NewBuildRepo(db *DB) *BuildRepo { return &BuildRepo{db} }

// Save upserts a Build row — safe to call on re-runs.
func (r *BuildRepo) Save(ctx context.Context, b *domain.Build) error {
	configJSON, err := json.Marshal(b.ConfigSnapshot)
	if err != nil {
		return fmt.Errorf("repository: marshal config snapshot: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		r.db.Ph(`INSERT INTO builds (id, project_id, build_id, created_at, report_url, total, passed, failed, skipped, status, uploaded_by, config_snapshot)
		          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		          ON CONFLICT(project_id, build_id) DO UPDATE SET
		              report_url      = excluded.report_url,
		              passed          = excluded.passed,
		              failed          = excluded.failed,
		              skipped         = excluded.skipped,
		              total           = excluded.total,
		              status          = excluded.status,
		              uploaded_by     = excluded.uploaded_by,
		              config_snapshot = excluded.config_snapshot`),
		b.ID, b.ProjectID, b.BuildID,
		b.CreatedAt.UTC().Format(time.RFC3339),
		b.ReportURL, b.Total, b.Passed, b.Failed, b.Skipped, b.Status, b.UploadedBy,
		string(configJSON),
	)
	if err != nil {
		return fmt.Errorf("repository: save build: %w", err)
	}
	return nil
}

func (r *BuildRepo) GetByBuildID(ctx context.Context, projectID, buildID string) (*domain.Build, error) {
	var b domain.Build
	var createdAt, configJSON string
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, project_id, build_id, created_at, report_url, passed, failed, skipped, total, status, uploaded_by, config_snapshot
		         FROM builds WHERE project_id = ? AND build_id = ? LIMIT 1`),
		projectID, buildID,
	).Scan(&b.ID, &b.ProjectID, &b.BuildID, &createdAt, &b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: get build by id: %w", err)
	}
	if b.CreatedAt, err = parseBuildTime(createdAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(configJSON), &b.ConfigSnapshot); err != nil {
		b.ConfigSnapshot = map[string]any{}
	}
	return &b, nil
}

// BatchStatsByProject returns build counts and latest builds for all given project IDs
// in exactly two queries, eliminating the 2N+1 query pattern in ListSummaries (M-08).
func (r *BuildRepo) BatchStatsByProject(ctx context.Context, projectIDs []string) (map[string]*domain.ProjectBatchStats, error) {
	result := make(map[string]*domain.ProjectBatchStats, len(projectIDs))
	if len(projectIDs) == 0 {
		return result, nil
	}

	args := make([]any, len(projectIDs))
	for i, id := range projectIDs {
		args[i] = id
		result[id] = &domain.ProjectBatchStats{}
	}

	// Query 1: counts per project.
	countRows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT project_id, COUNT(*) FROM builds WHERE project_id IN (`+r.db.InList(len(projectIDs))+`) GROUP BY project_id`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: batch count builds: %w", err)
	}
	defer countRows.Close()
	for countRows.Next() {
		var pid string
		var cnt int
		if err := countRows.Scan(&pid, &cnt); err != nil {
			return nil, err
		}
		if s, ok := result[pid]; ok {
			s.Count = cnt
		}
	}
	if err := countRows.Err(); err != nil {
		return nil, err
	}

	// Query 2: latest build per project using a portable correlated MAX subquery.
	latestRows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT b.id, b.project_id, b.build_id, b.created_at, b.report_url,
		                b.passed, b.failed, b.skipped, b.total, b.status, b.uploaded_by, b.config_snapshot
		         FROM builds b
		         INNER JOIN (
		             SELECT project_id, MAX(created_at) AS max_at
		             FROM builds WHERE project_id IN (`+r.db.InList(len(projectIDs))+`)
		             GROUP BY project_id
		         ) m ON b.project_id = m.project_id AND b.created_at = m.max_at`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: batch latest builds: %w", err)
	}
	defer latestRows.Close()
	for latestRows.Next() {
		var b domain.Build
		var createdAt, configJSON string
		if err := latestRows.Scan(&b.ID, &b.ProjectID, &b.BuildID, &createdAt,
			&b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON); err != nil {
			return nil, err
		}
		if t, err := parseTimestamp(createdAt); err == nil {
			b.CreatedAt = t
		}
		if err := json.Unmarshal([]byte(configJSON), &b.ConfigSnapshot); err != nil {
			b.ConfigSnapshot = map[string]any{}
		}
		if s, ok := result[b.ProjectID]; ok {
			bCopy := b
			s.Latest = &bCopy
		}
	}
	return result, latestRows.Err()
}

func (r *BuildRepo) CountByProject(ctx context.Context, projectID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT COUNT(*) FROM builds WHERE project_id = ?`),
		projectID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("repository: count builds: %w", err)
	}
	return count, nil
}

func (r *BuildRepo) LatestByProject(ctx context.Context, projectID string) (*domain.Build, error) {
	var b domain.Build
	var createdAt, configJSON string
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, project_id, build_id, created_at, report_url, passed, failed, skipped, total, status, uploaded_by, config_snapshot
		         FROM builds WHERE project_id = ? ORDER BY created_at DESC LIMIT 1`),
		projectID,
	).Scan(&b.ID, &b.ProjectID, &b.BuildID, &createdAt, &b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: latest build: %w", err)
	}
	if b.CreatedAt, err = parseBuildTime(createdAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(configJSON), &b.ConfigSnapshot); err != nil {
		b.ConfigSnapshot = map[string]any{}
	}
	return &b, nil
}

func (r *BuildRepo) ListByProject(ctx context.Context, projectID string) ([]*domain.Build, error) {
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT id, project_id, build_id, created_at, report_url, passed, failed, skipped, total, status, uploaded_by, config_snapshot
		          FROM builds WHERE project_id = ? ORDER BY created_at DESC LIMIT 1000`),
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list builds: %w", err)
	}
	defer rows.Close()

	var builds []*domain.Build
	for rows.Next() {
		var b domain.Build
		var createdAt, configJSON string
		if err := rows.Scan(&b.ID, &b.ProjectID, &b.BuildID, &createdAt, &b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON); err != nil {
			return nil, err
		}
		if b.CreatedAt, err = parseBuildTime(createdAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(configJSON), &b.ConfigSnapshot); err != nil {
			b.ConfigSnapshot = map[string]any{}
		}
		builds = append(builds, &b)
	}
	return builds, rows.Err()
}

func (r *BuildRepo) ListByProjectPaged(ctx context.Context, projectID, filter string, limit, offset int) ([]*domain.Build, error) {
	where, args := buildFilterWhere(projectID, filter)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT id, project_id, build_id, created_at, report_url, passed, failed, skipped, total, status, uploaded_by, config_snapshot
		          FROM builds WHERE `+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list builds paged: %w", err)
	}
	defer rows.Close()

	var builds []*domain.Build
	for rows.Next() {
		var b domain.Build
		var createdAt, configJSON string
		if err := rows.Scan(&b.ID, &b.ProjectID, &b.BuildID, &createdAt, &b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON); err != nil {
			return nil, err
		}
		if b.CreatedAt, err = parseBuildTime(createdAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(configJSON), &b.ConfigSnapshot); err != nil {
			b.ConfigSnapshot = map[string]any{}
		}
		builds = append(builds, &b)
	}
	return builds, rows.Err()
}

func (r *BuildRepo) CountByProjectFiltered(ctx context.Context, projectID, filter string) (int, error) {
	where, args := buildFilterWhere(projectID, filter)
	var count int
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT COUNT(*) FROM builds WHERE `+where),
		args...,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("repository: count builds filtered: %w", err)
	}
	return count, nil
}

func (r *BuildRepo) StatsForProject(ctx context.Context, projectID string) (*domain.BuildStats, error) {
	var stats domain.BuildStats
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT
		  COUNT(*) AS total_runs,
		  COALESCE(SUM(failed), 0) AS total_failed,
		  CASE WHEN SUM(total) > 0 THEN CAST(SUM(passed)*100/SUM(total) AS INTEGER) ELSE 0 END AS avg_rate
		FROM builds WHERE project_id = ?`),
		projectID,
	).Scan(&stats.TotalRuns, &stats.TotalFailed, &stats.AvgRate)
	if err != nil {
		return nil, fmt.Errorf("repository: stats for project: %w", err)
	}

	// Latest pass rate from the most recent build.
	var latestTotal, latestPassed int
	err = r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT total, passed FROM builds WHERE project_id = ? ORDER BY created_at DESC LIMIT 1`),
		projectID,
	).Scan(&latestTotal, &latestPassed)
	if err == nil && latestTotal > 0 {
		stats.LatestRate = latestPassed * 100 / latestTotal
	}
	return &stats, nil
}

func (r *BuildRepo) Delete(ctx context.Context, projectID, buildID string) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`DELETE FROM builds WHERE project_id = ? AND build_id = ?`),
		projectID, buildID,
	)
	if err != nil {
		return fmt.Errorf("repository: delete build: %w", err)
	}
	return nil
}

func (r *BuildRepo) DeleteByProject(ctx context.Context, projectID string) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`DELETE FROM builds WHERE project_id = ?`),
		projectID,
	)
	if err != nil {
		return fmt.Errorf("repository: delete builds by project: %w", err)
	}
	return nil
}

// parseBuildTime parses a timestamp stored for a Build row, accepting both
// RFC3339 and SQLite's legacy datetime format (M-11).
func parseBuildTime(s string) (time.Time, error) {
	return parseTimestamp(s)
}

// buildFilterWhere returns the WHERE clause fragment and args for a filter.
func buildFilterWhere(projectID, filter string) (string, []any) {
	switch filter {
	case "passed":
		return "project_id = ? AND failed = 0 AND total > 0", []any{projectID}
	case "failed":
		return "project_id = ? AND failed > 0", []any{projectID}
	default:
		return "project_id = ?", []any{projectID}
	}
}
