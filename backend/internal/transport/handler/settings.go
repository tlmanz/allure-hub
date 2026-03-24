package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
	kit "github.com/tlmanz/authkit"
)

const settingsPageSize = 20

// SettingsHandler exposes API key management, user-tracking, and retention endpoints.
// All routes require a valid OAuth session (no API key auth allowed here).
type SettingsHandler struct {
	apiKeySvc  *usecase.APIKeyService
	userRepo   domain.TrackedUserRepository
	provider   *kit.LayeredPolicyProvider
	cleanupSvc *usecase.CleanupService
	log        *zap.Logger
}

func NewSettingsHandler(
	apiKeySvc *usecase.APIKeyService,
	userRepo domain.TrackedUserRepository,
	provider *kit.LayeredPolicyProvider,
	cleanupSvc *usecase.CleanupService,
	log *zap.Logger,
) *SettingsHandler {
	return &SettingsHandler{apiKeySvc: apiKeySvc, userRepo: userRepo, provider: provider, cleanupSvc: cleanupSvc, log: log}
}

// ListAPIKeys returns a page of API keys matching an optional search query.
//
//	GET /api/settings/apikeys?search=&offset=
func (h *SettingsHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	keys, total, err := h.apiKeySvc.Search(r.Context(), search, settingsPageSize, offset)
	if err != nil {
		h.log.Error("list api keys failed", zap.Error(err))
		http.Error(w, "failed to list api keys", http.StatusInternalServerError)
		return
	}
	if keys == nil {
		keys = []*domain.APIKey{}
	}
	writeJSON(w, map[string]any{
		"keys":   keys,
		"total":  total,
		"limit":  settingsPageSize,
		"offset": offset,
	})
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

// ListUsers returns a page of OAuth users matching an optional search query.
//
//	GET /api/settings/users?search=&offset=
func (h *SettingsHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	users, err := h.userRepo.Search(r.Context(), search, settingsPageSize, offset)
	if err != nil {
		h.log.Error("list tracked users failed", zap.Error(err))
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}
	total, err := h.userRepo.CountSearch(r.Context(), search)
	if err != nil {
		h.log.Error("count tracked users failed", zap.Error(err))
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []*domain.TrackedUser{}
	}
	writeJSON(w, map[string]any{
		"users":  users,
		"total":  total,
		"limit":  settingsPageSize,
		"offset": offset,
	})
}

// SetUserRole stores a role override for a user. Only admins may call this.
// The role must be defined in policy.yaml. Changes take effect on next login.
//
//	PATCH /api/settings/users/{email}/role
func (h *SettingsHandler) SetUserRole(w http.ResponseWriter, r *http.Request) {
	caller := kit.UserFromCtx(r.Context())
	if caller == nil || caller.Role != "admin" {
		http.Error(w, "forbidden: admin role required", http.StatusForbidden)
		return
	}

	email := r.PathValue("email")
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Role == "" {
		http.Error(w, "role is required", http.StatusBadRequest)
		return
	}

	// Derive permissions from the YAML policy so the handler is not coupled to role definitions.
	perms := h.provider.PermissionsForRole(req.Role)
	if perms == nil {
		http.Error(w, "unknown role", http.StatusBadRequest)
		return
	}

	if err := h.provider.SetOverride(r.Context(), email, req.Role, perms); err != nil {
		h.log.Error("set role override failed", zap.String("email", email), zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetRetention returns the current data retention settings.
//
//	GET /api/settings/retention
func (h *SettingsHandler) GetRetention(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.cleanupSvc.GetSettings(r.Context())
	if err != nil {
		h.log.Error("get retention settings failed", zap.Error(err))
		http.Error(w, "failed to get retention settings", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg)
}

// SetRetention updates the data retention settings.
//
//	PUT /api/settings/retention
func (h *SettingsHandler) SetRetention(w http.ResponseWriter, r *http.Request) {
	var req usecase.RetentionSettings
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := h.cleanupSvc.SetSettings(r.Context(), req); err != nil {
		h.log.Error("set retention settings failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetCleanupRuns returns the last N cleanup sweep run records.
//
//	GET /api/settings/retention/runs?limit=5
func (h *SettingsHandler) GetCleanupRuns(w http.ResponseWriter, r *http.Request) {
	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	runs, err := h.cleanupSvc.GetRecentRuns(r.Context(), limit)
	if err != nil {
		h.log.Error("get cleanup runs failed", zap.Error(err))
		http.Error(w, "failed to get cleanup runs", http.StatusInternalServerError)
		return
	}
	if runs == nil {
		runs = []*domain.CleanupRun{}
	}
	writeJSON(w, runs)
}

// ResetUserRole removes a user's role override, reverting them to the YAML baseline.
// Only admins may call this. Takes effect on next login.
//
//	DELETE /api/settings/users/{email}/role
func (h *SettingsHandler) ResetUserRole(w http.ResponseWriter, r *http.Request) {
	caller := kit.UserFromCtx(r.Context())
	if caller == nil || caller.Role != "admin" {
		http.Error(w, "forbidden: admin role required", http.StatusForbidden)
		return
	}

	email := r.PathValue("email")
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	if err := h.provider.DeleteOverride(r.Context(), email); err != nil {
		h.log.Error("delete role override failed", zap.String("email", email), zap.Error(err))
		http.Error(w, "failed to reset role", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
