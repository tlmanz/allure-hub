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
	warnings := b.GenerationWarnings
	if warnings == nil {
		warnings = []string{}
	}
	warningsJSON, err := json.Marshal(warnings)
	if err != nil {
		return fmt.Errorf("repository: marshal generation warnings: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		r.db.Ph(`INSERT INTO builds (id, env_id, project_id, build_id, created_at, report_url, total, passed, failed, skipped, status, uploaded_by, config_snapshot, generation_warnings)
		          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		          ON CONFLICT(env_id, project_id, build_id) DO UPDATE SET
		              report_url      = excluded.report_url,
		              passed          = excluded.passed,
		              failed          = excluded.failed,
		              skipped         = excluded.skipped,
		              total           = excluded.total,
		              status          = excluded.status,
		              uploaded_by     = excluded.uploaded_by,
		              config_snapshot = excluded.config_snapshot,
		              generation_warnings = excluded.generation_warnings`),
		b.ID, b.EnvID, b.ProjectID, b.BuildID,
		b.CreatedAt.UTC().Format(time.RFC3339),
		b.ReportURL, b.Total, b.Passed, b.Failed, b.Skipped, b.Status, b.UploadedBy,
		string(configJSON), string(warningsJSON),
	)
	if err != nil {
		return fmt.Errorf("repository: save build: %w", err)
	}
	return nil
}

func (r *BuildRepo) GetByBuildID(ctx context.Context, envID, projectID, buildID string) (*domain.Build, error) {
	var b domain.Build
	var createdAt, configJSON, warningsJSON string
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, env_id, project_id, build_id, created_at, report_url, passed, failed, skipped, total, status, uploaded_by, config_snapshot, generation_warnings
		         FROM builds WHERE env_id = ? AND project_id = ? AND build_id = ? LIMIT 1`),
		envID, projectID, buildID,
	).Scan(&b.ID, &b.EnvID, &b.ProjectID, &b.BuildID, &createdAt, &b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON, &warningsJSON)
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
	if err := json.Unmarshal([]byte(warningsJSON), &b.GenerationWarnings); err != nil {
		b.GenerationWarnings = []string{}
	}
	return &b, nil
}

// BatchStatsByProject returns build counts and latest builds for all given project IDs
// in exactly two queries, eliminating the 2N+1 query pattern in ListSummaries (M-08).
func (r *BuildRepo) BatchStatsByProject(ctx context.Context, envID string, projectIDs []string) (map[string]*domain.ProjectBatchStats, error) {
	result := make(map[string]*domain.ProjectBatchStats, len(projectIDs))
	if len(projectIDs) == 0 {
		return result, nil
	}

	args := make([]any, len(projectIDs)+1)
	args[0] = envID
	for i, id := range projectIDs {
		args[i+1] = id
		result[id] = &domain.ProjectBatchStats{}
	}

	// Query 1: counts per project.
	countRows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT project_id, COUNT(*) FROM builds WHERE env_id = ? AND project_id IN (`+r.db.InList(len(projectIDs))+`) GROUP BY project_id`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: batch count builds: %w", err)
	}
	defer countRows.Close()
	if err := scanBatchCountRows(countRows, result); err != nil {
		return nil, err
	}

	// Query 2: latest build per project using a portable correlated MAX subquery.
	// args has envID + projectIDs; append envID again for the join condition.
	latestRows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT b.id, b.env_id, b.project_id, b.build_id, b.created_at, b.report_url,
		                b.passed, b.failed, b.skipped, b.total, b.status, b.uploaded_by, b.config_snapshot, b.generation_warnings
		         FROM builds b
		         INNER JOIN (
		             SELECT project_id, MAX(created_at) AS max_at
		             FROM builds WHERE env_id = ? AND project_id IN (`+r.db.InList(len(projectIDs))+`)
		             GROUP BY project_id
		         ) m ON b.project_id = m.project_id AND b.created_at = m.max_at AND b.env_id = ?`),
		append(args, envID)...,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: batch latest builds: %w", err)
	}
	defer latestRows.Close()
	if err := scanBatchLatestRows(latestRows, result); err != nil {
		return nil, err
	}
	return result, latestRows.Err()
}

