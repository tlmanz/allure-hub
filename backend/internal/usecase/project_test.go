package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
)

func TestProjectService_Create_Valid(t *testing.T) {
	projRepo := newMemProjectRepo()
	fs := &memFS{}
	svc := usecase.NewProjectService(projRepo, &memBuildRepo{}, &memSessionRepo{}, fs)

	p, err := svc.Create(context.Background(), "env-1", "my-proj", "My Proj")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID != "my-proj" || p.EnvironmentID != "env-1" {
		t.Errorf("unexpected project: %+v", p)
	}
	if len(fs.initedIDs) != 1 || fs.initedIDs[0] != "my-proj" {
		t.Errorf("InitProject not called correctly, got: %v", fs.initedIDs)
	}
}

func TestProjectService_Create_InvalidID(t *testing.T) {
	svc := usecase.NewProjectService(newMemProjectRepo(), &memBuildRepo{}, &memSessionRepo{}, &memFS{})
	_, err := svc.Create(context.Background(), "env-1", "My.Project", "My Project")
	if !errors.Is(err, domain.ErrInvalidProjectID) {
		t.Errorf("want ErrInvalidProjectID, got %v", err)
	}
}

func TestProjectService_Create_FSError_Rollback(t *testing.T) {
	projRepo := newMemProjectRepo()
	fs := &memFS{initErr: errors.New("disk full")}
	svc := usecase.NewProjectService(projRepo, &memBuildRepo{}, &memSessionRepo{}, fs)

	ctx := context.Background()
	_, err := svc.Create(ctx, "env-1", "my-proj", "My Proj")
	if err == nil {
		t.Fatal("expected error from fs.InitProject, got nil")
	}
	if _, getErr := projRepo.Get(ctx, "env-1", "my-proj"); !errors.Is(getErr, domain.ErrProjectNotFound) {
		t.Errorf("expected project rolled back, got: %v", getErr)
	}
}

func TestProjectService_ListSummaries_NoBuilds_Inactive(t *testing.T) {
	projRepo := newMemProjectRepo()
	ctx := context.Background()
	_ = projRepo.Create(ctx, &domain.Project{ID: "proj-1", EnvironmentID: "env-1", Name: "P1"})

	svc := usecase.NewProjectService(projRepo, &memBuildRepo{}, &memSessionRepo{}, &memFS{})
	summaries, err := svc.ListSummaries(ctx, "env-1")
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("want 1 summary, got %d", len(summaries))
	}
	if summaries[0].LastStatus != "inactive" {
		t.Errorf("LastStatus = %q, want inactive", summaries[0].LastStatus)
	}
	if summaries[0].BuildCount != 0 {
		t.Errorf("BuildCount = %d, want 0", summaries[0].BuildCount)
	}
}

func TestProjectService_ListSummaries_PassedStatus(t *testing.T) {
	projRepo := newMemProjectRepo()
	ctx := context.Background()
	_ = projRepo.Create(ctx, &domain.Project{ID: "proj-1", EnvironmentID: "env-1", Name: "P1"})

	now := time.Now().UTC()
	builds := &memBuildRepo{
		batchStats: map[string]*domain.ProjectBatchStats{
			"proj-1": {
				Count: 3,
				Latest: &domain.Build{
					ID:        "b1",
					Status:    "passed",
					Passed:    10,
					Failed:    0,
					Total:     10,
					CreatedAt: now,
				},
			},
		},
	}
	svc := usecase.NewProjectService(projRepo, builds, &memSessionRepo{}, &memFS{})
	summaries, err := svc.ListSummaries(ctx, "env-1")
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if summaries[0].LastStatus != "passed" {
		t.Errorf("LastStatus = %q, want passed", summaries[0].LastStatus)
	}
	if summaries[0].BuildCount != 3 {
		t.Errorf("BuildCount = %d, want 3", summaries[0].BuildCount)
	}
	if summaries[0].LastPassed != 10 {
		t.Errorf("LastPassed = %d, want 10", summaries[0].LastPassed)
	}
}

func TestProjectService_ListSummaries_FailedStatus(t *testing.T) {
	projRepo := newMemProjectRepo()
	ctx := context.Background()
	_ = projRepo.Create(ctx, &domain.Project{ID: "proj-2", EnvironmentID: "env-1", Name: "P2"})

	now := time.Now().UTC()
	builds := &memBuildRepo{
		batchStats: map[string]*domain.ProjectBatchStats{
			"proj-2": {
				Count: 1,
				Latest: &domain.Build{
					ID:        "b2",
					Status:    "failed",
					Passed:    5,
					Failed:    3,
					Total:     8,
					CreatedAt: now,
				},
			},
		},
	}
	svc := usecase.NewProjectService(projRepo, builds, &memSessionRepo{}, &memFS{})
	summaries, _ := svc.ListSummaries(ctx, "env-1")
	if summaries[0].LastStatus != "failed" {
		t.Errorf("LastStatus = %q, want failed", summaries[0].LastStatus)
	}
	if summaries[0].LastFailed != 3 {
		t.Errorf("LastFailed = %d, want 3", summaries[0].LastFailed)
	}
}

func TestProjectService_ListSummaries_BrokenStatus(t *testing.T) {
	projRepo := newMemProjectRepo()
	ctx := context.Background()
	_ = projRepo.Create(ctx, &domain.Project{ID: "proj-3", EnvironmentID: "env-1", Name: "P3"})

	now := time.Now().UTC()
	builds := &memBuildRepo{
		batchStats: map[string]*domain.ProjectBatchStats{
			"proj-3": {
				Count: 1,
				Latest: &domain.Build{
					ID:        "b3",
					Status:    "broken",
					Passed:    2,
					Failed:    1,
					Total:     3,
					CreatedAt: now,
				},
			},
		},
	}
	svc := usecase.NewProjectService(projRepo, builds, &memSessionRepo{}, &memFS{})
	summaries, _ := svc.ListSummaries(ctx, "env-1")
	if summaries[0].LastStatus != "failed" {
		t.Errorf("broken should map to failed, got %q", summaries[0].LastStatus)
	}
}

func TestProjectService_Delete_CascadesBuilds(t *testing.T) {
	projRepo := newMemProjectRepo()
	fs := &memFS{}
	ctx := context.Background()
	_ = projRepo.Create(ctx, &domain.Project{ID: "proj-del", EnvironmentID: "env-1", Name: "To Delete"})

	svc := usecase.NewProjectService(projRepo, &memBuildRepo{}, &memSessionRepo{}, fs)
	if err := svc.Delete(ctx, "env-1", "proj-del"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(fs.removedIDs) != 1 || fs.removedIDs[0] != "proj-del" {
		t.Errorf("RemoveProject not called with correct id, got: %v", fs.removedIDs)
	}
}
