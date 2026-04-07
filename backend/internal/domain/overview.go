package domain

// OverviewFilter scopes overview queries to a specific environment and/or project.
type OverviewFilter struct {
	EnvID     string
	ProjectID string
}

// OverviewStats holds all data needed to render the analytics overview dashboard.
type OverviewStats struct {
	Summary            OverviewSummary    `json:"summary"`
	DailyTrends        []DailyTrend       `json:"dailyTrends"`
	TopFailingProjects []ProjectFailStats `json:"topFailingProjects"`
	RecentBuilds       []*Build           `json:"recentBuilds"`
	// ProjectBuildTrend contains pass/fail/skipped per build for the last 30 builds
	// within the current filter scope. Most meaningful when filtered to a single project.
	ProjectBuildTrend []BuildTrend `json:"projectBuildTrend"`
}

// OverviewSummary contains system-wide aggregate counts.
type OverviewSummary struct {
	TotalEnvironments int `json:"totalEnvironments"`
	TotalProjects     int `json:"totalProjects"`
	TotalBuilds       int `json:"totalBuilds"`
	TotalPassed       int `json:"totalPassed"`
	TotalFailed       int `json:"totalFailed"`
	OverallPassRate   int `json:"overallPassRate"`
}

// DailyTrend holds aggregated test result counts for a single day.
type DailyTrend struct {
	Date       string `json:"date"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	Skipped    int    `json:"skipped"`
	BuildCount int    `json:"buildCount"`
}

// ProjectFailStats holds failure metrics for one project, used in the top-failing list.
type ProjectFailStats struct {
	EnvID       string `json:"envId"`
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
	EnvName     string `json:"envName"`
	TotalFailed int    `json:"totalFailed"`
	TotalBuilds int    `json:"totalBuilds"`
	PassRate    int    `json:"passRate"`
}

// BuildTrend holds pass/fail/skipped counts for a single build, used in the per-project build trend chart.
type BuildTrend struct {
	BuildID   string `json:"buildId"`
	CreatedAt string `json:"createdAt"`
	Passed    int    `json:"passed"`
	Failed    int    `json:"failed"`
	Skipped   int    `json:"skipped"`
}
