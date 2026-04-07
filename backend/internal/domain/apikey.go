package domain

import (
	"errors"
	"time"
)

var ErrAPIKeyNotFound = errors.New("api key not found")

// APIKey is a long-lived bearer token for programmatic access (CI/CD pipelines).
// The plaintext key is shown once at creation time; only the SHA-256 hash is stored.
type APIKey struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	CreatedBy            string     `json:"createdBy"` // email of the admin who created it
	Role                 string     `json:"role"`
	KeyHash              string     `json:"-"` // hex(SHA-256(plaintext)) - never sent to clients
	LastUsedAt           *time.Time `json:"lastUsedAt"`
	CreatedAt            time.Time  `json:"createdAt"`
	ExpiresAt            *time.Time `json:"expiresAt,omitempty"`
	IsActive             bool       `json:"isActive"`
	AutoCreateEnvProject bool       `json:"autoCreateEnvProject"` // auto-create env/project on upload if missing
}
