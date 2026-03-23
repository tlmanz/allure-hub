package usecase

import (
	"context"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// EnvironmentService orchestrates environment lifecycle use-cases.
type EnvironmentService struct {
	repo        domain.EnvironmentRepository
	projectRepo domain.ProjectRepository
	buildRepo   domain.BuildRepository
	sessionRepo domain.UploadSessionRepository
	fs          FileStorage
}

func NewEnvironmentService(repo domain.EnvironmentRepository, projectRepo domain.ProjectRepository, buildRepo domain.BuildRepository, sessionRepo domain.UploadSessionRepository, fs FileStorage) *EnvironmentService {
	return &EnvironmentService{repo: repo, projectRepo: projectRepo, buildRepo: buildRepo, sessionRepo: sessionRepo, fs: fs}
}

func (s *EnvironmentService) Create(ctx context.Context, id, name, icon string) (*domain.Environment, error) {
	e, err := domain.NewEnvironment(id, name, icon)
	if err != nil {
		return nil, err
	}
	return e, s.repo.Create(ctx, e)
}

func (s *EnvironmentService) List(ctx context.Context) ([]*domain.Environment, error) {
	envs, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	// Fetch all project counts in one query instead of N+1 (M-09).
	ids := make([]string, len(envs))
	for i, e := range envs {
		ids[i] = e.ID
	}
	counts, err := s.repo.CountProjectsBatch(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, e := range envs {
		e.ProjectCount = counts[e.ID]
	}
	return envs, nil
}

func (s *EnvironmentService) Update(ctx context.Context, id, name, icon string) (*domain.Environment, error) {
	if name == "" {
		return nil, domain.ErrMissingName
	}
	if icon == "" {
		icon = domain.DefaultEnvironmentIcon
	}
	if err := s.repo.Update(ctx, id, name, icon); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}

// Delete removes an environment, all its projects, their builds, upload sessions, and files.
func (s *EnvironmentService) Delete(ctx context.Context, id string) error {
	projects, err := s.projectRepo.List(ctx, id)
	if err != nil {
		return err
	}
	for _, p := range projects {
		_ = s.buildRepo.DeleteByProject(ctx, id, p.ID) // best-effort
		_ = s.projectRepo.Delete(ctx, id, p.ID)        // best-effort
		_ = s.fs.RemoveProject(id, p.ID)               // best-effort
	}
	_ = s.sessionRepo.DeleteByEnv(ctx, id) // best-effort
	return s.repo.Delete(ctx, id)
}
