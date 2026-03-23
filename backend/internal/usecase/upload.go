package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/tlmanz/allure-hub/internal/domain"
	"go.uber.org/zap"
)

// UploadService orchestrates chunked-upload sessions.
type UploadService struct {
	reportSvc       *ReportService
	fs              FileStorage
	sessionRepo     domain.UploadSessionRepository
	envRepo         domain.EnvironmentRepository
	projectRepo     domain.ProjectRepository
	bus             *EventBus
	assembleTempDir string // directory for assembled zip temp file; defaults to chunk parent dir
	log             *zap.Logger
}

func NewUploadService(reportSvc *ReportService, fs FileStorage, sessionRepo domain.UploadSessionRepository, envRepo domain.EnvironmentRepository, projectRepo domain.ProjectRepository, bus *EventBus, assembleTempDir string, log *zap.Logger) *UploadService {
	return &UploadService{
		reportSvc:       reportSvc,
		fs:              fs,
		sessionRepo:     sessionRepo,
		envRepo:         envRepo,
		projectRepo:     projectRepo,
		bus:             bus,
		assembleTempDir: assembleTempDir,
		log:             log,
	}
}

// InitUpload creates the staging directory, persists a new session, and returns the upload ID.
func (s *UploadService) InitUpload(ctx context.Context, envID, projectID, buildID, fileName, uploadedBy string, totalSize int64, totalChunks int) (string, error) {
	log := s.log.With(zap.String("projectId", projectID), zap.String("buildId", buildID), zap.String("envId", envID))

	uploadID := uuid.New().String()
	sess := &domain.UploadSession{
		ID:          uuid.New().String(),
		UploadID:    uploadID,
		BuildID:     buildID,
		ProjectID:   projectID,
		EnvID:       envID,
		FileName:    fileName,
		TotalSize:   totalSize,
		TotalChunks: totalChunks,
		UploadedBy:  uploadedBy,
		Phase:       domain.PhaseUploading,
		StartedAt:   time.Now().UTC(),
	}
	if err := s.sessionRepo.Create(ctx, sess); err == nil {
		s.bus.Publish(sess)
	}

	// Validate after creating the session so a failed session is always
	// persisted to the DB and visible in the frontend via SSE.
	if err := s.validateEnvAndProject(ctx, envID, projectID); err != nil {
		log.Debug("init upload: validation failed", zap.Error(err))
		s.failSessionByID(ctx, sess, err.Error())
		return "", err
	}

	if err := os.MkdirAll(s.fs.ChunkDir(envID, projectID, uploadID), 0755); err != nil {
		s.failSessionByID(ctx, sess, fmt.Sprintf("create chunk dir: %v", err))
		return "", err
	}
	meta := UploadMeta{
		UploadID:    uploadID,
		BuildID:     buildID,
		TotalSize:   totalSize,
		TotalChunks: totalChunks,
	}
	if err := s.fs.WriteUploadMeta(envID, projectID, uploadID, meta); err != nil {
		s.failSessionByID(ctx, sess, fmt.Sprintf("write upload meta: %v", err))
		return "", err
	}

	log.Debug("init upload: session created", zap.String("uploadId", uploadID), zap.Int("totalChunks", totalChunks))
	return uploadID, nil
}

// SaveChunk writes a single chunk to its slot file and updates received count.
func (s *UploadService) SaveChunk(ctx context.Context, envID, projectID, uploadID string, index, total int, body io.Reader) error {
	f, err := os.Create(s.fs.ChunkPath(envID, projectID, uploadID, index))
	if err != nil {
		return fmt.Errorf("create chunk file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, body); err != nil {
		return fmt.Errorf("write chunk: %w", err)
	}

	// Atomically increment received_chunks and broadcast (M-05: eliminates the
	// racy read-modify-write that caused lost updates under concurrent chunks).
	if sess, _ := s.sessionRepo.IncrementReceivedChunks(ctx, uploadID); sess != nil {
		s.bus.Publish(sess)
	}
	return nil
}

// ChunksReceived returns how many chunk files exist in the upload directory.
func (s *UploadService) ChunksReceived(ctx context.Context, envID, projectID, uploadID string) (int, error) {
	entries, err := os.ReadDir(s.fs.ChunkDir(envID, projectID, uploadID))
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == "" {
			count++
		}
	}
	return count, nil
}

