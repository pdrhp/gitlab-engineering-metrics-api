package middleware

import (
	"context"
	"net/http"

	"gitlab-engineering-metrics-api/internal/auth"
	"gitlab-engineering-metrics-api/internal/http/responses"
)

// ClientInfo holds authenticated client information
type ClientInfo struct {
	ClientID string
}

// clientInfoContextKey is the context key for client info
type clientInfoContextKey string

const (
	// ClientIDHeader is the header name for client ID
	ClientIDHeader = "X-Client-ID"
	// ClientSecretHeader is the header name for client secret
	ClientSecretHeader = "X-Client-Secret"
	// ClientInfoContextKey is the context key for client info
	ClientInfoContextKey clientInfoContextKey = "client_info"
)

// Auth creates an authentication middleware with the given validator
func Auth(validator *auth.Validator) func(http.Handler) http.Handler {
	if validator == nil {
		panic("auth middleware: validator is nil")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract credentials from headers
			clientID := r.Header.Get(ClientIDHeader)
			clientSecret := r.Header.Get(ClientSecretHeader)

			// Validate credentials
			if !validator.Validate(clientID, clientSecret) {
				requestID := GetRequestID(r.Context())
				responses.Unauthorized(w, requestID)
				return
			}

			// Store client info in context
			clientInfo := &ClientInfo{
				ClientID: clientID,
			}
			ctx := context.WithValue(r.Context(), ClientInfoContextKey, clientInfo)

			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClientInfo retrieves the client info from the context
func GetClientInfo(ctx context.Context) *ClientInfo {
	if ctx == nil {
		return nil
	}
	if info, ok := ctx.Value(ClientInfoContextKey).(*ClientInfo); ok {
		return info
	}
	return nil
}
