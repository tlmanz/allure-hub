package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
)

func newEnvSvc(envRepo *memEnvRepo, projRepo *memProjectRepo, builds *memBuildRepo, fs *memFS) *usecase.EnvironmentService {
	return usecase.NewEnvironmentService(envRepo, projRepo, builds, &memSessionRepo{}, fs)
}

func TestEnvironmentService_Create_Valid(t *testing.T) {
	svc := newEnvSvc(newMemEnvRepo(), newMemProjectRepo(), &memBuildRepo{}, &memFS{})
	env, err := svc.Create(context.Background(), "my-env", "My Env", "rocket")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if env.ID != "my-env" || env.Name != "My Env" || env.Icon != "rocket" {
		t.Errorf("unexpected env: %+v", env)
	}
}

func TestEnvironmentService_Create_InvalidID(t *testing.T) {
	svc := newEnvSvc(newMemEnvRepo(), newMemProjectRepo(), &memBuildRepo{}, &memFS{})
	_, err := svc.Create(context.Background(), "My Env!", "My Env", "")
	if !errors.Is(err, domain.ErrInvalidEnvironmentID) {
		t.Errorf("want ErrInvalidEnvironmentID, got %v", err)
	}
}

func TestEnvironmentService_Create_MissingName(t *testing.T) {
	svc := newEnvSvc(newMemEnvRepo(), newMemProjectRepo(), &memBuildRepo{}, &memFS{})
	_, err := svc.Create(context.Background(), "my-env", "", "")
	if !errors.Is(err, domain.ErrMissingName) {
		t.Errorf("want ErrMissingName, got %v", err)
	}
}

func TestEnvironmentService_Create_Duplicate(t *testing.T) {
	svc := newEnvSvc(newMemEnvRepo(), newMemProjectRepo(), &memBuildRepo{}, &memFS{})
	ctx := context.Background()
	_, _ = svc.Create(ctx, "my-env", "My Env", "")
	_, err := svc.Create(ctx, "my-env", "My Env 2", "")
	if !errors.Is(err, domain.ErrEnvironmentExists) {
		t.Errorf("want ErrEnvironmentExists, got %v", err)
	}
}

func TestEnvironmentService_List_PopulatesProjectCounts(t *testing.T) {
	envRepo := newMemEnvRepo()
	svc := newEnvSvc(envRepo, newMemProjectRepo(), &memBuildRepo{}, &memFS{})
	ctx := context.Background()
	_, _ = svc.Create(ctx, "env-a", "Env A", "")
	_, _ = svc.Create(ctx, "env-b", "Env B", "")

	envs, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(envs) != 2 {
		t.Fatalf("want 2 envs, got %d", len(envs))
	}
	for _, e := range envs {
		if e.ProjectCount != 0 {
			t.Errorf("env %s: ProjectCount = %d, want 0", e.ID, e.ProjectCount)
		}
	}
}

func TestEnvironmentService_Update_MissingName(t *testing.T) {
	svc := newEnvSvc(newMemEnvRepo(), newMemProjectRepo(), &memBuildRepo{}, &memFS{})
	_, err := svc.Update(context.Background(), "env-1", "", "")
	if !errors.Is(err, domain.ErrMissingName) {
		t.Errorf("want ErrMissingName, got %v", err)
	}
}

func TestEnvironmentService_Update_DefaultIcon(t *testing.T) {
	envRepo := newMemEnvRepo()
	svc := newEnvSvc(envRepo, newMemProjectRepo(), &memBuildRepo{}, &memFS{})
	ctx := context.Background()
	_, _ = svc.Create(ctx, "env-1", "Env 1", "old-icon")

	updated, err := svc.Update(ctx, "env-1", "New Name", "")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Icon != domain.DefaultEnvironmentIcon {
		t.Errorf("Icon = %q, want default %q", updated.Icon, domain.DefaultEnvironmentIcon)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
}

func TestEnvironmentService_Delete_CascadesProjects(t *testing.T) {
	envRepo := newMemEnvRepo()
	projRepo := newMemProjectRepo()
	fs := &memFS{}
	ctx := context.Background()

	svc := newEnvSvc(envRepo, projRepo, &memBuildRepo{}, fs)
	_, _ = svc.Create(ctx, "env-1", "Env 1", "")

	projSvc := usecase.NewProjectService(projRepo, &memBuildRepo{}, &memSessionRepo{}, fs)
	_, _ = projSvc.Create(ctx, "env-1", "proj-a", "Proj A")

	if err := svc.Delete(ctx, "env-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := projRepo.Get(ctx, "env-1", "proj-a"); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("expected project deleted, got err: %v", err)
	}
	if len(fs.removedIDs) == 0 {
		t.Error("expected RemoveProject to be called")
	}
}
