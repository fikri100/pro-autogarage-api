package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type PortalService struct {
	db          *sql.DB
	waService   WhatsAppService
	bookingRepo *repository.BookingRepository
}

func NewPortalService(db *sql.DB, waService WhatsAppService, bookingRepo *repository.BookingRepository) *PortalService {
	return &PortalService{
		db:          db,
		waService:   waService,
		bookingRepo: bookingRepo,
	}
}

// SendOTP generates a 6-digit OTP, stores it in customer_otps, and sends it via WhatsApp
func (s *PortalService) SendOTP(ctx context.Context, phone string) error {
	if phone == "" {
		return errors.New("phone number is required")
	}

	// 1. Generate 6-digit numeric OTP (guaranteed range 100000-999999)
	rand.Seed(time.Now().UnixNano())
	otp := fmt.Sprintf("%d", rand.Intn(900000)+100000)
	expiredAt := time.Now().Add(5 * time.Minute)

	// 2. Insert into customer_otps table
	query := `
		INSERT INTO customer_otps (phone, otp_code, expired_at)
		VALUES ($1, $2, $3)
	`
	_, err := s.db.ExecContext(ctx, query, phone, otp, expiredAt)
	if err != nil {
		return fmt.Errorf("failed to save OTP code: %w", err)
	}

	// 3. Send via WhatsApp (Mock prints to console)
	err = s.waService.SendOTP(ctx, phone, otp)
	if err != nil {
		return fmt.Errorf("failed to send OTP message: %w", err)
	}

	return nil
}

// VerifyOTP checks if the provided OTP code is correct and not expired
func (s *PortalService) VerifyOTP(ctx context.Context, phone string, code string) error {
	if phone == "" || code == "" {
		return errors.New("phone number and OTP code are required")
	}

	query := `
		SELECT id FROM customer_otps
		WHERE phone = $1 AND otp_code = $2 AND expired_at > $3 AND is_verified = FALSE
		ORDER BY id DESC LIMIT 1
	`
	var id int
	err := s.db.QueryRowContext(ctx, query, phone, code, time.Now()).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("invalid or expired OTP code")
		}
		return err
	}

	// Mark OTP as verified
	updateQuery := `UPDATE customer_otps SET is_verified = TRUE WHERE id = $1`
	_, err = s.db.ExecContext(ctx, updateQuery, id)
	return err
}

// Register registers a new customer after verifying their phone was OTP-verified
func (s *PortalService) Register(ctx context.Context, req domain.RegisterCustomerRequest) (*domain.Customer, error) {
	if req.Name == "" || req.Phone == "" || req.Username == "" || req.Password == "" {
		return nil, errors.New("name, phone, username, and password are required")
	}

	// 1. Verify that phone number has been verified via OTP recently (within 30 mins)
	verifyCheckQuery := `
		SELECT 1 FROM customer_otps
		WHERE phone = $1 AND is_verified = TRUE AND created_at > $2
		LIMIT 1
	`
	var verified int
	err := s.db.QueryRowContext(ctx, verifyCheckQuery, req.Phone, time.Now().Add(-30*time.Minute)).Scan(&verified)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("phone number is not verified via WhatsApp OTP yet")
		}
		return nil, err
	}

	// 2. Check if phone number is already registered in customers table
	var exists int
	phoneCheckQuery := `SELECT 1 FROM customers WHERE phone = $1 AND status = 'Y' LIMIT 1`
	err = s.db.QueryRowContext(ctx, phoneCheckQuery, req.Phone).Scan(&exists)
	if err == nil {
		return nil, errors.New("WhatsApp number is already registered")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// 3. Check if username is already taken
	usernameCheckQuery := `SELECT 1 FROM customers WHERE username = $1 LIMIT 1`
	err = s.db.QueryRowContext(ctx, usernameCheckQuery, req.Username).Scan(&exists)
	if err == nil {
		return nil, errors.New("username is already taken")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// 4. Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 5. Insert new customer record
	insertQuery := `
		INSERT INTO customers (name, phone, username, password, address, is_self_service, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, TRUE, 'SELF_SERVICE', 'SELF_SERVICE')
		RETURNING id, status, created_at, updated_at
	`
	var c domain.Customer
	c.Name = req.Name
	c.Phone = req.Phone
	c.Username = &req.Username
	c.Address = &req.Address
	c.IsSelfService = true
	createdByVal := "SELF_SERVICE"
	c.CreatedBy = &createdByVal
	c.UpdatedBy = &createdByVal

	err = s.db.QueryRowContext(ctx, insertQuery, c.Name, c.Phone, req.Username, string(hashedPassword), req.Address).
		Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to save customer record: %w", err)
	}

	return &c, nil
}

