package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
	"github.com/tlmanz/allure-hub/internal/usecase"
	"github.com/tlmanz/allure-hub/internal/storage"
)

// maxJSONBytes caps the body size for small JSON request payloads (1 MB).
const maxJSONBytes int64 = 1 << 20

type ReportHandler struct {
	reportSvc      *usecase.ReportService
	uploadSvc      *usecase.UploadService
	maxChunkBytes  int64
	maxUploadBytes int64
	log            *zap.Logger
}

func NewReportHandler(reportSvc *usecase.ReportService, uploadSvc *usecase.UploadService, maxChunkBytes, maxUploadBytes int64, log *zap.Logger) *ReportHandler {
	return &ReportHandler{reportSvc: reportSvc, uploadSvc: uploadSvc, maxChunkBytes: maxChunkBytes, maxUploadBytes: maxUploadBytes, log: log}
}

// UploadResultsStream handles a raw zip body streamed directly to disk.
//
//	POST /api/environments/:envId/projects/:projectId/results?buildId=xxx
func (h *ReportHandler) UploadResultsStream(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("envId")
	projectID := r.PathValue("projectId")
	buildID := r.URL.Query().Get("buildId")
	if !validatePathParam(w, "envId", envID) {
		return
	}
	if !validatePathParam(w, "projectId", projectID) {
		return
	}
	if !validatePathParam(w, "buildId", buildID) {
		return
	}
	fileName := r.Header.Get("X-Filename")
	if fileName == "" {
		fileName = "results.zip"
	}
	totalSize, _ := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)
	body := http.MaxBytesReader(w, r.Body, h.maxUploadBytes)
	if err := h.uploadSvc.TrackStreamUpload(r.Context(), envID, projectID, buildID, fileName, totalSize, body); err != nil {
		if errors.Is(err, storage.ErrUploadTooLarge) {
			http.Error(w, "upload exceeds maximum allowed size", http.StatusRequestEntityTooLarge)
			return
		}
		h.log.Error("upload results stream failed", zap.String("projectId", projectID), zap.String("buildId", buildID), zap.Error(err))
		http.Error(w, "failed to save results", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "received"})
}

