package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/service"
)

type PortalHandler struct {
	service *service.PortalService
}

func NewPortalHandler(service *service.PortalService) *PortalHandler {
	return &PortalHandler{service: service}
}

// SendOTP handles generating and sending OTP to customer
func (h *PortalHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req domain.SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	err := h.service.SendOTP(r.Context(), req.Phone)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success", "message": "OTP has been sent to WhatsApp"}`))
}

// VerifyOTP handles verifying the OTP code submitted by customer
func (h *PortalHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req domain.VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	err := h.service.VerifyOTP(r.Context(), req.Phone, req.OTPCode)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success", "message": "OTP verified successfully"}`))
}

// Register handles new customer registration
func (h *PortalHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	customer, err := h.service.Register(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(customer)
}

// Login handles customer login for portal
func (h *PortalHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.PortalLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	customer, err := h.service.Login(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customer)
}

// CreateBooking handles booking creations from portal
func (h *PortalHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	var req domain.PortalBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	booking, err := h.service.CreateBooking(r.Context(), customerID, req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(booking)
}

// GetBookings retrieves bookings made by current authenticated customer
func (h *PortalHandler) GetBookings(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	bookings, err := h.service.GetBookings(r.Context(), customerID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookings)
}

// GetDashboardSummary handles retrieving metrics for portal homepage
func (h *PortalHandler) GetDashboardSummary(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	summary, err := h.service.GetDashboardSummary(r.Context(), customerID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetVehicles handles retrieving registered vehicles for logged-in customer
func (h *PortalHandler) GetVehicles(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	vehicles, err := h.service.GetVehicles(r.Context(), customerID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vehicles)
}

// AddVehicle handles adding a new vehicle for logged-in customer
func (h *PortalHandler) AddVehicle(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	var req domain.PortalVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	vehicle, err := h.service.AddVehicle(r.Context(), customerID, req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vehicle)
}

// UpdateVehicle handles modifying vehicle details for logged-in customer
func (h *PortalHandler) UpdateVehicle(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	vehicleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid ID format"}`, http.StatusBadRequest)
		return
	}

	var req domain.PortalVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	err = h.service.UpdateVehicle(r.Context(), customerID, vehicleID, req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success", "message": "Kendaraan berhasil diperbarui"}`))
}

// DeleteVehicle handles soft-deleting a vehicle for logged-in customer
func (h *PortalHandler) DeleteVehicle(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	vehicleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid ID format"}`, http.StatusBadRequest)
		return
	}

	err = h.service.DeleteVehicle(r.Context(), customerID, vehicleID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success", "message": "Kendaraan berhasil dihapus"}`))
}

// GetProfile handles profile data retrieval
func (h *PortalHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	profile, err := h.service.GetProfile(r.Context(), customerID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// UpdateProfile handles modifying profile info and credentials
func (h *PortalHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	var req domain.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	err = h.service.UpdateProfile(r.Context(), customerID, req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success", "message": "Profil berhasil diperbarui"}`))
}

// CancelBooking handles canceling a pending booking
func (h *PortalHandler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.Header.Get("X-Customer-ID")
	customerID, err := strconv.Atoi(custIDStr)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized: missing or invalid X-Customer-ID header"}`, http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	bookingID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid ID format"}`, http.StatusBadRequest)
		return
	}

	err = h.service.CancelBooking(r.Context(), customerID, bookingID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success", "message": "Reservasi berhasil dibatalkan"}`))
}