// Login validates credentials for customer portal
func (s *PortalService) Login(ctx context.Context, req domain.PortalLoginRequest) (*domain.Customer, error) {
	if req.UsernameOrPhone == "" || req.Password == "" {
		return nil, errors.New("username/phone and password are required")
	}

	query := `
		SELECT id, name, phone, username, password, is_self_service, status, created_by, created_at, updated_by, updated_at
		FROM customers
		WHERE (LOWER(username) = LOWER($1) OR phone = $1) AND status = 'Y'
	`
	var c domain.Customer
	var passwordHash string
	err := s.db.QueryRowContext(ctx, query, req.UsernameOrPhone).Scan(
		&c.ID, &c.Name, &c.Phone, &c.Username, &passwordHash, &c.IsSelfService,
		&c.Status, &c.CreatedBy, &c.CreatedAt, &c.UpdatedBy, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid username/phone or password")
		}
		return nil, err
	}

	// Compare bcrypt password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid username/phone or password")
	}

	return &c, nil
}

// CreateBooking handles vehicle auto-linking and bookings creation
func (s *PortalService) CreateBooking(ctx context.Context, customerID int, req domain.PortalBookingRequest) (*domain.Booking, error) {
	if req.LicensePlate == "" || req.BookingDate == "" || req.BookingTime == "" {
		return nil, errors.New("licensePlate, bookingDate, and bookingTime are required")
	}

	// Start atomic DB Transaction to ensure data integrity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Check if vehicle plate already exists
	var vehicleID int
	var vehicleCustID int
	vehicleQuery := `SELECT id, customer_id FROM vehicles WHERE license_plate = $1 AND status = 'Y' LIMIT 1`
	err = tx.QueryRowContext(ctx, vehicleQuery, req.LicensePlate).Scan(&vehicleID, &vehicleCustID)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Vehicle does not exist -> Auto-Link: register it under current customer
			insertVehicleQuery := `
				INSERT INTO vehicles (customer_id, license_plate, brand, model, year_made, transmission, created_by, updated_by)
				VALUES ($1, $2, $3, $4, $5, $6, 'SELF_SERVICE', 'SELF_SERVICE')
				RETURNING id
			`
			err = tx.QueryRowContext(ctx, insertVehicleQuery, customerID, req.LicensePlate, req.Brand, req.Model, req.YearMade, req.Transmission).Scan(&vehicleID)
			if err != nil {
				return nil, fmt.Errorf("failed to register vehicle: %w", err)
			}
		} else {
			return nil, err
		}
	}

	// Check if date and time slot is already taken by another active booking
	var count int
	checkQuery := `SELECT COUNT(1) FROM bookings WHERE booking_date = $1 AND booking_time = $2 AND status = 'Y' AND operational_status != 'CANCELLED'`
	err = tx.QueryRowContext(ctx, checkQuery, req.BookingDate, req.BookingTime).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("jadwal tanggal dan jam ini sudah dipesan")
	}

	// 2. Insert booking record in PENDING state
	insertBookingQuery := `
		INSERT INTO bookings (customer_id, vehicle_id, booking_date, booking_time, complaints, operational_status, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, 'PENDING', 'SELF_SERVICE', 'SELF_SERVICE')
		RETURNING id
	`
	var bookingID int
	err = tx.QueryRowContext(ctx, insertBookingQuery, customerID, vehicleID, req.BookingDate, req.BookingTime, req.Complaints).Scan(&bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to save booking: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Load full details for response using BookingRepository
	return s.bookingRepo.FindByID(ctx, bookingID)
}

