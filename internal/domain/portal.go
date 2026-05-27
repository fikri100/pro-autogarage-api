package domain

// SendOTPRequest payload for requesting an OTP
type SendOTPRequest struct {
	Phone string `json:"phone"`
}

// VerifyOTPRequest payload for verifying an OTP
type VerifyOTPRequest struct {
	Phone   string `json:"phone"`
	OTPCode string `json:"otpCode"`
}

// RegisterCustomerRequest payload for new customer self-registration
type RegisterCustomerRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Username string `json:"username"`
	Password string `json:"password"`
	Address  string `json:"address"`
}

// PortalLoginRequest payload for logging into the customer portal
type PortalLoginRequest struct {
	UsernameOrPhone string `json:"usernameOrPhone"`
	Password        string `json:"password"`
}

// PortalBookingRequest payload for self-service booking creations
type PortalBookingRequest struct {
	LicensePlate string  `json:"licensePlate"`
	Brand        string  `json:"brand"`
	Model        string  `json:"model"`
	YearMade     int     `json:"yearMade"`
	Transmission string  `json:"transmission"`
	BookingDate  string  `json:"bookingDate"` // YYYY-MM-DD
	BookingTime  string  `json:"bookingTime"` // HH:MM
	Complaints   *string `json:"complaints"`
}

// PortalDashboardSummary represents metrics for portal homepage
type PortalDashboardSummary struct {
	TotalVehicles  int `json:"totalVehicles"`
	ActiveBookings int `json:"activeBookings"`
	TotalHistory   int `json:"totalHistory"`
}

// PortalVehicleRequest represents vehicle payload for CRUD operations
type PortalVehicleRequest struct {
	LicensePlate string `json:"licensePlate"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	YearMade     int    `json:"yearMade"`
	Transmission string `json:"transmission"`
}

// PortalVehicleResponse represents vehicle details in JSON response
type PortalVehicleResponse struct {
	ID           int    `json:"id"`
	CustomerID   int    `json:"customerId"`
	LicensePlate string `json:"licensePlate"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	YearMade     int    `json:"yearMade"`
	Transmission string `json:"transmission"`
}

// UpdateProfileRequest represents profile update payload
type UpdateProfileRequest struct {
	Name            string  `json:"name"`
	Username        string  `json:"username"`
	Address         string  `json:"address"`
	CurrentPassword *string `json:"currentPassword,omitempty"`
	NewPassword     *string `json:"newPassword,omitempty"`
}

