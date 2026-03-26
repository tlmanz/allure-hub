package usecase

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func (s *ReportService) SaveResultsStream(ctx context.Context, envID, projectID, buildID string, r io.Reader) error {
	return s.fs.SaveResultsStream(envID, projectID, buildID, r)
}

// Generate stitches history, runs allure generate, persists the Build record,
// and saves the new history back for the next run.
func (s *ReportService) Generate(ctx context.Context, envID, projectID, buildID string, opts GenerateOptions) (reportURL string, retErr error) {
	log := s.log.With(zap.String("projectId", projectID), zap.String("buildId", buildID))
	log.Debug("generate: starting")

	resultsDir := s.fs.ResultsDir(envID, projectID, buildID)
	reportDir := s.fs.ReportDir(envID, projectID, buildID)
	persistentHistory := s.fs.HistoryDir(envID, projectID)
	historyFile := s.fs.HistoryFile(envID, projectID)
	log.Debug("generate: paths resolved",
		zap.String("resultsDir", resultsDir),
		zap.String("reportDir", reportDir),
		zap.String("historyFile", historyFile),
	)

	// Transition session to generating.
	s.transitionSession(ctx, projectID, buildID, domain.PhaseGenerating, "")
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("panic during generate: %v", r)
			s.transitionSession(ctx, projectID, buildID, domain.PhaseFailed, msg)
			panic(r)
		}
		// Safety net: ensure any returned error cannot leave the session in
		// "generating" due to a missed transition in future code paths.
		if retErr != nil {
			s.transitionSession(ctx, projectID, buildID, domain.PhaseFailed, retErr.Error())
		}
	}()

	// Inject previous history so trend charts work.
	historyDest := filepath.Join(resultsDir, "history")
	if _, err := os.Stat(persistentHistory); err == nil {
		log.Debug("generate: injecting history", zap.String("src", persistentHistory), zap.String("dst", historyDest))
		if err := copyDir(persistentHistory, historyDest); err != nil {
			return "", fmt.Errorf("inject history: %w", err)
		}
	} else {
		log.Debug("generate: no previous history found, skipping injection")
	}

	log.Debug("generate: invoking allure CLI", zap.String("resultsDir", resultsDir), zap.String("reportDir", reportDir))
	genResult, err := s.gen.Generate(resultsDir, reportDir, historyFile, opts)
	if err != nil {
		return "", err
	}
	log.Debug("generate: allure CLI finished")

	// Persist the new history for the next build.
	newHistory := s.gen.HistoryDir(reportDir)
	if _, err := os.Stat(newHistory); err == nil {
		log.Debug("generate: saving new history", zap.String("src", newHistory), zap.String("dst", persistentHistory))
		os.RemoveAll(persistentHistory)
		if err := copyDir(newHistory, persistentHistory); err != nil {
			return "", fmt.Errorf("save history: %w", err)
		}
	} else {
		log.Debug("generate: no new history dir found in report output")
	}

	reportURL = fmt.Sprintf("/reports/%s/%s/%s/index.html", envID, projectID, buildID)
	passed, failed, skipped, total, status := parseSummary(reportDir)
	log.Debug("generate: summary parsed", zap.Int("passed", passed), zap.Int("failed", failed), zap.Int("total", total), zap.String("status", status))

	// Carry the uploader identity from the upload session onto the build record.
	uploadedBy := ""
	if sess, _ := s.sessionRepo.GetByBuild(ctx, projectID, buildID); sess != nil {
		uploadedBy = sess.UploadedBy
	}

	build := &domain.Build{
		ID:                 uuid.New().String(),
		EnvID:              envID,
		ProjectID:          projectID,
		BuildID:            buildID,
		CreatedAt:          time.Now().UTC(),
		ReportURL:          reportURL,
		Passed:             passed,
		Failed:             failed,
		Skipped:            skipped,
		Total:              total,
		Status:             status,
		UploadedBy:         uploadedBy,
		ConfigSnapshot:     genResult.ConfigSnapshot,
		GenerationWarnings: genResult.Warnings,
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
	s.finishSession(ctx, projectID, buildID, reportURL, passed, failed, skipped, total, genResult.Warnings)

	return reportURL, nil
}

func (s *ReportService) List(ctx context.Context, envID, projectID string) ([]*domain.Build, error) {
	return s.buildRepo.ListByProject(ctx, envID, projectID)
}

func (s *ReportService) ListPaged(ctx context.Context, envID, projectID, filter string, limit, offset int) ([]*domain.Build, int, error) {
	builds, err := s.buildRepo.ListByProjectPaged(ctx, envID, projectID, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.buildRepo.CountByProjectFiltered(ctx, envID, projectID, filter)
	if err != nil {
		return nil, 0, err
	}
	return builds, total, nil
}

func (s *ReportService) Stats(ctx context.Context, envID, projectID string) (*domain.BuildStats, error) {
	return s.buildRepo.StatsForProject(ctx, envID, projectID)
}

func (s *ReportService) DeleteBuild(ctx context.Context, envID, projectID, buildID string) error {
	// Fetch the build before deleting so we can match its history entry.
	build, _ := s.buildRepo.GetByBuildID(ctx, envID, projectID, buildID)

	_ = os.RemoveAll(s.fs.ResultsDir(envID, projectID, buildID))
	_ = os.RemoveAll(s.fs.ReportDir(envID, projectID, buildID))

	if err := s.buildRepo.Delete(ctx, envID, projectID, buildID); err != nil {
		return err
	}

	// Best-effort: remove the matching entry from history.jsonl.
	if build != nil {
		_ = removeHistoryEntry(s.fs.HistoryFile(envID, projectID), build.CreatedAt)
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

func (s *ReportService) finishSession(ctx context.Context, projectID, buildID, reportURL string, passed, failed, skipped, total int, warnings []string) {
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
	if len(warnings) > 0 {
		sess.Error = strings.Join(warnings, "\n")
	} else {
		sess.Error = ""
	}
	if err := s.sessionRepo.Update(ctx, sess); err == nil {
		s.bus.Publish(sess)
	}
}