// GetBookings retrieves self bookings for customer portal
func (s *PortalService) GetBookings(ctx context.Context, customerID int) ([]*domain.Booking, error) {
	query := `
		SELECT 
			b.id, b.customer_id, c.name, c.phone, b.vehicle_id, v.license_plate, v.brand, v.model,
			b.booking_date, b.booking_time, b.complaints, b.operational_status, b.status,
			b.created_by, b.created_at, b.updated_by, b.updated_at,
			v.customer_id AS vehicle_customer_id, vc.name AS vehicle_owner_name,
			wo.estimated_completion
		FROM bookings b
		JOIN customers c ON b.customer_id = c.id
		JOIN vehicles v ON b.vehicle_id = v.id
		JOIN customers vc ON v.customer_id = vc.id
		LEFT JOIN work_orders wo ON wo.booking_id = b.id AND wo.status = 'Y'
		WHERE b.customer_id = $1 AND b.status = 'Y'
		ORDER BY b.booking_date DESC, b.booking_time DESC
	`
	rows, err := s.db.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*domain.Booking
	for rows.Next() {
		var b domain.Booking
		var bDate time.Time
		var bTime string
		var estimatedCompletion sql.NullTime

		if err := rows.Scan(
			&b.ID, &b.CustomerID, &b.CustomerName, &b.CustomerPhone, &b.VehicleID, &b.LicensePlate, &b.VehicleBrand, &b.VehicleModel,
			&bDate, &bTime, &b.Complaints, &b.OperationalStatus, &b.Status,
			&b.CreatedBy, &b.CreatedAt, &b.UpdatedBy, &b.UpdatedAt,
			&b.VehicleCustomerID, &b.VehicleOwnerName,
			&estimatedCompletion,
		); err != nil {
			return nil, err
		}

		b.BookingDate = bDate.Format("2006-01-02")
		b.BookingTime = bTime[:5]
		if estimatedCompletion.Valid {
			b.EstimatedCompletion = &estimatedCompletion.Time
		}

		// Load estimation details and total amount
		itemsQuery := `
			SELECT 
				td.id, td.transaction_id, td.product_id, p.code, p.name, p.item_type, p.category,
				td.quantity, td.price_at_transaction, td.subtotal, t.total_amount
			FROM transaction_details td
			JOIN transactions t ON td.transaction_id = t.id
			JOIN work_orders wo ON t.work_order_id = wo.id
			JOIN products p ON td.product_id = p.id
			WHERE wo.booking_id = $1 AND td.status = 'Y' AND t.status = 'Y' AND wo.status = 'Y'
		`
		rowsItems, errItems := s.db.QueryContext(ctx, itemsQuery, b.ID)
		if errItems == nil {
			var items []*domain.TransactionDetail
			var totalAmt float64
			hasItems := false

			for rowsItems.Next() {
				var d domain.TransactionDetail
				if err := rowsItems.Scan(
					&d.ID, &d.TransactionID, &d.ProductID, &d.ProductCode, &d.ProductName, &d.ProductType, &d.ProductCategory,
					&d.Quantity, &d.PriceAtTransaction, &d.Subtotal, &totalAmt,
				); err == nil {
					items = append(items, &d)
					hasItems = true
				}
			}
			rowsItems.Close()

			if hasItems {
				b.EstimationItems = items
				b.TotalAmount = &totalAmt
			} else {
				b.EstimationItems = []*domain.TransactionDetail{}
			}
		} else {
			b.EstimationItems = []*domain.TransactionDetail{}
		}

		bookings = append(bookings, &b)
	}

	if bookings == nil {
		bookings = []*domain.Booking{}
	}
	return bookings, nil
}

// GetDashboardSummary retrieves summary metrics for the portal home
func (s *PortalService) GetDashboardSummary(ctx context.Context, customerID int) (*domain.PortalDashboardSummary, error) {
	var summary domain.PortalDashboardSummary

	// 1. Total vehicles
	queryVehicles := `SELECT COUNT(1) FROM vehicles WHERE customer_id = $1 AND status = 'Y'`
	err := s.db.QueryRowContext(ctx, queryVehicles, customerID).Scan(&summary.TotalVehicles)
	if err != nil {
		return nil, fmt.Errorf("failed to count vehicles: %w", err)
	}

	// 2. Active bookings (PENDING, CONFIRMED)
	queryActive := `SELECT COUNT(1) FROM bookings WHERE customer_id = $1 AND operational_status IN ('PENDING', 'CONFIRMED') AND status = 'Y'`
	err = s.db.QueryRowContext(ctx, queryActive, customerID).Scan(&summary.ActiveBookings)
	if err != nil {
		return nil, fmt.Errorf("failed to count active bookings: %w", err)
	}

	// 3. Total history bookings
	queryHistory := `SELECT COUNT(1) FROM bookings WHERE customer_id = $1 AND status = 'Y'`
	err = s.db.QueryRowContext(ctx, queryHistory, customerID).Scan(&summary.TotalHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to count total history: %w", err)
	}

	return &summary, nil
}

