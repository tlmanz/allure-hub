package usecase

import (
	"context"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// ProjectSummary enriches a Project with aggregated build statistics.
type ProjectSummary struct {
	*domain.Project
	BuildCount  int        `json:"buildCount"`
	LastStatus  string     `json:"lastStatus"` // passed | failed | inactive
	LastBuildAt *time.Time `json:"lastBuildAt,omitempty"`
	LastTotal   int        `json:"lastTotal"`
	LastPassed  int        `json:"lastPassed"`
	LastFailed  int        `json:"lastFailed"`
}

// ProjectService orchestrates project lifecycle use-cases.
type ProjectService struct {
	repo        domain.ProjectRepository
	buildRepo   domain.BuildRepository
	sessionRepo domain.UploadSessionRepository
	fs          FileStorage
}

func NewProjectService(repo domain.ProjectRepository, buildRepo domain.BuildRepository, sessionRepo domain.UploadSessionRepository, fs FileStorage) *ProjectService {
	return &ProjectService{repo: repo, buildRepo: buildRepo, sessionRepo: sessionRepo, fs: fs}
}

func (s *ProjectService) Create(ctx context.Context, envID, id, name string) (*domain.Project, error) {
	p, err := domain.NewProject(envID, id, name)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	if err := s.fs.InitProject(id); err != nil {
		_ = s.repo.Delete(ctx, envID, id) // best-effort rollback
		return nil, err
	}
	return p, nil
}

func (s *ProjectService) List(ctx context.Context, envID string) ([]*domain.Project, error) {
	return s.repo.List(ctx, envID)
}

// ListSummaries returns all projects in an environment enriched with build stats.
// Uses BatchStatsByProject to fetch all stats in 2 queries instead of 2N+1 (M-08).
func (s *ProjectService) ListSummaries(ctx context.Context, envID string) ([]*ProjectSummary, error) {
	projects, err := s.repo.List(ctx, envID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(projects))
	for i, p := range projects {
		ids[i] = p.ID
	}
	batchStats, err := s.buildRepo.BatchStatsByProject(ctx, ids)
	if err != nil {
		return nil, err
	}
	summaries := make([]*ProjectSummary, 0, len(projects))
	for _, p := range projects {
		stats := batchStats[p.ID]
		sum := &ProjectSummary{Project: p, LastStatus: "inactive"}
		if stats != nil {
			sum.BuildCount = stats.Count
			if stats.Latest != nil {
				t := stats.Latest.CreatedAt
				sum.LastBuildAt = &t
				sum.LastTotal = stats.Latest.Total
				sum.LastPassed = stats.Latest.Passed
				sum.LastFailed = stats.Latest.Failed
				switch stats.Latest.Status {
				case "failed", "broken":
					sum.LastStatus = "failed"
				case "passed":
					sum.LastStatus = "passed"
				default:
					if stats.Latest.Failed > 0 {
						sum.LastStatus = "failed"
					} else if stats.Latest.Passed > 0 {
						sum.LastStatus = "passed"
					} else {
						sum.LastStatus = "active"
					}
				}
			}
		}
		summaries = append(summaries, sum)
	}
	return summaries, nil
}

func (s *ProjectService) Delete(ctx context.Context, envID, id string) error {
	if err := s.repo.Delete(ctx, envID, id); err != nil {
		return err
	}
	_ = s.sessionRepo.DeleteByProject(ctx, id) // best-effort
	return s.fs.RemoveProject(id)
}
