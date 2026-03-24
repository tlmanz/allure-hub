package usecase

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
)

const (
	SettingRetentionDays = "retention_days"
	SettingIntervalHours = "cleanup_interval_hours"
	SettingDryRun        = "cleanup_dry_run"
)

// RetentionSettings holds all three cleanup configuration values.
type RetentionSettings struct {
	RetentionDays int  `json:"retentionDays"`
	IntervalHours int  `json:"intervalHours"`
	DryRun        bool `json:"dryRun"`
}

// CleanupService periodically deletes expired builds (filesystem + DB).
type CleanupService struct {
	buildRepo    domain.BuildRepository
	settingsRepo domain.SystemSettingsRepository
	runRepo      domain.CleanupRunRepository
	fs           FileStorage
	log          *zap.Logger
}

func NewCleanupService(
	buildRepo domain.BuildRepository,
	settingsRepo domain.SystemSettingsRepository,
	runRepo domain.CleanupRunRepository,
	fs FileStorage,
	log *zap.Logger,
) *CleanupService {
	return &CleanupService{
		buildRepo:    buildRepo,
		settingsRepo: settingsRepo,
		runRepo:      runRepo,
		fs:           fs,
		log:          log,
	}
}

// GetSettings reads all three retention settings from the DB.
func (s *CleanupService) GetSettings(ctx context.Context) (RetentionSettings, error) {
	days, err := s.intSetting(ctx, SettingRetentionDays, 90)
	if err != nil {
		return RetentionSettings{}, err
	}
	hours, err := s.intSetting(ctx, SettingIntervalHours, 6)
	if err != nil {
		return RetentionSettings{}, err
	}
	dryRunStr, err := s.settingsRepo.Get(ctx, SettingDryRun)
	if err != nil {
		return RetentionSettings{}, fmt.Errorf("cleanup: get dry_run setting: %w", err)
	}
	dryRun, _ := strconv.ParseBool(dryRunStr)

	return RetentionSettings{
		RetentionDays: days,
		IntervalHours: hours,
		DryRun:        dryRun,
	}, nil
}

// SetSettings persists all three retention settings to the DB.
func (s *CleanupService) SetSettings(ctx context.Context, cfg RetentionSettings) error {
	if cfg.RetentionDays < 1 {
		return fmt.Errorf("cleanup: retentionDays must be at least 1")
	}
	if cfg.IntervalHours < 1 {
		return fmt.Errorf("cleanup: intervalHours must be at least 1")
	}
	if err := s.settingsRepo.Set(ctx, SettingRetentionDays, strconv.Itoa(cfg.RetentionDays)); err != nil {
		return err
	}
	if err := s.settingsRepo.Set(ctx, SettingIntervalHours, strconv.Itoa(cfg.IntervalHours)); err != nil {
		return err
	}
	return s.settingsRepo.Set(ctx, SettingDryRun, strconv.FormatBool(cfg.DryRun))
}

// Sweep performs one cleanup pass: deletes all builds older than retentionDays.
// Per-build errors are logged and skipped; the sweep never aborts on partial failure.
// The outcome is always persisted to cleanup_runs.
func (s *CleanupService) Sweep(ctx context.Context) error {
	run := &domain.CleanupRun{
		ID:        uuid.New().String(),
		StartedAt: time.Now().UTC(),
		Status:    "success",
	}

	sweepErr := s.sweep(ctx, run)

	run.FinishedAt = time.Now().UTC()
	if sweepErr != nil {
		run.Status = "failed"
		run.ErrorMessage = sweepErr.Error()
	}

	if err := s.runRepo.Save(ctx, run); err != nil {
		s.log.Warn("cleanup: failed to persist run record", zap.Error(err))
	}
	return sweepErr
}

// sweep is the inner implementation; results are written into run.
func (s *CleanupService) sweep(ctx context.Context, run *domain.CleanupRun) error {
	cfg, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}
	run.DryRun = cfg.DryRun
	cutoff := time.Now().UTC().Add(-time.Duration(cfg.RetentionDays) * 24 * time.Hour)

	s.log.Info("cleanup: starting sweep",
		zap.Int("retention_days", cfg.RetentionDays),
		zap.Time("cutoff", cutoff),
		zap.Bool("dry_run", cfg.DryRun),
	)

	expired, err := s.buildRepo.ListExpiredBuilds(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("cleanup: list expired builds: %w", err)
	}

	if len(expired) == 0 {
		s.log.Info("cleanup: no expired builds found")
		return nil
	}

	s.log.Info("cleanup: found expired builds", zap.Int("count", len(expired)))

	for _, b := range expired {
		blog := s.log.With(
			zap.String("buildId", b.BuildID),
			zap.String("projectId", b.ProjectID),
			zap.String("envId", b.EnvID),
			zap.Time("createdAt", b.CreatedAt),
		)

		if cfg.DryRun {
			blog.Info("cleanup: dry-run — would delete build")
			run.DeletedCount++
			continue
		}

		if err := s.deleteBuild(ctx, b, blog); err != nil {
			blog.Warn("cleanup: failed to delete build, skipping", zap.Error(err))
			run.SkippedCount++
			continue
		}
		run.DeletedCount++
	}

	s.log.Info("cleanup: sweep complete",
		zap.Int("deleted", run.DeletedCount),
		zap.Int("skipped", run.SkippedCount),
	)
	return nil
}

// GetRecentRuns returns the most recent cleanup run records.
func (s *CleanupService) GetRecentRuns(ctx context.Context, limit int) ([]*domain.CleanupRun, error) {
	return s.runRepo.ListRecent(ctx, limit)
}

// deleteBuild removes filesystem dirs first then the DB record.
// If FS deletion fails the DB record is kept so the build remains accessible.
func (s *CleanupService) deleteBuild(ctx context.Context, b *domain.Build, log *zap.Logger) error {
	resultsDir := s.fs.ResultsDir(b.EnvID, b.ProjectID, b.BuildID)
	reportDir := s.fs.ReportDir(b.EnvID, b.ProjectID, b.BuildID)

	if err := os.RemoveAll(resultsDir); err != nil {
		return fmt.Errorf("remove results dir %s: %w", resultsDir, err)
	}
	if err := os.RemoveAll(reportDir); err != nil {
		log.Warn("cleanup: failed to remove report dir, proceeding with DB delete",
			zap.String("dir", reportDir), zap.Error(err))
	}

	if err := s.buildRepo.Delete(ctx, b.EnvID, b.ProjectID, b.BuildID); err != nil {
		return fmt.Errorf("delete build record: %w", err)
	}
	return nil
}

// intSetting reads a positive integer from the settings store, falling back to def on parse errors.
func (s *CleanupService) intSetting(ctx context.Context, key string, def int) (int, error) {
	val, err := s.settingsRepo.Get(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("cleanup: get setting %q: %w", key, err)
	}
	if val == "" {
		return def, nil
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		s.log.Warn("cleanup: invalid setting value, using default",
			zap.String("key", key), zap.String("value", val), zap.Int("default", def))
		return def, nil
	}
	return n, nil
}
