package logger

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
)

// Info logs standard informational messages
func Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("%s[INFO]%s %s", Blue, Reset, msg)
}

// Error logs error messages along with request context
func Error(r *http.Request, err error, status int) {
	if r != nil {
		log.Printf("%s[ERROR] Status %d | %s %s%s\n%s--> Detail: %v%s",
			Red, status, r.Method, r.URL.Path, Reset,
			Red, err, Reset,
		)
	} else {
		log.Printf("%s[ERROR] Status %d%s | %v", Red, status, Reset, err)
	}
}

// Success logs request completion successfully (2xx status codes)
func Success(r *http.Request, status int, duration time.Duration) {
	log.Printf("%s[REQ] %d%s | %s | %s %s | %s%v%s",
		Green, status, Reset,
		r.RemoteAddr,
		r.Method, r.URL.Path,
		Gray, duration, Reset,
	)
}

// Warn logs warnings (4xx status codes)
func Warn(r *http.Request, status int, duration time.Duration, errMsg string) {
	log.Printf("%s[REQ] %d%s | %s | %s %s | %s%v%s\n%s--> Client Error: %s%s",
		Yellow, status, Reset,
		r.RemoteAddr,
		r.Method, r.URL.Path,
		Gray, duration, Reset,
		Yellow, errMsg, Reset,
	)
}

// Fatal logs critical issues and panics
func Fatal(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Fatalf("%s[FATAL]%s %s", Magenta, Reset, msg)
}
