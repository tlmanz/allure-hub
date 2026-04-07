package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tlmanz/allure-hub/internal/usecase"
)

func TestAPIKeyService_Create_ValidRoles(t *testing.T) {
	for _, role := range []string{"admin", "developer", "viewer"} {
		t.Run(role, func(t *testing.T) {
			svc := usecase.NewAPIKeyService(newMemAPIKeyRepo())
			res, err := svc.Create(context.Background(), "ci-key", role, "admin@example.com", false)
			if err != nil {
				t.Fatalf("Create(%q): %v", role, err)
			}
			if !strings.HasPrefix(res.Plaintext, "ah_") {
				t.Errorf("Plaintext %q does not have ah_ prefix", res.Plaintext)
			}
			if res.Key.Role != role {
				t.Errorf("Role = %q, want %q", res.Key.Role, role)
			}
			if !res.Key.IsActive {
				t.Error("IsActive = false, want true")
			}
			if res.Key.Name != "ci-key" {
				t.Errorf("Name = %q, want ci-key", res.Key.Name)
			}
		})
	}
}

func TestAPIKeyService_Create_InvalidRole(t *testing.T) {
	svc := usecase.NewAPIKeyService(newMemAPIKeyRepo())
	_, err := svc.Create(context.Background(), "key", "superadmin", "admin@example.com", false)
	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}
}

func TestAPIKeyService_Create_RepoError(t *testing.T) {
	repo := &memAPIKeyRepo{err: errAny}
	svc := usecase.NewAPIKeyService(repo)
	_, err := svc.Create(context.Background(), "key", "viewer", "admin@example.com", false)
	if err == nil {
		t.Fatal("expected repo error to propagate, got nil")
	}
}

func TestAPIKeyService_List(t *testing.T) {
	svc := usecase.NewAPIKeyService(newMemAPIKeyRepo())
	ctx := context.Background()
	_, _ = svc.Create(ctx, "key-1", "admin", "a@b.com", false)
	_, _ = svc.Create(ctx, "key-2", "viewer", "a@b.com", false)

	keys, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("List: got %d keys, want 2", len(keys))
	}
}

func TestAPIKeyService_Revoke(t *testing.T) {
	svc := usecase.NewAPIKeyService(newMemAPIKeyRepo())
	ctx := context.Background()
	res, _ := svc.Create(ctx, "ci-key", "developer", "admin@example.com", false)

	if err := svc.Revoke(ctx, res.Key.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	keys, _ := svc.List(ctx)
	if len(keys) != 1 || keys[0].IsActive {
		t.Errorf("expected key to be inactive after revoke, got: %+v", keys)
	}
}

func TestAPIKeyService_Delete(t *testing.T) {
	svc := usecase.NewAPIKeyService(newMemAPIKeyRepo())
	ctx := context.Background()
	res, _ := svc.Create(ctx, "ci-key", "viewer", "admin@example.com", false)

	if err := svc.Delete(ctx, res.Key.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	keys, _ := svc.List(ctx)
	if len(keys) != 0 {
		t.Errorf("expected no keys after delete, got %d", len(keys))
	}
}
