package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
)

// SettingsHandler exposes API key management and user-tracking endpoints.
// All routes require a valid OAuth session (no API key auth allowed here).
type SettingsHandler struct {
	apiKeySvc *usecase.APIKeyService
	userRepo  domain.TrackedUserRepository
	log       *zap.Logger
}

func NewSettingsHandler(apiKeySvc *usecase.APIKeyService, userRepo domain.TrackedUserRepository, log *zap.Logger) *SettingsHandler {
	return &SettingsHandler{apiKeySvc: apiKeySvc, userRepo: userRepo, log: log}
}

// ListAPIKeys returns all API keys (without key hashes).
//
//	GET /api/settings/apikeys
func (h *SettingsHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.apiKeySvc.List(r.Context())
	if err != nil {
		h.log.Error("list api keys failed", zap.Error(err))
		http.Error(w, "failed to list api keys", http.StatusInternalServerError)
		return
	}
	if keys == nil {
		keys = []*domain.APIKey{}
	}
	writeJSON(w, keys)
}

// CreateAPIKey generates a new API key and returns the plaintext once.
//
//	POST /api/settings/apikeys
func (h *SettingsHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Role == "" {
		req.Role = "developer"
	}

	creator := callerIdentity(r)
	result, err := h.apiKeySvc.Create(r.Context(), req.Name, req.Role, creator)
	if err != nil {
		h.log.Error("create api key failed", zap.String("name", req.Name), zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]any{
		"key":       result.Key,
		"plaintext": result.Plaintext,
	})
}

// RevokeAPIKey soft-deletes a key by setting is_active = false.
//
//	DELETE /api/settings/apikeys/{id}?action=revoke  (default)
//	DELETE /api/settings/apikeys/{id}?action=delete  (permanent)
func (h *SettingsHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validatePathParam(w, "id", id) {
		return
	}
	action := r.URL.Query().Get("action")
	var err error
	if action == "delete" {
		err = h.apiKeySvc.Delete(r.Context(), id)
	} else {
		err = h.apiKeySvc.Revoke(r.Context(), id)
	}
	if err != nil {
		h.log.Error("revoke/delete api key failed", zap.String("id", id), zap.Error(err))
		http.Error(w, "failed to revoke api key", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListUsers returns all OAuth users who have logged into allure-hub.
//
//	GET /api/settings/users
func (h *SettingsHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.List(r.Context())
	if err != nil {
		h.log.Error("list tracked users failed", zap.Error(err))
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []*domain.TrackedUser{}
	}
	writeJSON(w, users)
}
