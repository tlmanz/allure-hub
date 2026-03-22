// Package transport wires HTTP routes to application service handlers.
package transport

import (
	"net/http"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
	"github.com/tlmanz/allure-hub/internal/transport/handler"
	"github.com/tlmanz/allure-hub/internal/transport/middleware"
	"github.com/tlmanz/allure-hub/pkg/authkit"
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
	rcfg RouterConfig,
	log *zap.Logger,
) http.Handler {
	mux := http.NewServeMux()

	eh := handler.NewEnvironmentHandler(envSvc, log)
	ph := handler.NewProjectHandler(projectSvc, log)
	rh := handler.NewReportHandler(reportSvc, uploadSvc, rcfg.MaxChunkBytes, rcfg.MaxUploadBytes, log)
	uh := handler.NewUploadSessionHandler(sessionRepo, uploadSvc, bus, log)
	hh := handler.NewHealthHandler(db)

	// Auth routes (no session required)
	mux.HandleFunc("GET /auth/{provider}", auth.BeginAuth)
	mux.HandleFunc("GET /auth/{provider}/callback", auth.Callback)
	mux.HandleFunc("POST /auth/logout", auth.Logout)
	mux.HandleFunc("GET /auth/me", auth.Me)

	// Environments
	mux.Handle("GET /api/environments", auth.RequireAuth(http.HandlerFunc(eh.List)))
	mux.Handle("POST /api/environments", auth.Require(authkit.PermManage)(http.HandlerFunc(eh.Create)))
	mux.Handle("PATCH /api/environments/{envId}", auth.Require(authkit.PermManage)(http.HandlerFunc(eh.Update)))
	mux.Handle("DELETE /api/environments/{envId}", auth.Require(authkit.PermManage)(http.HandlerFunc(eh.Delete)))

	// Projects (scoped by environment)
	mux.Handle("GET /api/environments/{envId}/projects", auth.RequireAuth(http.HandlerFunc(ph.List)))
	mux.Handle("POST /api/environments/{envId}/projects", auth.Require(authkit.PermManage)(http.HandlerFunc(ph.Create)))
	mux.Handle("DELETE /api/environments/{envId}/projects/{projectId}", auth.Require(authkit.PermManage)(http.HandlerFunc(ph.Delete)))

	// Results — Strategy A: single streaming upload
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/results", auth.Require(authkit.PermUpload)(http.HandlerFunc(rh.UploadResultsStream)))

	// Results — Strategy B: chunked upload
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/uploads", auth.Require(authkit.PermUpload)(http.HandlerFunc(rh.InitChunkedUpload)))
	mux.Handle("PUT /api/environments/{envId}/projects/{projectId}/uploads/{uploadId}", auth.Require(authkit.PermUpload)(http.HandlerFunc(rh.UploadChunk)))
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/uploads/{uploadId}/complete", auth.Require(authkit.PermUpload)(http.HandlerFunc(rh.CompleteChunkedUpload)))

	// Report generation + listing
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/reports", auth.Require(authkit.PermUpload)(http.HandlerFunc(rh.GenerateReport)))
	mux.Handle("GET /api/environments/{envId}/projects/{projectId}/reports", auth.RequireAuth(http.HandlerFunc(rh.ListReports)))
	mux.Handle("GET /api/environments/{envId}/projects/{projectId}/reports/stats", auth.RequireAuth(http.HandlerFunc(rh.ReportStats)))
	mux.Handle("DELETE /api/environments/{envId}/projects/{projectId}/reports/{buildId}", auth.Require(authkit.PermManage)(http.HandlerFunc(rh.DeleteReport)))

	// Upload session tracking (all uploads across all projects/envs)
	mux.Handle("GET /api/uploads", auth.RequireAuth(http.HandlerFunc(uh.List)))
	mux.Handle("GET /api/uploads/stream", auth.RequireAuth(http.HandlerFunc(uh.Stream)))
	mux.Handle("DELETE /api/uploads/{id}", auth.Require(authkit.PermManage)(http.HandlerFunc(uh.Delete)))

	// Health + version (unprotected — needed for monitoring)
	mux.HandleFunc("GET /api/healthz", hh.Check)
	mux.HandleFunc("GET /api/version", hh.Info)

	// Serve generated Allure report HTML from the data directory.
	mux.Handle("/reports/", reportsHandler(rcfg.DataDir))

	// Serve the compiled frontend SPA (catch-all, must be last).
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

func reportsHandler(dataDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/reports/")
		parts := strings.SplitN(p, "/", 2)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			http.NotFound(w, r)
			return
		}

		// Prevent path traversal: resolve the final path and ensure it stays
		// within the expected reports directory.
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

		// Security headers for generated Allure report HTML.
		// Allure bundles are pre-compiled SPAs that require 'unsafe-inline' and
		// 'unsafe-eval' — these cannot be removed without breaking the reports.
		// We tighten every other directive to minimise the attack surface.
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
