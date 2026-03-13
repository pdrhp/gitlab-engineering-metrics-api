package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"gitlab-engineering-metrics-api/internal/http/responses"
)

// Recovery middleware catches panics and returns 500 error
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get request ID from context
					requestID := GetRequestID(r.Context())

					// Log the panic with stack trace
					logger.Error("Panic recovered",
						slog.String("request_id", requestID),
						slog.String("error", fmt.Sprintf("%v", err)),
						slog.String("stack", string(debug.Stack())),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)

					// Return 500 error to client
					responses.InternalServerError(w, requestID)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
