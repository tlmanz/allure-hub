package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
	kit "github.com/tlmanz/authkit"
)

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

const npmAllureLatestURL = "https://registry.npmjs.org/allure/latest"
const npmCacheTTL = time.Hour

const settingsPageSize = 20

const diskCacheTTL = time.Minute

const (
	diskNotificationThresholdKey     = "disk_notification_threshold_percent"
	defaultDiskNotificationThreshold = 85
)

// SettingsHandler exposes API key management, user-tracking, and retention endpoints.
// All routes require a valid OAuth session (no API key auth allowed here).
type SettingsHandler struct {
	apiKeySvc    *usecase.APIKeyService
	userRepo     domain.TrackedUserRepository
	settingsRepo domain.SystemSettingsRepository
	provider     *kit.LayeredPolicyProvider
	cleanupSvc   *usecase.CleanupService
	allureBin    string
	dataDir      string
	log          *zap.Logger

	npmMu          sync.Mutex
	npmLatest      string
	npmLatestFetch time.Time

	diskMu      sync.Mutex
	diskCached  *diskUsage
	diskCacheAt time.Time
}

type diskUsage struct {
	UsedBytes  int64       `json:"usedBytes"`
	FreeBytes  int64       `json:"freeBytes"`
	TotalBytes int64       `json:"totalBytes"`
	Breakdown  []diskEntry `json:"breakdown"`
}

type diskEntry struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

func NewSettingsHandler(
	apiKeySvc *usecase.APIKeyService,
	userRepo domain.TrackedUserRepository,
	settingsRepo domain.SystemSettingsRepository,
	provider *kit.LayeredPolicyProvider,
	cleanupSvc *usecase.CleanupService,
	allureBin string,
	dataDir string,
	log *zap.Logger,
) *SettingsHandler {
	return &SettingsHandler{
		apiKeySvc:    apiKeySvc,
		userRepo:     userRepo,
		settingsRepo: settingsRepo,
		provider:     provider,
		cleanupSvc:   cleanupSvc,
		allureBin:    allureBin,
		dataDir:      dataDir,
		log:          log,
	}
}

// computeDiskUsage walks dataDir and returns usage stats. Results are cached
// for diskCacheTTL to keep the endpoint fast under repeated requests.
func (h *SettingsHandler) computeDiskUsage() *diskUsage {
	h.diskMu.Lock()
	defer h.diskMu.Unlock()

	if h.diskCached != nil && time.Since(h.diskCacheAt) < diskCacheTTL {
		return h.diskCached
	}

	// Sum all file sizes under dataDir.
	var usedBytes int64
	_ = filepath.Walk(h.dataDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		usedBytes += info.Size()
		return nil
	})

	// Filesystem free/total via syscall.
	var freeBytes, totalBytes int64
	var stat syscall.Statfs_t
	if err := syscall.Statfs(h.dataDir, &stat); err == nil {
		freeBytes = int64(stat.Bavail) * int64(stat.Bsize)
		totalBytes = int64(stat.Blocks) * int64(stat.Bsize)
	}

	// Per-project breakdown (env/project, two levels deep).
	var breakdown []diskEntry
	envDirs, _ := os.ReadDir(h.dataDir)
	for _, envDir := range envDirs {
		if !envDir.IsDir() {
			continue
		}
		projDirs, _ := os.ReadDir(filepath.Join(h.dataDir, envDir.Name()))
		for _, projDir := range projDirs {
			if !projDir.IsDir() {
				continue
			}
			projPath := filepath.Join(h.dataDir, envDir.Name(), projDir.Name())
			var projBytes int64
			_ = filepath.Walk(projPath, func(_ string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				projBytes += info.Size()
				return nil
			})
			breakdown = append(breakdown, diskEntry{
				Path:  envDir.Name() + "/" + projDir.Name(),
				Bytes: projBytes,
			})
		}
	}
	sort.Slice(breakdown, func(i, j int) bool { return breakdown[i].Bytes > breakdown[j].Bytes })
	if len(breakdown) > 20 {
		breakdown = breakdown[:20]
	}
	if breakdown == nil {
		breakdown = []diskEntry{}
	}

	h.diskCached = &diskUsage{
		UsedBytes:  usedBytes,
		FreeBytes:  freeBytes,
		TotalBytes: totalBytes,
		Breakdown:  breakdown,
	}
	h.diskCacheAt = time.Now()
	return h.diskCached
}

// GetDiskUsage returns data-directory disk usage and a per-project breakdown.
//
//	GET /api/settings/disk
func (h *SettingsHandler) GetDiskUsage(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.computeDiskUsage())
}

