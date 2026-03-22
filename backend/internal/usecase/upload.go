package usecase

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/tlmanz/allure-hub/internal/domain"
)

// UploadService orchestrates chunked-upload sessions.
type UploadService struct {
	reportSvc       *ReportService
	fs              FileStorage
	sessionRepo     domain.UploadSessionRepository
	bus             *EventBus
	assembleTempDir string // directory for assembled zip temp file; defaults to chunk parent dir
}

func NewUploadService(reportSvc *ReportService, fs FileStorage, sessionRepo domain.UploadSessionRepository, bus *EventBus, assembleTempDir string) *UploadService {
	return &UploadService{
		reportSvc:       reportSvc,
		fs:              fs,
		sessionRepo:     sessionRepo,
		bus:             bus,
		assembleTempDir: assembleTempDir,
	}
}

// InitUpload creates the staging directory, persists a new session, and returns the upload ID.
func (s *UploadService) InitUpload(ctx context.Context, envID, projectID, buildID, fileName string, totalSize int64, totalChunks int) (string, error) {
	uploadID := uuid.New().String()
	if err := os.MkdirAll(s.fs.ChunkDir(projectID, uploadID), 0755); err != nil {
		return "", err
	}
	meta := UploadMeta{
		UploadID:    uploadID,
		BuildID:     buildID,
		TotalSize:   totalSize,
		TotalChunks: totalChunks,
	}
	if err := s.fs.WriteUploadMeta(projectID, uploadID, meta); err != nil {
		return "", err
	}

	sess := &domain.UploadSession{
		ID:          uuid.New().String(),
		UploadID:    uploadID,
		BuildID:     buildID,
		ProjectID:   projectID,
		EnvID:       envID,
		FileName:    fileName,
		TotalSize:   totalSize,
		TotalChunks: totalChunks,
		Phase:       domain.PhaseUploading,
		StartedAt:   time.Now().UTC(),
	}
	if err := s.sessionRepo.Create(ctx, sess); err != nil {
		// Non-fatal: tracking failure must not block the upload itself.
		_ = err
	} else {
		s.bus.Publish(sess)
	}

	return uploadID, nil
}

// SaveChunk writes a single chunk to its slot file and updates received count.
func (s *UploadService) SaveChunk(ctx context.Context, projectID, uploadID string, index, total int, body io.Reader) error {
	f, err := os.Create(s.fs.ChunkPath(projectID, uploadID, index))
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
func (s *UploadService) ChunksReceived(ctx context.Context, projectID, uploadID string) (int, error) {
	entries, err := os.ReadDir(s.fs.ChunkDir(projectID, uploadID))
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
	// Transition to assembling.
	if sess, _ := s.sessionRepo.GetByUploadID(ctx, uploadID); sess != nil {
		sess.Phase = domain.PhaseAssembling
		if err := s.sessionRepo.Update(ctx, sess); err == nil {
			s.bus.Publish(sess)
		}
	}

	meta, err := s.fs.ReadUploadMeta(projectID, uploadID)
	if err != nil {
		s.failSession(ctx, uploadID, fmt.Sprintf("read upload meta: %v", err))
		return fmt.Errorf("read upload meta: %w", err)
	}

	chunkDir := s.fs.ChunkDir(projectID, uploadID)
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

	for _, c := range chunks {
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

	assembled.Seek(0, 0)
	if err := s.reportSvc.SaveResultsStream(ctx, projectID, meta.BuildID, assembled); err != nil {
		s.failSession(ctx, uploadID, fmt.Sprintf("unzip assembled: %v", err))
		return fmt.Errorf("unzip assembled: %w", err)
	}
	return os.RemoveAll(chunkDir)
}

// TrackStreamUpload creates a session for a single-shot streaming upload,
// delegates to ReportService.SaveResultsStream, and marks the session done/failed.
func (s *UploadService) TrackStreamUpload(ctx context.Context, envID, projectID, buildID, fileName string, totalSize int64, body io.Reader) error {
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
		Phase:       domain.PhaseUploading,
		StartedAt:   time.Now().UTC(),
	}
	if err := s.sessionRepo.Create(ctx, sess); err == nil {
		s.bus.Publish(sess)
	}

	if err := s.reportSvc.SaveResultsStream(ctx, projectID, buildID, body); err != nil {
		s.failSessionByID(ctx, sess, err.Error())
		return err
	}

	now := time.Now().UTC()
	sess.Phase = domain.PhaseAssembling
	sess.ReceivedChunks = 1
	sess.CompletedAt = &now
	if err := s.sessionRepo.Update(ctx, sess); err == nil {
		s.bus.Publish(sess)
	}
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
		_ = os.RemoveAll(s.fs.ChunkDir(sess.ProjectID, sess.UploadID))
		_ = os.RemoveAll(s.fs.ResultsDir(sess.ProjectID, sess.BuildID))
		_ = os.RemoveAll(s.fs.ReportDir(sess.ProjectID, sess.BuildID))
	}
	return s.sessionRepo.Delete(ctx, id)
}

// ── internal helpers ──────────────────────────────────────────────────────────

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
	if err := s.sessionRepo.Update(ctx, sess); err == nil {
		s.bus.Publish(sess)
	}
}
