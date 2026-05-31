package handler

import (
	"encoding/json"
	"fmt"
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

// Login HTTP Handler
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		if err.Error() == "Username atau password salah" {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetAllMenus HTTP Handler
func (h *UserHandler) GetAllMenus(w http.ResponseWriter, r *http.Request) {
	menus, err := h.service.GetAllMenus(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(menus)
}

// GetRoleMenus HTTP Handler
func (h *UserHandler) GetRoleMenus(w http.ResponseWriter, r *http.Request) {
	roleIDStr := r.PathValue("id")
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	menuIDs, err := h.service.GetRoleMenus(r.Context(), roleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(menuIDs)
}

// UpdateRoleMenus HTTP Handler
func (h *UserHandler) UpdateRoleMenus(w http.ResponseWriter, r *http.Request) {
	roleIDStr := r.PathValue("id")
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	var req domain.RoleMenuMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.service.UpdateRoleMenus(r.Context(), roleID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success"}`)
}

// CreateRole HTTP Handler
func (h *UserHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req domain.RoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateRole(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": %d, "roleName": "%s"}`, id, req.RoleName)
}

// UpdateRole HTTP Handler
func (h *UserHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var req domain.RoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.service.UpdateRole(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success"}`)
}

// DeleteRole HTTP Handler
func (h *UserHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	err = h.service.DeleteRole(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success"}`)
}

// CreateEmployee HTTP Handler
func (h *UserHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req domain.EmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateEmployee(r.Context(), req, "admin")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": %d}`, id)
}

// UpdateEmployee HTTP Handler
func (h *UserHandler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var req domain.EmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.service.UpdateEmployee(r.Context(), id, req, "admin")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success"}`)
}

// DeleteEmployee HTTP Handler
func (h *UserHandler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	err = h.service.DeleteEmployee(r.Context(), id, "admin")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success"}`)
}


