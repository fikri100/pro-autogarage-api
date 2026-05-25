package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"pro-autogarage-api/internal/domain"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// Insert creates a new product or service record
func (r *ProductRepository) Insert(ctx context.Context, p *domain.Product) error {
	query := `
		INSERT INTO products (code, name, item_type, category, purchase_price, sale_price, stock_quantity, min_stock_limit, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, status, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		p.Code, p.Name, p.ItemType, p.Category, p.PurchasePrice, p.SalePrice, p.StockQuantity, p.MinStockLimit, p.CreatedBy, p.UpdatedBy,
	).Scan(&p.ID, &p.Status, &p.CreatedAt, &p.UpdatedAt)

	return err
}

// FindAll retrieves products with optional dynamic filters
func (r *ProductRepository) FindAll(ctx context.Context, search string, itemType string, lowStock bool) ([]*domain.Product, error) {
	query := `
		SELECT id, code, name, item_type, category, purchase_price, sale_price, stock_quantity, min_stock_limit, status, created_by, created_at, updated_by, updated_at
		FROM products
		WHERE status = 'Y'
	`
	var args []interface{}
	placeholderIdx := 1

	if search != "" {
		query += fmt.Sprintf(" AND (code ILIKE $%d OR name ILIKE $%d OR category ILIKE $%d)", placeholderIdx, placeholderIdx, placeholderIdx)
		args = append(args, "%"+search+"%")
		placeholderIdx++
	}

	if itemType != "" {
		query += fmt.Sprintf(" AND item_type = $%d", placeholderIdx)
		args = append(args, itemType)
		placeholderIdx++
	}

	if lowStock {
		query += " AND item_type = 'SPR' AND stock_quantity <= min_stock_limit"
	}

	query += " ORDER BY id DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(
			&p.ID, &p.Code, &p.Name, &p.ItemType, &p.Category,
			&p.PurchasePrice, &p.SalePrice, &p.StockQuantity, &p.MinStockLimit,
			&p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedBy, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		products = append(products, &p)
	}
	return products, nil
}

// FindByID retrieves a product by ID
func (r *ProductRepository) FindByID(ctx context.Context, id int) (*domain.Product, error) {
	query := `
		SELECT id, code, name, item_type, category, purchase_price, sale_price, stock_quantity, min_stock_limit, status, created_by, created_at, updated_by, updated_at
		FROM products
		WHERE id = $1 AND status = 'Y'
	`
	var p domain.Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Code, &p.Name, &p.ItemType, &p.Category,
		&p.PurchasePrice, &p.SalePrice, &p.StockQuantity, &p.MinStockLimit,
		&p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedBy, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	return &p, nil
}

// IsCodeExists checks if a product code already exists (excluding a specific ID for updates)
func (r *ProductRepository) IsCodeExists(ctx context.Context, code string, excludeID int) (bool, error) {
	query := `
		SELECT COUNT(1) FROM products
		WHERE code = $1 AND id <> $2 AND status = 'Y'
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, code, excludeID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Update modifies an existing product record
func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	query := `
		UPDATE products
		SET code = $1, name = $2, item_type = $3, category = $4, purchase_price = $5, sale_price = $6, stock_quantity = $7, min_stock_limit = $8, updated_by = $9, updated_at = CURRENT_TIMESTAMP
		WHERE id = $10 AND status = 'Y'
	`
	res, err := r.db.ExecContext(ctx, query,
		p.Code, p.Name, p.ItemType, p.Category, p.PurchasePrice, p.SalePrice, p.StockQuantity, p.MinStockLimit, p.UpdatedBy, p.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("product not found or no changes made")
	}
	return nil
}

// SoftDelete soft deletes a product by setting status to 'N'
func (r *ProductRepository) SoftDelete(ctx context.Context, id int, updatedBy string) error {
	query := `
		UPDATE products
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
		return errors.New("product not found")
	}
	return nil
}
