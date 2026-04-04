// Package transport wires HTTP routes to application service handlers.
package transport

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/transport/handler"
	"github.com/tlmanz/allure-hub/internal/transport/middleware"
	"github.com/tlmanz/allure-hub/internal/transport/routes"
	"github.com/tlmanz/allure-hub/internal/usecase"
	localauth "github.com/tlmanz/allure-hub/pkg/authkit"
	kit "github.com/tlmanz/authkit"
	notify "github.com/tlmanz/go-notify"
)

// Pinger is the minimal interface NewRouter needs for health checks.
type Pinger interface{ Ping() error }

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
	AllureBin      string
}

// OverviewRepository is the minimal interface NewRouter needs for the overview handler.
type OverviewRepository interface {
	GetStats(ctx context.Context) (*domain.OverviewStats, error)
}

// NewRouter builds and returns the HTTP handler tree.
func NewRouter(
	db Pinger,
	envSvc *usecase.EnvironmentService,
	projectSvc *usecase.ProjectService,
	reportSvc *usecase.ReportService,
	uploadSvc *usecase.UploadService,
	sessionRepo domain.UploadSessionRepository,
	bus *usecase.EventBus,
	notifier *notify.Notifier,
	auth *kit.Auth,
	provider *kit.LayeredPolicyProvider,
	apiKeySvc *usecase.APIKeyService,
	userRepo domain.TrackedUserRepository,
	settingsRepo domain.SystemSettingsRepository,
	cleanupSvc *usecase.CleanupService,
	overviewRepo OverviewRepository,
	rcfg RouterConfig,
	log *zap.Logger,
) http.Handler {
	mux := http.NewServeMux()

	environmentHandler := handler.NewEnvironmentHandler(envSvc, log)
	projectHandler := handler.NewProjectHandler(projectSvc, log)
	reportHandler := handler.NewReportHandler(reportSvc, uploadSvc, rcfg.MaxChunkBytes, rcfg.MaxUploadBytes, log)
	uploadHandler := handler.NewUploadSessionHandler(sessionRepo, uploadSvc, bus, log)
	healthHandler := handler.NewHealthHandler(db)
	settingsHandler := handler.NewSettingsHandler(apiKeySvc, userRepo, settingsRepo, provider, cleanupSvc, rcfg.AllureBin, rcfg.DataDir, log)
	overviewHandler := handler.NewOverviewHandler(overviewRepo, log)

	routes.RegisterAuthRoutes(mux, auth, userRepo)
	routes.RegisterEnvironmentRoutes(mux, auth, environmentHandler, projectHandler)
	routes.RegisterReportRoutes(mux, auth, reportHandler, uploadHandler)
	routes.RegisterSettingsRoutes(mux, auth, settingsHandler)
	routes.RegisterSystemRoutes(mux, auth, overviewHandler, healthHandler)
	if notifier != nil {
		mux.Handle("/api/notifications/", auth.Require(localauth.PermView)(notifier.Handler("/api/notifications")))
	}

	// Serve generated Allure reports - requires PermView (session or API key).
	mux.Handle("/reports/", auth.Require(localauth.PermView)(reportsHandler(rcfg.DataDir)))
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
