package domain

import (
	"errors"
	"time"
)

var (
	ErrEnvironmentNotFound  = errors.New("environment not found")
	ErrEnvironmentExists    = errors.New("environment already exists")
	ErrInvalidEnvironmentID = errors.New("environment id must be lowercase letters, numbers, and hyphens only")
)

// DefaultEnvironmentIcon is used when no icon is specified.
const DefaultEnvironmentIcon = "deployed_code"

// Environment is a logical grouping that contains multiple Projects.
type Environment struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Icon         string    `json:"icon"`
	CreatedAt    time.Time `json:"createdAt"`
	ProjectCount int       `json:"projectCount"`
}

// NewEnvironment constructs and validates an Environment entity.
func NewEnvironment(id, name, icon string) (*Environment, error) {
	if !idPattern.MatchString(id) {
		return nil, ErrInvalidEnvironmentID
	}
	if name == "" {
		return nil, ErrMissingName
	}
	if icon == "" {
		icon = DefaultEnvironmentIcon
	}
	return &Environment{
		ID:        id,
		Name:      name,
		Icon:      icon,
		CreatedAt: time.Now().UTC(),
	}, nil
}
