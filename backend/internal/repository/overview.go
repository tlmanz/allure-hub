package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// OverviewRepo implements overview analytics queries.
type OverviewRepo struct{ db *DB }

func NewOverviewRepo(db *DB) *OverviewRepo { return &OverviewRepo{db} }

// buildWhere constructs a WHERE clause from (column, value) pairs, skipping empty values.
func buildWhere(pairs ...string) (string, []any) {
	var conds []string
	var args []any
	for i := 0; i+1 < len(pairs); i += 2 {
		col, val := pairs[i], pairs[i+1]
		if val != "" {
			conds = append(conds, col+" = ?")
			args = append(args, val)
		}
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// GetStats returns the full set of analytics data for the overview dashboard.
func (r *OverviewRepo) GetStats(ctx context.Context, f domain.OverviewFilter) (*domain.OverviewStats, error) {
	summary, err := r.getSummary(ctx, f)
	if err != nil {
		return nil, err
	}
	trends, err := r.getDailyTrends(ctx, f)
	if err != nil {
		return nil, err
	}
	topFailing, err := r.getTopFailingProjects(ctx, f)
	if err != nil {
		return nil, err
	}
	recent, err := r.getRecentBuilds(ctx, f)
	if err != nil {
		return nil, err
	}
	return &domain.OverviewStats{
		Summary:            *summary,
		DailyTrends:        trends,
		TopFailingProjects: topFailing,
		RecentBuilds:       recent,
	}, nil
}

func (r *OverviewRepo) getSummary(ctx context.Context, f domain.OverviewFilter) (*domain.OverviewSummary, error) {
	var s domain.OverviewSummary

	// environments count (filter by env if set)
	envWhere, envArgs := buildWhere("id", f.EnvID)
	err := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT COUNT(*) FROM environments`+envWhere),
		envArgs...,
	).Scan(&s.TotalEnvironments)
	if err != nil {
		return nil, fmt.Errorf("repository: overview environments count: %w", err)
	}

	// projects count (filter by environment_id and/or id)
	projWhere, projArgs := buildWhere("environment_id", f.EnvID, "id", f.ProjectID)
	err = r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT COUNT(*) FROM projects`+projWhere),
		projArgs...,
	).Scan(&s.TotalProjects)
	if err != nil {
		return nil, fmt.Errorf("repository: overview projects count: %w", err)
	}

	// builds summary (filter by env_id and/or project_id)
	buildWhere, buildArgs := buildWhere("env_id", f.EnvID, "project_id", f.ProjectID)
	var totalTests int
	err = r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT COUNT(*), COALESCE(SUM(passed), 0), COALESCE(SUM(failed), 0) FROM builds`+buildWhere),
		buildArgs...,
	).Scan(&s.TotalBuilds, &s.TotalPassed, &s.TotalFailed)
	if err != nil {
		return nil, fmt.Errorf("repository: overview builds summary: %w", err)
	}
	totalTests = s.TotalPassed + s.TotalFailed
	if totalTests > 0 {
		s.OverallPassRate = s.TotalPassed * 100 / totalTests
	}
	return &s, nil
}

// getDailyTrends returns per-day pass/fail/skipped totals for the last 30 days.
func (r *OverviewRepo) getDailyTrends(ctx context.Context, f domain.OverviewFilter) ([]domain.DailyTrend, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -29).Format("2006-01-02")

	extraWhere := ""
	args := []any{cutoff}
	if f.EnvID != "" {
		extraWhere += " AND env_id = ?"
		args = append(args, f.EnvID)
	}
	if f.ProjectID != "" {
		extraWhere += " AND project_id = ?"
		args = append(args, f.ProjectID)
	}

	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT
		  SUBSTR(created_at, 1, 10) AS day,
		  COALESCE(SUM(passed), 0),
		  COALESCE(SUM(failed), 0),
		  COALESCE(SUM(skipped), 0),
		  COUNT(*)
		FROM builds
		WHERE SUBSTR(created_at, 1, 10) >= ?`+extraWhere+`
		GROUP BY day
		ORDER BY day ASC`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: overview daily trends: %w", err)
	}
	defer rows.Close()

	var trends []domain.DailyTrend
	for rows.Next() {
		var t domain.DailyTrend
		if err := rows.Scan(&t.Date, &t.Passed, &t.Failed, &t.Skipped, &t.BuildCount); err != nil {
			return nil, fmt.Errorf("repository: overview trend scan: %w", err)
		}
		trends = append(trends, t)
	}
	if trends == nil {
		trends = []domain.DailyTrend{}
	}
	return trends, rows.Err()
}

