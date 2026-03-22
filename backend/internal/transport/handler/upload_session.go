package handler

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
)

// UploadSessionHandler exposes upload session tracking endpoints.
type UploadSessionHandler struct {
	sessionRepo domain.UploadSessionRepository
	uploadSvc   *usecase.UploadService
	bus         *usecase.EventBus
	log         *zap.Logger
}

func NewUploadSessionHandler(sessionRepo domain.UploadSessionRepository, uploadSvc *usecase.UploadService, bus *usecase.EventBus, log *zap.Logger) *UploadSessionHandler {
	return &UploadSessionHandler{sessionRepo: sessionRepo, uploadSvc: uploadSvc, bus: bus, log: log}
}

// List returns the 100 most recent upload sessions.
//
//	GET /api/uploads
func (h *UploadSessionHandler) List(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.sessionRepo.ListRecent(r.Context(), 100)
	if err != nil {
		h.log.Error("list upload sessions failed", zap.Error(err))
		http.Error(w, "failed to list sessions", http.StatusInternalServerError)
		return
	}
	if sessions == nil {
		sessions = []*domain.UploadSession{}
	}
	writeJSON(w, sessions)
}

// Delete removes a single upload session record.
//
//	DELETE /api/uploads/{id}
func (h *UploadSessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validatePathParam(w, "id", id) {
		return
	}
	if err := h.uploadSvc.DeleteSession(r.Context(), id); err != nil {
		h.log.Error("delete upload session failed", zap.Error(err), zap.String("id", id))
		http.Error(w, "failed to delete session", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Stream opens an SSE connection that receives session_updated events in real time.
//
//	GET /api/uploads/stream
func (h *UploadSessionHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx proxy buffering

	id, ch := h.bus.Subscribe()
	defer h.bus.Unsubscribe(id)

	// Send a ping immediately so the client knows the connection is alive.
	w.Write([]byte(": ping\n\n"))
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write(msg)
			flusher.Flush()
		case <-ticker.C:
			w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
