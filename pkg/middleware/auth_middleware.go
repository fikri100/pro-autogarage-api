package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"pro-autogarage-api/pkg/utils"
)

// AuthMiddleware intercepts API requests to validate JWT tokens
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 1. Skip authentication for public endpoints
		if path == "/api/login" ||
			path == "/api/health" ||
			path == "/api/params" ||
			path == "/api/portal/send-otp" ||
			path == "/api/portal/verify-otp" ||
			path == "/api/portal/register" ||
			path == "/api/portal/login" {
			next.ServeHTTP(w, r)
			return
		}

		// 2. Customer Portal routes authentication
		if strings.HasPrefix(path, "/api/portal/") {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Unauthorized: missing token"}`))
				return
			}

			claims, err := utils.ValidateCustomerToken(tokenStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Unauthorized: ` + err.Error() + `"}`))
				return
			}

			// Automatically set/override X-Customer-ID request header from verified claims
			r.Header.Set("X-Customer-ID", strconv.Itoa(claims.CustomerID))

			next.ServeHTTP(w, r)
			return
		}

		// 3. Admin Dashboard routes authentication
		if strings.HasPrefix(path, "/api/") {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Unauthorized: missing token"}`))
				return
			}

			_, err := utils.ValidateAdminToken(tokenStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Unauthorized: ` + err.Error() + `"}`))
				return
			}

			next.ServeHTTP(w, r)
			return
		}

		// Fallback for non-API routes (if any)
		next.ServeHTTP(w, r)
	})
}

// extractToken retrieves the bearer token from the Authorization header
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
