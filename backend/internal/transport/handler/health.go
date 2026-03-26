package handler

import (
	"net/http"
	"time"
)

// Build-time variables injected via -ldflags.
var (
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = "unknown"
)

var startTime = time.Now()

type HealthHandler struct {
	db interface{ Ping() error }
}

func NewHealthHandler(db interface{ Ping() error }) *HealthHandler {
	return &HealthHandler{db: db}
}

// Check returns JSON health status.
//
//	GET /healthz
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := h.db.Ping(); err != nil {
		// Do not expose raw error — it may contain hostnames or DSN details (L-07).
		dbStatus = "unavailable"
	}

	status := "ok"
	if dbStatus != "ok" {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	if status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	writeJSON(w, map[string]any{
		"status":  status,
		"uptime":  time.Since(startTime).Round(time.Second).String(),
		"db":      dbStatus,
		"version": Version,
	})
}

// Info returns build-time version info.
//
//	GET /api/version
func (h *HealthHandler) Info(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"version":   Version,
		"buildTime": BuildTime,
		"goVersion": GoVersion,
	})
}
