package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/usecase"
)

type ProjectHandler struct {
	svc *usecase.ProjectService
	log *zap.Logger
}

func NewProjectHandler(svc *usecase.ProjectService, log *zap.Logger) *ProjectHandler {
	return &ProjectHandler{svc: svc, log: log}
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("envId")
	if !validatePathParam(w, "envId", envID) {
		return
	}
	summaries, err := h.svc.ListSummaries(r.Context(), envID)
	if err != nil {
		h.log.Error("list projects failed", zap.Error(err))
		http.Error(w, "failed to list projects", http.StatusInternalServerError)
		return
	}
	writeJSON(w, summaries)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("envId")
	if !validatePathParam(w, "envId", envID) {
		return
	}
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	p, err := h.svc.Create(r.Context(), envID, req.ID, req.Name)
	if err != nil {
		h.log.Error("create project failed", zap.Error(err))
		http.Error(w, "failed to create project", http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, p)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("envId")
	projectID := r.PathValue("projectId")
	if !validatePathParam(w, "envId", envID) {
		return
	}
	if !validatePathParam(w, "projectId", projectID) {
		return
	}
	if err := h.svc.Delete(r.Context(), envID, projectID); err != nil {
		h.log.Error("delete project failed", zap.String("projectId", projectID), zap.Error(err))
		http.Error(w, "failed to delete project", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Response may be partially written; can only log, not re-send.
		_ = err
	}
}
