package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"pro-autogarage-api/internal/domain"
)

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// Insert creates a new booking record
func (r *BookingRepository) Insert(ctx context.Context, b *domain.Booking) error {
	query := `
		INSERT INTO bookings (customer_id, vehicle_id, booking_date, booking_time, complaints, operational_status, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, status, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		b.CustomerID, b.VehicleID, b.BookingDate, b.BookingTime, b.Complaints, b.OperationalStatus, b.CreatedBy, b.UpdatedBy,
	).Scan(&b.ID, &b.Status, &b.CreatedAt, &b.UpdatedAt)

	return err
}

// FindAll retrieves bookings with joins to customers and vehicles
func (r *BookingRepository) FindAll(ctx context.Context, statusFilter string) ([]*domain.Booking, error) {
	query := `
		SELECT 
			b.id, b.customer_id, c.name, c.phone, b.vehicle_id, v.license_plate, v.brand, v.model,
			b.booking_date, b.booking_time, b.complaints, b.operational_status, b.status,
			b.created_by, b.created_at, b.updated_by, b.updated_at
		FROM bookings b
		JOIN customers c ON b.customer_id = c.id
		JOIN vehicles v ON b.vehicle_id = v.id
		WHERE b.status = 'Y'
	`
	var args []interface{}
	if statusFilter != "" {
		query += " AND b.operational_status = $1"
		args = append(args, statusFilter)
	}
	query += " ORDER BY b.booking_date DESC, b.booking_time DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*domain.Booking
	for rows.Next() {
		var b domain.Booking
		var bDate time.Time
		var bTime string // time in Postgres

		if err := rows.Scan(
			&b.ID, &b.CustomerID, &b.CustomerName, &b.CustomerPhone, &b.VehicleID, &b.LicensePlate, &b.VehicleBrand, &b.VehicleModel,
			&bDate, &bTime, &b.Complaints, &b.OperationalStatus, &b.Status,
			&b.CreatedBy, &b.CreatedAt, &b.UpdatedBy, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}

		b.BookingDate = bDate.Format("2006-01-02")
		b.BookingTime = bTime[:5] // Get HH:MM
		bookings = append(bookings, &b)
	}

	return bookings, nil
}

// FindByID retrieves a booking by ID
func (r *BookingRepository) FindByID(ctx context.Context, id int) (*domain.Booking, error) {
	query := `
		SELECT 
			b.id, b.customer_id, c.name, c.phone, b.vehicle_id, v.license_plate, v.brand, v.model,
			b.booking_date, b.booking_time, b.complaints, b.operational_status, b.status,
			b.created_by, b.created_at, b.updated_by, b.updated_at
		FROM bookings b
		JOIN customers c ON b.customer_id = c.id
		JOIN vehicles v ON b.vehicle_id = v.id
		WHERE b.id = $1 AND b.status = 'Y'
	`
	var b domain.Booking
	var bDate time.Time
	var bTime string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.CustomerID, &b.CustomerName, &b.CustomerPhone, &b.VehicleID, &b.LicensePlate, &b.VehicleBrand, &b.VehicleModel,
		&bDate, &bTime, &b.Complaints, &b.OperationalStatus, &b.Status,
		&b.CreatedBy, &b.CreatedAt, &b.UpdatedBy, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("booking not found")
		}
		return nil, err
	}

	b.BookingDate = bDate.Format("2006-01-02")
	b.BookingTime = bTime[:5]
	return &b, nil
}

// UpdateStatus changes the operational status of a booking
func (r *BookingRepository) UpdateStatus(ctx context.Context, id int, status string, updatedBy string) error {
	query := `
		UPDATE bookings
		SET operational_status = $1, updated_by = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND status = 'Y'
	`
	res, err := r.db.ExecContext(ctx, query, status, updatedBy, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("booking not found")
	}
	return nil
}
