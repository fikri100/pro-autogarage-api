package domain

import "time"

type Booking struct {
	ID                int       `json:"id"`
	CustomerID        int       `json:"customerId"`
	CustomerName      string    `json:"customerName"`
	CustomerPhone     string    `json:"customerPhone"`
	VehicleID         int       `json:"vehicleId"`
	LicensePlate      string    `json:"licensePlate"`
	VehicleBrand      string    `json:"vehicleBrand"`
	VehicleModel      string    `json:"vehicleModel"`
	BookingDate       string    `json:"bookingDate"` // format: YYYY-MM-DD
	BookingTime       string    `json:"bookingTime"` // format: HH:MM
	Complaints        *string   `json:"complaints"`
	OperationalStatus string    `json:"operationalStatus"` // PENDING, CONFIRMED, CANCELLED
	Status            string    `json:"status"`
	CreatedBy         *string   `json:"createdBy"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedBy         *string   `json:"updatedBy"`
	UpdatedAt         time.Time `json:"updatedAt"`
	VehicleCustomerID int       `json:"vehicleCustomerId"`
	VehicleOwnerName  string    `json:"vehicleOwnerName"`
}

type BookingRequest struct {
	CustomerID  int     `json:"customerId"`
	VehicleID   int     `json:"vehicleId"`
	BookingDate string  `json:"bookingDate"`
	BookingTime string  `json:"bookingTime"`
	Complaints  *string `json:"complaints"`
}
