package handler

import (
	"encoding/json"
	"net/http"

	"pro-autogarage-api/internal/service"
)

type DashboardHandler struct {
	service *service.DashboardService
}

func NewDashboardHandler(service *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

// GetDashboardSummary handles fetching consolidated admin dashboard metrics
func (h *DashboardHandler) GetDashboardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetDashboardSummary(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
