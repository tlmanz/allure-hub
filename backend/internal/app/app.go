// Package app wires all application dependencies and owns the server lifecycle.
// main() delegates to New() + Run() so the entry point stays minimal.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/allure"
	"github.com/tlmanz/allure-hub/internal/apikey"
	"github.com/tlmanz/allure-hub/internal/repository"
	"github.com/tlmanz/allure-hub/internal/storage"
	"github.com/tlmanz/allure-hub/internal/transport"
	"github.com/tlmanz/allure-hub/internal/usecase"
	"github.com/tlmanz/allure-hub/pkg/config"
	"github.com/tlmanz/authkit"
)

// App holds all wired application components and manages the server lifecycle.
type App struct {
	log        *zap.Logger
	db         *repository.DB
	srv        *http.Server
	cfg        config.Config
	auth       *authkit.Auth
	provider   *authkit.LayeredPolicyProvider
	cleanupSvc *usecase.CleanupService
}

// New wires all dependencies - repositories, services, HTTP router - and
// returns an App ready to serve. It does not start listening yet.
func New(cfg config.Config, log *zap.Logger) (*App, error) {
	// ── Database ──────────────────────────────────────────────────────────────
	db, err := repository.Open(cfg.DB.Driver, cfg.DB.DSN, repository.PoolConfig{
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.DB.ConnMaxIdleTime,
	}, log)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	log.Info("database connected", zap.String("driver", cfg.DB.Driver))

	// ── Repositories ─────────────────────────────────────────────────────────
	envRepo := repository.NewEnvironmentRepo(db)
	projectRepo := repository.NewProjectRepo(db)
	buildRepo := repository.NewBuildRepo(db)
	sessionRepo := repository.NewUploadSessionRepo(db)
	userRepo := repository.NewTrackedUserRepo(db)
	apiKeyRepo := repository.NewAPIKeyRepo(db)
	settingsRepo := repository.NewSystemSettingsRepo(db)
	cleanupRunRepo := repository.NewCleanupRunRepo(db)
	overviewRepo := repository.NewOverviewRepo(db)

	// ── Infrastructure adapters ───────────────────────────────────────────────
	fs := storage.NewFilesystem(
		cfg.Storage.DataDir,
		cfg.Storage.MaxUploadBytes,
		cfg.Storage.MaxDecompressedBytes,
		cfg.Storage.MaxZipEntries,
	)
	gen := allure.NewGenerator(
		cfg.Allure.Bin,
		cfg.Allure.ConfigPath,
		cfg.Allure.MaxConcurrency,
		cfg.Allure.Timeout,
		log,
	)
	bus := usecase.NewEventBus()
	notifier, closeNotifyRepo, err := newNotifier(cfg.Notify, log)
	if err != nil {
		return nil, err
	}
	bus.SetNotifier(notifier)

	// ── Services ──────────────────────────────────────────────────────────────
	envSvc := usecase.NewEnvironmentService(envRepo, projectRepo, buildRepo, sessionRepo, fs)
	projectSvc := usecase.NewProjectService(projectRepo, buildRepo, sessionRepo, fs)
	reportSvc := usecase.NewReportService(buildRepo, sessionRepo, bus, fs, gen, log)
	uploadSvc := usecase.NewUploadService(reportSvc, fs, sessionRepo, envRepo, projectRepo, settingsRepo, bus, cfg.Storage.AssembleTempDir, log)
	cleanupSvc := usecase.NewCleanupService(buildRepo, settingsRepo, cleanupRunRepo, fs, log)

	// ── API keys ──────────────────────────────────────────────────────────────
	// Initialised before auth so keyStore can be passed as APIKeyValidator.
	keyStore := apikey.NewStore(apiKeyRepo)
	apiKeySvc := usecase.NewAPIKeyService(apiKeyRepo)

	// ── Auth ──────────────────────────────────────────────────────────────────
	auth, provider, providerCount, err := newAuth(cfg.Auth, db, keyStore, log)
	if err != nil {
		return nil, err
	}
	log.Info("authkit initialised", zap.Int("providers", providerCount))

	// ── HTTP layer ────────────────────────────────────────────────────────────
	router := transport.NewRouter(
		db,
		envSvc, projectSvc, reportSvc, uploadSvc,
		sessionRepo, bus, notifier,
		auth, provider,
		apiKeySvc, userRepo, settingsRepo,
		cleanupSvc,
		overviewRepo,
		transport.RouterConfig{
			DataDir:        cfg.Storage.DataDir,
			WebDir:         cfg.Server.WebDir,
			MaxChunkBytes:  cfg.Storage.MaxChunkBytes,
			MaxUploadBytes: cfg.Storage.MaxUploadBytes,
			CORSOrigins:    cfg.CORS.AllowedOrigins,
			RateLimitRate:  cfg.RateLimit.Rate,
			RateLimitBurst: cfg.RateLimit.Burst,
			TrustProxy:     cfg.RateLimit.TrustProxy,
			AllureBin:      cfg.Allure.Bin,
		},
		log,
	)

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           router,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}
	srv.RegisterOnShutdown(bus.Shutdown)
	srv.RegisterOnShutdown(notifier.Close)
	srv.RegisterOnShutdown(closeNotifyRepo)

	return &App{log: log, db: db, srv: srv, cfg: cfg, auth: auth, provider: provider, cleanupSvc: cleanupSvc}, nil
}

// runCleanupWorker runs an initial sweep after a short startup delay then ticks
// on cfg.Cleanup.Interval, calling CleanupService.Sweep each cycle.
func (a *App) runCleanupWorker(ctx context.Context) {
	// Short delay so DB migrations settle before the first sweep.
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}

	if err := a.cleanupSvc.Sweep(ctx); err != nil {
		a.log.Error("cleanup sweep failed", zap.Error(err))
	}

	ticker := time.NewTicker(a.cfg.Cleanup.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			a.log.Info("cleanup worker stopped")
			return
		case <-ticker.C:
			if err := a.cleanupSvc.Sweep(ctx); err != nil {
				a.log.Error("cleanup sweep failed", zap.Error(err))
			}
		}
	}
}

// Run starts the HTTP server and blocks until ctx is cancelled (e.g. SIGINT),
// then performs a graceful shutdown within the configured ShutdownTimeout.
// It closes the database connection before returning.
func (a *App) Run(ctx context.Context) error {
	go a.auth.WatchRBAC(ctx, 30*time.Second)

	if a.cfg.Cleanup.Interval > 0 {
		go a.runCleanupWorker(ctx)
	}

	errCh := make(chan error, 1)
	go func() {
		a.log.Info("allure-hub listening", zap.String("addr", a.srv.Addr))
		if err := a.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		_ = a.db.Close()
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
		a.log.Info("shutdown signal received, draining connections")
	}

	// Use a fresh context for graceful shutdown; the root context is already
	// canceled by SIGINT/SIGTERM, which would otherwise cause immediate failure.
	sdCtx, cancel := context.WithTimeout(context.Background(), a.cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := a.srv.Shutdown(sdCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			a.log.Warn("graceful shutdown timed out", zap.Duration("timeout", a.cfg.Server.ShutdownTimeout))
		} else {
			a.log.Error("forced shutdown", zap.Error(err))
		}
	}
	_ = a.db.Close()
	a.log.Info("server stopped")
	return nil
}
