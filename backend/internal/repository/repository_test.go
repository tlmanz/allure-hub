package repository_test

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/repository"
)

func TestSQLiteRepositories(t *testing.T) {
	testRepositories(t, "sqlite", ":memory:")
}

func TestPostgresRepositories(t *testing.T) {
	t.Skip("set POSTGRES_DSN env and remove this skip to run Postgres integration tests")
}

func testRepositories(t *testing.T, driver, dsn string) {
	t.Helper()
	ctx := context.Background()

	db, err := repository.Open(driver, dsn, repository.PoolConfig{
		MaxOpenConns: 5, MaxIdleConns: 2,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	envRepo := repository.NewEnvironmentRepo(db)
	projectRepo := repository.NewProjectRepo(db)
	buildRepo := repository.NewBuildRepo(db)

	// ── Environments ─────────────────────────────────────────────────────────

	env := &domain.Environment{ID: "test-env", Name: "Test Env", CreatedAt: time.Now().UTC()}
	if err := envRepo.Create(ctx, env); err != nil {
		t.Fatalf("Create environment: %v", err)
	}

	// ── Projects ─────────────────────────────────────────────────────────────

	p := &domain.Project{ID: "my-project", EnvironmentID: "test-env", Name: "My Project", CreatedAt: time.Now().UTC()}
	if err := projectRepo.Create(ctx, p); err != nil {
		t.Fatalf("Create project: %v", err)
	}

	got, err := projectRepo.Get(ctx, "test-env", "my-project")
	if err != nil {
		t.Fatalf("Get project: %v", err)
	}
	if got.Name != "My Project" {
		t.Errorf("Get: name = %q, want %q", got.Name, "My Project")
	}

	list, err := projectRepo.List(ctx, "test-env")
	if err != nil {
		t.Fatalf("List projects: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List: got %d projects, want 1", len(list))
	}

	// ── Builds ───────────────────────────────────────────────────────────────

	b := &domain.Build{
		ID:        "build-uuid-1",
		EnvID:     "test-env",
		ProjectID: "my-project",
		BuildID:   "build-001",
		CreatedAt: time.Now().UTC(),
		ReportURL: "/reports/test-env/my-project/build-001/index.html",
		Passed:    42,
		Failed:    3,
	}
	if err := buildRepo.Save(ctx, b); err != nil {
		t.Fatalf("Save build: %v", err)
	}

	builds, err := buildRepo.ListByProject(ctx, "test-env", "my-project")
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(builds) != 1 {
		t.Fatalf("ListByProject: got %d builds, want 1", len(builds))
	}
	if builds[0].Passed != 42 || builds[0].Failed != 3 {
		t.Errorf("build stats: passed=%d failed=%d, want 42/3", builds[0].Passed, builds[0].Failed)
	}

	// Upsert - re-save with updated stats
	b.Passed = 50
	b.Failed = 0
	if err := buildRepo.Save(ctx, b); err != nil {
		t.Fatalf("Save build upsert: %v", err)
	}
	builds, _ = buildRepo.ListByProject(ctx, "test-env", "my-project")
	if builds[0].Passed != 50 {
		t.Errorf("upsert: passed = %d, want 50", builds[0].Passed)
	}

	// ── Delete project (cascades to builds) ──────────────────────────────────

	if err := projectRepo.Delete(ctx, "test-env", "my-project"); err != nil {
		t.Fatalf("Delete project: %v", err)
	}
	list, _ = projectRepo.List(ctx, "test-env")
	if len(list) != 0 {
		t.Errorf("after delete: %d projects remain, want 0", len(list))
	}
}
