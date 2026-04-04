package handler

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/domain"
)

// overviewRepo is the minimal interface the OverviewHandler needs.
type overviewRepo interface {
	GetStats(ctx context.Context) (*domain.OverviewStats, error)
}

// OverviewHandler serves the analytics overview dashboard endpoint.
type OverviewHandler struct {
	repo overviewRepo
	log  *zap.Logger
}

func NewOverviewHandler(repo overviewRepo, log *zap.Logger) *OverviewHandler {
	return &OverviewHandler{repo: repo, log: log}
}

// GetStats handles GET /api/overview
func (h *OverviewHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.GetStats(r.Context())
	if err != nil {
		h.log.Error("overview stats failed", zap.Error(err))
		http.Error(w, "failed to load overview stats", http.StatusInternalServerError)
		return
	}
	writeJSON(w, stats)
}
