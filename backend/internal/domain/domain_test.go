package domain_test

import (
	"errors"
	"testing"

	"github.com/tlmanz/allure-hub/internal/domain"
)

func TestNewEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		envName  string
		icon     string
		wantErr  error
		wantIcon string
	}{
		{"valid with icon", "my-env", "My Env", "rocket", nil, "rocket"},
		{"valid empty icon defaults", "my-env", "My Env", "", nil, domain.DefaultEnvironmentIcon},
		{"missing name", "my-env", "", "rocket", domain.ErrMissingName, ""},
		{"invalid id uppercase", "My-Env", "My Env", "", domain.ErrInvalidEnvironmentID, ""},
		{"invalid id spaces", "my env", "My Env", "", domain.ErrInvalidEnvironmentID, ""},
		{"invalid id starts with hyphen", "-env", "My Env", "", domain.ErrInvalidEnvironmentID, ""},
		{"valid id with numbers", "env-01", "Env 01", "", nil, domain.DefaultEnvironmentIcon},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e, err := domain.NewEnvironment(tc.id, tc.envName, tc.icon)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("NewEnvironment(%q, %q, %q) err = %v, want %v", tc.id, tc.envName, tc.icon, err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if e.ID != tc.id {
				t.Errorf("ID = %q, want %q", e.ID, tc.id)
			}
			if e.Name != tc.envName {
				t.Errorf("Name = %q, want %q", e.Name, tc.envName)
			}
			if e.Icon != tc.wantIcon {
				t.Errorf("Icon = %q, want %q", e.Icon, tc.wantIcon)
			}
			if e.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})
	}
}

func TestNewProject(t *testing.T) {
	tests := []struct {
		name     string
		envID    string
		id       string
		projName string
		wantErr  error
	}{
		{"valid", "env-1", "my-project", "My Project", nil},
		{"missing name", "env-1", "proj", "", domain.ErrMissingName},
		{"invalid id uppercase", "env-1", "MyProject", "My Project", domain.ErrInvalidProjectID},
		{"invalid id with dot", "env-1", "my.project", "My Project", domain.ErrInvalidProjectID},
		{"valid alphanumeric", "env-1", "proj01", "Proj 01", nil},
		{"valid with hyphen", "env-1", "my-proj-1", "My Proj", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := domain.NewProject(tc.envID, tc.id, tc.projName)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("NewProject err = %v, want %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if p.ID != tc.id {
				t.Errorf("ID = %q, want %q", p.ID, tc.id)
			}
			if p.EnvironmentID != tc.envID {
				t.Errorf("EnvironmentID = %q, want %q", p.EnvironmentID, tc.envID)
			}
			if p.Name != tc.projName {
				t.Errorf("Name = %q, want %q", p.Name, tc.projName)
			}
			if p.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})
	}
}