// AssembleUpload concatenates all chunks into a zip and extracts to results/.
func (s *UploadService) AssembleUpload(ctx context.Context, projectID, uploadID string) error {
	log := s.log.With(zap.String("projectId", projectID), zap.String("uploadId", uploadID))
	log.Debug("assemble: starting")

	// Transition to assembling; capture envID from session for FS path scoping.
	envID := "default"
	if sess, _ := s.sessionRepo.GetByUploadID(ctx, uploadID); sess != nil {
		if sess.EnvID != "" {
			envID = sess.EnvID
		}
		sess.Phase = domain.PhaseAssembling
		if err := s.sessionRepo.Update(ctx, sess); err == nil {
			s.bus.Publish(sess)
		}
	}

	meta, err := s.fs.ReadUploadMeta(envID, projectID, uploadID)
	if err != nil {
		s.failSession(ctx, uploadID, fmt.Sprintf("read upload meta: %v", err))
		return fmt.Errorf("read upload meta: %w", err)
	}
	log.Debug("assemble: meta loaded", zap.String("buildId", meta.BuildID), zap.Int("totalChunks", meta.TotalChunks), zap.Int64("totalSize", meta.TotalSize))

	chunkDir := s.fs.ChunkDir(envID, projectID, uploadID)
	entries, err := os.ReadDir(chunkDir)
	if err != nil {
		s.failSession(ctx, uploadID, err.Error())
		return err
	}

	type indexedEntry struct {
		path  string
		index int
	}
	var chunks []indexedEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		idx, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		chunks = append(chunks, indexedEntry{filepath.Join(chunkDir, e.Name()), idx})
	}
	sort.Slice(chunks, func(i, j int) bool { return chunks[i].index < chunks[j].index })
	log.Debug("assemble: chunks discovered", zap.Int("found", len(chunks)), zap.Int("expected", meta.TotalChunks))

	if len(chunks) != meta.TotalChunks {
		msg := fmt.Sprintf("expected %d chunks, have %d", meta.TotalChunks, len(chunks))
		s.failSession(ctx, uploadID, msg)
		return fmt.Errorf("%s", msg)
	}

	tempDir := s.assembleTempDir
	if tempDir == "" {
		tempDir = filepath.Dir(chunkDir)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		s.failSession(ctx, uploadID, fmt.Sprintf("create temp dir: %v", err))
		return fmt.Errorf("create temp dir: %w", err)
	}
	assembled, err := os.CreateTemp(tempDir, "allure-assembled-*.zip")
	if err != nil {
		s.failSession(ctx, uploadID, err.Error())
		return err
	}
	defer os.Remove(assembled.Name())
	defer assembled.Close()
	log.Debug("assemble: temp file created", zap.String("path", assembled.Name()))

	for _, c := range chunks {
		log.Debug("assemble: copying chunk", zap.Int("index", c.index))
		cf, err := os.Open(c.path)
		if err != nil {
			s.failSession(ctx, uploadID, fmt.Sprintf("open chunk %d: %v", c.index, err))
			return fmt.Errorf("open chunk %d: %w", c.index, err)
		}
		_, err = io.Copy(assembled, cf)
		cf.Close()
		if err != nil {
			s.failSession(ctx, uploadID, fmt.Sprintf("copy chunk %d: %v", c.index, err))
			return fmt.Errorf("copy chunk %d: %w", c.index, err)
		}
	}
	log.Debug("assemble: all chunks concatenated, starting unzip")

	assembled.Seek(0, 0)
	if err := s.reportSvc.SaveResultsStream(ctx, envID, projectID, meta.BuildID, assembled); err != nil {
		s.failSession(ctx, uploadID, fmt.Sprintf("unzip assembled: %v", err))
		return fmt.Errorf("unzip assembled: %w", err)
	}
	log.Debug("assemble: unzip complete, cleaning chunk dir")
	return os.RemoveAll(chunkDir)
}

