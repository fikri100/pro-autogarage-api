package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/service"
)

type CashflowHandler struct {
	service *service.CashflowService
}

func NewCashflowHandler(service *service.CashflowService) *CashflowHandler {
	return &CashflowHandler{service: service}
}

// CreateManualCashflow handles manual entry of cashflows (primarily EXP)
func (h *CashflowHandler) CreateManualCashflow(w http.ResponseWriter, r *http.Request) {
	var req domain.CashflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	creator := "system" // Mocked authenticated staff

	cashflow, err := h.service.CreateManualCashflow(r.Context(), req, creator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cashflow)
}

// GetAllCashflows retrieves list of active cashflows with filter
func (h *CashflowHandler) GetAllCashflows(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	typeFilter := query.Get("type")
	categoryFilter := query.Get("category")
	startDate := query.Get("startDate")
	endDate := query.Get("endDate")

	list, err := h.service.GetAllCashflows(r.Context(), typeFilter, categoryFilter, startDate, endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if list == nil {
		list = []*domain.Cashflow{}
	}
	json.NewEncoder(w).Encode(list)
}

// DeleteCashflow soft deletes a manual cashflow entry
func (h *CashflowHandler) DeleteCashflow(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	updatedBy := "system"

	err = h.service.DeleteCashflow(r.Context(), id, updatedBy)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "cannot delete") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetFinanceSummary retrieves KPI aggregates
func (h *CashflowHandler) GetFinanceSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetFinanceSummary(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetChartData aggregates cashflows for visualization
func (h *CashflowHandler) GetChartData(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	period := query.Get("period")

	chartItems, err := h.service.GetChartData(r.Context(), period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chartItems)
}
