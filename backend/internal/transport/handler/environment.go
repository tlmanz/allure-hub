package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/usecase"
)

type EnvironmentHandler struct {
	svc *usecase.EnvironmentService
	log *zap.Logger
}

func NewEnvironmentHandler(svc *usecase.EnvironmentService, log *zap.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{svc: svc, log: log}
}

func (h *EnvironmentHandler) List(w http.ResponseWriter, r *http.Request) {
	envs, err := h.svc.List(r.Context())
	if err != nil {
		h.log.Error("list environments failed", zap.Error(err))
		http.Error(w, "failed to list environments", http.StatusInternalServerError)
		return
	}
	writeJSON(w, envs)
}

func (h *EnvironmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Icon string `json:"icon"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	e, err := h.svc.Create(r.Context(), req.ID, req.Name, req.Icon)
	if err != nil {
		h.log.Error("create environment failed", zap.Error(err))
		http.Error(w, "failed to create environment", http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, e)
}

func (h *EnvironmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("envId")
	if !validatePathParam(w, "envId", envID) {
		return
	}
	var req struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	e, err := h.svc.Update(r.Context(), envID, req.Name, req.Icon)
	if err != nil {
		h.log.Error("update environment failed", zap.String("envId", envID), zap.Error(err))
		http.Error(w, "failed to update environment", http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, e)
}

func (h *EnvironmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("envId")
	if !validatePathParam(w, "envId", envID) {
		return
	}
	if envID == "default" {
		http.Error(w, "the default environment cannot be deleted", http.StatusForbidden)
		return
	}
	if err := h.svc.Delete(r.Context(), envID); err != nil {
		h.log.Error("delete environment failed", zap.String("envId", envID), zap.Error(err))
		http.Error(w, "failed to delete environment", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
