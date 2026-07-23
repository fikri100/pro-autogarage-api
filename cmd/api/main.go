package main

import (
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
	vehicleRepo := repository.NewVehicleRepository(db)
	userRepo := repository.NewUserRepository(db)
	productRepo := repository.NewProductRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	workOrderRepo := repository.NewWorkOrderRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	cashflowRepo := repository.NewCashflowRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db)
	paramRepo := repository.NewParamRepository(db)

	// Initialize Services
	customerService := service.NewCustomerService(customerRepo, vehicleRepo)
	vehicleService := service.NewVehicleService(vehicleRepo)
	userService := service.NewUserService(userRepo)
	productService := service.NewProductService(productRepo)
	bookingService := service.NewBookingService(bookingRepo, workOrderRepo)
	workOrderService := service.NewWorkOrderService(workOrderRepo, transactionRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	cashflowService := service.NewCashflowService(cashflowRepo)
	dashboardService := service.NewDashboardService(dashboardRepo)
	paramService := service.NewParamService(paramRepo)

	// Initialize Portal & Security Middleware
	portalAuthRateLimiter := middleware.NewIPRateLimiter(15, time.Minute)
	portalGeneralRateLimiter := middleware.NewIPRateLimiter(100, time.Minute)
	waService := service.NewMockWhatsAppService()
	portalService := service.NewPortalService(db, waService, bookingRepo)

	// Initialize Handlers
	customerHandler := handler.NewCustomerHandler(customerService)
	vehicleHandler := handler.NewVehicleHandler(vehicleService)
	userHandler := handler.NewUserHandler(userService)
	productHandler := handler.NewProductHandler(productService)
	bookingHandler := handler.NewBookingHandler(bookingService)
	workOrderHandler := handler.NewWorkOrderHandler(workOrderService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	cashflowHandler := handler.NewCashflowHandler(cashflowService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	paramHandler := handler.NewParamHandler(paramService)
	portalHandler := handler.NewPortalHandler(portalService)
	exportHandler := handler.NewExportHandler(db)

	// Initialize standard Go 1.22+ ServeMux
	mux := http.NewServeMux()

	// Base API route
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok", "db_connected": true}`)
	})
	mux.HandleFunc("GET /api/params", paramHandler.GetParamsByGroup)

	// Customer Routes
	mux.HandleFunc("POST /api/customers", customerHandler.CreateCustomer)
	mux.HandleFunc("GET /api/customers", customerHandler.GetAllCustomers)
	mux.HandleFunc("GET /api/customers/{id}", customerHandler.GetCustomerByID)
	mux.HandleFunc("PUT /api/customers/{id}", customerHandler.UpdateCustomer)
	mux.HandleFunc("DELETE /api/customers/{id}", customerHandler.DeleteCustomer)

	// Vehicle Routes (Full CRUD for Customer Vehicles & Autocomplete)
	mux.HandleFunc("GET /api/vehicles", vehicleHandler.GetVehicles)
	mux.HandleFunc("POST /api/vehicles", vehicleHandler.CreateVehicle)
	mux.HandleFunc("PUT /api/vehicles/{id}", vehicleHandler.UpdateVehicle)
	mux.HandleFunc("DELETE /api/vehicles/{id}", vehicleHandler.DeleteVehicle)

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
	mux.HandleFunc("GET /api/bookings/booked-slots", bookingHandler.GetBookedSlots)
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

	// Export Routes (xlsx & csv download)
	mux.HandleFunc("GET /api/export/cashflows", exportHandler.ExportCashflowsExcel)
	mux.HandleFunc("GET /api/export/customers", exportHandler.ExportCustomersCSV)
	mux.HandleFunc("GET /api/export/products", exportHandler.ExportProductsCSV)
	mux.HandleFunc("GET /api/export/employees", exportHandler.ExportEmployeesCSV)

	// Dashboard Routes
	mux.HandleFunc("GET /api/dashboard/summary", dashboardHandler.GetDashboardSummary)

	// User & Role Routes
	mux.HandleFunc("GET /api/users", userHandler.GetAllUsers)
	mux.HandleFunc("POST /api/users", userHandler.CreateUser)
	mux.HandleFunc("PUT /api/users/{id}", userHandler.UpdateUser)

	mux.HandleFunc("POST /api/login", userHandler.Login)
	mux.HandleFunc("GET /api/roles", userHandler.GetRoles)

	// Menu & Role-Menu Management Endpoints
	mux.HandleFunc("GET /api/menus", userHandler.GetAllMenus)
	mux.HandleFunc("GET /api/roles/{id}/menus", userHandler.GetRoleMenus)
	mux.HandleFunc("PUT /api/roles/{id}/menus", userHandler.UpdateRoleMenus)

	// Role CRUD Extensions
	mux.HandleFunc("POST /api/roles", userHandler.CreateRole)
	mux.HandleFunc("PUT /api/roles/{id}", userHandler.UpdateRole)
	mux.HandleFunc("DELETE /api/roles/{id}", userHandler.DeleteRole)

	// Category CRUD
	mux.HandleFunc("GET /api/categories", productHandler.GetCategories)
	mux.HandleFunc("POST /api/categories", productHandler.CreateCategory)
	mux.HandleFunc("PUT /api/categories/{id}", productHandler.UpdateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", productHandler.DeleteCategory)

	// Extended Employee CRUD routes
	mux.HandleFunc("GET /api/employees", userHandler.GetEmployees)
	mux.HandleFunc("POST /api/employees", userHandler.CreateEmployee)
	mux.HandleFunc("PUT /api/employees/{id}", userHandler.UpdateEmployee)
	mux.HandleFunc("DELETE /api/employees/{id}", userHandler.DeleteEmployee)

	// Server configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Wrap mux with AuthMiddleware, then LoggerAndRecovery
	handlerStack := middleware.LoggerAndRecovery(middleware.AuthMiddleware(mux))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handlerStack,
	}

	log.Printf("Starting Pro Auto Garage API on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
