package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// NewPostgresDB initializes the database connection pool
func NewPostgresDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("[OK] Berhasil terhubung ke database PostgreSQL!")
	
	// Auto migrate schema for param_id columns
	migrateSchema(db)

	return db, nil
}

func migrateSchema(db *sql.DB) {
	queries := []string{
		// 1. Ensure essential status params exist
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'WORK_ORDER_STATUS', 'IN_PROGRESS', 'IN_PROGRESS' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'WORK_ORDER_STATUS' AND kode_param = 'IN_PROGRESS');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'WORK_ORDER_STATUS', 'COMPLETED', 'COMPLETED' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'WORK_ORDER_STATUS' AND kode_param = 'COMPLETED');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'WORK_ORDER_STATUS', 'PAID', 'PAID' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'WORK_ORDER_STATUS' AND kode_param = 'PAID');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'WORK_ORDER_STATUS', 'CANCELLED', 'CANCELLED' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'WORK_ORDER_STATUS' AND kode_param = 'CANCELLED');`,
		`UPDATE params SET nama_param = kode_param WHERE group_param = 'WORK_ORDER_STATUS';`,

		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'PAYMENT_STATUS', 'UNPAID', 'Belum Dibayar' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'PAYMENT_STATUS' AND kode_param = 'UNPAID');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'PAYMENT_STATUS', 'PAID', 'Sudah Dibayar' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'PAYMENT_STATUS' AND kode_param = 'PAID');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'PAYMENT_STATUS', 'CANCELLED', 'Dibatalkan' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'PAYMENT_STATUS' AND kode_param = 'CANCELLED');`,

		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'BOOKING_STATUS', 'PENDING', 'PENDING' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'BOOKING_STATUS' AND kode_param = 'PENDING');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'BOOKING_STATUS', 'CONFIRMED', 'CONFIRMED' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'BOOKING_STATUS' AND kode_param = 'CONFIRMED');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'BOOKING_STATUS', 'IN_PROGRESS', 'IN_PROGRESS' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'BOOKING_STATUS' AND kode_param = 'IN_PROGRESS');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'BOOKING_STATUS', 'COMPLETED', 'COMPLETED' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'BOOKING_STATUS' AND kode_param = 'COMPLETED');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'BOOKING_STATUS', 'CANCELLED', 'CANCELLED' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'BOOKING_STATUS' AND kode_param = 'CANCELLED');`,
		`UPDATE params SET nama_param = kode_param WHERE group_param = 'BOOKING_STATUS';`,

		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'CASHFLOW_TYPE', 'INC', 'Pemasukan' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'CASHFLOW_TYPE' AND kode_param = 'INC');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'CASHFLOW_TYPE', 'EXP', 'Pengeluaran' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'CASHFLOW_TYPE' AND kode_param = 'EXP');`,

		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'ITEM_TYPE', 'SPR', 'Sparepart' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'ITEM_TYPE' AND kode_param = 'SPR');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'ITEM_TYPE', 'SRV', 'Jasa' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'ITEM_TYPE' AND kode_param = 'SRV');`,

		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'VEHICLE_TRANSMISSION', 'MANUAL', 'Manual' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'VEHICLE_TRANSMISSION' AND kode_param = 'MANUAL');`,
		`INSERT INTO params (group_param, kode_param, nama_param) 
		 SELECT 'VEHICLE_TRANSMISSION', 'AUTOMATIC', 'Automatic' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'VEHICLE_TRANSMISSION' AND kode_param = 'AUTOMATIC');`,

		// 2. ALTER bookings table
		`DO $$ 
		BEGIN 
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='bookings' AND column_name='operational_status') THEN
				ALTER TABLE bookings RENAME COLUMN operational_status TO old_operational_status;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='bookings' AND column_name='operational_status_id') THEN
				ALTER TABLE bookings ADD COLUMN operational_status_id INT REFERENCES params(id);
			END IF;
		END $$;`,

		`DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='bookings' AND column_name='old_operational_status') THEN
				UPDATE bookings b SET operational_status_id = p.id FROM params p WHERE p.group_param = 'BOOKING_STATUS' AND (p.kode_param = b.old_operational_status OR p.nama_param = b.old_operational_status) AND b.operational_status_id IS NULL;
			END IF;
		END $$;`,

		// 3. ALTER products table
		`DO $$ 
		BEGIN 
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='products' AND column_name='item_type') THEN
				ALTER TABLE products RENAME COLUMN item_type TO old_item_type;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='products' AND column_name='item_type_id') THEN
				ALTER TABLE products ADD COLUMN item_type_id INT REFERENCES params(id);
			END IF;
		END $$;`,

		`DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='products' AND column_name='old_item_type') THEN
				UPDATE products prod SET item_type_id = p.id FROM params p WHERE p.group_param = 'ITEM_TYPE' AND (p.kode_param = prod.old_item_type OR p.nama_param = prod.old_item_type) AND prod.item_type_id IS NULL;
			END IF;
		END $$;`,

		// 4. ALTER cashflows table
		`DO $$ 
		BEGIN 
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='cashflows' AND column_name='cashflow_type') THEN
				ALTER TABLE cashflows RENAME COLUMN cashflow_type TO old_cashflow_type;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='cashflows' AND column_name='cashflow_type_id') THEN
				ALTER TABLE cashflows ADD COLUMN cashflow_type_id INT REFERENCES params(id);
			END IF;
		END $$;`,

		`DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='cashflows' AND column_name='old_cashflow_type') THEN
				UPDATE cashflows c SET cashflow_type_id = p.id FROM params p WHERE p.group_param = 'CASHFLOW_TYPE' AND (p.kode_param = c.old_cashflow_type OR p.nama_param = c.old_cashflow_type) AND c.cashflow_type_id IS NULL;
			END IF;
		END $$;`,

		// 5. ALTER transactions table
		`DO $$ 
		BEGIN 
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='transactions' AND column_name='payment_status') THEN
				ALTER TABLE transactions RENAME COLUMN payment_status TO old_payment_status;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='transactions' AND column_name='payment_status_id') THEN
				ALTER TABLE transactions ADD COLUMN payment_status_id INT REFERENCES params(id);
			END IF;
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='transactions' AND column_name='payment_method') THEN
				ALTER TABLE transactions RENAME COLUMN payment_method TO old_payment_method;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='transactions' AND column_name='payment_method_id') THEN
				ALTER TABLE transactions ADD COLUMN payment_method_id INT REFERENCES params(id);
			END IF;
		END $$;`,

		`DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='transactions' AND column_name='old_payment_status') THEN
				UPDATE transactions t SET payment_status_id = p.id FROM params p WHERE p.group_param = 'PAYMENT_STATUS' AND (p.kode_param = t.old_payment_status OR p.nama_param = t.old_payment_status) AND t.payment_status_id IS NULL;
			END IF;
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='transactions' AND column_name='old_payment_method') THEN
				UPDATE transactions t SET payment_method_id = p.id FROM params p WHERE p.group_param = 'PAYMENT_METHOD' AND (p.kode_param = t.old_payment_method OR p.nama_param = t.old_payment_method OR t.old_payment_method ILIKE '%' || p.nama_param || '%') AND t.payment_method_id IS NULL;
			END IF;
		END $$;`,

		// 6. ALTER work_orders table
		`DO $$ 
		BEGIN 
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='work_orders' AND column_name='work_status') THEN
				ALTER TABLE work_orders RENAME COLUMN work_status TO old_work_status;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='work_orders' AND column_name='work_status_id') THEN
				ALTER TABLE work_orders ADD COLUMN work_status_id INT REFERENCES params(id);
			END IF;
		END $$;`,

		`DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='work_orders' AND column_name='old_work_status') THEN
				UPDATE work_orders wo SET work_status_id = p.id FROM params p WHERE p.group_param = 'WORK_ORDER_STATUS' AND (p.kode_param = wo.old_work_status OR p.nama_param = wo.old_work_status) AND wo.work_status_id IS NULL;
			END IF;
		END $$;`,

		// 7. Seed EMPLOYEE_POSITION params
		`INSERT INTO params (group_param, kode_param, nama_param)
		 SELECT 'EMPLOYEE_POSITION', 'LEAD_MECHANIC', 'Lead Mechanic' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'EMPLOYEE_POSITION' AND kode_param = 'LEAD_MECHANIC');`,
		`INSERT INTO params (group_param, kode_param, nama_param)
		 SELECT 'EMPLOYEE_POSITION', 'MECHANIC', 'Mekanik' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'EMPLOYEE_POSITION' AND kode_param = 'MECHANIC');`,
		`INSERT INTO params (group_param, kode_param, nama_param)
		 SELECT 'EMPLOYEE_POSITION', 'SERVICE_ADVISOR', 'Service Advisor' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'EMPLOYEE_POSITION' AND kode_param = 'SERVICE_ADVISOR');`,
		`INSERT INTO params (group_param, kode_param, nama_param)
		 SELECT 'EMPLOYEE_POSITION', 'CASHIER', 'Kasir' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'EMPLOYEE_POSITION' AND kode_param = 'CASHIER');`,
		`INSERT INTO params (group_param, kode_param, nama_param)
		 SELECT 'EMPLOYEE_POSITION', 'MANAGER', 'Manajer' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'EMPLOYEE_POSITION' AND kode_param = 'MANAGER');`,
		`INSERT INTO params (group_param, kode_param, nama_param)
		 SELECT 'EMPLOYEE_POSITION', 'ADMIN', 'Admin' WHERE NOT EXISTS (SELECT 1 FROM params WHERE group_param = 'EMPLOYEE_POSITION' AND kode_param = 'ADMIN');`,

		// 8. ALTER employees table
		`DO $$ 
		BEGIN 
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='employees' AND column_name='position') THEN
				ALTER TABLE employees RENAME COLUMN position TO old_position;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='employees' AND column_name='position_id') THEN
				ALTER TABLE employees ADD COLUMN position_id INT REFERENCES params(id);
			END IF;
		END $$;`,

		`DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='employees' AND column_name='old_position') THEN
				UPDATE employees e SET position_id = p.id FROM params p WHERE p.group_param = 'EMPLOYEE_POSITION' AND (p.kode_param = e.old_position OR p.nama_param = e.old_position OR e.old_position ILIKE '%' || p.nama_param || '%') AND e.position_id IS NULL;
			END IF;
		END $$;`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			log.Printf("[MIGRATION WARNING] %v", err)
		}
	}
	log.Println("[OK] Auto-migrasi skema param_id ke database PostgreSQL berhasil!")
}
