package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/service"
)

type VehicleHandler struct {
	service *service.VehicleService
}

func NewVehicleHandler(service *service.VehicleService) *VehicleHandler {
	return &VehicleHandler{service: service}
}

func (h *VehicleHandler) GetVehicles(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.URL.Query().Get("customerId")
	var custID int
	var err error
	if custIDStr != "" {
		custID, err = strconv.Atoi(custIDStr)
		if err != nil {
			http.Error(w, "Invalid customerId format", http.StatusBadRequest)
			return
		}
	}
	vehicles, err := h.service.GetAllVehicles(r.Context(), custID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vehicles)
}

func (h *VehicleHandler) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	var req domain.VehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := h.service.CreateVehicle(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": %d}`, id)
}

func (h *VehicleHandler) UpdateVehicle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	var req domain.UpdateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.service.UpdateVehicle(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success"}`)
}

func (h *VehicleHandler) DeleteVehicle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	err = h.service.DeleteVehicle(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success"}`)
}