// getTopFailingProjects returns the 5 projects with the most cumulative failures.
func (r *OverviewRepo) getTopFailingProjects(ctx context.Context, f domain.OverviewFilter) ([]domain.ProjectFailStats, error) {
	extraWhere := ""
	var args []any
	if f.EnvID != "" {
		extraWhere += " AND b.env_id = ?"
		args = append(args, f.EnvID)
	}
	if f.ProjectID != "" {
		extraWhere += " AND b.project_id = ?"
		args = append(args, f.ProjectID)
	}

	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT
		  b.env_id,
		  b.project_id,
		  p.name,
		  e.name,
		  COALESCE(SUM(b.failed), 0) AS total_failed,
		  COUNT(b.id) AS build_count,
		  CASE WHEN SUM(b.total) > 0
		    THEN CAST(SUM(b.passed) * 100 / SUM(b.total) AS INTEGER)
		    ELSE 0
		  END AS pass_rate
		FROM builds b
		JOIN projects p ON p.environment_id = b.env_id AND p.id = b.project_id
		JOIN environments e ON e.id = b.env_id
		WHERE 1=1`+extraWhere+`
		GROUP BY b.env_id, b.project_id, p.name, e.name
		HAVING COALESCE(SUM(b.failed), 0) > 0
		ORDER BY total_failed DESC
		LIMIT 5`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: overview top failing: %w", err)
	}
	defer rows.Close()

	var result []domain.ProjectFailStats
	for rows.Next() {
		var s domain.ProjectFailStats
		if err := rows.Scan(&s.EnvID, &s.ProjectID, &s.ProjectName, &s.EnvName,
			&s.TotalFailed, &s.TotalBuilds, &s.PassRate); err != nil {
			return nil, fmt.Errorf("repository: overview top failing scan: %w", err)
		}
		result = append(result, s)
	}
	if result == nil {
		result = []domain.ProjectFailStats{}
	}
	return result, rows.Err()
}

// getRecentBuilds returns the 10 most recent builds across all projects.
func (r *OverviewRepo) getRecentBuilds(ctx context.Context, f domain.OverviewFilter) ([]*domain.Build, error) {
	extraWhere := ""
	var args []any
	if f.EnvID != "" {
		extraWhere += " AND b.env_id = ?"
		args = append(args, f.EnvID)
	}
	if f.ProjectID != "" {
		extraWhere += " AND b.project_id = ?"
		args = append(args, f.ProjectID)
	}

	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT b.id, b.env_id, b.project_id, b.build_id, b.created_at,
		       b.report_url, b.passed, b.failed, b.skipped, b.total,
		       b.status, b.uploaded_by, b.config_snapshot, b.generation_warnings
		FROM builds b
		WHERE 1=1`+extraWhere+`
		ORDER BY b.created_at DESC
		LIMIT 10`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: overview recent builds: %w", err)
	}
	defer rows.Close()

	var builds []*domain.Build
	for rows.Next() {
		var b domain.Build
		var createdAt, configJSON, warningsJSON string
		if err := rows.Scan(&b.ID, &b.EnvID, &b.ProjectID, &b.BuildID, &createdAt,
			&b.ReportURL, &b.Passed, &b.Failed, &b.Skipped, &b.Total,
			&b.Status, &b.UploadedBy, &configJSON, &warningsJSON); err != nil {
			return nil, fmt.Errorf("repository: overview recent builds scan: %w", err)
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
		builds = append(builds, &b)
	}
	if builds == nil {
		builds = []*domain.Build{}
	}
	return builds, rows.Err()
}
