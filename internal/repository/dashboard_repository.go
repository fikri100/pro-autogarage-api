package repository

import (
	"context"
	"database/sql"
	"time"

	"pro-autogarage-api/internal/domain"
)

type DashboardRepository struct {
	db *sql.DB
}

func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// GetStats calculates KPI stats for dashboard summary cards
func (r *DashboardRepository) GetStats(ctx context.Context) (domain.DashboardStats, error) {
	var stats domain.DashboardStats

	// 1. Total Customers
	custQuery := `SELECT COUNT(1) FROM customers WHERE status = 'Y'`
	err := r.db.QueryRowContext(ctx, custQuery).Scan(&stats.TotalCustomers)
	if err != nil {
		return stats, err
	}

	// 2. Active Work Orders
	woQuery := `SELECT COUNT(1) FROM work_orders wo LEFT JOIN params p ON wo.work_status_id = p.id WHERE wo.status = 'Y' AND COALESCE(p.kode_param, '') IN ('IN_PROGRESS', 'COMPLETED')`
	err = r.db.QueryRowContext(ctx, woQuery).Scan(&stats.ActiveWorkOrders)
	if err != nil {
		return stats, err
	}

	// 3. Today's Revenue
	revQuery := `
		SELECT COALESCE(SUM(t.total_amount), 0)
		FROM transactions t
		LEFT JOIN params p ON t.payment_status_id = p.id
		WHERE t.status = 'Y' AND COALESCE(p.kode_param, '') = 'PAID' AND t.transaction_date::date = CURRENT_DATE
	`
	err = r.db.QueryRowContext(ctx, revQuery).Scan(&stats.TodayRevenue)
	if err != nil {
		return stats, err
	}

	// 4. Pending Bookings
	bookQuery := `SELECT COUNT(1) FROM bookings b LEFT JOIN params p ON b.operational_status_id = p.id WHERE b.status = 'Y' AND COALESCE(p.kode_param, '') = 'PENDING'`
	err = r.db.QueryRowContext(ctx, bookQuery).Scan(&stats.PendingBookings)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

// GetRecentBookings fetches bookings waiting for verification
func (r *DashboardRepository) GetRecentBookings(ctx context.Context, limit int) ([]*domain.Booking, error) {
	query := `
		SELECT 
			b.id, b.customer_id, c.name, c.phone, b.vehicle_id, v.license_plate, v.brand, v.model,
			b.booking_date, b.booking_time, b.complaints, COALESCE(b.operational_status_id, 0), COALESCE(p.kode_param, ''), b.status,
			b.created_by, b.created_at, b.updated_by, b.updated_at
		FROM bookings b
		JOIN customers c ON b.customer_id = c.id
		JOIN vehicles v ON b.vehicle_id = v.id
		LEFT JOIN params p ON b.operational_status_id = p.id
		WHERE b.status = 'Y' AND COALESCE(p.kode_param, '') = 'PENDING'
		ORDER BY b.booking_date ASC, b.booking_time ASC
		LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*domain.Booking
	for rows.Next() {
		var b domain.Booking
		var bDate time.Time
		var bTime string

		err := rows.Scan(
			&b.ID, &b.CustomerID, &b.CustomerName, &b.CustomerPhone, &b.VehicleID, &b.LicensePlate, &b.VehicleBrand, &b.VehicleModel,
			&bDate, &bTime, &b.Complaints, &b.OperationalStatusID, &b.OperationalStatus, &b.Status,
			&b.CreatedBy, &b.CreatedAt, &b.UpdatedBy, &b.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		b.BookingDate = bDate.Format("2006-01-02")
		b.BookingTime = bTime[:5]
		bookings = append(bookings, &b)
	}

	if bookings == nil {
		bookings = []*domain.Booking{}
	}

	return bookings, nil
}

// GetActiveWorkOrders fetches active work orders (not paid yet)
func (r *DashboardRepository) GetActiveWorkOrders(ctx context.Context) ([]*domain.WorkOrder, error) {
	query := `
		SELECT 
			wo.id, wo.booking_id, wo.vehicle_id, v.license_plate, v.brand, v.model,
			c.id, c.name, wo.mechanic_id, e.name, wo.start_time, wo.end_time,
			COALESCE(wo.work_status_id, 0), COALESCE(p.kode_param, ''), wo.notes, wo.status, wo.created_by, wo.created_at, wo.updated_by, wo.updated_at
		FROM work_orders wo
		JOIN vehicles v ON wo.vehicle_id = v.id
		JOIN customers c ON v.customer_id = c.id
		LEFT JOIN employees e ON wo.mechanic_id = e.id
		LEFT JOIN params p ON wo.work_status_id = p.id
		WHERE wo.status = 'Y' AND COALESCE(p.kode_param, '') IN ('IN_PROGRESS', 'COMPLETED')
		ORDER BY wo.id DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wos []*domain.WorkOrder
	for rows.Next() {
		var wo domain.WorkOrder
		var mechName sql.NullString
		err := rows.Scan(
			&wo.ID, &wo.BookingID, &wo.VehicleID, &wo.LicensePlate, &wo.VehicleBrand, &wo.VehicleModel,
			&wo.CustomerID, &wo.CustomerName, &wo.MechanicID, &mechName, &wo.StartTime, &wo.EndTime,
			&wo.WorkStatusID, &wo.WorkStatus, &wo.Notes, &wo.Status, &wo.CreatedBy, &wo.CreatedAt, &wo.UpdatedBy, &wo.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if mechName.Valid {
			wo.MechanicName = &mechName.String
		}
		wos = append(wos, &wo)
	}

	if wos == nil {
		wos = []*domain.WorkOrder{}
	}

	return wos, nil
}
