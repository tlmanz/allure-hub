package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// UploadSessionRepo implements domain.UploadSessionRepository using SQL.
type UploadSessionRepo struct{ db *DB }

func NewUploadSessionRepo(db *DB) *UploadSessionRepo { return &UploadSessionRepo{db} }

func (r *UploadSessionRepo) Create(ctx context.Context, s *domain.UploadSession) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`INSERT INTO upload_sessions
		  (id, upload_id, build_id, project_id, env_id, file_name,
		   total_size, total_chunks, received_chunks, phase, failed_at_phase, error,
		   uploaded_by, started_at, completed_at, report_url, passed, failed, skipped, total)
		  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		s.ID, s.UploadID, s.BuildID, s.ProjectID, s.EnvID, s.FileName,
		s.TotalSize, s.TotalChunks, s.ReceivedChunks, string(s.Phase), string(s.FailedAtPhase), s.Error,
		s.UploadedBy, s.StartedAt.UTC().Format(time.RFC3339), nullTime(s.CompletedAt),
		s.ReportURL, s.Passed, s.Failed, s.Skipped, s.Total,
	)
	if err != nil {
		return fmt.Errorf("repository: create upload session: %w", err)
	}
	return nil
}

func (r *UploadSessionRepo) Update(ctx context.Context, s *domain.UploadSession) error {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`UPDATE upload_sessions SET
		  upload_id = ?, build_id = ?, project_id = ?, env_id = ?,
		  file_name = ?, total_size = ?, total_chunks = ?, received_chunks = ?,
		  phase = ?, failed_at_phase = ?, error = ?, uploaded_by = ?, completed_at = ?,
		  report_url = ?, passed = ?, failed = ?, skipped = ?, total = ?
		  WHERE id = ?`),
		s.UploadID, s.BuildID, s.ProjectID, s.EnvID,
		s.FileName, s.TotalSize, s.TotalChunks, s.ReceivedChunks,
		string(s.Phase), string(s.FailedAtPhase), s.Error, s.UploadedBy, nullTime(s.CompletedAt),
		s.ReportURL, s.Passed, s.Failed, s.Skipped, s.Total,
		s.ID,
	)
	if err != nil {
		return fmt.Errorf("repository: update upload session: %w", err)
	}
	return nil
}

// IncrementReceivedChunks atomically increments the received_chunks counter
// and returns the updated session. This is a single round-trip to the DB and
// avoids the lost-update race in concurrent chunk uploads (M-05).
func (r *UploadSessionRepo) IncrementReceivedChunks(ctx context.Context, uploadID string) (*domain.UploadSession, error) {
	_, err := r.db.ExecContext(ctx,
		r.db.Ph(`UPDATE upload_sessions SET received_chunks = received_chunks + 1 WHERE upload_id = ?`),
		uploadID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: increment received chunks: %w", err)
	}
	return r.GetByUploadID(ctx, uploadID)
}

func (r *UploadSessionRepo) GetByUploadID(ctx context.Context, uploadID string) (*domain.UploadSession, error) {
	row := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, upload_id, build_id, project_id, env_id, file_name,
		         total_size, total_chunks, received_chunks, phase, failed_at_phase, error,
		         uploaded_by, started_at, completed_at, report_url, passed, failed, skipped, total
		         FROM upload_sessions WHERE upload_id = ? LIMIT 1`),
		uploadID,
	)
	return scanSession(row)
}

