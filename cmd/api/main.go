package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	
	"pro-autogarage-api/internal/handler"
	"pro-autogarage-api/internal/repository"
	"pro-autogarage-api/internal/service"
	"pro-autogarage-api/pkg/database"

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

	// Initialize Services
	customerService := service.NewCustomerService(customerRepo)
	userService := service.NewUserService(userRepo)

	// Initialize Handlers
	customerHandler := handler.NewCustomerHandler(customerService)
	userHandler := handler.NewUserHandler(userService)

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
		Handler: mux,
	}

	log.Printf("Starting Pro Auto Garage API on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
