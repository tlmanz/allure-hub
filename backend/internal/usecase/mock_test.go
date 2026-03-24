package usecase_test

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
)

// errAny is a generic sentinel used to simulate repository failures.
var errAny = errors.New("unexpected error")

// ─── Environment Repo ─────────────────────────────────────────────────────────

type memEnvRepo struct {
	envs map[string]*domain.Environment
	err  error
}

func newMemEnvRepo() *memEnvRepo {
	return &memEnvRepo{envs: make(map[string]*domain.Environment)}
}

func (r *memEnvRepo) Create(_ context.Context, e *domain.Environment) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.envs[e.ID]; ok {
		return domain.ErrEnvironmentExists
	}
	cp := *e
	r.envs[e.ID] = &cp
	return nil
}

func (r *memEnvRepo) Get(_ context.Context, id string) (*domain.Environment, error) {
	if r.err != nil {
		return nil, r.err
	}
	e, ok := r.envs[id]
	if !ok {
		return nil, domain.ErrEnvironmentNotFound
	}
	cp := *e
	return &cp, nil
}

func (r *memEnvRepo) List(_ context.Context) ([]*domain.Environment, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := make([]*domain.Environment, 0, len(r.envs))
	for _, e := range r.envs {
		cp := *e
		out = append(out, &cp)
	}
	return out, nil
}

func (r *memEnvRepo) CountProjectsBatch(_ context.Context, ids []string) (map[string]int, error) {
	if r.err != nil {
		return nil, r.err
	}
	m := make(map[string]int, len(ids))
	for _, id := range ids {
		m[id] = 0
	}
	return m, nil
}

func (r *memEnvRepo) Update(_ context.Context, id, name, icon string) error {
	if r.err != nil {
		return r.err
	}
	e, ok := r.envs[id]
	if !ok {
		return domain.ErrEnvironmentNotFound
	}
	e.Name = name
	e.Icon = icon
	return nil
}

func (r *memEnvRepo) Delete(_ context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	delete(r.envs, id)
	return nil
}

// ─── Project Repo ─────────────────────────────────────────────────────────────

type memProjectRepo struct {
	projects  map[string]*domain.Project
	deleteErr error
}

func newMemProjectRepo() *memProjectRepo {
	return &memProjectRepo{projects: make(map[string]*domain.Project)}
}

func (r *memProjectRepo) Create(_ context.Context, p *domain.Project) error {
	if _, ok := r.projects[p.ID]; ok {
		return domain.ErrProjectExists
	}
	cp := *p
	r.projects[p.ID] = &cp
	return nil
}

