package repository

import (
	"context"
	"database/sql"
	"errors"
	"pro-autogarage-api/internal/domain"
)

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

// Insert creates a new customer record
func (r *CustomerRepository) Insert(ctx context.Context, c *domain.Customer) error {
	query := `
		INSERT INTO customers (name, phone, address, email, username, is_self_service, password, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, status, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		c.Name, c.Phone, c.Address, c.Email, c.Username, c.IsSelfService, c.Password, c.CreatedBy, c.UpdatedBy,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)

	return err
}

// FindAll retrieves active customers with pagination and search
func (r *CustomerRepository) FindAll(ctx context.Context, search string, limit int, offset int) ([]*domain.Customer, int, error) {
	// First, get the total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM customers 
		WHERE status = 'Y' AND (name ILIKE $1 OR phone ILIKE $1)
	`
	var total int
	searchParam := "%" + search + "%"
	err := r.db.QueryRowContext(ctx, countQuery, searchParam).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Then, get the paginated data
	query := `
		SELECT id, name, phone, address, email, username, is_self_service, status, created_by, created_at, updated_by, updated_at
		FROM customers
		WHERE status = 'Y' AND (name ILIKE $1 OR phone ILIKE $1)
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, searchParam, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var customers []*domain.Customer
	for rows.Next() {
		var c domain.Customer
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Phone, &c.Address, &c.Email, &c.Username, &c.IsSelfService,
			&c.Status, &c.CreatedBy, &c.CreatedAt, &c.UpdatedBy, &c.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		customers = append(customers, &c)
	}
	return customers, total, nil
}

// FindByID retrieves a specific customer by ID
func (r *CustomerRepository) FindByID(ctx context.Context, id int) (*domain.Customer, error) {
	query := `
		SELECT id, name, phone, address, email, username, is_self_service, status, created_by, created_at, updated_by, updated_at
		FROM customers
		WHERE id = $1 AND status = 'Y'
	`
	var c domain.Customer
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Phone, &c.Address, &c.Email, &c.Username, &c.IsSelfService,
		&c.Status, &c.CreatedBy, &c.CreatedAt, &c.UpdatedBy, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}
	return &c, nil
}

// Update modifies an existing customer
func (r *CustomerRepository) Update(ctx context.Context, c *domain.Customer) error {
	query := `
		UPDATE customers
		SET name = $1, phone = $2, address = $3, email = $4, username = $5, is_self_service = $6, updated_by = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND status = 'Y'
	`
	res, err := r.db.ExecContext(ctx, query,
		c.Name, c.Phone, c.Address, c.Email, c.Username, c.IsSelfService, c.UpdatedBy, c.ID,
	)
	if err != nil {
		return err
	}
	
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("customer not found or no changes made")
	}
	return nil
}

// SoftDelete sets the customer status to 'N'
func (r *CustomerRepository) SoftDelete(ctx context.Context, id int, updatedBy string) error {
	query := `
		UPDATE customers
		SET status = 'N', updated_by = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND status = 'Y'
	`
	res, err := r.db.ExecContext(ctx, query, updatedBy, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("customer not found")
	}
	return nil
}
