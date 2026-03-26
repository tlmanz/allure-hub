package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tlmanz/allure-hub/internal/apikey"
	"github.com/tlmanz/allure-hub/internal/domain"
)

// APIKeyService manages API key lifecycle: create, list, revoke, delete.
type APIKeyService struct {
	repo domain.APIKeyRepository
}

func NewAPIKeyService(repo domain.APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// CreateResult holds the newly created key record and the plaintext key.
// The plaintext is shown once to the caller; it is never stored.
type CreateResult struct {
	Key       *domain.APIKey
	Plaintext string
}

// Create generates a new API key, persists the hash, and returns the plaintext once.
func (s *APIKeyService) Create(ctx context.Context, name, role, createdBy string) (*CreateResult, error) {
	switch role {
	case "admin", "developer", "viewer":
	default:
		return nil, fmt.Errorf("invalid role %q: must be admin, developer, or viewer", role)
	}

	plaintext, hash, err := apikey.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	k := &domain.APIKey{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedBy: createdBy,
		Role:      role,
		KeyHash:   hash,
		CreatedAt: time.Now().UTC(),
		IsActive:  true,
	}
	if err := s.repo.Create(ctx, k); err != nil {
		return nil, fmt.Errorf("persist api key: %w", err)
	}
	return &CreateResult{Key: k, Plaintext: plaintext}, nil
}

// List returns all API keys (hashes are excluded from the returned structs).
func (s *APIKeyService) List(ctx context.Context) ([]*domain.APIKey, error) {
	return s.repo.List(ctx)
}

// Search returns a page of keys matching query, plus the total match count.
func (s *APIKeyService) Search(ctx context.Context, query string, limit, offset int) ([]*domain.APIKey, int, error) {
	keys, err := s.repo.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountSearch(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return keys, total, nil
}

// Revoke soft-deletes a key by setting is_active = false.
func (s *APIKeyService) Revoke(ctx context.Context, id string) error {
	return s.repo.Revoke(ctx, id)
}

// Delete permanently removes a key record.
func (s *APIKeyService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
