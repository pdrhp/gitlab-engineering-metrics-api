package responses

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// writeError writes an error response with the given status code and message
func writeError(w http.ResponseWriter, statusCode int, code, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log the error - can't write to response since headers already sent
		log.Printf("failed to encode error response: %v", err)
	}
}

// BadRequest returns a 400 Bad Request response
func BadRequest(w http.ResponseWriter, requestID, message string) {
	if message == "" {
		message = "Bad request"
	}
	writeError(w, http.StatusBadRequest, "BAD_REQUEST", message, requestID)
}

// Unauthorized returns a 401 Unauthorized response
func Unauthorized(w http.ResponseWriter, requestID string) {
	writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", requestID)
}

// NotFound returns a 404 Not Found response
func NotFound(w http.ResponseWriter, requestID, resource string) {
	message := "Resource not found"
	if resource != "" {
		message = resource + " not found"
	}
	writeError(w, http.StatusNotFound, "NOT_FOUND", message, requestID)
}

// UnprocessableEntity returns a 422 Unprocessable Entity response
func UnprocessableEntity(w http.ResponseWriter, requestID, message string) {
	if message == "" {
		message = "Validation failed"
	}
	writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message, requestID)
}

// InternalServerError returns a 500 Internal Server Error response
func InternalServerError(w http.ResponseWriter, requestID string) {
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", requestID)
}
