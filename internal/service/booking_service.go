package service

import (
	"context"
	"errors"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type BookingService struct {
	bookingRepo   *repository.BookingRepository
	workOrderRepo *repository.WorkOrderRepository
}

func NewBookingService(bookingRepo *repository.BookingRepository, workOrderRepo *repository.WorkOrderRepository) *BookingService {
	return &BookingService{
		bookingRepo:   bookingRepo,
		workOrderRepo: workOrderRepo,
	}
}

// CreateBooking validates and saves a booking
func (s *BookingService) CreateBooking(ctx context.Context, req domain.BookingRequest, createdBy string) (*domain.Booking, error) {
	if req.CustomerID <= 0 || req.VehicleID <= 0 || req.BookingDate == "" || req.BookingTime == "" {
		return nil, errors.New("customerId, vehicleId, bookingDate, and bookingTime are required")
	}

	// Check if date and time slot is already taken by another active booking
	bookedSlots, err := s.bookingRepo.GetBookedSlotsByDate(ctx, req.BookingDate)
	if err != nil {
		return nil, err
	}
	for _, slot := range bookedSlots {
		if slot == req.BookingTime || (len(req.BookingTime) >= 5 && slot == req.BookingTime[:5]) {
			return nil, errors.New("jadwal tanggal dan jam ini sudah dipesan")
		}
	}

	booking := &domain.Booking{
		CustomerID:        req.CustomerID,
		VehicleID:         req.VehicleID,
		BookingDate:       req.BookingDate,
		BookingTime:       req.BookingTime,
		Complaints:        req.Complaints,
		OperationalStatus: "PENDING",
		CreatedBy:         &createdBy,
		UpdatedBy:         &createdBy,
	}

	err = s.bookingRepo.Insert(ctx, booking)
	if err != nil {
		return nil, err
	}

	// Load full details for response
	return s.bookingRepo.FindByID(ctx, booking.ID)
}

// GetAllBookings retrieves bookings with details, pagination, and search
func (s *BookingService) GetAllBookings(ctx context.Context, search string, statusFilter string, customerID int, page int, limit int) ([]*domain.Booking, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.bookingRepo.FindAll(ctx, search, statusFilter, customerID, limit, offset)
}

// ConfirmBooking transitions booking status to CONFIRMED and spawns an active Work Order
func (s *BookingService) ConfirmBooking(ctx context.Context, id int, adminUser string) error {
	booking, err := s.bookingRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if booking.OperationalStatus != "PENDING" {
		return errors.New("only PENDING bookings can be confirmed")
	}

	// Update status to CONFIRMED
	err = s.bookingRepo.UpdateStatus(ctx, id, "CONFIRMED", adminUser)
	if err != nil {
		return err
	}

	// Auto spawn WorkOrder in IN_PROGRESS state
	wo := &domain.WorkOrder{
		BookingID:  &id,
		VehicleID:  booking.VehicleID,
		WorkStatus: "IN_PROGRESS",
		Notes:      booking.Complaints,
		CreatedBy:  &adminUser,
		UpdatedBy:  &adminUser,
	}

	err = s.workOrderRepo.Insert(ctx, wo)
	return err
}

// CancelBooking transitions booking status to CANCELLED
func (s *BookingService) CancelBooking(ctx context.Context, id int, adminUser string) error {
	booking, err := s.bookingRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if booking.OperationalStatus != "PENDING" {
		return errors.New("only PENDING bookings can be cancelled")
	}

	return s.bookingRepo.UpdateStatus(ctx, id, "CANCELLED", adminUser)
}

// GetBookedSlots retrieves all booked times for a specific date
func (s *BookingService) GetBookedSlots(ctx context.Context, date string) ([]string, error) {
	if date == "" {
		return nil, errors.New("date is required")
	}
	return s.bookingRepo.GetBookedSlotsByDate(ctx, date)
}
