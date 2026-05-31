package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/service"
)

type WorkOrderHandler struct {
	service *service.WorkOrderService
}

func NewWorkOrderHandler(service *service.WorkOrderService) *WorkOrderHandler {
	return &WorkOrderHandler{service: service}
}

func (h *WorkOrderHandler) CreateWorkOrder(w http.ResponseWriter, r *http.Request) {
	var req domain.WorkOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	adminUser := "system"
	wo, err := h.service.CreateWorkOrder(r.Context(), req, adminUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wo)
}

func (h *WorkOrderHandler) GetAllActiveWorkOrders(w http.ResponseWriter, r *http.Request) {
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

	wos, total, err := h.service.GetAllActiveWorkOrders(r.Context(), search, page, limit)
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

func (h *WorkOrderHandler) GetWorkOrderByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	wo, err := h.service.GetWorkOrder(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wo)
}

func (h *WorkOrderHandler) AssignMechanic(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var req struct {
		MechanicID *int    `json:"mechanicId"`
		Notes      *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	adminUser := "system"
	err = h.service.AssignMechanic(r.Context(), id, req.MechanicID, req.Notes, adminUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkOrderHandler) SaveEstimation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var req []domain.TransactionDetail
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	adminUser := "system"
	// Map slice of concrete values to pointers
	var details []*domain.TransactionDetail
	for i := range req {
		details = append(details, &req[i])
	}

	err = h.service.SaveEstimation(r.Context(), id, details, adminUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkOrderHandler) CompleteWorkOrder(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	adminUser := "system"
	err = h.service.CompleteWorkOrder(r.Context(), id, adminUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkOrderHandler) UpdateEstimation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var req domain.UpdateEstimationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	adminUser := "system"
	err = h.service.UpdateEstimation(r.Context(), id, req, adminUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
