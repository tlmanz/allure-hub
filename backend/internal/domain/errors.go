package domain

import (
	"errors"
	"regexp"
)

// Sentinel errors shared across domain entities.
var (
	ErrMissingName = errors.New("name is required")
)

// idPattern is the shared validation rule for all entity IDs:
// lowercase letters, digits, and hyphens; must start with a letter or digit.
var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*$`)
