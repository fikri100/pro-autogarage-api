package domain

import "time"

type WorkOrder struct {
	ID                  int        `json:"id"`
	BookingID           *int       `json:"bookingId"` // nullable if walk-in
	VehicleID           int        `json:"vehicleId"`
	LicensePlate        string     `json:"licensePlate"`
	VehicleBrand        string     `json:"vehicleBrand"`
	VehicleModel        string     `json:"vehicleModel"`
	CustomerID          int        `json:"customerId"`
	CustomerName        string     `json:"customerName"`
	MechanicID          *int       `json:"mechanicId"` // nullable initially
	MechanicName        *string    `json:"mechanicName"`
	StartTime           time.Time  `json:"startTime"`
	EndTime             *time.Time `json:"endTime"`
	EstimatedMinutes    *int       `json:"estimatedMinutes"`
	EstimatedCompletion *time.Time `json:"estimatedCompletion"`
	WorkStatusID        int        `json:"workStatusId"`
	WorkStatus          string     `json:"workStatus"` // IN_PROGRESS, COMPLETED, PAID
	Notes               *string    `json:"notes"`
	Status              string     `json:"status"`
	CreatedBy           *string    `json:"createdBy"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedBy           *string    `json:"updatedBy"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type WorkOrderRequest struct {
	BookingID  *int    `json:"bookingId"`
	VehicleID  int     `json:"vehicleId"`
	MechanicID *int    `json:"mechanicId"`
	Notes      *string `json:"notes"`
}

type UpdateEstimationRequest struct {
	EstimatedMinutes int `json:"estimatedMinutes"`
}
