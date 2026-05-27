package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// GetAllUsers HTTP Handler
func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.GetAllUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if users == nil {
		users = []*domain.User{}
	}
	json.NewEncoder(w).Encode(users)
}

// CreateUser HTTP Handler
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req domain.UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	err := h.service.CreateUser(r.Context(), req, "system")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// UpdateUser HTTP Handler
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var req domain.UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	err := h.service.UpdateUser(r.Context(), id, req, "system")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetRoles HTTP Handler
func (h *UserHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.GetRoles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

// GetEmployees HTTP Handler
func (h *UserHandler) GetEmployees(w http.ResponseWriter, r *http.Request) {
	emps, err := h.service.GetEmployees(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(emps)
}
