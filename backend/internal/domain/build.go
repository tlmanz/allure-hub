package domain

import (
	"errors"
	"time"
)

var ErrBuildNotFound = errors.New("build not found")

// Build represents a generated Allure report for a specific test run.
type Build struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"projectId"`
	BuildID        string         `json:"buildId"`
	CreatedAt      time.Time      `json:"createdAt"`
	ReportURL      string         `json:"reportUrl"`
	Passed         int            `json:"passed"`
	Failed         int            `json:"failed"`
	Skipped        int            `json:"skipped"`
	Total          int            `json:"total"`
	Status         string         `json:"status"`
	// UploadedBy is the email (OAuth) or "apikey:<name>" of whoever uploaded these results.
	UploadedBy     string         `json:"uploadedBy"`
	// ConfigSnapshot is the effective allurerc.yml config used for generation,
	// with server-controlled keys (output, historyPath) excluded.
	ConfigSnapshot map[string]any `json:"configSnapshot"`
}

// BuildStats holds aggregate metrics for a project across all builds.
type BuildStats struct {
	TotalRuns   int `json:"totalRuns"`
	LatestRate  int `json:"latestRate"`
	AvgRate     int `json:"avgRate"`
	TotalFailed int `json:"totalFailed"`
}

// ProjectBatchStats holds the build count and latest build for one project.
// Used by BatchStatsByProject to return all projects' stats in two queries (M-08).
type ProjectBatchStats struct {
	Count  int
	Latest *Build
}
