package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "pro_auto_garage_super_secret_key"
	}
	jwtSecret = []byte(secret)
}

// SetJWTSecret overrides the default secret key if needed (useful for configuration updates)
func SetJWTSecret(secret string) {
	if secret != "" {
		jwtSecret = []byte(secret)
	}
}

// GetJWTSecret returns the current secret key
func GetJWTSecret() []byte {
	return jwtSecret
}

// AdminClaims represents custom JWT claims for administrators and staff
type AdminClaims struct {
	UserID     int    `json:"userId"`
	Username   string `json:"username"`
	RoleID     int    `json:"roleId"`
	RoleName   string `json:"roleName"`
	EmployeeID int    `json:"employeeId"`
	TokenType  string `json:"tokenType"` // should be "admin"
	jwt.RegisteredClaims
}

// CustomerClaims represents custom JWT claims for customer portal users
type CustomerClaims struct {
	CustomerID int    `json:"customerId"`
	Phone      string `json:"phone"`
	TokenType  string `json:"tokenType"` // should be "customer"
	jwt.RegisteredClaims
}

// GenerateAdminToken generates a JWT token for admin staff
func GenerateAdminToken(userID int, username string, roleID int, roleName string, employeeID int, duration time.Duration) (string, error) {
	claims := AdminClaims{
		UserID:     userID,
		Username:   username,
		RoleID:     roleID,
		RoleName:   roleName,
		EmployeeID: employeeID,
		TokenType:  "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GenerateCustomerToken generates a JWT token for portal customers
func GenerateCustomerToken(customerID int, phone string, duration time.Duration) (string, error) {
	claims := CustomerClaims{
		CustomerID: customerID,
		Phone:      phone,
		TokenType:  "customer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateAdminToken parses and validates an admin JWT token
func ValidateAdminToken(tokenStr string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid claims or token")
	}

	if claims.TokenType != "admin" {
		return nil, errors.New("invalid token type")
	}

	return claims, nil
}

// ValidateCustomerToken parses and validates a customer JWT token
func ValidateCustomerToken(tokenStr string) (*CustomerClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomerClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomerClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid claims or token")
	}

	if claims.TokenType != "customer" {
		return nil, errors.New("invalid token type")
	}

	return claims, nil
}
