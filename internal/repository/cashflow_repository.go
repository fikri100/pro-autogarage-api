package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pro-autogarage-api/internal/domain"
)

type CashflowRepository struct {
	db *sql.DB
}

func NewCashflowRepository(db *sql.DB) *CashflowRepository {
	return &CashflowRepository{db: db}
}

// Insert creates a new manual cashflow record
func (r *CashflowRepository) Insert(ctx context.Context, c *domain.Cashflow) error {
	query := `
		INSERT INTO cashflows (cashflow_type, amount, category, description, flow_date, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, status, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		c.CashflowType, c.Amount, c.Category, c.Description, c.FlowDate, c.CreatedBy, c.UpdatedBy,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)

	return err
}

// FindAll retrieves all active cashflow records based on filters
func (r *CashflowRepository) FindAll(ctx context.Context, typeFilter string, categoryFilter string, startDate string, endDate string) ([]*domain.Cashflow, error) {
	query := `
		SELECT id, cashflow_type, amount, category, description, transaction_id, flow_date, status, created_by, created_at, updated_by, updated_at
		FROM cashflows
		WHERE status = 'Y'
	`
	var args []interface{}
	placeholderCount := 1

	if typeFilter != "" {
		query += fmt.Sprintf(" AND cashflow_type = $%d", placeholderCount)
		args = append(args, typeFilter)
		placeholderCount++
	}

	if categoryFilter != "" {
		query += fmt.Sprintf(" AND category = $%d", placeholderCount)
		args = append(args, categoryFilter)
		placeholderCount++
	}

	if startDate != "" && endDate != "" {
		query += fmt.Sprintf(" AND flow_date::date BETWEEN $%d AND $%d", placeholderCount, placeholderCount+1)
		args = append(args, startDate, endDate)
		placeholderCount += 2
	} else if startDate != "" {
		query += fmt.Sprintf(" AND flow_date::date >= $%d", placeholderCount)
		args = append(args, startDate)
		placeholderCount++
	} else if endDate != "" {
		query += fmt.Sprintf(" AND flow_date::date <= $%d", placeholderCount)
		args = append(args, endDate)
		placeholderCount++
	}

	query += " ORDER BY flow_date DESC, id DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Cashflow
	for rows.Next() {
		var c domain.Cashflow
		var desc sql.NullString
		var transID sql.NullInt64
		err := rows.Scan(
			&c.ID, &c.CashflowType, &c.Amount, &c.Category, &desc, &transID,
			&c.FlowDate, &c.Status, &c.CreatedBy, &c.CreatedAt, &c.UpdatedBy, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if desc.Valid {
			c.Description = &desc.String
		}
		if transID.Valid {
			idVal := int(transID.Int64)
			c.TransactionID = &idVal
		}
		list = append(list, &c)
	}

	return list, nil
}

// SoftDelete soft deletes a manual cashflow entry
func (r *CashflowRepository) SoftDelete(ctx context.Context, id int, updatedBy string) error {
	query := `
		UPDATE cashflows
		SET status = 'N', updated_by = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND status = 'Y' AND transaction_id IS NULL
	` // Only allow manual entries (no associated transactions) to be deleted
	res, err := r.db.ExecContext(ctx, query, updatedBy, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("cashflow entry not found or cannot delete transaction-linked cashflow")
	}
	return nil
}

// GetSummary calculates total income, expenses, and Gross Profit
func (r *CashflowRepository) GetSummary(ctx context.Context) (*domain.FinanceSummary, error) {
	summary := &domain.FinanceSummary{}

	// 1. Get Income and Expense sums
	cashQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN cashflow_type = 'INC' THEN amount END), 0) as total_income,
			COALESCE(SUM(CASE WHEN cashflow_type = 'EXP' THEN amount END), 0) as total_expense
		FROM cashflows
		WHERE status = 'Y'
	`
	err := r.db.QueryRowContext(ctx, cashQuery).Scan(&summary.TotalIncome, &summary.TotalExpense)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch income/expense: %w", err)
	}
	summary.NetCashflow = summary.TotalIncome - summary.TotalExpense

	// 2. Get detailed service & sparepart revenue & COGS for Gross Profit
	detailsQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN p.item_type = 'SRV' THEN td.quantity * td.price_at_transaction END), 0) as total_jasa,
			COALESCE(SUM(CASE WHEN p.item_type = 'SPR' THEN td.quantity * td.price_at_transaction END), 0) as total_sparepart_sales,
			COALESCE(SUM(CASE WHEN p.item_type = 'SPR' THEN td.quantity * p.purchase_price END), 0) as total_sparepart_cogs
		FROM transaction_details td
		JOIN transactions t ON td.transaction_id = t.id
		JOIN products p ON td.product_id = p.id
		WHERE t.payment_status = 'PAID' AND t.status = 'Y' AND td.status = 'Y'
	`
	err = r.db.QueryRowContext(ctx, detailsQuery).Scan(
		&summary.TotalServiceRevenue,
		&summary.TotalSparepartSales,
		&summary.TotalSparepartCOGS,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch operational profit details: %w", err)
	}

	// Laba Kotor = Total Jasa + (Total Sparepart Sales - Total Sparepart COGS)
	summary.GrossProfit = summary.TotalServiceRevenue + (summary.TotalSparepartSales - summary.TotalSparepartCOGS)

	return summary, nil
}

// GetChartData returns grouped income & expense for charting
func (r *CashflowRepository) GetChartData(ctx context.Context, period string) ([]*domain.FinanceChartItem, error) {
	var query string
	if period == "monthly" {
		query = `
			SELECT 
				TO_CHAR(flow_date, 'YYYY-MM') as label,
				COALESCE(SUM(CASE WHEN cashflow_type = 'INC' THEN amount END), 0) as income,
				COALESCE(SUM(CASE WHEN cashflow_type = 'EXP' THEN amount END), 0) as expense
			FROM cashflows
			WHERE status = 'Y' AND flow_date >= CURRENT_DATE - INTERVAL '12 months'
			GROUP BY TO_CHAR(flow_date, 'YYYY-MM')
			ORDER BY label ASC
		`
	} else { // daily
		query = `
			SELECT 
				TO_CHAR(flow_date, 'YYYY-MM-DD') as label,
				COALESCE(SUM(CASE WHEN cashflow_type = 'INC' THEN amount END), 0) as income,
				COALESCE(SUM(CASE WHEN cashflow_type = 'EXP' THEN amount END), 0) as expense
			FROM cashflows
			WHERE status = 'Y' AND flow_date >= CURRENT_DATE - INTERVAL '30 days'
			GROUP BY TO_CHAR(flow_date, 'YYYY-MM-DD')
			ORDER BY label ASC
		`
	}

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.FinanceChartItem
	for rows.Next() {
		var item domain.FinanceChartItem
		if err := rows.Scan(&item.Label, &item.Income, &item.Expense); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}

	// If empty, return an empty array instead of null
	if list == nil {
		list = []*domain.FinanceChartItem{}
	}

	return list, nil
}
