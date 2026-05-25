package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	
	"pro-autogarage-api/internal/handler"
	"pro-autogarage-api/internal/repository"
	"pro-autogarage-api/internal/service"
	"pro-autogarage-api/pkg/database"
	"pro-autogarage-api/pkg/middleware"

	"github.com/joho/godotenv"
)

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

	// Initialize Handlers
	customerHandler := handler.NewCustomerHandler(customerService)
	userHandler := handler.NewUserHandler(userService)
	productHandler := handler.NewProductHandler(productService)
	bookingHandler := handler.NewBookingHandler(bookingService)
	workOrderHandler := handler.NewWorkOrderHandler(workOrderService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	cashflowHandler := handler.NewCashflowHandler(cashflowService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)

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

	// Vehicle Routes (Lightweight Fetch for Booking/WO Autocomplete)
	mux.HandleFunc("GET /api/vehicles", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), "SELECT id, customer_id, license_plate, brand, model FROM vehicles WHERE status = 'Y'")
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
		}

		var vehicles []Vehicle
		for rows.Next() {
			var v Vehicle
			var brand, model sql.NullString
			if err := rows.Scan(&v.ID, &v.CustomerID, &v.LicensePlate, &brand, &model); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if brand.Valid {
				v.Brand = brand.String
			}
			if model.Valid {
				v.Model = model.String
			}
			vehicles = append(vehicles, v)
		}

		w.Header().Set("Content-Type", "application/json")
		if vehicles == nil {
			vehicles = []Vehicle{}
		}
		json.NewEncoder(w).Encode(vehicles)
	})

	// Product & Inventory Routes
	mux.HandleFunc("POST /api/products", productHandler.CreateProduct)
	mux.HandleFunc("GET /api/products", productHandler.GetAllProducts)
	mux.HandleFunc("GET /api/products/{id}", productHandler.GetProductByID)
	mux.HandleFunc("PUT /api/products/{id}", productHandler.UpdateProduct)
	mux.HandleFunc("DELETE /api/products/{id}", productHandler.DeleteProduct)

	// Booking Routes
	mux.HandleFunc("POST /api/bookings", bookingHandler.CreateBooking)
	mux.HandleFunc("GET /api/bookings", bookingHandler.GetAllBookings)
	mux.HandleFunc("PUT /api/bookings/{id}/confirm", bookingHandler.ConfirmBooking)
	mux.HandleFunc("PUT /api/bookings/{id}/cancel", bookingHandler.CancelBooking)

	// Work Order Routes
	mux.HandleFunc("POST /api/work-orders", workOrderHandler.CreateWorkOrder)
	mux.HandleFunc("GET /api/work-orders", workOrderHandler.GetAllActiveWorkOrders)
	mux.HandleFunc("GET /api/work-orders/{id}", workOrderHandler.GetWorkOrderByID)
	mux.HandleFunc("PUT /api/work-orders/{id}/assign", workOrderHandler.AssignMechanic)
	mux.HandleFunc("PUT /api/work-orders/{id}/estimate", workOrderHandler.SaveEstimation)
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
	
	mux.HandleFunc("GET /api/roles", userHandler.GetRoles)
	mux.HandleFunc("PUT /api/roles/{id}/permissions", userHandler.UpdateRolePermissions)
	
	mux.HandleFunc("GET /api/employees", userHandler.GetEmployees)

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
