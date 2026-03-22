package usecase

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/tlmanz/allure-hub/internal/domain"
)

// ReportService orchestrates report generation and listing use-cases.
type ReportService struct {
	buildRepo   domain.BuildRepository
	sessionRepo domain.UploadSessionRepository
	bus         *EventBus
	fs          FileStorage
	gen         Generator
	log         *zap.Logger
}

func NewReportService(buildRepo domain.BuildRepository, sessionRepo domain.UploadSessionRepository, bus *EventBus, fs FileStorage, gen Generator, log *zap.Logger) *ReportService {
	return &ReportService{
		buildRepo:   buildRepo,
		sessionRepo: sessionRepo,
		bus:         bus,
		fs:          fs,
		gen:         gen,
		log:         log,
	}
}

func (s *ReportService) SaveResultsStream(ctx context.Context, projectID, buildID string, r io.Reader) error {
	return s.fs.SaveResultsStream(projectID, buildID, r)
}

// Generate stitches history, runs allure generate, persists the Build record,
// and saves the new history back for the next run.
func (s *ReportService) Generate(ctx context.Context, envID, projectID, buildID string, opts GenerateOptions) (string, error) {
	resultsDir        := s.fs.ResultsDir(projectID, buildID)
	reportDir         := s.fs.ReportDir(projectID, buildID)
	persistentHistory := s.fs.HistoryDir(projectID)
	historyFile       := s.fs.HistoryFile(projectID)

	// Transition session to generating.
	s.transitionSession(ctx, projectID, buildID, domain.PhaseGenerating, "")

	// Inject previous history so trend charts work.
	historyDest := filepath.Join(resultsDir, "history")
	if _, err := os.Stat(persistentHistory); err == nil {
		if err := copyDir(persistentHistory, historyDest); err != nil {
			s.transitionSession(ctx, projectID, buildID, domain.PhaseFailed, fmt.Sprintf("inject history: %v", err))
			return "", fmt.Errorf("inject history: %w", err)
		}
	}

	configSnapshot, err := s.gen.Generate(resultsDir, reportDir, historyFile, opts)
	if err != nil {
		s.transitionSession(ctx, projectID, buildID, domain.PhaseFailed, err.Error())
		return "", err
	}

	// Persist the new history for the next build.
	newHistory := s.gen.HistoryDir(reportDir)
	if _, err := os.Stat(newHistory); err == nil {
		os.RemoveAll(persistentHistory)
		if err := copyDir(newHistory, persistentHistory); err != nil {
			return "", fmt.Errorf("save history: %w", err)
		}
	}

	reportURL := fmt.Sprintf("/reports/%s/%s/index.html", projectID, buildID)
	passed, failed, skipped, total, status := parseSummary(reportDir)

	build := &domain.Build{
		ID:             uuid.New().String(),
		ProjectID:      projectID,
		BuildID:        buildID,
		CreatedAt:      time.Now().UTC(),
		ReportURL:      reportURL,
		Passed:         passed,
		Failed:         failed,
		Skipped:        skipped,
		Total:          total,
		Status:         status,
		ConfigSnapshot: configSnapshot,
	}
	if err := s.buildRepo.Save(ctx, build); err != nil {
		// Non-fatal: report generated; warn and continue.
		s.log.Warn("persist build record failed",
			zap.String("projectId", projectID),
			zap.String("buildId", buildID),
			zap.Error(err),
		)
	}

	// Mark session done with final stats.
	s.finishSession(ctx, projectID, buildID, reportURL, passed, failed, skipped, total)

	return reportURL, nil
}

func (s *ReportService) List(ctx context.Context, projectID string) ([]*domain.Build, error) {
	return s.buildRepo.ListByProject(ctx, projectID)
}

func (s *ReportService) ListPaged(ctx context.Context, projectID, filter string, limit, offset int) ([]*domain.Build, int, error) {
	builds, err := s.buildRepo.ListByProjectPaged(ctx, projectID, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.buildRepo.CountByProjectFiltered(ctx, projectID, filter)
	if err != nil {
		return nil, 0, err
	}
	return builds, total, nil
}

func (s *ReportService) Stats(ctx context.Context, projectID string) (*domain.BuildStats, error) {
	return s.buildRepo.StatsForProject(ctx, projectID)
}

func (s *ReportService) DeleteBuild(ctx context.Context, projectID, buildID string) error {
	// Fetch the build before deleting so we can match its history entry.
	build, _ := s.buildRepo.GetByBuildID(ctx, projectID, buildID)

	_ = os.RemoveAll(s.fs.ResultsDir(projectID, buildID))
	_ = os.RemoveAll(s.fs.ReportDir(projectID, buildID))

	if err := s.buildRepo.Delete(ctx, projectID, buildID); err != nil {
		return err
	}

	// Best-effort: remove the matching entry from history.jsonl.
	if build != nil {
		_ = removeHistoryEntry(s.fs.HistoryFile(projectID), build.CreatedAt)
	}
	return nil
}

// ── session state machine ─────────────────────────────────────────────────────

func (s *ReportService) transitionSession(ctx context.Context, projectID, buildID string, phase domain.UploadPhase, errMsg string) {
	sess, err := s.sessionRepo.GetByBuild(ctx, projectID, buildID)
	if err != nil || sess == nil {
		return
	}
	if phase == domain.PhaseFailed {
		sess.FailedAtPhase = sess.Phase // capture which phase was active before failing
	}
	sess.Phase = phase
	sess.Error = errMsg
	if phase == domain.PhaseFailed {
		now := time.Now().UTC()
		sess.CompletedAt = &now
	}
	if err := s.sessionRepo.Update(ctx, sess); err == nil {
		s.bus.Publish(sess)
	}
}

func (s *ReportService) finishSession(ctx context.Context, projectID, buildID, reportURL string, passed, failed, skipped, total int) {
	sess, err := s.sessionRepo.GetByBuild(ctx, projectID, buildID)
	if err != nil || sess == nil {
		return
	}
	now := time.Now().UTC()
	sess.Phase = domain.PhaseDone
	sess.CompletedAt = &now
	sess.ReportURL = reportURL
	sess.Passed = passed
	sess.Failed = failed
	sess.Skipped = skipped
	sess.Total = total
	if err := s.sessionRepo.Update(ctx, sess); err == nil {
		s.bus.Publish(sess)
	}
}
