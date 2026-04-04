package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// OverviewRepo implements overview analytics queries.
type OverviewRepo struct{ db *DB }

func NewOverviewRepo(db *DB) *OverviewRepo { return &OverviewRepo{db} }

// GetStats returns the full set of analytics data for the overview dashboard.
func (r *OverviewRepo) GetStats(ctx context.Context) (*domain.OverviewStats, error) {
	summary, err := r.getSummary(ctx)
	if err != nil {
		return nil, err
	}
	trends, err := r.getDailyTrends(ctx)
	if err != nil {
		return nil, err
	}
	topFailing, err := r.getTopFailingProjects(ctx)
	if err != nil {
		return nil, err
	}
	recent, err := r.getRecentBuilds(ctx)
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

func (r *OverviewRepo) getSummary(ctx context.Context) (*domain.OverviewSummary, error) {
	var s domain.OverviewSummary
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM environments`).Scan(&s.TotalEnvironments)
	if err != nil {
		return nil, fmt.Errorf("repository: overview environments count: %w", err)
	}
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects`).Scan(&s.TotalProjects)
	if err != nil {
		return nil, fmt.Errorf("repository: overview projects count: %w", err)
	}
	var totalTests int
	err = r.db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*),
		  COALESCE(SUM(passed), 0),
		  COALESCE(SUM(failed), 0)
		FROM builds`).Scan(&s.TotalBuilds, &s.TotalPassed, &s.TotalFailed)
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
// SUBSTR(created_at, 1, 10) extracts the YYYY-MM-DD prefix from RFC3339 strings,
// and works identically for both SQLite and PostgreSQL TEXT columns.
func (r *OverviewRepo) getDailyTrends(ctx context.Context) ([]domain.DailyTrend, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -29).Format("2006-01-02")
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT
		  SUBSTR(created_at, 1, 10) AS day,
		  COALESCE(SUM(passed), 0),
		  COALESCE(SUM(failed), 0),
		  COALESCE(SUM(skipped), 0),
		  COUNT(*)
		FROM builds
		WHERE SUBSTR(created_at, 1, 10) >= ?
		GROUP BY day
		ORDER BY day ASC`),
		cutoff,
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
func (r *OverviewRepo) getTopFailingProjects(ctx context.Context) ([]domain.ProjectFailStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
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
		GROUP BY b.env_id, b.project_id, p.name, e.name
		HAVING COALESCE(SUM(b.failed), 0) > 0
		ORDER BY total_failed DESC
		LIMIT 5`)
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
func (r *OverviewRepo) getRecentBuilds(ctx context.Context) ([]*domain.Build, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.id, b.env_id, b.project_id, b.build_id, b.created_at,
		       b.report_url, b.passed, b.failed, b.skipped, b.total,
		       b.status, b.uploaded_by, b.config_snapshot, b.generation_warnings
		FROM builds b
		ORDER BY b.created_at DESC
		LIMIT 10`)
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
