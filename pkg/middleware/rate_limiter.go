package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// IPRateLimiter is an IP-based token-bucket/history rate limiter
type IPRateLimiter struct {
	mu      sync.Mutex
	history map[string][]time.Time
	limit   int
	window  time.Duration
}

// NewIPRateLimiter creates a new IPRateLimiter instance
func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		history: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
}

// Limit returns a middleware that limits requests per IP
func (l *IPRateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := l.getIP(r)

		l.mu.Lock()
		now := time.Now()

		// Filter out timestamps that are outside the time window
		var active []time.Time
		for _, t := range l.history[ip] {
			if now.Sub(t) < l.window {
				active = append(active, t)
			}
		}

		// Check if threshold is reached
		if len(active) >= l.limit {
			l.mu.Unlock()
			http.Error(w, `{"error": "Too many requests. Please try again later."}`, http.StatusTooManyRequests)
			return
		}

		// Record current timestamp and update history
		active = append(active, now)
		l.history[ip] = active
		l.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// getIP extracts the client IP address from request headers or RemoteAddr
func (l *IPRateLimiter) getIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}
	if idx := strings.Index(ip, ","); idx != -1 {
		ip = strings.TrimSpace(ip[:idx])
	}
	return ip
}