func normalizeDiskNotificationThreshold(raw int) (int, bool) {
	if raw < 0 || raw > 100 {
		return 0, false
	}
	return raw, true
}

// GetDiskNotificationThreshold returns the configured disk usage notification threshold percentage.
//
//	GET /api/settings/disk/notification-threshold
func (h *SettingsHandler) GetDiskNotificationThreshold(w http.ResponseWriter, r *http.Request) {
	raw, err := h.settingsRepo.Get(r.Context(), diskNotificationThresholdKey)
	if err != nil {
		h.log.Error("get disk notification threshold failed", zap.Error(err))
		http.Error(w, "failed to get disk notification threshold", http.StatusInternalServerError)
		return
	}
	threshold := defaultDiskNotificationThreshold
	if raw != "" {
		v, convErr := strconv.Atoi(raw)
		if convErr == nil {
			if normalized, ok := normalizeDiskNotificationThreshold(v); ok {
				threshold = normalized
			}
		}
	}
	writeJSON(w, map[string]int{"thresholdPercent": threshold})
}

// SetDiskNotificationThreshold updates the disk usage notification threshold percentage.
//
//	PUT /api/settings/disk/notification-threshold
func (h *SettingsHandler) SetDiskNotificationThreshold(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ThresholdPercent int `json:"thresholdPercent"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	threshold, ok := normalizeDiskNotificationThreshold(req.ThresholdPercent)
	if !ok {
		http.Error(w, "thresholdPercent must be between 0 and 100", http.StatusBadRequest)
		return
	}
	if err := h.settingsRepo.Set(r.Context(), diskNotificationThresholdKey, strconv.Itoa(threshold)); err != nil {
		h.log.Error("set disk notification threshold failed", zap.Int("thresholdPercent", threshold), zap.Error(err))
		http.Error(w, "failed to set disk notification threshold", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// fetchNPMLatest returns the latest published version of the allure npm package,
// caching the result for npmCacheTTL to avoid hitting the registry on every request.
// Returns an empty string on any error so the caller degrades gracefully.
func (h *SettingsHandler) fetchNPMLatest(ctx context.Context) string {
	h.npmMu.Lock()
	defer h.npmMu.Unlock()

	if h.npmLatest != "" && time.Since(h.npmLatestFetch) < npmCacheTTL {
		return h.npmLatest
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, npmAllureLatestURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return ""
	}
	defer resp.Body.Close()

	var payload struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&payload); err != nil || payload.Version == "" {
		return ""
	}

	h.npmLatest = payload.Version
	h.npmLatestFetch = time.Now()
	return h.npmLatest
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
		Name                 string `json:"name"`
		Role                 string `json:"role"`
		AutoCreateEnvProject bool   `json:"autoCreateEnvProject"`
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
	result, err := h.apiKeySvc.Create(r.Context(), req.Name, req.Role, creator, req.AutoCreateEnvProject)
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

// GetAllureVersion returns the currently installed Allure CLI version and the
// latest version published to npm (cached for 1 h). The "latest" field is an
// empty string if the registry is unreachable.
//
//	GET /api/settings/allure
func (h *SettingsHandler) GetAllureVersion(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, h.allureBin, "--version")
	out, err := cmd.Output()
	if err != nil {
		h.log.Error("allure --version failed", zap.Error(err))
		http.Error(w, "failed to get allure version", http.StatusInternalServerError)
		return
	}

	latest := h.fetchNPMLatest(r.Context())
	writeJSON(w, map[string]string{
		"version": strings.TrimSpace(string(out)),
		"latest":  latest,
	})
}

// UpdateAllureVersion installs a specific Allure CLI version via npm.
// Admin only. The version must be a valid semver string (e.g. "3.3.1").
//
//	PUT /api/settings/allure
func (h *SettingsHandler) UpdateAllureVersion(w http.ResponseWriter, r *http.Request) {
	caller := kit.UserFromCtx(r.Context())
	if caller == nil || caller.Role != "admin" {
		http.Error(w, "forbidden: admin role required", http.StatusForbidden)
		return
	}

	var req struct {
		Version string `json:"version"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Version == "" {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if !semverRe.MatchString(req.Version) {
		http.Error(w, "version must be a valid semver (e.g. 3.3.1)", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "npm", "install", "-g", "allure@"+req.Version)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		h.log.Error("npm install allure failed",
			zap.String("version", req.Version),
			zap.String("stderr", stderr.String()),
			zap.Error(err),
		)
		http.Error(w, "failed to install allure "+req.Version+": "+strings.TrimSpace(stderr.String()), http.StatusInternalServerError)
		return
	}

	h.log.Info("allure version updated", zap.String("version", req.Version))
	writeJSON(w, map[string]string{"version": req.Version})
}

