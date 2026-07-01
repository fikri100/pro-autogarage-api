package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/service"
)

type ProductHandler struct {
	service *service.ProductService
}

func NewProductHandler(service *service.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

// CreateProduct HTTP Handler
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req domain.ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	adminUser := "system"

	product, err := h.service.CreateProduct(r.Context(), req, adminUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

// GetAllProducts HTTP Handler with query filtering
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("q")
	itemType := r.URL.Query().Get("type")
	lowStockStr := r.URL.Query().Get("low_stock")
	lowStock := false

	if lowStockStr == "true" {
		lowStock = true
	}

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	products, total, err := h.service.GetAllProducts(r.Context(), searchQuery, itemType, lowStock, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if products == nil {
		products = []*domain.Product{} // return empty list instead of null
	}
	
	pageStart := (page - 1) * limit + 1
	if total == 0 {
		pageStart = 0
	}
	pageEnd := pageStart + len(products) - 1
	if pageStart > 0 && len(products) == 0 {
		pageEnd = 0
	}

	response := domain.PaginatedResponse[*domain.Product]{
		Data: products,
		PageResponse: domain.PageResponse{
			PageStart: pageStart,
			PageEnd:   pageEnd,
			Limit:     limit,
			Total:     total,
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

// GetProductByID HTTP Handler
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	product, err := h.service.GetProductByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// UpdateProduct HTTP Handler
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var req domain.ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	adminUser := "system"

	err = h.service.UpdateProduct(r.Context(), id, req, adminUser)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteProduct HTTP Handler
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	adminUser := "system"

	err = h.service.DeleteProduct(r.Context(), id, adminUser)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RestockProduct HTTP Handler
func (h *ProductHandler) RestockProduct(w http.ResponseWriter, r *http.Request) {
	var req domain.RestockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	adminUser := "system"

	err := h.service.RestockProduct(r.Context(), req, adminUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success", "message": "produk berhasil di-restock"}`)
}

// GetStockLogs HTTP Handler
func (h *ProductHandler) GetStockLogs(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	logs, err := h.service.GetStockLogs(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// GetCategories HTTP Handler
func (h *ProductHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	itemTypeIDStr := r.URL.Query().Get("item_type_id")
	var itemTypeID int
	if itemTypeIDStr != "" {
		itemTypeID, _ = strconv.Atoi(itemTypeIDStr)
	}
	categories, err := h.service.GetAllCategories(r.Context(), itemTypeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// CreateCategory HTTP Handler
func (h *ProductHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req domain.CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := h.service.CreateCategory(r.Context(), req.Name, req.ItemTypeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": %d, "name": "%s", "itemTypeId": %d}`, id, req.Name, req.ItemTypeID)
}

// UpdateCategory HTTP Handler
func (h *ProductHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	var req domain.CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.service.UpdateCategory(r.Context(), id, req.Name, req.ItemTypeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success"}`)
}

// DeleteCategory HTTP Handler
func (h *ProductHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	err = h.service.DeleteCategory(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success"}`)
}


