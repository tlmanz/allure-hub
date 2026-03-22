package domain

import (
	"errors"
	"time"
)

var (
	ErrProjectNotFound  = errors.New("project not found")
	ErrProjectExists    = errors.New("project already exists")
	ErrInvalidProjectID = errors.New("project id must be lowercase letters, numbers, and hyphens only")
)

// Project is the core domain entity for a test project.
type Project struct {
	ID            string    `json:"id"`
	EnvironmentID string    `json:"environmentId"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"createdAt"`
}

// NewProject constructs and validates a Project entity.
func NewProject(envID, id, name string) (*Project, error) {
	if !idPattern.MatchString(id) {
		return nil, ErrInvalidProjectID
	}
	if name == "" {
		return nil, ErrMissingName
	}
	return &Project{
		ID:            id,
		EnvironmentID: envID,
		Name:          name,
		CreatedAt:     time.Now().UTC(),
	}, nil
}
