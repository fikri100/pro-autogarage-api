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

// RestockProductTx processes product restocking in an atomic database transaction
func (r *ProductRepository) RestockProductTx(ctx context.Context, req domain.RestockRequest, creator string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Fetch current product details and lock row for update
	var itemType string
	var currentStock int
	var currentHPP float64
	var prodName string

	query := `
		SELECT item_type, stock_quantity, purchase_price, name 
		FROM products 
		WHERE id = $1 AND status = 'Y' 
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, query, req.ProductID).Scan(&itemType, &currentStock, &currentHPP, &prodName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("produk tidak ditemukan")
		}
		return err
	}

	if itemType != "SPR" {
		return errors.New("hanya produk bertipe SPAREPART (SPR) yang dapat di-restock")
	}

	// 2. Calculate new Moving Average HPP
	var newHPP float64
	newStock := currentStock + req.Quantity

	if newStock > 0 {
		// MA Formula: ((Q_curr * HPP_curr) + (Q_new * Price_new)) / (Q_curr + Q_new)
		totalCost := (float64(currentStock) * currentHPP) + (float64(req.Quantity) * req.PurchasePrice)
		newHPP = totalCost / float64(newStock)
	} else {
		newHPP = req.PurchasePrice
	}

	// 3. Update products table
	updateQuery := `
		UPDATE products
		SET stock_quantity = $1, purchase_price = $2, updated_by = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`
	_, err = tx.ExecContext(ctx, updateQuery, newStock, newHPP, creator, req.ProductID)
	if err != nil {
		return err
	}

	// 4. Insert Stock Log ('IN')
	logQuery := `
		INSERT INTO stock_logs (product_id, log_type, quantity, reference_id, created_by, updated_by)
		VALUES ($1, 'IN', $2, $3, $4, $5)
	`
	_, err = tx.ExecContext(ctx, logQuery, req.ProductID, req.Quantity, req.ReferenceID, creator, creator)
	if err != nil {
		return err
	}

	// 5. Optionally record expense in cashflows
	if req.RecordExpense {
		totalExpense := float64(req.Quantity) * req.PurchasePrice
		desc := fmt.Sprintf("Pembelian Restock: %s (%d pcs @ Rp %.0f) - Ref: %s", prodName, req.Quantity, req.PurchasePrice, req.ReferenceID)
		
		cashflowQuery := `
			INSERT INTO cashflows (cashflow_type, amount, category, description, created_by, updated_by)
			VALUES ('EXP', $1, 'STOCK', $2, $3, $4)
		`
		_, err = tx.ExecContext(ctx, cashflowQuery, totalExpense, desc, creator, creator)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// FindStockLogsByProductID retrieves the list of stock logs for a product
func (r *ProductRepository) FindStockLogsByProductID(ctx context.Context, prodID int) ([]*domain.StockLog, error) {
	query := `
		SELECT id, product_id, log_type, quantity, COALESCE(reference_id, ''), status, COALESCE(created_by, ''), created_at, COALESCE(updated_by, ''), updated_at
		FROM stock_logs
		WHERE product_id = $1 AND status = 'Y'
		ORDER BY id DESC
	`
	rows, err := r.db.QueryContext(ctx, query, prodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.StockLog
	for rows.Next() {
		var l domain.StockLog
		err := rows.Scan(
			&l.ID, &l.ProductID, &l.LogType, &l.Quantity, &l.ReferenceID,
			&l.Status, &l.CreatedBy, &l.CreatedAt, &l.UpdatedBy, &l.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}

	if logs == nil {
		logs = []*domain.StockLog{}
	}

	return logs, nil
}

// FindAllCategories retrieves all active categories
func (r *ProductRepository) FindAllCategories(ctx context.Context) ([]domain.Category, error) {
	query := "SELECT id, name FROM categories WHERE status = 'Y' ORDER BY id ASC"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []domain.Category{}
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

// InsertCategory inserts a new category record
func (r *ProductRepository) InsertCategory(ctx context.Context, name string) (int, error) {
	query := `
		INSERT INTO categories (name, created_by, updated_by)
		VALUES ($1, 'admin', 'admin')
		RETURNING id
	`
	var id int
	err := r.db.QueryRowContext(ctx, query, name).Scan(&id)
	return id, err
}

// UpdateCategory updates an existing category name
func (r *ProductRepository) UpdateCategory(ctx context.Context, id int, name string) error {
	query := `
		UPDATE categories 
		SET name = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND status = 'Y'
	`
	_, err := r.db.ExecContext(ctx, query, name, id)
	return err
}

// DeleteCategory soft deletes a category by setting status to 'N'
func (r *ProductRepository) DeleteCategory(ctx context.Context, id int) error {
	query := `
		UPDATE categories 
		SET status = 'N', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
