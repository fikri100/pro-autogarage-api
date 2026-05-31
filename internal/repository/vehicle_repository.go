package repository

import (
	"context"
	"database/sql"
	"pro-autogarage-api/internal/domain"
)

type VehicleRepository struct {
	db *sql.DB
}

func NewVehicleRepository(db *sql.DB) *VehicleRepository {
	return &VehicleRepository{db: db}
}

func (r *VehicleRepository) FindAll(ctx context.Context, customerID int) ([]domain.Vehicle, error) {
	var rows *sql.Rows
	var err error
	if customerID > 0 {
		rows, err = r.db.QueryContext(ctx, "SELECT id, customer_id, license_plate, brand, model, year_made, transmission FROM vehicles WHERE status = 'Y' AND customer_id = $1 ORDER BY id DESC", customerID)
	} else {
		rows, err = r.db.QueryContext(ctx, "SELECT id, customer_id, license_plate, brand, model, year_made, transmission FROM vehicles WHERE status = 'Y' ORDER BY id DESC")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []domain.Vehicle
	for rows.Next() {
		var v domain.Vehicle
		var brand, model, trans sql.NullString
		var year sql.NullInt32
		if err := rows.Scan(&v.ID, &v.CustomerID, &v.LicensePlate, &brand, &model, &year, &trans); err != nil {
			return nil, err
		}
		if brand.Valid {
			v.Brand = brand.String
		}
		if model.Valid {
			v.Model = model.String
		}
		if year.Valid {
			v.YearMade = int(year.Int32)
		}
		if trans.Valid {
			v.Transmission = trans.String
		}
		vehicles = append(vehicles, v)
	}
	if vehicles == nil {
		vehicles = []domain.Vehicle{}
	}
	return vehicles, nil
}

func (r *VehicleRepository) Insert(ctx context.Context, v *domain.VehicleRequest) (int, error) {
	query := `
		INSERT INTO vehicles (customer_id, license_plate, brand, model, year_made, transmission, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, 'admin')
		RETURNING id
	`
	var id int
	err := r.db.QueryRowContext(ctx, query, v.CustomerID, v.LicensePlate, v.Brand, v.Model, v.YearMade, v.Transmission).Scan(&id)
	return id, err
}

func (r *VehicleRepository) Update(ctx context.Context, id int, v *domain.UpdateVehicleRequest) error {
	query := `
		UPDATE vehicles 
		SET license_plate = $1, brand = $2, model = $3, year_made = $4, transmission = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query, v.LicensePlate, v.Brand, v.Model, v.YearMade, v.Transmission, id)
	return err
}

func (r *VehicleRepository) Delete(ctx context.Context, id int) error {
	query := `
		UPDATE vehicles 
		SET status = 'N', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
