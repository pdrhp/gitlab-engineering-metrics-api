package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDContextKey is the context key for request ID
	RequestIDContextKey contextKey = "request_id"
)

// RequestID middleware generates or extracts a request ID from the request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get existing request ID from header
		requestID := r.Header.Get(RequestIDHeader)

		// If not provided, generate a new one
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to context
		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)

		// Add request ID to response header
		w.Header().Set(RequestIDHeader, requestID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return requestID
	}
	return ""
}
