package domain

import "time"

// Customer represents the database entity
type Customer struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Phone         string    `json:"phone"`
	Address       *string   `json:"address"` // nullable
	Email         *string   `json:"email"`   // nullable
	Username      *string   `json:"username"` // nullable/unique
	IsSelfService bool      `json:"isSelfService"`
	Password      *string   `json:"-"` // never expose password in JSON
	Status        string    `json:"status"`
	CreatedBy     *string   `json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedBy     *string   `json:"updatedBy"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Token         string    `json:"token,omitempty"`
}

// CustomerRequest represents the payload from the client
type CustomerRequest struct {
	Name          string  `json:"name"`
	Phone         string  `json:"phone"`
	Address       *string `json:"address"`
	Email         *string `json:"email"`
	Username      *string `json:"username"`
	IsSelfService bool    `json:"isSelfService"`
	Password      *string `json:"password"`

	// Initial Vehicle details (Optional)
	LicensePlate string `json:"licensePlate"`
	Plate        string `json:"plate"` // alias
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	YearMade     int    `json:"yearMade"`
	Year         int    `json:"year"` // alias
	Transmission string `json:"transmission"`
}

// Vehicle represents a customer's vehicle
type Vehicle struct {
	ID           int    `json:"id"`
	CustomerID   int    `json:"customerId"`
	LicensePlate string `json:"licensePlate"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	YearMade     int    `json:"yearMade"`
	Transmission string `json:"transmission"`
}

// VehicleRequest represents the request payload to add a vehicle
type VehicleRequest struct {
	CustomerID   int    `json:"customerId"`
	LicensePlate string `json:"licensePlate"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	YearMade     int    `json:"yearMade"`
	Transmission string `json:"transmission"`
}

// UpdateVehicleRequest represents the request payload to update a vehicle
type UpdateVehicleRequest struct {
	LicensePlate string `json:"licensePlate"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	YearMade     int    `json:"yearMade"`
	Transmission string `json:"transmission"`
}