// GetVehicles retrieves list of vehicles for customer
func (s *PortalService) GetVehicles(ctx context.Context, customerID int) ([]*domain.PortalVehicleResponse, error) {
	query := `
		SELECT id, customer_id, license_plate, brand, model, year_made, transmission
		FROM vehicles
		WHERE customer_id = $1 AND status = 'Y'
		ORDER BY id DESC
	`
	rows, err := s.db.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []*domain.PortalVehicleResponse
	for rows.Next() {
		var v domain.PortalVehicleResponse
		var brand, model, trans sql.NullString
		var year sql.NullInt32

		if err := rows.Scan(&v.ID, &v.CustomerID, &v.LicensePlate, &brand, &model, &year, &trans); err != nil {
			return nil, err
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

		vehicles = append(vehicles, &v)
	}

	if vehicles == nil {
		vehicles = []*domain.PortalVehicleResponse{}
	}
	return vehicles, nil
}

// AddVehicle adds a new vehicle for customer
func (s *PortalService) AddVehicle(ctx context.Context, customerID int, req domain.PortalVehicleRequest) (*domain.PortalVehicleResponse, error) {
	if req.LicensePlate == "" || req.Brand == "" || req.Model == "" {
		return nil, errors.New("plat nomor, brand, dan model wajib diisi")
	}

	// Check if plate already registered under another active vehicle
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM vehicles WHERE license_plate = $1 AND status = 'Y' LIMIT 1", req.LicensePlate).Scan(&exists)
	if err == nil {
		return nil, errors.New("nomor polisi sudah terdaftar di sistem")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	query := `
		INSERT INTO vehicles (customer_id, license_plate, brand, model, year_made, transmission, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, 'SELF_SERVICE', 'SELF_SERVICE')
		RETURNING id
	`
	var id int
	err = s.db.QueryRowContext(ctx, query, customerID, req.LicensePlate, req.Brand, req.Model, req.YearMade, req.Transmission).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to save vehicle: %w", err)
	}

	return &domain.PortalVehicleResponse{
		ID:           id,
		CustomerID:   customerID,
		LicensePlate: req.LicensePlate,
		Brand:        req.Brand,
		Model:        req.Model,
		YearMade:     req.YearMade,
		Transmission: req.Transmission,
	}, nil
}

// UpdateVehicle updates an existing vehicle for customer
func (s *PortalService) UpdateVehicle(ctx context.Context, customerID int, vehicleID int, req domain.PortalVehicleRequest) error {
	if req.LicensePlate == "" || req.Brand == "" || req.Model == "" {
		return errors.New("plat nomor, brand, dan model wajib diisi")
	}

	// Check ownership
	var ownerID int
	err := s.db.QueryRowContext(ctx, "SELECT customer_id FROM vehicles WHERE id = $1 AND status = 'Y'", vehicleID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("kendaraan tidak ditemukan")
		}
		return err
	}
	if ownerID != customerID {
		return errors.New("tidak diizinkan mengakses data kendaraan ini")
	}

	// Check if plate already registered under another active vehicle
	var exists int
	err = s.db.QueryRowContext(ctx, "SELECT 1 FROM vehicles WHERE license_plate = $1 AND status = 'Y' AND id != $2 LIMIT 1", req.LicensePlate, vehicleID).Scan(&exists)
	if err == nil {
		return errors.New("nomor polisi sudah terdaftar di sistem")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	query := `
		UPDATE vehicles
		SET license_plate = $1, brand = $2, model = $3, year_made = $4, transmission = $5, updated_by = 'SELF_SERVICE', updated_at = CURRENT_TIMESTAMP
		WHERE id = $6 AND customer_id = $7 AND status = 'Y'
	`
	_, err = s.db.ExecContext(ctx, query, req.LicensePlate, req.Brand, req.Model, req.YearMade, req.Transmission, vehicleID, customerID)
	return err
}

// DeleteVehicle performs soft-delete of customer vehicle
func (s *PortalService) DeleteVehicle(ctx context.Context, customerID int, vehicleID int) error {
	query := `
		UPDATE vehicles
		SET status = 'N', updated_by = 'SELF_SERVICE', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND customer_id = $2 AND status = 'Y'
	`
	res, err := s.db.ExecContext(ctx, query, vehicleID, customerID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("kendaraan tidak ditemukan atau tidak diizinkan untuk dihapus")
	}
	return nil
}

// GetProfile retrieves profile information
func (s *PortalService) GetProfile(ctx context.Context, customerID int) (*domain.Customer, error) {
	query := `
		SELECT id, name, phone, address, email, username, is_self_service, status, created_at, updated_at
		FROM customers
		WHERE id = $1 AND status = 'Y'
	`
	var c domain.Customer
	err := s.db.QueryRowContext(ctx, query, customerID).Scan(
		&c.ID, &c.Name, &c.Phone, &c.Address, &c.Email, &c.Username, &c.IsSelfService,
		&c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("pelanggan tidak ditemukan")
		}
		return nil, err
	}
	return &c, nil
}

