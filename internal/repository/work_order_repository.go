package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"pro-autogarage-api/internal/domain"
)

type WorkOrderRepository struct {
	db *sql.DB
}

func NewWorkOrderRepository(db *sql.DB) *WorkOrderRepository {
	return &WorkOrderRepository{db: db}
}

// Insert creates a new work order
func (r *WorkOrderRepository) Insert(ctx context.Context, wo *domain.WorkOrder) error {
	query := `
		INSERT INTO work_orders (booking_id, vehicle_id, mechanic_id, work_status, notes, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, status, start_time, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		wo.BookingID, wo.VehicleID, wo.MechanicID, wo.WorkStatus, wo.Notes, wo.CreatedBy, wo.UpdatedBy,
	).Scan(&wo.ID, &wo.Status, &wo.StartTime, &wo.CreatedAt, &wo.UpdatedAt)

	return err
}

// FindAllActive retrieves active work orders (not paid) with search and pagination
func (r *WorkOrderRepository) FindAllActive(ctx context.Context, search string, limit int, offset int) ([]*domain.WorkOrder, int, error) {
	// First, get the total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM work_orders wo
		JOIN vehicles v ON wo.vehicle_id = v.id
		JOIN customers c ON v.customer_id = c.id
		LEFT JOIN employees e ON wo.mechanic_id = e.id
		WHERE wo.status = 'Y' AND wo.work_status <> 'PAID'
	`
	var countArgs []interface{}
	placeholderCount := 1

	searchParam := "%" + search + "%"
	countQuery += fmt.Sprintf(" AND (v.license_plate ILIKE $%d OR c.name ILIKE $%d OR e.name ILIKE $%d)", placeholderCount, placeholderCount, placeholderCount)
	countArgs = append(countArgs, searchParam)
	placeholderCount++

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Then, get the paginated data
	query := `
		SELECT 
			wo.id, wo.booking_id, wo.vehicle_id, v.license_plate, v.brand, v.model,
			c.id, c.name, wo.mechanic_id, e.name, wo.start_time, wo.end_time,
			wo.estimated_minutes, wo.estimated_completion,
			wo.work_status, wo.notes, wo.status, wo.created_by, wo.created_at, wo.updated_by, wo.updated_at
		FROM work_orders wo
		JOIN vehicles v ON wo.vehicle_id = v.id
		JOIN customers c ON v.customer_id = c.id
		LEFT JOIN employees e ON wo.mechanic_id = e.id
		WHERE wo.status = 'Y' AND wo.work_status <> 'PAID'
	`
	var args []interface{}
	placeholderCount = 1

	query += fmt.Sprintf(" AND (v.license_plate ILIKE $%d OR c.name ILIKE $%d OR e.name ILIKE $%d)", placeholderCount, placeholderCount, placeholderCount)
	args = append(args, searchParam)
	placeholderCount++

	query += fmt.Sprintf(" ORDER BY wo.id DESC LIMIT $%d OFFSET $%d", placeholderCount, placeholderCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var wos []*domain.WorkOrder
	for rows.Next() {
		var wo domain.WorkOrder
		var mechName sql.NullString
		var estimatedMinutes sql.NullInt64
		var estimatedCompletion sql.NullTime
		if err := rows.Scan(
			&wo.ID, &wo.BookingID, &wo.VehicleID, &wo.LicensePlate, &wo.VehicleBrand, &wo.VehicleModel,
			&wo.CustomerID, &wo.CustomerName, &wo.MechanicID, &mechName, &wo.StartTime, &wo.EndTime,
			&estimatedMinutes, &estimatedCompletion,
			&wo.WorkStatus, &wo.Notes, &wo.Status, &wo.CreatedBy, &wo.CreatedAt, &wo.UpdatedBy, &wo.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		if mechName.Valid {
			wo.MechanicName = &mechName.String
		}
		if estimatedMinutes.Valid {
			v := int(estimatedMinutes.Int64)
			wo.EstimatedMinutes = &v
		}
		if estimatedCompletion.Valid {
			wo.EstimatedCompletion = &estimatedCompletion.Time
		}
		wos = append(wos, &wo)
	}
	return wos, total, nil
}

// FindByID retrieves a specific work order by ID
func (r *WorkOrderRepository) FindByID(ctx context.Context, id int) (*domain.WorkOrder, error) {
	query := `
		SELECT 
			wo.id, wo.booking_id, wo.vehicle_id, v.license_plate, v.brand, v.model,
			c.id, c.name, wo.mechanic_id, e.name, wo.start_time, wo.end_time,
			wo.estimated_minutes, wo.estimated_completion,
			wo.work_status, wo.notes, wo.status, wo.created_by, wo.created_at, wo.updated_by, wo.updated_at
		FROM work_orders wo
		JOIN vehicles v ON wo.vehicle_id = v.id
		JOIN customers c ON v.customer_id = c.id
		LEFT JOIN employees e ON wo.mechanic_id = e.id
		WHERE wo.id = $1 AND wo.status = 'Y'
	`
	var wo domain.WorkOrder
	var mechName sql.NullString
	var estimatedMinutes sql.NullInt64
	var estimatedCompletion sql.NullTime
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&wo.ID, &wo.BookingID, &wo.VehicleID, &wo.LicensePlate, &wo.VehicleBrand, &wo.VehicleModel,
		&wo.CustomerID, &wo.CustomerName, &wo.MechanicID, &mechName, &wo.StartTime, &wo.EndTime,
		&estimatedMinutes, &estimatedCompletion,
		&wo.WorkStatus, &wo.Notes, &wo.Status, &wo.CreatedBy, &wo.CreatedAt, &wo.UpdatedBy, &wo.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("work order not found")
		}
		return nil, err
	}

	if mechName.Valid {
		wo.MechanicName = &mechName.String
	}
	if estimatedMinutes.Valid {
		v := int(estimatedMinutes.Int64)
		wo.EstimatedMinutes = &v
	}
	if estimatedCompletion.Valid {
		wo.EstimatedCompletion = &estimatedCompletion.Time
	}
	return &wo, nil
}

// UpdateStatus changes work_status of a work order
func (r *WorkOrderRepository) UpdateStatus(ctx context.Context, id int, status string, updatedBy string) error {
	query := `
		UPDATE work_orders
		SET work_status = $1, updated_by = $2, updated_at = CURRENT_TIMESTAMP
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
		return errors.New("work order not found")
	}
	return nil
}

// AssignMechanic updates mechanic and optionally notes on a work order
func (r *WorkOrderRepository) AssignMechanic(ctx context.Context, id int, mechanicID *int, notes *string, status string, updatedBy string) error {
	query := `
		UPDATE work_orders
		SET mechanic_id = $1, notes = $2, work_status = $3, updated_by = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND status = 'Y'
	`
	res, err := r.db.ExecContext(ctx, query, mechanicID, notes, status, updatedBy, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("work order not found")
	}
	return nil
}

// UpdateEstimation sets estimated_minutes and auto-calculates estimated_completion from start_time
func (r *WorkOrderRepository) UpdateEstimation(ctx context.Context, id int, estimatedMinutes int, updatedBy string) error {
	query := `
		UPDATE work_orders
		SET 
			estimated_minutes    = $1,
			estimated_completion = start_time + ($1::integer * INTERVAL '1 minute'),
			updated_by           = $2,
			updated_at           = CURRENT_TIMESTAMP
		WHERE id = $3 AND status = 'Y'
	`
	res, err := r.db.ExecContext(ctx, query, estimatedMinutes, updatedBy, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("work order not found")
	}
	return nil
}
