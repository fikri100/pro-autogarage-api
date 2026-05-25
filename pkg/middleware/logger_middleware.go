package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"pro-autogarage-api/pkg/logger"
)

// responseWriter is a wrapper to capture status code in standard library middleware
type responseWriter struct {
	http.ResponseWriter
	statusCode    int
	wroteHeader   bool
	capturedError string
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(buf []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	if rw.statusCode >= 400 {
		rw.capturedError = string(buf)
	}
	return rw.ResponseWriter.Write(buf)
}

// LoggerAndRecovery is a central middleware that logs all incoming requests and recovers from panics
func LoggerAndRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		// Recover from panics to keep server alive and log trace
		defer func() {
			if err := recover(); err != nil {
				duration := time.Since(start)
				
				// Log Panic Error with traceback
				logger.Fatal("PANIC RECOVERED in request %s %s: %v\nStack Trace:\n%s",
					r.Method, r.URL.Path, err, debug.Stack(),
				)

				// Write 500 Internal Server Error
				http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
				
				// Print to log
				logger.Error(r, fmt.Errorf("panic: %v", err), http.StatusInternalServerError)
				logger.Warn(r, http.StatusInternalServerError, duration, "Panic recovered")
			}
		}()

		// Serve the request
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// Log request based on status code
		if rw.statusCode >= 500 {
			logger.Error(r, fmt.Errorf("%s", rw.capturedError), rw.statusCode)
		} else if rw.statusCode >= 400 {
			logger.Warn(r, rw.statusCode, duration, rw.capturedError)
		} else {
			logger.Success(r, rw.statusCode, duration)
		}
	})
}
