package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pro-autogarage-api/internal/domain"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// GenerateInvoiceNumber creates an invoice number: INV-YYYYMMDD-XXXX
func (r *TransactionRepository) GenerateInvoiceNumber(ctx context.Context) (string, error) {
	todayStr := time.Now().Format("20060102")
	prefix := "INV-" + todayStr + "-"

	// Count number of invoices today to get the next sequential number
	query := `SELECT COUNT(1) FROM transactions WHERE invoice_number LIKE $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, prefix+"%").Scan(&count)
	if err != nil {
		return "", err
	}

	seqNumber := fmt.Sprintf("%04d", count+1)
	return prefix + seqNumber, nil
}

// CreateUnpaidTransaction creates a draft invoice with estimation items
func (r *TransactionRepository) CreateUnpaidTransaction(ctx context.Context, woID int, details []*domain.TransactionDetail, createdBy string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Check if a transaction for this work order already exists
	var transID int
	var invoiceNumber string
	var exists bool

	checkQuery := `SELECT id, invoice_number FROM transactions WHERE work_order_id = $1 AND status = 'Y'`
	err = tx.QueryRowContext(ctx, checkQuery, woID).Scan(&transID, &invoiceNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			exists = false
		} else {
			return err
		}
	} else {
		exists = true
	}

	if !exists {
		// Generate invoice number
		todayStr := time.Now().Format("20060102")
		prefix := "INV-" + todayStr + "-"
		var count int
		countQuery := `SELECT COUNT(1) FROM transactions WHERE invoice_number LIKE $1`
		err = tx.QueryRowContext(ctx, countQuery, prefix+"%").Scan(&count)
		if err != nil {
			return err
		}
		invoiceNumber = prefix + fmt.Sprintf("%04d", count+1)

		var unpaidStatusID int
		_ = tx.QueryRowContext(ctx, "SELECT id FROM params WHERE group_param = 'PAYMENT_STATUS' AND kode_param = 'UNPAID' LIMIT 1").Scan(&unpaidStatusID)
		if unpaidStatusID == 0 {
			err = tx.QueryRowContext(ctx, "INSERT INTO params (group_param, kode_param, nama_param, created_by) VALUES ('PAYMENT_STATUS', 'UNPAID', 'Belum Dibayar', 'system') RETURNING id").Scan(&unpaidStatusID)
			if err != nil {
				return fmt.Errorf("failed to resolve UNPAID payment status param: %w", err)
			}
		}

		// Insert unpaid transaction
		insertQuery := `
			INSERT INTO transactions (work_order_id, invoice_number, total_amount, payment_status_id, created_by, updated_by)
			VALUES ($1, $2, 0, $3, $4, $5)
			RETURNING id
		`
		err = tx.QueryRowContext(ctx, insertQuery, woID, invoiceNumber, unpaidStatusID, createdBy, createdBy).Scan(&transID)
		if err != nil {
			return err
		}
	} else {
		// Delete old transaction details for updating
		deleteDetailsQuery := `DELETE FROM transaction_details WHERE transaction_id = $1`
		_, err = tx.ExecContext(ctx, deleteDetailsQuery, transID)
		if err != nil {
			return err
		}
	}

	// 2. Insert new transaction details
	insertDetailQuery := `
		INSERT INTO transaction_details (transaction_id, product_id, quantity, price_at_transaction, subtotal, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var totalAmount float64
	for _, d := range details {
		subtotal := float64(d.Quantity) * d.PriceAtTransaction
		totalAmount += subtotal

		_, err = tx.ExecContext(ctx, insertDetailQuery,
			transID, d.ProductID, d.Quantity, d.PriceAtTransaction, subtotal, createdBy, createdBy,
		)
		if err != nil {
			return err
		}
	}

	// 3. Update total amount in transaction
	updateTotalQuery := `UPDATE transactions SET total_amount = $1, updated_by = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
	_, err = tx.ExecContext(ctx, updateTotalQuery, totalAmount, createdBy, transID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetTransactionByWO gets transaction and items by WO ID
func (r *TransactionRepository) GetTransactionByWO(ctx context.Context, woID int) (*domain.Transaction, []*domain.TransactionDetail, error) {
	query := `
		SELECT t.id, t.work_order_id, t.invoice_number, t.total_amount, COALESCE(t.payment_status_id, 0), COALESCE(par.kode_param, ''), t.transaction_date, t.created_by, t.created_at
		FROM transactions t
		LEFT JOIN params par ON t.payment_status_id = par.id
		WHERE t.work_order_id = $1 AND t.status = 'Y'
	`
	var t domain.Transaction
	var payMethod sql.NullString
	err := r.db.QueryRowContext(ctx, query, woID).Scan(
		&t.ID, &t.WorkOrderID, &t.InvoiceNumber, &t.TotalAmount, &t.PaymentStatusID, &t.PaymentStatus, &t.TransactionDate, &t.CreatedBy, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errors.New("no draft invoice found for this work order")
		}
		return nil, nil, err
	}

	if payMethod.Valid {
		t.PaymentMethod = &payMethod.String
	}

	// Fetch details
	detailsQuery := `
		SELECT 
			td.id, td.transaction_id, td.product_id, p.code, p.name, COALESCE(par.kode_param, ''), p.category, p.purchase_price,
			td.quantity, td.price_at_transaction, td.subtotal, td.created_by, td.created_at
		FROM transaction_details td
		JOIN products p ON td.product_id = p.id
		LEFT JOIN params par ON p.item_type_id = par.id
		WHERE td.transaction_id = $1 AND td.status = 'Y'
	`
	rows, err := r.db.QueryContext(ctx, detailsQuery, t.ID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var details []*domain.TransactionDetail
	for rows.Next() {
		var d domain.TransactionDetail
		if err := rows.Scan(
			&d.ID, &d.TransactionID, &d.ProductID, &d.ProductCode, &d.ProductName, &d.ProductType, &d.ProductCategory, &d.PurchasePrice,
			&d.Quantity, &d.PriceAtTransaction, &d.Subtotal, &d.CreatedBy, &d.CreatedAt,
		); err != nil {
			return nil, nil, err
		}
		details = append(details, &d)
	}

	return &t, details, nil
}

// FindAllReadyForCashier gets COMPLETED work orders that do not have PAID transactions with search and pagination
func (r *TransactionRepository) FindAllReadyForCashier(ctx context.Context, search string, limit int, offset int) ([]*domain.WorkOrder, int, error) {
	// First, get the total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM work_orders wo
		JOIN vehicles v ON wo.vehicle_id = v.id
		JOIN customers c ON v.customer_id = c.id
		LEFT JOIN employees e ON wo.mechanic_id = e.id
		LEFT JOIN params par_wo ON wo.work_status_id = par_wo.id
		WHERE wo.status = 'Y' AND (par_wo.kode_param = 'COMPLETED' OR wo.work_status_id IS NULL)
		AND wo.id NOT IN (
			SELECT work_order_id FROM transactions t LEFT JOIN params par ON t.payment_status_id = par.id WHERE COALESCE(par.kode_param, '') = 'PAID' AND t.status = 'Y'
		)
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
			COALESCE(wo.work_status_id, 0), COALESCE(par_wo.kode_param, ''), wo.notes, wo.status, wo.created_by, wo.created_at
		FROM work_orders wo
		JOIN vehicles v ON wo.vehicle_id = v.id
		JOIN customers c ON v.customer_id = c.id
		LEFT JOIN employees e ON wo.mechanic_id = e.id
		LEFT JOIN params par_wo ON wo.work_status_id = par_wo.id
		WHERE wo.status = 'Y' AND (par_wo.kode_param = 'COMPLETED' OR wo.work_status_id IS NULL)
		AND wo.id NOT IN (
			SELECT work_order_id FROM transactions t LEFT JOIN params par ON t.payment_status_id = par.id WHERE COALESCE(par.kode_param, '') = 'PAID' AND t.status = 'Y'
		)
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
		if err := rows.Scan(
			&wo.ID, &wo.BookingID, &wo.VehicleID, &wo.LicensePlate, &wo.VehicleBrand, &wo.VehicleModel,
			&wo.CustomerID, &wo.CustomerName, &wo.MechanicID, &mechName, &wo.StartTime, &wo.EndTime,
			&wo.WorkStatusID, &wo.WorkStatus, &wo.Notes, &wo.Status, &wo.CreatedBy, &wo.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		if mechName.Valid {
			wo.MechanicName = &mechName.String
		}
		wos = append(wos, &wo)
	}
	return wos, total, nil
}

// FinalizePaymentTx performs the cashier finalized checkout in an atomic database transaction
func (r *TransactionRepository) FinalizePaymentTx(ctx context.Context, transID int, paymentMethodID int, paymentMethod string, discount float64, tax float64, total float64, details []*domain.TransactionDetail, createdBy string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Get Work Order ID and Invoice Number
	var woID int
	var invoiceNumber string
	selectTransQuery := `SELECT t.work_order_id, t.invoice_number FROM transactions t LEFT JOIN params par ON t.payment_status_id = par.id WHERE t.id = $1 AND (COALESCE(par.kode_param, '') = 'UNPAID' OR t.payment_status_id IS NULL)`
	err = tx.QueryRowContext(ctx, selectTransQuery, transID).Scan(&woID, &invoiceNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("invoice already paid or does not exist")
		}
		return err
	}

	// 2. Clear out existing transaction details so we sync with final items
	deleteDetailsQuery := `DELETE FROM transaction_details WHERE transaction_id = $1`
	_, err = tx.ExecContext(ctx, deleteDetailsQuery, transID)
	if err != nil {
		return err
	}

	// 3. Insert finalized transaction details
	insertDetailQuery := `
		INSERT INTO transaction_details (transaction_id, product_id, quantity, price_at_transaction, subtotal, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	for _, d := range details {
		subtotal := float64(d.Quantity) * d.PriceAtTransaction
		_, err = tx.ExecContext(ctx, insertDetailQuery,
			transID, d.ProductID, d.Quantity, d.PriceAtTransaction, subtotal, createdBy, createdBy,
		)
		if err != nil {
			return err
		}

		// 4. Reduce stock if Sparepart (SPR)
		var itemType string
		var currentStock int
		var code string
		selectProdQuery := `SELECT COALESCE(par.kode_param, ''), p.stock_quantity, p.code FROM products p LEFT JOIN params par ON p.item_type_id = par.id WHERE p.id = $1 AND p.status = 'Y' FOR UPDATE OF p`
		err = tx.QueryRowContext(ctx, selectProdQuery, d.ProductID).Scan(&itemType, &currentStock, &code)
		if err != nil {
			return err
		}

		if itemType == "SPR" {
			newStock := currentStock - d.Quantity
			if newStock < 0 {
				return fmt.Errorf("stok tidak cukup untuk produk %s. Stok fisik: %d, diminta: %d", code, currentStock, d.Quantity)
			}

			// Update stock_quantity in database
			updateStockQuery := `UPDATE products SET stock_quantity = $1, updated_by = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
			_, err = tx.ExecContext(ctx, updateStockQuery, newStock, createdBy, d.ProductID)
			if err != nil {
				return err
			}

			// Insert stock log
			insertLogQuery := `
				INSERT INTO stock_logs (product_id, log_type, quantity, reference_id, created_by, updated_by)
				VALUES ($1, 'OUT', $2, $3, $4, $5)
			`
			_, err = tx.ExecContext(ctx, insertLogQuery, d.ProductID, d.Quantity, invoiceNumber, createdBy, createdBy)
			if err != nil {
				return err
			}
		}
	}

	var paidStatusID int
	_ = tx.QueryRowContext(ctx, "SELECT id FROM params WHERE group_param = 'PAYMENT_STATUS' AND kode_param = 'PAID' LIMIT 1").Scan(&paidStatusID)

	var incCashflowID int
	_ = tx.QueryRowContext(ctx, "SELECT id FROM params WHERE group_param = 'CASHFLOW_TYPE' AND kode_param = 'INC' LIMIT 1").Scan(&incCashflowID)

	var paidWOStatusID int
	_ = tx.QueryRowContext(ctx, "SELECT id FROM params WHERE group_param = 'WORK_ORDER_STATUS' AND kode_param = 'PAID' LIMIT 1").Scan(&paidWOStatusID)

	pmID := paymentMethodID
	if pmID == 0 && paymentMethod != "" {
		_ = tx.QueryRowContext(ctx, "SELECT id FROM params WHERE group_param = 'PAYMENT_METHOD' AND (kode_param = $1 OR nama_param = $1 OR id::text = $1) AND status = 'Y' LIMIT 1", paymentMethod).Scan(&pmID)
	}

	// 5. Update transaction columns: paid status, discount, tax, total, method_id, and timestamps
	updateTransQuery := `
		UPDATE transactions
		SET payment_status_id = $1, payment_method_id = NULLIF($2, 0), total_amount = $3, transaction_date = CURRENT_TIMESTAMP, updated_by = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`
	_, err = tx.ExecContext(ctx, updateTransQuery, paidStatusID, pmID, total, createdBy, transID)
	if err != nil {
		return err
	}

	// 6. Record revenue to cashflows
	insertCashQuery := `
		INSERT INTO cashflows (cashflow_type_id, amount, category, description, transaction_id, created_by, updated_by)
		VALUES ($1, $2, 'SERVICE', $3, $4, $5, $6)
	`
	desc := "Pendapatan servis kendaraan dari Invoice " + invoiceNumber
	_, err = tx.ExecContext(ctx, insertCashQuery, incCashflowID, total, desc, transID, createdBy, createdBy)
	if err != nil {
		return err
	}

	// 7. Finalize Work Order status to PAID
	updateWOQuery := `
		UPDATE work_orders
		SET work_status_id = $1, end_time = CURRENT_TIMESTAMP, updated_by = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err = tx.ExecContext(ctx, updateWOQuery, paidWOStatusID, createdBy, woID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