func (r *UploadSessionRepo) GetByBuild(ctx context.Context, projectID, buildID string) (*domain.UploadSession, error) {
	row := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, upload_id, build_id, project_id, env_id, file_name,
		         total_size, total_chunks, received_chunks, phase, failed_at_phase, error,
		         uploaded_by, started_at, completed_at, report_url, passed, failed, skipped, total
		         FROM upload_sessions WHERE project_id = ? AND build_id = ?
		         ORDER BY started_at DESC LIMIT 1`),
		projectID, buildID,
	)
	return scanSession(row)
}

func (r *UploadSessionRepo) ListRecent(ctx context.Context, limit int) ([]*domain.UploadSession, error) {
	rows, err := r.db.QueryContext(ctx,
		r.db.Ph(`SELECT id, upload_id, build_id, project_id, env_id, file_name,
		         total_size, total_chunks, received_chunks, phase, failed_at_phase, error,
		         uploaded_by, started_at, completed_at, report_url, passed, failed, skipped, total
		         FROM upload_sessions ORDER BY started_at DESC LIMIT ?`),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: list upload sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.UploadSession
	for rows.Next() {
		s, err := scanSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *UploadSessionRepo) GetByID(ctx context.Context, id string) (*domain.UploadSession, error) {
	row := r.db.QueryRowContext(ctx,
		r.db.Ph(`SELECT id, upload_id, build_id, project_id, env_id, file_name,
		         total_size, total_chunks, received_chunks, phase, failed_at_phase, error,
		         uploaded_by, started_at, completed_at, report_url, passed, failed, skipped, total
		         FROM upload_sessions WHERE id = ? LIMIT 1`),
		id,
	)
	return scanSession(row)
}

func (r *UploadSessionRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.db.Ph(`DELETE FROM upload_sessions WHERE id = ?`), id)
	if err != nil {
		return fmt.Errorf("repository: delete upload session: %w", err)
	}
	return nil
}

func (r *UploadSessionRepo) DeleteByProject(ctx context.Context, projectID string) error {
	_, err := r.db.ExecContext(ctx, r.db.Ph(`DELETE FROM upload_sessions WHERE project_id = ?`), projectID)
	if err != nil {
		return fmt.Errorf("repository: delete upload sessions by project: %w", err)
	}
	return nil
}

func (r *UploadSessionRepo) DeleteByEnv(ctx context.Context, envID string) error {
	_, err := r.db.ExecContext(ctx, r.db.Ph(`DELETE FROM upload_sessions WHERE env_id = ?`), envID)
	if err != nil {
		return fmt.Errorf("repository: delete upload sessions by env: %w", err)
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func scanSession(row *sql.Row) (*domain.UploadSession, error) {
	var s domain.UploadSession
	var phase, failedAtPhase string
	var startedAt string
	var completedAt sql.NullString
	err := row.Scan(
		&s.ID, &s.UploadID, &s.BuildID, &s.ProjectID, &s.EnvID, &s.FileName,
		&s.TotalSize, &s.TotalChunks, &s.ReceivedChunks, &phase, &failedAtPhase, &s.Error,
		&s.UploadedBy, &startedAt, &completedAt, &s.ReportURL, &s.Passed, &s.Failed, &s.Skipped, &s.Total,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: scan upload session: %w", err)
	}
	s.Phase = domain.UploadPhase(phase)
	s.FailedAtPhase = domain.UploadPhase(failedAtPhase)
	if s.StartedAt, err = parseTimestamp(startedAt); err != nil {
		return nil, err
	}
	if completedAt.Valid {
		t, err := parseTimestamp(completedAt.String)
		if err != nil {
			return nil, err
		}
		s.CompletedAt = &t
	}
	return &s, nil
}

func scanSessionRow(rows *sql.Rows) (*domain.UploadSession, error) {
	var s domain.UploadSession
	var phase, failedAtPhase string
	var startedAt string
	var completedAt sql.NullString
	err := rows.Scan(
		&s.ID, &s.UploadID, &s.BuildID, &s.ProjectID, &s.EnvID, &s.FileName,
		&s.TotalSize, &s.TotalChunks, &s.ReceivedChunks, &phase, &failedAtPhase, &s.Error,
		&s.UploadedBy, &startedAt, &completedAt, &s.ReportURL, &s.Passed, &s.Failed, &s.Skipped, &s.Total,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: scan upload session row: %w", err)
	}
	s.Phase = domain.UploadPhase(phase)
	s.FailedAtPhase = domain.UploadPhase(failedAtPhase)
	if s.StartedAt, err = parseTimestamp(startedAt); err != nil {
		return nil, err
	}
	if completedAt.Valid {
		t, err := parseTimestamp(completedAt.String)
		if err != nil {
			return nil, err
		}
		s.CompletedAt = &t
	}
	return &s, nil
}

func nullTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(time.RFC3339), Valid: true}
}