// InitChunkedUpload registers a new chunked upload session.
//
//	POST /api/environments/:envId/projects/:projectId/uploads
func (h *ReportHandler) InitChunkedUpload(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("envId")
	projectID := r.PathValue("projectId")
	if !validatePathParam(w, "envId", envID) {
		return
	}
	if !validatePathParam(w, "projectId", projectID) {
		return
	}
	var req struct {
		BuildID     string `json:"buildId"`
		FileName    string `json:"fileName"`
		TotalSize   int64  `json:"totalSize"`
		TotalChunks int    `json:"totalChunks"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BuildID == "" {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if !validatePathParam(w, "buildId", req.BuildID) {
		return
	}
	fileName := req.FileName
	if fileName == "" {
		fileName = "results.zip"
	}
	uploadID, err := h.uploadSvc.InitUpload(r.Context(), envID, projectID, req.BuildID, fileName, req.TotalSize, req.TotalChunks)
	if err != nil {
		h.log.Error("init chunked upload failed", zap.String("projectId", projectID), zap.Error(err))
		http.Error(w, "failed to init upload", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"uploadId": uploadID})
}

// UploadChunk receives one chunk.
//
//	PUT /api/environments/:envId/projects/:projectId/uploads/:uploadId
func (h *ReportHandler) UploadChunk(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	uploadID := r.PathValue("uploadId")
	if !validatePathParam(w, "projectId", projectID) {
		return
	}
	if !validatePathParam(w, "uploadId", uploadID) {
		return
	}

	chunkIndex, err := strconv.Atoi(r.Header.Get("X-Chunk-Index"))
	if err != nil || chunkIndex < 0 {
		http.Error(w, "X-Chunk-Index header required (non-negative integer)", http.StatusBadRequest)
		return
	}
	totalChunks, err := strconv.Atoi(r.Header.Get("X-Total-Chunks"))
	if err != nil || totalChunks <= 0 {
		http.Error(w, "X-Total-Chunks header required (positive integer)", http.StatusBadRequest)
		return
	}
	body := http.MaxBytesReader(w, r.Body, h.maxChunkBytes)
	if err := h.uploadSvc.SaveChunk(r.Context(), projectID, uploadID, chunkIndex, totalChunks, body); err != nil {
		h.log.Error("save chunk failed", zap.String("uploadId", uploadID), zap.Int("chunkIndex", chunkIndex), zap.Error(err))
		http.Error(w, "failed to save chunk", http.StatusInternalServerError)
		return
	}
	received, err := h.uploadSvc.ChunksReceived(r.Context(), projectID, uploadID)
	if err != nil {
		h.log.Error("count chunks failed", zap.String("uploadId", uploadID), zap.Error(err))
		http.Error(w, "failed to count chunks", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"chunkIndex": chunkIndex, "received": received, "total": totalChunks})
}

// CompleteChunkedUpload assembles all chunks and unzips into results/.
//
//	POST /api/environments/:envId/projects/:projectId/uploads/:uploadId/complete
func (h *ReportHandler) CompleteChunkedUpload(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	uploadID := r.PathValue("uploadId")
	if !validatePathParam(w, "projectId", projectID) {
		return
	}
	if !validatePathParam(w, "uploadId", uploadID) {
		return
	}
	if err := h.uploadSvc.AssembleUpload(r.Context(), projectID, uploadID); err != nil {
		h.log.Error("assemble upload failed", zap.String("uploadId", uploadID), zap.Error(err))
		http.Error(w, "failed to assemble upload", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "assembled"})
}

// GenerateReport triggers allure generate for a given build.
//
//	POST /api/environments/:envId/projects/:projectId/reports
func (h *ReportHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("envId")
	projectID := r.PathValue("projectId")
	if !validatePathParam(w, "envId", envID) {
		return
	}
	if !validatePathParam(w, "projectId", projectID) {
		return
	}
	var req struct {
		BuildID      string         `json:"buildId"`
		ReportConfig map[string]any `json:"reportConfig"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BuildID == "" {
		http.Error(w, "buildId required", http.StatusBadRequest)
		return
	}
	if !validatePathParam(w, "buildId", req.BuildID) {
		return
	}

	opts := usecase.GenerateOptions{Overrides: req.ReportConfig}

	reportURL, err := h.reportSvc.Generate(r.Context(), envID, projectID, req.BuildID, opts)
	if err != nil {
		h.log.Error("generate report failed", zap.String("projectId", projectID), zap.String("buildId", req.BuildID), zap.Error(err))
		http.Error(w, "failed to generate report", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"reportUrl": reportURL})
}

// ListReports returns builds for a project with optional pagination and filter.
//
//	GET /api/environments/:envId/projects/:projectId/reports?limit=15&offset=0&filter=all
func (h *ReportHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	if !validatePathParam(w, "projectId", projectID) {
		return
	}

	q := r.URL.Query()
	limitStr := q.Get("limit")
	offsetStr := q.Get("offset")
	filter := q.Get("filter")

	// If no limit is specified, return all (legacy behaviour).
	if limitStr == "" {
		builds, err := h.reportSvc.List(r.Context(), projectID)
		if err != nil {
			h.log.Error("list reports failed", zap.String("projectId", projectID), zap.Error(err))
			http.Error(w, "failed to list reports", http.StatusInternalServerError)
			return
		}
		writeJSON(w, builds)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
		return
	}
	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	}

	builds, total, err := h.reportSvc.ListPaged(r.Context(), projectID, filter, limit, offset)
	if err != nil {
		h.log.Error("list reports paged failed", zap.String("projectId", projectID), zap.Error(err))
		http.Error(w, "failed to list reports", http.StatusInternalServerError)
		return
	}
	if builds == nil {
		builds = []*domain.Build{}
	}
	writeJSON(w, map[string]any{
		"builds": builds,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteReport removes a build record and its associated files.
//
//	DELETE /api/environments/:envId/projects/:projectId/reports/:buildId
func (h *ReportHandler) DeleteReport(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	buildID := r.PathValue("buildId")
	if !validatePathParam(w, "projectId", projectID) {
		return
	}
	if !validatePathParam(w, "buildId", buildID) {
		return
	}
	if err := h.reportSvc.DeleteBuild(r.Context(), projectID, buildID); err != nil {
		h.log.Error("delete report failed", zap.String("projectId", projectID), zap.String("buildId", buildID), zap.Error(err))
		http.Error(w, "failed to delete report", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ReportStats returns aggregate build statistics for a project.
//
//	GET /api/environments/:envId/projects/:projectId/reports/stats
func (h *ReportHandler) ReportStats(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	if !validatePathParam(w, "projectId", projectID) {
		return
	}
	stats, err := h.reportSvc.Stats(r.Context(), projectID)
	if err != nil {
		h.log.Error("report stats failed", zap.String("projectId", projectID), zap.Error(err))
		http.Error(w, "failed to get stats", http.StatusInternalServerError)
		return
	}
	writeJSON(w, stats)
}
