package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/service"
)

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(service *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

// GetReadyWorkOrders HTTP Handler
func (h *TransactionHandler) GetReadyWorkOrders(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	search := r.URL.Query().Get("search")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	wos, total, err := h.service.GetReadyWorkOrders(r.Context(), search, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if wos == nil {
		wos = []*domain.WorkOrder{}
	}

	pageStart := (page - 1) * limit + 1
	if total == 0 {
		pageStart = 0
	}
	pageEnd := pageStart + len(wos) - 1
	if pageStart > 0 && len(wos) == 0 {
		pageEnd = 0
	}

	response := domain.PaginatedResponse[*domain.WorkOrder]{
		Data: wos,
		PageResponse: domain.PageResponse{
			PageStart: pageStart,
			PageEnd:   pageEnd,
			Limit:     limit,
			Total:     total,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// GetTransactionByWO HTTP Handler
func (h *TransactionHandler) GetTransactionByWO(w http.ResponseWriter, r *http.Request) {
	woIDStr := r.PathValue("woId")
	woID, err := strconv.Atoi(woIDStr)
	if err != nil {
		http.Error(w, "Invalid WO ID format", http.StatusBadRequest)
		return
	}

	t, details, err := h.service.GetTransactionByWO(r.Context(), woID)
	if err != nil {
		if strings.Contains(err.Error(), "no draft invoice") || strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Attach details directly to transaction for client ease
	t.Details = details

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// FinalizePayment HTTP Handler
func (h *TransactionHandler) FinalizePayment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var req domain.PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	adminUser := "system"
	err = h.service.FinalizePayment(r.Context(), id, req, adminUser)
	if err != nil {
		if strings.Contains(err.Error(), "already paid") || strings.Contains(err.Error(), "does not exist") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