// TrackStreamUpload creates a session for a single-shot streaming upload,
// delegates to ReportService.SaveResultsStream, and marks the session done/failed.
func (s *UploadService) TrackStreamUpload(ctx context.Context, envID, projectID, buildID, fileName, uploadedBy string, totalSize int64, body io.Reader) error {
	log := s.log.With(zap.String("projectId", projectID), zap.String("buildId", buildID), zap.String("envId", envID))
	log.Debug("stream upload: starting", zap.String("fileName", fileName), zap.Int64("totalSize", totalSize))

	sessID := uuid.New().String()
	sess := &domain.UploadSession{
		ID:          sessID,
		UploadID:    sessID, // reuse session ID as the upload identifier
		BuildID:     buildID,
		ProjectID:   projectID,
		EnvID:       envID,
		FileName:    fileName,
		TotalSize:   totalSize,
		TotalChunks: 1,
		UploadedBy:  uploadedBy,
		Phase:       domain.PhaseUploading,
		StartedAt:   time.Now().UTC(),
	}
	if err := s.sessionRepo.Create(ctx, sess); err == nil {
		s.bus.Publish(sess)
	}

	if err := s.validateEnvAndProject(ctx, envID, projectID); err != nil {
		s.failSessionByID(ctx, sess, err.Error())
		return err
	}
	log.Debug("stream upload: env+project validated, starting save")

	if err := s.reportSvc.SaveResultsStream(ctx, envID, projectID, buildID, body); err != nil {
		log.Debug("stream upload: save failed", zap.Error(err))
		s.failSessionByID(ctx, sess, err.Error())
		return err
	}
	log.Debug("stream upload: save complete, transitioning to assembling")

	// Use background context — the request context may be cancelled if the
	// client closed the connection, but we still need to persist the final state.
	sess.Phase = domain.PhaseAssembling
	sess.ReceivedChunks = 1
	if err := s.sessionRepo.Update(context.Background(), sess); err == nil {
		s.bus.Publish(sess)
	}

	// Auto-generate report after stream upload.
	log.Debug("stream upload: auto-generating report")
	if _, err := s.reportSvc.Generate(context.Background(), envID, projectID, buildID, GenerateOptions{}); err != nil {
		log.Error("stream upload: auto-generate failed", zap.Error(err))
		s.failSessionByID(context.Background(), sess, fmt.Sprintf("auto-generate: %v", err))
		return err
	}
	log.Debug("stream upload: report generated")
	return nil
}

// DeleteSession removes an upload session from the DB and cleans up all
// associated files: chunk staging dir, extracted results, and generated report.
func (s *UploadService) DeleteSession(ctx context.Context, id string) error {
	sess, err := s.sessionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sess != nil {
		_ = os.RemoveAll(s.fs.ChunkDir(sess.EnvID, sess.ProjectID, sess.UploadID))
		_ = os.RemoveAll(s.fs.ResultsDir(sess.EnvID, sess.ProjectID, sess.BuildID))
		_ = os.RemoveAll(s.fs.ReportDir(sess.EnvID, sess.ProjectID, sess.BuildID))
	}
	return s.sessionRepo.Delete(ctx, id)
}

// ── internal helpers ──────────────────────────────────────────────────────────

// validateEnvAndProject checks that the environment and project both exist.
// Returns a descriptive error (wrapping the domain sentinel) if either is missing.
func (s *UploadService) validateEnvAndProject(ctx context.Context, envID, projectID string) error {
	if _, err := s.envRepo.Get(ctx, envID); err != nil {
		if errors.Is(err, domain.ErrEnvironmentNotFound) {
			return fmt.Errorf("environment %q not found: %w", envID, domain.ErrEnvironmentNotFound)
		}
		return fmt.Errorf("look up environment: %w", err)
	}
	if _, err := s.projectRepo.Get(ctx, envID, projectID); err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found in environment %q: %w", projectID, envID, domain.ErrProjectNotFound)
		}
		return fmt.Errorf("look up project: %w", err)
	}
	return nil
}

func (s *UploadService) failSession(ctx context.Context, uploadID, msg string) {
	if sess, _ := s.sessionRepo.GetByUploadID(ctx, uploadID); sess != nil {
		s.failSessionByID(ctx, sess, msg)
	}
}

func (s *UploadService) failSessionByID(ctx context.Context, sess *domain.UploadSession, msg string) {
	now := time.Now().UTC()
	sess.FailedAtPhase = sess.Phase // capture which phase was active before failing
	sess.Phase = domain.PhaseFailed
	sess.Error = msg
	sess.CompletedAt = &now
	// Use a background context so a cancelled request context (client disconnect)
	// does not silently prevent the session from being marked as failed.
	bg := context.Background()
	if err := s.sessionRepo.Update(bg, sess); err == nil {
		s.bus.Publish(sess)
	}
}
