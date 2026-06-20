package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// FindAll retrieves bookings with joins to customers and vehicles with pagination and search
func (r *BookingRepository) FindAll(ctx context.Context, search string, statusFilter string, customerID int, limit int, offset int) ([]*domain.Booking, int, error) {
	// First, get the total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM bookings b
		JOIN customers c ON b.customer_id = c.id
		JOIN vehicles v ON b.vehicle_id = v.id
		WHERE b.status = 'Y'
	`
	var countArgs []interface{}
	placeholderCount := 1

	if customerID > 0 {
		countQuery += fmt.Sprintf(" AND b.customer_id = $%d", placeholderCount)
		countArgs = append(countArgs, customerID)
		placeholderCount++
	}

	searchParam := "%" + search + "%"
	countQuery += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.phone ILIKE $%d OR v.license_plate ILIKE $%d)", placeholderCount, placeholderCount, placeholderCount)
	countArgs = append(countArgs, searchParam)
	placeholderCount++

	if statusFilter != "" {
		countQuery += fmt.Sprintf(" AND b.operational_status = $%d", placeholderCount)
		countArgs = append(countArgs, statusFilter)
		placeholderCount++
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Then, get the paginated data
	query := `
		SELECT 
			b.id, b.customer_id, c.name, c.phone, b.vehicle_id, v.license_plate, v.brand, v.model,
			b.booking_date, b.booking_time, b.complaints, b.operational_status, b.status,
			b.created_by, b.created_at, b.updated_by, b.updated_at,
			v.customer_id AS vehicle_customer_id, vc.name AS vehicle_owner_name
		FROM bookings b
		JOIN customers c ON b.customer_id = c.id
		JOIN vehicles v ON b.vehicle_id = v.id
		JOIN customers vc ON v.customer_id = vc.id
		WHERE b.status = 'Y'
	`
	var args []interface{}
	placeholderCount = 1

	if customerID > 0 {
		query += fmt.Sprintf(" AND b.customer_id = $%d", placeholderCount)
		args = append(args, customerID)
		placeholderCount++
	}

	query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.phone ILIKE $%d OR v.license_plate ILIKE $%d)", placeholderCount, placeholderCount, placeholderCount)
	args = append(args, searchParam)
	placeholderCount++

	if statusFilter != "" {
		query += fmt.Sprintf(" AND b.operational_status = $%d", placeholderCount)
		args = append(args, statusFilter)
		placeholderCount++
	}
	
	// Dynamic priority order: PENDING first, then date DESC, time DESC
	query += fmt.Sprintf(" ORDER BY CASE WHEN b.operational_status = 'PENDING' THEN 0 ELSE 1 END ASC, b.booking_date DESC, b.booking_time DESC LIMIT $%d OFFSET $%d", placeholderCount, placeholderCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
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
			&b.VehicleCustomerID, &b.VehicleOwnerName,
		); err != nil {
			return nil, 0, err
		}

		b.BookingDate = bDate.Format("2006-01-02")
		b.BookingTime = bTime[:5] // Get HH:MM
		bookings = append(bookings, &b)
	}

	return bookings, total, nil
}

// FindByID retrieves a booking by ID
func (r *BookingRepository) FindByID(ctx context.Context, id int) (*domain.Booking, error) {
	query := `
		SELECT 
			b.id, b.customer_id, c.name, c.phone, b.vehicle_id, v.license_plate, v.brand, v.model,
			b.booking_date, b.booking_time, b.complaints, b.operational_status, b.status,
			b.created_by, b.created_at, b.updated_by, b.updated_at,
			v.customer_id AS vehicle_customer_id, vc.name AS vehicle_owner_name
		FROM bookings b
		JOIN customers c ON b.customer_id = c.id
		JOIN vehicles v ON b.vehicle_id = v.id
		JOIN customers vc ON v.customer_id = vc.id
		WHERE b.id = $1 AND b.status = 'Y'
	`
	var b domain.Booking
	var bDate time.Time
	var bTime string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.CustomerID, &b.CustomerName, &b.CustomerPhone, &b.VehicleID, &b.LicensePlate, &b.VehicleBrand, &b.VehicleModel,
		&bDate, &bTime, &b.Complaints, &b.OperationalStatus, &b.Status,
		&b.CreatedBy, &b.CreatedAt, &b.UpdatedBy, &b.UpdatedAt,
		&b.VehicleCustomerID, &b.VehicleOwnerName,
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