func (r *memProjectRepo) Get(_ context.Context, envID, id string) (*domain.Project, error) {
	p, ok := r.projects[id]
	if !ok || p.EnvironmentID != envID {
		return nil, domain.ErrProjectNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *memProjectRepo) List(_ context.Context, envID string) ([]*domain.Project, error) {
	var out []*domain.Project
	for _, p := range r.projects {
		if p.EnvironmentID == envID {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *memProjectRepo) Delete(_ context.Context, _ string, id string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.projects, id)
	return nil
}

// ─── Build Repo ───────────────────────────────────────────────────────────────

type memBuildRepo struct {
	batchStats map[string]*domain.ProjectBatchStats
}

func (r *memBuildRepo) Save(_ context.Context, _ *domain.Build) error { return nil }
func (r *memBuildRepo) GetByBuildID(_ context.Context, _, _, _ string) (*domain.Build, error) {
	return nil, domain.ErrBuildNotFound
}
func (r *memBuildRepo) BatchStatsByProject(_ context.Context, _ string, ids []string) (map[string]*domain.ProjectBatchStats, error) {
	if r.batchStats != nil {
		return r.batchStats, nil
	}
	return make(map[string]*domain.ProjectBatchStats, len(ids)), nil
}
func (r *memBuildRepo) ListByProject(_ context.Context, _, _ string) ([]*domain.Build, error) {
	return nil, nil
}
func (r *memBuildRepo) ListByProjectPaged(_ context.Context, _, _, _ string, _, _ int) ([]*domain.Build, error) {
	return nil, nil
}
func (r *memBuildRepo) CountByProjectFiltered(_ context.Context, _, _, _ string) (int, error) {
	return 0, nil
}
func (r *memBuildRepo) CountByProject(_ context.Context, _, _ string) (int, error) { return 0, nil }
func (r *memBuildRepo) LatestByProject(_ context.Context, _, _ string) (*domain.Build, error) {
	return nil, domain.ErrBuildNotFound
}
func (r *memBuildRepo) StatsForProject(_ context.Context, _, _ string) (*domain.BuildStats, error) {
	return &domain.BuildStats{}, nil
}
func (r *memBuildRepo) Delete(_ context.Context, _, _, _ string) error                               { return nil }
func (r *memBuildRepo) DeleteByProject(_ context.Context, _, _ string) error                        { return nil }
func (r *memBuildRepo) ListExpiredBuilds(_ context.Context, _ time.Time) ([]*domain.Build, error) { return nil, nil }

// ─── Upload Session Repo ──────────────────────────────────────────────────────

type memSessionRepo struct{}

func (r *memSessionRepo) Create(_ context.Context, _ *domain.UploadSession) error { return nil }
func (r *memSessionRepo) Update(_ context.Context, _ *domain.UploadSession) error { return nil }
func (r *memSessionRepo) IncrementReceivedChunks(_ context.Context, _ string) (*domain.UploadSession, error) {
	return nil, nil
}
func (r *memSessionRepo) GetByUploadID(_ context.Context, _ string) (*domain.UploadSession, error) {
	return nil, nil
}
func (r *memSessionRepo) GetByBuild(_ context.Context, _, _ string) (*domain.UploadSession, error) {
	return nil, nil
}
func (r *memSessionRepo) ListRecent(_ context.Context, _ int) ([]*domain.UploadSession, error) {
	return nil, nil
}
func (r *memSessionRepo) GetByID(_ context.Context, _ string) (*domain.UploadSession, error) {
	return nil, nil
}
func (r *memSessionRepo) Delete(_ context.Context, _ string) error          { return nil }
func (r *memSessionRepo) DeleteByProject(_ context.Context, _ string) error { return nil }
func (r *memSessionRepo) DeleteByEnv(_ context.Context, _ string) error     { return nil }

// ─── FileStorage ──────────────────────────────────────────────────────────────

type memFS struct {
	initErr    error
	removedIDs []string
	initedIDs  []string
}

func (f *memFS) InitProject(envID, id string) error {
	if f.initErr != nil {
		return f.initErr
	}
	f.initedIDs = append(f.initedIDs, id)
	return nil
}
func (f *memFS) RemoveProject(envID, id string) error {
	f.removedIDs = append(f.removedIDs, id)
	return nil
}
func (f *memFS) SaveResultsStream(_, _, _ string, _ io.Reader) error        { return nil }
func (f *memFS) ResultsDir(_, _, _ string) string                           { return "" }
func (f *memFS) ReportDir(_, _, _ string) string                            { return "" }
func (f *memFS) HistoryDir(_, _ string) string                              { return "" }
func (f *memFS) HistoryFile(_, _ string) string                             { return "" }
func (f *memFS) ChunkDir(_, _, _ string) string                             { return "" }
func (f *memFS) ChunkPath(_, _, _ string, _ int) string                     { return "" }
func (f *memFS) WriteUploadMeta(_, _, _ string, _ usecase.UploadMeta) error { return nil }
func (f *memFS) ReadUploadMeta(_, _, _ string) (usecase.UploadMeta, error) {
	return usecase.UploadMeta{}, nil
}

// ─── API Key Repo ─────────────────────────────────────────────────────────────

type memAPIKeyRepo struct {
	keys map[string]*domain.APIKey
	err  error
}

func newMemAPIKeyRepo() *memAPIKeyRepo {
	return &memAPIKeyRepo{keys: make(map[string]*domain.APIKey)}
}

func (r *memAPIKeyRepo) Create(_ context.Context, k *domain.APIKey) error {
	if r.err != nil {
		return r.err
	}
	cp := *k
	r.keys[k.ID] = &cp
	return nil
}
func (r *memAPIKeyRepo) GetByHash(_ context.Context, h string) (*domain.APIKey, error) {
	for _, k := range r.keys {
		if k.KeyHash == h {
			return k, nil
		}
	}
	return nil, nil
}
func (r *memAPIKeyRepo) List(_ context.Context) ([]*domain.APIKey, error) {
	out := make([]*domain.APIKey, 0, len(r.keys))
	for _, k := range r.keys {
		cp := *k
		out = append(out, &cp)
	}
	return out, nil
}
func (r *memAPIKeyRepo) Search(_ context.Context, _ string, _, _ int) ([]*domain.APIKey, error) {
	return r.List(context.Background())
}
func (r *memAPIKeyRepo) CountSearch(_ context.Context, _ string) (int, error) {
	return len(r.keys), nil
}
func (r *memAPIKeyRepo) UpdateLastUsed(_ context.Context, _ string) error { return nil }
func (r *memAPIKeyRepo) Revoke(_ context.Context, id string) error {
	if k, ok := r.keys[id]; ok {
		k.IsActive = false
	}
	return nil
}
func (r *memAPIKeyRepo) Delete(_ context.Context, id string) error {
	delete(r.keys, id)
	return nil
}
