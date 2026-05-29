package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	
	"pro-autogarage-api/internal/handler"
	"pro-autogarage-api/internal/repository"
	"pro-autogarage-api/internal/service"
	"pro-autogarage-api/pkg/database"
	"pro-autogarage-api/pkg/middleware"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type MenuItem struct {
	ID         int        `json:"id"`
	Label      string     `json:"label"`
	Icon       string     `json:"icon"`
	RouterLink string     `json:"routerLink"`
	ParentID   *int       `json:"parentId,omitempty"`
	Children   []MenuItem `json:"children"`
	IsOpen     bool       `json:"isOpen"`
}

func getMenusForRole(db *sql.DB, roleID int) ([]MenuItem, error) {
	query := `
		SELECT m.id, m.label, COALESCE(m.icon, ''), COALESCE(m.router_link, ''), m.parent_id
		FROM menus m
		JOIN role_menus rm ON m.id = rm.menu_id
		WHERE rm.role_id = $1 AND m.status = 'Y'
		ORDER BY m.parent_id ASC NULLS FIRST, m.sort_order ASC, m.id ASC
	`
	rows, err := db.Query(query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type FlatMenu struct {
		ID         int
		Label      string
		Icon       string
		RouterLink string
		ParentID   *int
	}

	var flatMenus []FlatMenu
	for rows.Next() {
		var fm FlatMenu
		var parentID sql.NullInt64
		if err := rows.Scan(&fm.ID, &fm.Label, &fm.Icon, &fm.RouterLink, &parentID); err != nil {
			return nil, err
		}
		if parentID.Valid {
			val := int(parentID.Int64)
			fm.ParentID = &val
		}
		flatMenus = append(flatMenus, fm)
	}

	menuMap := make(map[int]*MenuItem)
	var rootMenus []MenuItem

	for _, fm := range flatMenus {
		menuMap[fm.ID] = &MenuItem{
			ID:         fm.ID,
			Label:      fm.Label,
			Icon:       fm.Icon,
			RouterLink: fm.RouterLink,
			ParentID:   fm.ParentID,
			Children:   []MenuItem{},
			IsOpen:     false,
		}
	}

	for _, fm := range flatMenus {
		item := menuMap[fm.ID]
		if fm.ParentID == nil {
			rootMenus = append(rootMenus, *item)
		} else {
			parentItem, exists := menuMap[*fm.ParentID]
			if exists {
				parentItem.Children = append(parentItem.Children, *item)
			}
		}
	}

	var finalRootMenus []MenuItem
	for _, rm := range rootMenus {
		finalRootMenus = append(finalRootMenus, *menuMap[rm.ID])
	}

	if finalRootMenus == nil {
		finalRootMenus = []MenuItem{}
	}

	return finalRootMenus, nil
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found or error reading it")
	}

	// Initialize Database Connection
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	db, err := database.NewPostgresDB(dbURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Ensure categories table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL UNIQUE,
			status CHAR(1) DEFAULT 'Y' CHECK (status IN ('Y', 'N')),
			created_by VARCHAR(50) DEFAULT 'system',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_by VARCHAR(50) DEFAULT 'system',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create categories table: %v", err)
	}

	// Ensure menus table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS menus (
			id SERIAL PRIMARY KEY,
			label VARCHAR(100) NOT NULL,
			icon VARCHAR(100),
			router_link VARCHAR(100),
			parent_id INT REFERENCES menus(id) ON DELETE SET NULL,
			sort_order INT DEFAULT 0,
			status CHAR(1) DEFAULT 'Y' CHECK (status IN ('Y', 'N')),
			created_by VARCHAR(50) DEFAULT 'system',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_by VARCHAR(50) DEFAULT 'system',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create menus table: %v", err)
	}

	// Ensure role_menus table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS role_menus (
			role_id INT REFERENCES roles(id) ON DELETE CASCADE,
			menu_id INT REFERENCES menus(id) ON DELETE CASCADE,
			PRIMARY KEY (role_id, menu_id)
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create role_menus table: %v", err)
	}

	// Migrate: add estimated_minutes & estimated_completion to work_orders (idempotent)
	_, err = db.Exec(`
		ALTER TABLE work_orders
			ADD COLUMN IF NOT EXISTS estimated_minutes    INT DEFAULT NULL,
			ADD COLUMN IF NOT EXISTS estimated_completion TIMESTAMPTZ DEFAULT NULL
	`)
	if err != nil {
		log.Fatalf("Failed to migrate work_orders table (estimated fields): %v", err)
	}

	// Auto seed menus if empty
	var menuCount int
	err = db.QueryRow("SELECT COUNT(*) FROM menus").Scan(&menuCount)
	if err != nil {
		log.Fatalf("Failed to check menus count: %v", err)
	}
	if menuCount == 0 {
		log.Println("Seeding menus table dynamically...")
		seedSQL := `
			DO $$
			DECLARE
				v_master_id INT;
			BEGIN
				-- Root menus
				INSERT INTO menus (id, label, icon, router_link, parent_id, sort_order, created_by) 
				VALUES (1, 'Dashboard', 'pi pi-home', '/dashboard', NULL, 1, 'system');
				
				INSERT INTO menus (id, label, icon, router_link, parent_id, sort_order, created_by) 
				VALUES (2, 'Master Data', 'pi pi-database', '', NULL, 2, 'system')
				RETURNING id INTO v_master_id;

				INSERT INTO menus (id, label, icon, router_link, parent_id, sort_order, created_by) 
				VALUES (3, 'Inventory', 'pi pi-box', '/inventory', NULL, 3, 'system');
				
				INSERT INTO menus (id, label, icon, router_link, parent_id, sort_order, created_by) 
				VALUES (4, 'Booking', 'pi pi-calendar', '/booking', NULL, 4, 'system');
				
				INSERT INTO menus (id, label, icon, router_link, parent_id, sort_order, created_by) 
				VALUES (5, 'Work Order', 'pi pi-wrench', '/work-order', NULL, 5, 'system');
				
				INSERT INTO menus (id, label, icon, router_link, parent_id, sort_order, created_by) 
				VALUES (6, 'Cashier', 'pi pi-wallet', '/cashier', NULL, 6, 'system');
				
				INSERT INTO menus (id, label, icon, router_link, parent_id, sort_order, created_by) 
				VALUES (7, 'Cashflow', 'pi pi-money-bill', '/cashflow', NULL, 7, 'system');
				
				INSERT INTO menus (id, label, icon, router_link, parent_id, sort_order, created_by) 
				VALUES (8, 'Reports', 'pi pi-chart-bar', '/reports', NULL, 8, 'system');

				-- Reset sequence to prevent key violation on next inserts
				PERFORM setval('menus_id_seq', 8);

				-- Children of Master Data
				INSERT INTO menus (label, icon, router_link, parent_id, sort_order, created_by) VALUES
				('Pelanggan', 'pi pi-user', '/customers', v_master_id, 1, 'system'),
				('Karyawan', 'pi pi-id-card', '/master/employee', v_master_id, 2, 'system'),
				('Role', 'pi pi-shield', '/master/role', v_master_id, 3, 'system'),
				('Category', 'pi pi-tag', '/master/category', v_master_id, 4, 'system'),
				('User Access', 'pi pi-users', '/user-access', v_master_id, 5, 'system');

				-- Map seeded menus to Super Admin (role_id 1)
				INSERT INTO role_menus (role_id, menu_id)
				SELECT 1, id FROM menus;

				-- Reset sequence to prevent key violation later
				PERFORM setval('menus_id_seq', (SELECT MAX(id) FROM menus));
			END $$;
		`
		_, err = db.Exec(seedSQL)
		if err != nil {
			log.Fatalf("Failed to seed menus: %v", err)
		}
		log.Println("[OK] Berhasil melakukan seeding table menus & role_menus!")
	}


	// Initialize Repositories
	customerRepo := repository.NewCustomerRepository(db)
	userRepo := repository.NewUserRepository(db)
	productRepo := repository.NewProductRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	workOrderRepo := repository.NewWorkOrderRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	cashflowRepo := repository.NewCashflowRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db)

	// Initialize Services
	customerService := service.NewCustomerService(customerRepo)
	userService := service.NewUserService(userRepo)
	productService := service.NewProductService(productRepo)
	bookingService := service.NewBookingService(bookingRepo, workOrderRepo)
	workOrderService := service.NewWorkOrderService(workOrderRepo, transactionRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	cashflowService := service.NewCashflowService(cashflowRepo)
	dashboardService := service.NewDashboardService(dashboardRepo)

	// Initialize Portal & Security Middleware
	portalAuthRateLimiter := middleware.NewIPRateLimiter(15, time.Minute)
	portalGeneralRateLimiter := middleware.NewIPRateLimiter(100, time.Minute)
	waService := service.NewMockWhatsAppService()
	portalService := service.NewPortalService(db, waService, bookingRepo)

	// Initialize Handlers
	customerHandler := handler.NewCustomerHandler(customerService)
	userHandler := handler.NewUserHandler(userService)
	productHandler := handler.NewProductHandler(productService)
	bookingHandler := handler.NewBookingHandler(bookingService)
	workOrderHandler := handler.NewWorkOrderHandler(workOrderService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	cashflowHandler := handler.NewCashflowHandler(cashflowService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	portalHandler := handler.NewPortalHandler(portalService)

	// Initialize standard Go 1.22+ ServeMux
	mux := http.NewServeMux()

	// Base API route
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok", "db_connected": true}`)
	})

	// Customer Routes
	mux.HandleFunc("POST /api/customers", customerHandler.CreateCustomer)
	mux.HandleFunc("GET /api/customers", customerHandler.GetAllCustomers)
	mux.HandleFunc("GET /api/customers/{id}", customerHandler.GetCustomerByID)
	mux.HandleFunc("PUT /api/customers/{id}", customerHandler.UpdateCustomer)
	mux.HandleFunc("DELETE /api/customers/{id}", customerHandler.DeleteCustomer)

	// Vehicle Routes (Full CRUD for Customer Vehicles & Autocomplete)
	mux.HandleFunc("GET /api/vehicles", func(w http.ResponseWriter, r *http.Request) {
		custID := r.URL.Query().Get("customerId")
		var rows *sql.Rows
		var err error
		if custID != "" {
			rows, err = db.QueryContext(r.Context(), "SELECT id, customer_id, license_plate, brand, model, year_made, transmission FROM vehicles WHERE status = 'Y' AND customer_id = $1 ORDER BY id DESC", custID)
		} else {
			rows, err = db.QueryContext(r.Context(), "SELECT id, customer_id, license_plate, brand, model, year_made, transmission FROM vehicles WHERE status = 'Y' ORDER BY id DESC")
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type Vehicle struct {
			ID           int    `json:"id"`
			CustomerID   int    `json:"customerId"`
			LicensePlate string `json:"licensePlate"`
			Brand        string `json:"brand"`
			Model        string `json:"model"`
			YearMade     int    `json:"yearMade"`
			Transmission string `json:"transmission"`
		}

		var vehicles []Vehicle
		for rows.Next() {
			var v Vehicle
			var brand, model, trans sql.NullString
			var year sql.NullInt32
			if err := rows.Scan(&v.ID, &v.CustomerID, &v.LicensePlate, &brand, &model, &year, &trans); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
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

		w.Header().Set("Content-Type", "application/json")
		if vehicles == nil {
			vehicles = []Vehicle{}
		}
		json.NewEncoder(w).Encode(vehicles)
	})

	mux.HandleFunc("POST /api/vehicles", func(w http.ResponseWriter, r *http.Request) {
		type Req struct {
			CustomerID   int    `json:"customerId"`
			LicensePlate string `json:"licensePlate"`
			Brand        string `json:"brand"`
			Model        string `json:"model"`
			YearMade     int    `json:"yearMade"`
			Transmission string `json:"transmission"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		query := `
			INSERT INTO vehicles (customer_id, license_plate, brand, model, year_made, transmission, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, 'admin')
			RETURNING id
		`
		var id int
		err := db.QueryRowContext(r.Context(), query, req.CustomerID, req.LicensePlate, req.Brand, req.Model, req.YearMade, req.Transmission).Scan(&id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id": %d}`, id)
	})

	mux.HandleFunc("PUT /api/vehicles/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		type Req struct {
			LicensePlate string `json:"licensePlate"`
			Brand        string `json:"brand"`
			Model        string `json:"model"`
			YearMade     int    `json:"yearMade"`
			Transmission string `json:"transmission"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		query := `
			UPDATE vehicles 
			SET license_plate = $1, brand = $2, model = $3, year_made = $4, transmission = $5, updated_at = CURRENT_TIMESTAMP
			WHERE id = $6
		`
		_, err := db.ExecContext(r.Context(), query, req.LicensePlate, req.Brand, req.Model, req.YearMade, req.Transmission, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success"}`)
	})

	mux.HandleFunc("DELETE /api/vehicles/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		query := `
			UPDATE vehicles 
			SET status = 'N', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, err := db.ExecContext(r.Context(), query, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success"}`)
	})

	// Product & Inventory Routes
	mux.HandleFunc("POST /api/products", productHandler.CreateProduct)
	mux.HandleFunc("GET /api/products", productHandler.GetAllProducts)
	mux.HandleFunc("POST /api/products/restock", productHandler.RestockProduct)
	mux.HandleFunc("GET /api/products/{id}/stock-logs", productHandler.GetStockLogs)
	mux.HandleFunc("GET /api/products/{id}", productHandler.GetProductByID)
	mux.HandleFunc("PUT /api/products/{id}", productHandler.UpdateProduct)
	mux.HandleFunc("DELETE /api/products/{id}", productHandler.DeleteProduct)

	// Booking Routes
	mux.HandleFunc("POST /api/bookings", bookingHandler.CreateBooking)
	mux.HandleFunc("GET /api/bookings", bookingHandler.GetAllBookings)
	mux.HandleFunc("PUT /api/bookings/{id}/confirm", bookingHandler.ConfirmBooking)
	mux.HandleFunc("PUT /api/bookings/{id}/cancel", bookingHandler.CancelBooking)

	// Portal Customer Self-Service Routes (secured with IP-based rate limiting)
	mux.Handle("POST /api/portal/send-otp", portalAuthRateLimiter.Limit(http.HandlerFunc(portalHandler.SendOTP)))
	mux.Handle("POST /api/portal/verify-otp", portalAuthRateLimiter.Limit(http.HandlerFunc(portalHandler.VerifyOTP)))
	mux.Handle("POST /api/portal/register", portalAuthRateLimiter.Limit(http.HandlerFunc(portalHandler.Register)))
	mux.Handle("POST /api/portal/login", portalAuthRateLimiter.Limit(http.HandlerFunc(portalHandler.Login)))
	mux.Handle("POST /api/portal/bookings", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.CreateBooking)))
	mux.Handle("GET /api/portal/bookings", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.GetBookings)))
	mux.Handle("GET /api/portal/dashboard/summary", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.GetDashboardSummary)))
	mux.Handle("GET /api/portal/vehicles", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.GetVehicles)))
	mux.Handle("POST /api/portal/vehicles", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.AddVehicle)))
	mux.Handle("PUT /api/portal/vehicles/{id}", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.UpdateVehicle)))
	mux.Handle("DELETE /api/portal/vehicles/{id}", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.DeleteVehicle)))
	mux.Handle("GET /api/portal/profile", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.GetProfile)))
	mux.Handle("PUT /api/portal/profile", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.UpdateProfile)))
	mux.Handle("PUT /api/portal/bookings/{id}/cancel", portalGeneralRateLimiter.Limit(http.HandlerFunc(portalHandler.CancelBooking)))

	// Work Order Routes
	mux.HandleFunc("POST /api/work-orders", workOrderHandler.CreateWorkOrder)
	mux.HandleFunc("GET /api/work-orders", workOrderHandler.GetAllActiveWorkOrders)
	mux.HandleFunc("GET /api/work-orders/{id}", workOrderHandler.GetWorkOrderByID)
	mux.HandleFunc("PUT /api/work-orders/{id}/assign", workOrderHandler.AssignMechanic)
	mux.HandleFunc("PUT /api/work-orders/{id}/estimate", workOrderHandler.SaveEstimation)
	mux.HandleFunc("PUT /api/work-orders/{id}/estimation", workOrderHandler.UpdateEstimation)
	mux.HandleFunc("PUT /api/work-orders/{id}/complete", workOrderHandler.CompleteWorkOrder)

	// Transaction & Cashier Routes
	mux.HandleFunc("GET /api/transactions/ready-for-cashier", transactionHandler.GetReadyWorkOrders)
	mux.HandleFunc("GET /api/transactions/wo/{woId}", transactionHandler.GetTransactionByWO)
	mux.HandleFunc("POST /api/transactions/{id}/pay", transactionHandler.FinalizePayment)

	// Cashflow & Finance Routes
	mux.HandleFunc("GET /api/cashflows", cashflowHandler.GetAllCashflows)
	mux.HandleFunc("POST /api/cashflows", cashflowHandler.CreateManualCashflow)
	mux.HandleFunc("DELETE /api/cashflows/{id}", cashflowHandler.DeleteCashflow)
	mux.HandleFunc("GET /api/finance/summary", cashflowHandler.GetFinanceSummary)
	mux.HandleFunc("GET /api/finance/chart", cashflowHandler.GetChartData)

	// Dashboard Routes
	mux.HandleFunc("GET /api/dashboard/summary", dashboardHandler.GetDashboardSummary)

	// User & Role Routes
	mux.HandleFunc("GET /api/users", userHandler.GetAllUsers)
	mux.HandleFunc("POST /api/users", userHandler.CreateUser)
	mux.HandleFunc("PUT /api/users/{id}", userHandler.UpdateUser)
	
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		type Req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
		
		query := `
			SELECT u.id, u.username, u.password, u.role_id, r.role_name, u.employee_id, e.name as employee_name, u.status
			FROM users u
			JOIN roles r ON u.role_id = r.id
			JOIN employees e ON u.employee_id = e.id
			WHERE LOWER(u.username) = LOWER($1) AND u.status = 'Y'
		`
		type UserResponse struct {
			ID           int        `json:"id"`
			Username     string     `json:"username"`
			PasswordHash string     `json:"-"`
			RoleID       int        `json:"roleId"`
			RoleName     string     `json:"roleName"`
			EmployeeID   int        `json:"employeeId"`
			EmployeeName string     `json:"employeeName"`
			Status       string     `json:"status"`
			Menus        []MenuItem `json:"menus"`
		}
		
		var u UserResponse
		err := db.QueryRowContext(r.Context(), query, req.Username).Scan(
			&u.ID, &u.Username, &u.PasswordHash, &u.RoleID, &u.RoleName, &u.EmployeeID, &u.EmployeeName, &u.Status,
		)
		if err == sql.ErrNoRows {
			http.Error(w, "Username atau password salah", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Verify password using bcrypt
		err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password))
		if err != nil {
			// Fallback: check plain text for default seeded database records
			if u.PasswordHash != req.Password {
				http.Error(w, "Username atau password salah", http.StatusUnauthorized)
				return
			}
		}

		// Fetch menus for role
		menus, err := getMenusForRole(db, u.RoleID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Gagal memuat menu: %v", err), http.StatusInternalServerError)
			return
		}
		u.Menus = menus
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	})
	
	mux.HandleFunc("GET /api/roles", userHandler.GetRoles)

	// Menu & Role-Menu Management Endpoints
	mux.HandleFunc("GET /api/menus", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), "SELECT id, label, COALESCE(icon, ''), COALESCE(router_link, ''), parent_id FROM menus WHERE status = 'Y' ORDER BY parent_id ASC NULLS FIRST, sort_order ASC, id ASC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type MenuResponse struct {
			ID         int    `json:"id"`
			Label      string `json:"label"`
			Icon       string `json:"icon"`
			RouterLink string `json:"routerLink"`
			ParentID   *int   `json:"parentId,omitempty"`
		}

		var menus []MenuResponse
		for rows.Next() {
			var m MenuResponse
			var parentID sql.NullInt64
			if err := rows.Scan(&m.ID, &m.Label, &m.Icon, &m.RouterLink, &parentID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if parentID.Valid {
				val := int(parentID.Int64)
				m.ParentID = &val
			}
			menus = append(menus, m)
		}
		w.Header().Set("Content-Type", "application/json")
		if menus == nil {
			menus = []MenuResponse{}
		}
		json.NewEncoder(w).Encode(menus)
	})

	mux.HandleFunc("GET /api/roles/{id}/menus", func(w http.ResponseWriter, r *http.Request) {
		roleID := r.PathValue("id")
		rows, err := db.QueryContext(r.Context(), "SELECT menu_id FROM role_menus WHERE role_id = $1", roleID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var menuIDs []int
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			menuIDs = append(menuIDs, id)
		}
		w.Header().Set("Content-Type", "application/json")
		if menuIDs == nil {
			menuIDs = []int{}
		}
		json.NewEncoder(w).Encode(menuIDs)
	})

	mux.HandleFunc("PUT /api/roles/{id}/menus", func(w http.ResponseWriter, r *http.Request) {
		roleID := r.PathValue("id")
		type Req struct {
			MenuIDs []int `json:"menuIds"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Delete existing menu mapping
		_, err = tx.ExecContext(r.Context(), "DELETE FROM role_menus WHERE role_id = $1", roleID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Insert new menu mapping
		for _, mID := range req.MenuIDs {
			_, err = tx.ExecContext(r.Context(), "INSERT INTO role_menus (role_id, menu_id) VALUES ($1, $2)", roleID, mID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success"}`)
	})
	
	// Role CRUD Extensions
	mux.HandleFunc("POST /api/roles", func(w http.ResponseWriter, r *http.Request) {
		type Req struct {
			RoleName string `json:"roleName"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.RoleName == "" {
			http.Error(w, "Nama role wajib diisi", http.StatusBadRequest)
			return
		}
		
		defaultPermissions := `{"dashboard":{"create":"N","read":"Y","update":"N","delete":"N"},"master":{"create":"N","read":"Y","update":"N","delete":"N"},"inventory":{"create":"N","read":"Y","update":"N","delete":"N"},"cashier":{"create":"N","read":"Y","update":"N","delete":"N"},"reports":{"create":"N","read":"Y","update":"N","delete":"N"}}`
		
		query := `
			INSERT INTO roles (role_name, permissions, created_by, updated_by)
			VALUES ($1, $2::jsonb, 'admin', 'admin')
			RETURNING id
		`
		var id int
		err := db.QueryRowContext(r.Context(), query, req.RoleName, defaultPermissions).Scan(&id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id": %d, "roleName": "%s"}`, id, req.RoleName)
	})

	mux.HandleFunc("PUT /api/roles/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		type Req struct {
			RoleName string `json:"roleName"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.RoleName == "" {
			http.Error(w, "Nama role wajib diisi", http.StatusBadRequest)
			return
		}
		
		query := `
			UPDATE roles 
			SET role_name = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND status = 'Y'
		`
		_, err := db.ExecContext(r.Context(), query, req.RoleName, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success"}`)
	})

	mux.HandleFunc("DELETE /api/roles/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		query := `
			UPDATE roles 
			SET status = 'N', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, err := db.ExecContext(r.Context(), query, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success"}`)
	})

	// Category CRUD
	mux.HandleFunc("GET /api/categories", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), "SELECT id, name FROM categories WHERE status = 'Y' ORDER BY id ASC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		
		type Category struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		list := []Category{}
		for rows.Next() {
			var c Category
			if err := rows.Scan(&c.ID, &c.Name); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			list = append(list, c)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	})

	mux.HandleFunc("POST /api/categories", func(w http.ResponseWriter, r *http.Request) {
		type Req struct {
			Name string `json:"name"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "Nama kategori wajib diisi", http.StatusBadRequest)
			return
		}
		
		query := `
			INSERT INTO categories (name, created_by, updated_by)
			VALUES ($1, 'admin', 'admin')
			RETURNING id
		`
		var id int
		err := db.QueryRowContext(r.Context(), query, req.Name).Scan(&id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id": %d, "name": "%s"}`, id, req.Name)
	})

	mux.HandleFunc("PUT /api/categories/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		type Req struct {
			Name string `json:"name"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "Nama kategori wajib diisi", http.StatusBadRequest)
			return
		}
		
		query := `
			UPDATE categories 
			SET name = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND status = 'Y'
		`
		_, err := db.ExecContext(r.Context(), query, req.Name, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success"}`)
	})

	mux.HandleFunc("DELETE /api/categories/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		query := `
			UPDATE categories 
			SET status = 'N', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, err := db.ExecContext(r.Context(), query, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success"}`)
	})
	
	// Extended Employee CRUD routes
	mux.HandleFunc("GET /api/employees", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), "SELECT id, name, phone, COALESCE(address, ''), position FROM employees WHERE status = 'Y' ORDER BY name ASC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		
		type Employee struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Phone    string `json:"phone"`
			Address  string `json:"address"`
			Position string `json:"position"`
		}
		list := []Employee{}
		for rows.Next() {
			var e Employee
			if err := rows.Scan(&e.ID, &e.Name, &e.Phone, &e.Address, &e.Position); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			list = append(list, e)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	})

	mux.HandleFunc("POST /api/employees", func(w http.ResponseWriter, r *http.Request) {
		type Req struct {
			Name     string `json:"name"`
			Phone    string `json:"phone"`
			Address  string `json:"address"`
			Position string `json:"position"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Phone == "" || req.Position == "" {
			http.Error(w, "Nama, Nomor Telepon, dan Jabatan wajib diisi", http.StatusBadRequest)
			return
		}
		
		query := `
			INSERT INTO employees (name, phone, address, position, status, created_by, updated_by)
			VALUES ($1, $2, $3, $4, 'Y', 'admin', 'admin')
			RETURNING id
		`
		var id int
		err := db.QueryRowContext(r.Context(), query, req.Name, req.Phone, req.Address, req.Position).Scan(&id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id": %d}`, id)
	})

	mux.HandleFunc("PUT /api/employees/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		type Req struct {
			Name     string `json:"name"`
			Phone    string `json:"phone"`
			Address  string `json:"address"`
			Position string `json:"position"`
		}
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Phone == "" || req.Position == "" {
			http.Error(w, "Nama, Nomor Telepon, dan Jabatan wajib diisi", http.StatusBadRequest)
			return
		}
		
		query := `
			UPDATE employees 
			SET name = $1, phone = $2, address = $3, position = $4, updated_at = CURRENT_TIMESTAMP
			WHERE id = $5 AND status = 'Y'
		`
		_, err := db.ExecContext(r.Context(), query, req.Name, req.Phone, req.Address, req.Position, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success"}`)
	})

	mux.HandleFunc("DELETE /api/employees/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		query := `
			UPDATE employees 
			SET status = 'N', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, err := db.ExecContext(r.Context(), query, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success"}`)
	})

	// Server configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: middleware.LoggerAndRecovery(mux),
	}

	log.Printf("Starting Pro Auto Garage API on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