func scanBatchCountRows(rows *sql.Rows, result map[string]*domain.ProjectBatchStats) error {
	for rows.Next() {
		var pid string
		var cnt int
		if err := rows.Scan(&pid, &cnt); err != nil {
			return err
		}
		if s, ok := result[pid]; ok {
			s.Count = cnt
		}
	}
	return rows.Err()
}

func scanBatchLatestRows(rows *sql.Rows, result map[string]*domain.ProjectBatchStats) error {
	for rows.Next() {
		var b domain.Build
		var createdAt, configJSON, warningsJSON string
		if err := rows.Scan(&b.ID, &b.EnvID, &b.ProjectID, &b.BuildID, &createdAt,
			&b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON, &warningsJSON); err != nil {
			return err
		}
		if t, err := parseTimestamp(createdAt); err == nil {
			b.CreatedAt = t
		}
		if err := json.Unmarshal([]byte(configJSON), &b.ConfigSnapshot); err != nil {
			b.ConfigSnapshot = map[string]any{}
		}
		if err := json.Unmarshal([]byte(warningsJSON), &b.GenerationWarnings); err != nil {
			b.GenerationWarnings = []string{}
		}
		if s, ok := result[b.ProjectID]; ok {
			bCopy := b
			s.Latest = &bCopy
		}
	}
	return rows.Err()
}

func (r *BuildRepo) CountByProject(ctx context.Context, envID, projectID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT COUNT(*) FROM builds WHERE env_id = ? AND project_id = ?`),
		envID, projectID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("repository: count builds: %w", err)
	}
	return count, nil
}

func (r *BuildRepo) LatestByProject(ctx context.Context, envID, projectID string) (*domain.Build, error) {
	var b domain.Build
	var createdAt, configJSON, warningsJSON string
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, env_id, project_id, build_id, created_at, report_url, passed, failed, skipped, total, status, uploaded_by, config_snapshot, generation_warnings
		         FROM builds WHERE env_id = ? AND project_id = ? ORDER BY created_at DESC LIMIT 1`),
		envID, projectID,
	).Scan(&b.ID, &b.EnvID, &b.ProjectID, &b.BuildID, &createdAt, &b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON, &warningsJSON)
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
	if err := json.Unmarshal([]byte(warningsJSON), &b.GenerationWarnings); err != nil {
		b.GenerationWarnings = []string{}
	}
	return &b, nil
}

