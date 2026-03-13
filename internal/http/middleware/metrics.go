package middleware

import (
	"net/http"
	"time"

	"gitlab-engineering-metrics-api/internal/observability"
)

// responseWriterWithStatus wraps http.ResponseWriter to capture the status code
type responseWriterWithStatus struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWithStatus) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Metrics middleware records request metrics using the observability collector
func Metrics(collector *observability.MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriterWithStatus{ResponseWriter: w, statusCode: http.StatusOK}

			// Call next handler
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Record metrics
			collector.RecordRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}
