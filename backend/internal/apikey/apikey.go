// Package apikey provides bearer token API key authentication for programmatic
// access (CI/CD pipelines, scripts). Keys are stored with only their SHA-256
// hash in the database - the plaintext is shown once at creation and never
// persisted.
//
// Key format:  ah_<64 hex chars>   (ah_ prefix aids secret scanning)
// Header:      Authorization: Bearer ah_<key>
//
//	X-API-Key: ah_<key>   (alternative)
//
// Store implements authkit.APIKeyValidator so it can be passed directly to
// authkit.Config.APIKeyValidator. Authkit then handles header extraction and
// context injection using the same context key as OAuth session users, meaning
// kit.UserFromCtx works transparently for both auth paths.
package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
	kit "github.com/tlmanz/authkit"
)

// Generate creates a new API key string and its SHA-256 hash.
// The key is NOT persisted - callers must store the hash via APIKeyRepository.
func Generate() (key, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	key = "ah_" + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(key))
	hash = hex.EncodeToString(sum[:])
	return
}

// HashKey returns the hex-encoded SHA-256 hash of key.
func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Store wraps an APIKeyRepository for request-time validation.
// It implements authkit.APIKeyValidator.
type Store struct {
	repo domain.APIKeyRepository
}

// NewStore creates a Store backed by the given repository.
func NewStore(repo domain.APIKeyRepository) *Store {
	return &Store{repo: repo}
}

// ValidateKey implements authkit.APIKeyValidator.
// It hashes rawKey, looks it up in the database, and returns a synthetic
// *kit.User on success. Returns nil, nil when the key is not found, inactive,
// or expired. Returns nil, err only for unexpected infrastructure failures.
func (s *Store) ValidateKey(ctx context.Context, rawKey string) (*kit.User, error) {
	if !strings.HasPrefix(rawKey, "ah_") {
		return nil, nil
	}
	rec, err := s.repo.GetByHash(ctx, HashKey(rawKey))
	if err != nil {
		return nil, err
	}
	if rec == nil || !rec.IsActive {
		return nil, nil
	}
	if rec.ExpiresAt != nil && time.Now().After(*rec.ExpiresAt) {
		return nil, nil
	}
	// Update last_used_at asynchronously - non-blocking.
	go s.repo.UpdateLastUsed(ctx, rec.ID)
	return &kit.User{
		Email:    "apikey:" + rec.Name,
		Name:     rec.Name,
		Provider: "apikey",
		Role:     rec.Role,
	}, nil
}