func (r *BuildRepo) ListByProject(ctx context.Context, envID, projectID string) ([]*domain.Build, error) {
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT id, env_id, project_id, build_id, created_at, report_url, passed, failed, skipped, total, status, uploaded_by, config_snapshot, generation_warnings
		          FROM builds WHERE env_id = ? AND project_id = ? ORDER BY created_at DESC LIMIT 1000`),
		envID, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list builds: %w", err)
	}
	defer rows.Close()

	var builds []*domain.Build
	for rows.Next() {
		var b domain.Build
		var createdAt, configJSON, warningsJSON string
		if err := rows.Scan(&b.ID, &b.EnvID, &b.ProjectID, &b.BuildID, &createdAt, &b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON, &warningsJSON); err != nil {
			return nil, err
		}
		if b.CreatedAt, err = parseBuildTime(createdAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(configJSON), &b.ConfigSnapshot); err != nil {
			b.ConfigSnapshot = map[string]any{}
		}
		if err := json.Unmarshal([]byte(warningsJSON), &b.GenerationWarnings); err != nil {
			b.GenerationWarnings = []string{}
		}
		builds = append(builds, &b)
	}
	return builds, rows.Err()
}

func (r *BuildRepo) ListByProjectPaged(ctx context.Context, envID, projectID, filter string, limit, offset int) ([]*domain.Build, error) {
	where, args := buildFilterWhere(envID, projectID, filter)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT id, env_id, project_id, build_id, created_at, report_url, passed, failed, skipped, total, status, uploaded_by, config_snapshot, generation_warnings
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
		var createdAt, configJSON, warningsJSON string
		if err := rows.Scan(&b.ID, &b.EnvID, &b.ProjectID, &b.BuildID, &createdAt, &b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total, &b.Status, &b.UploadedBy, &configJSON, &warningsJSON); err != nil {
			return nil, err
		}
		if b.CreatedAt, err = parseBuildTime(createdAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(configJSON), &b.ConfigSnapshot); err != nil {
			b.ConfigSnapshot = map[string]any{}
		}
		if err := json.Unmarshal([]byte(warningsJSON), &b.GenerationWarnings); err != nil {
			b.GenerationWarnings = []string{}
		}
		builds = append(builds, &b)
	}
	return builds, rows.Err()
}

func (r *BuildRepo) CountByProjectFiltered(ctx context.Context, envID, projectID, filter string) (int, error) {
	where, args := buildFilterWhere(envID, projectID, filter)
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

func (r *BuildRepo) StatsForProject(ctx context.Context, envID, projectID string) (*domain.BuildStats, error) {
	var stats domain.BuildStats
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT
		  COUNT(*) AS total_runs,
		  COALESCE(SUM(failed), 0) AS total_failed,
		  CASE WHEN SUM(total) > 0 THEN CAST(SUM(passed)*100/SUM(total) AS INTEGER) ELSE 0 END AS avg_rate
		FROM builds WHERE env_id = ? AND project_id = ?`),
		envID, projectID,
	).Scan(&stats.TotalRuns, &stats.TotalFailed, &stats.AvgRate)
	if err != nil {
		return nil, fmt.Errorf("repository: stats for project: %w", err)
	}

	// Latest pass rate from the most recent build.
	var latestTotal, latestPassed int
	err = r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT total, passed FROM builds WHERE env_id = ? AND project_id = ? ORDER BY created_at DESC LIMIT 1`),
		envID, projectID,
	).Scan(&latestTotal, &latestPassed)
	if err == nil && latestTotal > 0 {
		stats.LatestRate = latestPassed * 100 / latestTotal
	}
	return &stats, nil
}

func (r *BuildRepo) Delete(ctx context.Context, envID, projectID, buildID string) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`DELETE FROM builds WHERE env_id = ? AND project_id = ? AND build_id = ?`),
		envID, projectID, buildID,
	)
	if err != nil {
		return fmt.Errorf("repository: delete build: %w", err)
	}
	return nil
}

func (r *BuildRepo) DeleteByProject(ctx context.Context, envID, projectID string) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`DELETE FROM builds WHERE env_id = ? AND project_id = ?`),
		envID, projectID,
	)
	if err != nil {
		return fmt.Errorf("repository: delete builds by project: %w", err)
	}
	return nil
}

// ListExpiredBuilds returns all builds whose created_at is older than cutoff,
// selecting only the fields needed by the cleanup worker.
func (r *BuildRepo) ListExpiredBuilds(ctx context.Context, cutoff time.Time) ([]*domain.Build, error) {
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT id, env_id, project_id, build_id, created_at
		          FROM builds WHERE created_at < ?
		          ORDER BY created_at ASC`),
		cutoff.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list expired builds: %w", err)
	}
	defer rows.Close()

	var builds []*domain.Build
	for rows.Next() {
		var b domain.Build
		var createdAt string
		if err := rows.Scan(&b.ID, &b.EnvID, &b.ProjectID, &b.BuildID, &createdAt); err != nil {
			return nil, fmt.Errorf("repository: scan expired build: %w", err)
		}
		var parseErr error
		if b.CreatedAt, parseErr = parseBuildTime(createdAt); parseErr != nil {
			return nil, parseErr
		}
		builds = append(builds, &b)
	}
	return builds, rows.Err()
}

// parseBuildTime parses a timestamp stored for a Build row, accepting both
// RFC3339 and SQLite's legacy datetime format (M-11).
func parseBuildTime(s string) (time.Time, error) {
	return parseTimestamp(s)
}

// buildFilterWhere returns the WHERE clause fragment and args for a filter.
func buildFilterWhere(envID, projectID, filter string) (string, []any) {
	switch filter {
	case "passed":
		return "env_id = ? AND project_id = ? AND failed = 0 AND total > 0", []any{envID, projectID}
	case "failed":
		return "env_id = ? AND project_id = ? AND failed > 0", []any{envID, projectID}
	default:
		return "env_id = ? AND project_id = ?", []any{envID, projectID}
	}
}