// UpdateProfile updates customer information including optional password change
func (s *PortalService) UpdateProfile(ctx context.Context, customerID int, req domain.UpdateProfileRequest) error {
	if req.Name == "" || req.Username == "" {
		return errors.New("nama dan username wajib diisi")
	}

	// Check username uniqueness if changed
	var currentUsername sql.NullString
	var currentPasswordHash string
	err := s.db.QueryRowContext(ctx, "SELECT username, password FROM customers WHERE id = $1 AND status = 'Y'", customerID).Scan(&currentUsername, &currentPasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("pelanggan tidak ditemukan")
		}
		return err
	}

	if !currentUsername.Valid || currentUsername.String != req.Username {
		var exists int
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM customers WHERE username = $1 LIMIT 1", req.Username).Scan(&exists)
		if err == nil {
			return errors.New("username sudah digunakan oleh pengguna lain")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if req.NewPassword != nil && *req.NewPassword != "" {
		if req.CurrentPassword == nil || *req.CurrentPassword == "" {
			return errors.New("kata sandi saat ini wajib diisi untuk mengubah kata sandi baru")
		}
		
		err = bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(*req.CurrentPassword))
		if err != nil {
			return errors.New("kata sandi saat ini salah")
		}

		newHash, err := bcrypt.GenerateFromPassword([]byte(*req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash new password: %w", err)
		}

		query := `
			UPDATE customers
			SET name = $1, username = $2, password = $3, address = $4, updated_by = 'SELF_SERVICE', updated_at = CURRENT_TIMESTAMP
			WHERE id = $5 AND status = 'Y'
		`
		_, err = tx.ExecContext(ctx, query, req.Name, req.Username, string(newHash), req.Address, customerID)
		if err != nil {
			return err
		}
	} else {
		query := `
			UPDATE customers
			SET name = $1, username = $2, address = $3, updated_by = 'SELF_SERVICE', updated_at = CURRENT_TIMESTAMP
			WHERE id = $4 AND status = 'Y'
		`
		_, err = tx.ExecContext(ctx, query, req.Name, req.Username, req.Address, customerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CancelBooking cancels a pending booking for customer
func (s *PortalService) CancelBooking(ctx context.Context, customerID int, bookingID int) error {
	var currentStatus string
	var dbCustomerID int
	err := s.db.QueryRowContext(ctx, "SELECT customer_id, operational_status FROM bookings WHERE id = $1 AND status = 'Y'", bookingID).Scan(&dbCustomerID, &currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("booking tidak ditemukan")
		}
		return err
	}
	if dbCustomerID != customerID {
		return errors.New("tidak diizinkan membatalkan booking ini")
	}
	if currentStatus != "PENDING" {
		return errors.New("hanya booking dengan status 'Menunggu Persetujuan' yang dapat dibatalkan")
	}

	_, err = s.db.ExecContext(ctx, "UPDATE bookings SET operational_status = 'CANCELLED', updated_by = 'SELF_SERVICE', updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND status = 'Y'", bookingID)
	return err
}

