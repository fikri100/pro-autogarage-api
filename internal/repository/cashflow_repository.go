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
	if c.CashflowTypeID == 0 && c.CashflowType != "" {
		_ = r.db.QueryRowContext(ctx, "SELECT id FROM params WHERE group_param = 'CASHFLOW_TYPE' AND (kode_param = $1 OR nama_param = $1) AND status = 'Y' LIMIT 1", c.CashflowType).Scan(&c.CashflowTypeID)
	}

	query := `
		INSERT INTO cashflows (cashflow_type_id, amount, category, description, flow_date, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, status, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		c.CashflowTypeID, c.Amount, c.Category, c.Description, c.FlowDate, c.CreatedBy, c.UpdatedBy,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)

	return err
}

// FindAll retrieves active cashflow records based on filters with pagination and search
func (r *CashflowRepository) FindAll(ctx context.Context, typeFilter string, categoryFilter string, startDate string, endDate string, search string, limit int, offset int) ([]*domain.Cashflow, int, error) {
	// First, get the total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM cashflows c
		LEFT JOIN params par ON c.cashflow_type_id = par.id
		WHERE c.status = 'Y'
	`
	var countArgs []interface{}
	placeholderCount := 1

	if typeFilter != "" {
		countQuery += fmt.Sprintf(" AND (par.kode_param = $%d OR par.nama_param = $%d)", placeholderCount, placeholderCount)
		countArgs = append(countArgs, typeFilter)
		placeholderCount++
	}

	if categoryFilter != "" {
		countQuery += fmt.Sprintf(" AND c.category = $%d", placeholderCount)
		countArgs = append(countArgs, categoryFilter)
		placeholderCount++
	}

	if startDate != "" && endDate != "" {
		countQuery += fmt.Sprintf(" AND c.flow_date::date BETWEEN $%d AND $%d", placeholderCount, placeholderCount+1)
		countArgs = append(countArgs, startDate, endDate)
		placeholderCount += 2
	} else if startDate != "" {
		countQuery += fmt.Sprintf(" AND c.flow_date::date >= $%d", placeholderCount)
		countArgs = append(countArgs, startDate)
		placeholderCount++
	} else if endDate != "" {
		countQuery += fmt.Sprintf(" AND c.flow_date::date <= $%d", placeholderCount)
		countArgs = append(countArgs, endDate)
		placeholderCount++
	}

	searchParam := "%" + search + "%"
	countQuery += fmt.Sprintf(" AND (c.description ILIKE $%d OR c.category ILIKE $%d)", placeholderCount, placeholderCount)
	countArgs = append(countArgs, searchParam)
	placeholderCount++

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Then, get the paginated data
	query := `
		SELECT c.id, COALESCE(c.cashflow_type_id, 0), COALESCE(par.kode_param, ''), c.amount, c.category, c.description, c.transaction_id, c.flow_date, c.status, c.created_by, c.created_at, c.updated_by, c.updated_at
		FROM cashflows c
		LEFT JOIN params par ON c.cashflow_type_id = par.id
		WHERE c.status = 'Y'
	`
	var args []interface{}
	placeholderCount = 1

	if typeFilter != "" {
		query += fmt.Sprintf(" AND (par.kode_param = $%d OR par.nama_param = $%d)", placeholderCount, placeholderCount)
		args = append(args, typeFilter)
		placeholderCount++
	}

	if categoryFilter != "" {
		query += fmt.Sprintf(" AND c.category = $%d", placeholderCount)
		args = append(args, categoryFilter)
		placeholderCount++
	}

	if startDate != "" && endDate != "" {
		query += fmt.Sprintf(" AND c.flow_date::date BETWEEN $%d AND $%d", placeholderCount, placeholderCount+1)
		args = append(args, startDate, endDate)
		placeholderCount += 2
	} else if startDate != "" {
		query += fmt.Sprintf(" AND c.flow_date::date >= $%d", placeholderCount)
		args = append(args, startDate)
		placeholderCount++
	} else if endDate != "" {
		query += fmt.Sprintf(" AND c.flow_date::date <= $%d", placeholderCount)
		args = append(args, endDate)
		placeholderCount++
	}

	query += fmt.Sprintf(" AND (c.description ILIKE $%d OR c.category ILIKE $%d)", placeholderCount, placeholderCount)
	args = append(args, searchParam)
	placeholderCount++

	query += fmt.Sprintf(" ORDER BY c.flow_date DESC, c.id DESC LIMIT $%d OFFSET $%d", placeholderCount, placeholderCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*domain.Cashflow
	for rows.Next() {
		var c domain.Cashflow
		var desc sql.NullString
		var transID sql.NullInt64
		err := rows.Scan(
			&c.ID, &c.CashflowTypeID, &c.CashflowType, &c.Amount, &c.Category, &desc, &transID,
			&c.FlowDate, &c.Status, &c.CreatedBy, &c.CreatedAt, &c.UpdatedBy, &c.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
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

	return list, total, nil
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
			COALESCE(SUM(CASE WHEN par.kode_param = 'INC' THEN c.amount END), 0) as total_income,
			COALESCE(SUM(CASE WHEN par.kode_param = 'EXP' THEN c.amount END), 0) as total_expense
		FROM cashflows c
		LEFT JOIN params par ON c.cashflow_type_id = par.id
		WHERE c.status = 'Y'
	`
	err := r.db.QueryRowContext(ctx, cashQuery).Scan(&summary.TotalIncome, &summary.TotalExpense)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch income/expense: %w", err)
	}
	summary.NetCashflow = summary.TotalIncome - summary.TotalExpense

	// 2. Get detailed service & sparepart revenue & COGS for Gross Profit
	detailsQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN par_item.kode_param = 'SRV' THEN td.quantity * td.price_at_transaction END), 0) as total_jasa,
			COALESCE(SUM(CASE WHEN par_item.kode_param = 'SPR' THEN td.quantity * td.price_at_transaction END), 0) as total_sparepart_sales,
			COALESCE(SUM(CASE WHEN par_item.kode_param = 'SPR' THEN td.quantity * p.purchase_price END), 0) as total_sparepart_cogs
		FROM transaction_details td
		JOIN transactions t ON td.transaction_id = t.id
		JOIN products p ON td.product_id = p.id
		LEFT JOIN params par_item ON p.item_type_id = par_item.id
		LEFT JOIN params par_pay ON t.payment_status_id = par_pay.id
		WHERE (par_pay.kode_param = 'PAID' OR t.payment_status_id IS NULL) AND t.status = 'Y' AND td.status = 'Y'
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
				TO_CHAR(c.flow_date, 'YYYY-MM') as label,
				COALESCE(SUM(CASE WHEN par.kode_param = 'INC' THEN c.amount END), 0) as income,
				COALESCE(SUM(CASE WHEN par.kode_param = 'EXP' THEN c.amount END), 0) as expense
			FROM cashflows c
			LEFT JOIN params par ON c.cashflow_type_id = par.id
			WHERE c.status = 'Y' AND c.flow_date >= CURRENT_DATE - INTERVAL '12 months'
			GROUP BY TO_CHAR(c.flow_date, 'YYYY-MM')
			ORDER BY label ASC
		`
	} else { // daily
		query = `
			SELECT 
				TO_CHAR(c.flow_date, 'YYYY-MM-DD') as label,
				COALESCE(SUM(CASE WHEN par.kode_param = 'INC' THEN c.amount END), 0) as income,
				COALESCE(SUM(CASE WHEN par.kode_param = 'EXP' THEN c.amount END), 0) as expense
			FROM cashflows c
			LEFT JOIN params par ON c.cashflow_type_id = par.id
			WHERE c.status = 'Y' AND c.flow_date >= CURRENT_DATE - INTERVAL '30 days'
			GROUP BY TO_CHAR(c.flow_date, 'YYYY-MM-DD')
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
