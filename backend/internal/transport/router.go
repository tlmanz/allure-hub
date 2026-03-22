// Package transport wires HTTP routes to application service handlers.
package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
	"github.com/tlmanz/allure-hub/internal/transport/handler"
	"github.com/tlmanz/allure-hub/internal/transport/middleware"
	localauth "github.com/tlmanz/allure-hub/pkg/authkit"
	kit "github.com/tlmanz/authkit"
)

// DB is the minimal interface NewRouter needs for health checks.
type DB interface{ Ping() error }

// RouterConfig groups the tuneable parameters for the HTTP handler tree.
type RouterConfig struct {
	DataDir        string
	WebDir         string
	MaxChunkBytes  int64
	MaxUploadBytes int64
	CORSOrigins    string
	RateLimitRate  float64
	RateLimitBurst float64
	TrustProxy     bool
}

// NewRouter builds and returns the HTTP handler tree.
func NewRouter(
	db DB,
	envSvc *usecase.EnvironmentService,
	projectSvc *usecase.ProjectService,
	reportSvc *usecase.ReportService,
	uploadSvc *usecase.UploadService,
	sessionRepo domain.UploadSessionRepository,
	bus *usecase.EventBus,
	auth *kit.Auth,
	apiKeySvc *usecase.APIKeyService,
	userRepo domain.TrackedUserRepository,
	rcfg RouterConfig,
	log *zap.Logger,
) http.Handler {
	mux := http.NewServeMux()

	eh := handler.NewEnvironmentHandler(envSvc, log)
	ph := handler.NewProjectHandler(projectSvc, log)
	rh := handler.NewReportHandler(reportSvc, uploadSvc, rcfg.MaxChunkBytes, rcfg.MaxUploadBytes, log)
	uh := handler.NewUploadSessionHandler(sessionRepo, uploadSvc, bus, log)
	hh := handler.NewHealthHandler(db)
	sh := handler.NewSettingsHandler(apiKeySvc, userRepo, log)

	// Auth routes — OAuth flow (no API key auth here)
	mux.HandleFunc("GET /auth/{provider}", auth.BeginAuth)
	mux.HandleFunc("GET /auth/{provider}/callback", auth.Callback)
	mux.HandleFunc("POST /auth/logout", auth.Logout)
	// /auth/me: session-only, track the user on access
	mux.Handle("GET /auth/me", auth.RequireSessionAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := kit.UserFromCtx(r.Context())
		go upsertTrackedUser(context.Background(), userRepo, u)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	})))

	// Environments
	// GET  — API keys allowed (read-only dashboards, CI status checks)
	// POST/PATCH/DELETE — session-only management
	mux.Handle("GET /api/environments", auth.Require(localauth.PermView)(http.HandlerFunc(eh.List)))
	mux.Handle("POST /api/environments", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(eh.Create)))
	mux.Handle("PATCH /api/environments/{envId}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(eh.Update)))
	mux.Handle("DELETE /api/environments/{envId}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(eh.Delete)))

	// Projects
	mux.Handle("GET /api/environments/{envId}/projects", auth.Require(localauth.PermView)(http.HandlerFunc(ph.List)))
	mux.Handle("POST /api/environments/{envId}/projects", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(ph.Create)))
	mux.Handle("DELETE /api/environments/{envId}/projects/{projectId}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(ph.Delete)))

	// Results upload — API keys allowed (primary CI/CD path)
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/results", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.UploadResultsStream)))
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/uploads", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.InitChunkedUpload)))
	mux.Handle("PUT /api/environments/{envId}/projects/{projectId}/uploads/{uploadId}", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.UploadChunk)))
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/uploads/{uploadId}/complete", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.CompleteChunkedUpload)))

	// Report generation + listing
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/reports", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.GenerateReport)))
	mux.Handle("GET /api/environments/{envId}/projects/{projectId}/reports", auth.Require(localauth.PermView)(http.HandlerFunc(rh.ListReports)))
	mux.Handle("GET /api/environments/{envId}/projects/{projectId}/reports/stats", auth.Require(localauth.PermView)(http.HandlerFunc(rh.ReportStats)))
	mux.Handle("DELETE /api/environments/{envId}/projects/{projectId}/reports/{buildId}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(rh.DeleteReport)))

	// Upload session tracking
	mux.Handle("GET /api/uploads", auth.Require(localauth.PermView)(http.HandlerFunc(uh.List)))
	mux.Handle("GET /api/uploads/stream", auth.Require(localauth.PermView)(http.HandlerFunc(uh.Stream)))
	mux.Handle("DELETE /api/uploads/{id}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(uh.Delete)))

	// Settings — session-only (API keys must not manage themselves)
	mux.Handle("GET /api/settings/apikeys", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.ListAPIKeys)))
	mux.Handle("POST /api/settings/apikeys", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.CreateAPIKey)))
	mux.Handle("DELETE /api/settings/apikeys/{id}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.RevokeAPIKey)))
	mux.Handle("GET /api/settings/users", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.ListUsers)))

	// Health + version (unprotected)
	mux.HandleFunc("GET /api/healthz", hh.Check)
	mux.HandleFunc("GET /api/version", hh.Info)

	// Serve generated Allure reports and the compiled SPA.
	mux.Handle("/reports/", reportsHandler(rcfg.DataDir))
	if rcfg.WebDir != "" {
		mux.Handle("/", spaHandler(rcfg.WebDir))
	}

	return middleware.Logger(log)(
		middleware.RateLimit(rcfg.RateLimitRate, rcfg.RateLimitBurst, rcfg.TrustProxy)(
			middleware.CORS(rcfg.CORSOrigins)(
				middleware.CSRF(mux),
			),
		),
	)
}

// upsertTrackedUser persists an OAuth user's login record asynchronously.
func upsertTrackedUser(ctx context.Context, repo domain.TrackedUserRepository, u *kit.User) {
	if repo == nil || u == nil {
		return
	}
	now := time.Now().UTC()
	_ = repo.Upsert(ctx, &domain.TrackedUser{
		Email:        u.Email,
		Name:         u.Name,
		AvatarURL:    u.AvatarURL,
		Provider:     u.Provider,
		Role:         u.Role,
		FirstLoginAt: now,
		LastLoginAt:  now,
	})
}

func reportsHandler(dataDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/reports/")
		parts := strings.SplitN(p, "/", 2)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			http.NotFound(w, r)
			return
		}

		target := filepath.Join(dataDir, parts[0], "reports", parts[1])
		absTarget, err := filepath.Abs(target)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		absBase, err := filepath.Abs(filepath.Join(dataDir, parts[0], "reports"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if !strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) && absTarget != absBase {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self' 'unsafe-inline' 'unsafe-eval' blob: data: https:; "+
				"img-src 'self' data: blob: https:; "+
				"font-src 'self' data: https:; "+
				"connect-src 'self' data: blob: https:; "+
				"worker-src blob: 'self'; "+
				"frame-ancestors 'self';",
		)
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		http.ServeFile(w, r, absTarget)
	})
}

func spaHandler(webDir string) http.Handler {
	fsys := http.Dir(webDir)
	fileServer := http.FileServer(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := fsys.Open(r.URL.Path)
		if err != nil {
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}
