package responses

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBadRequest(t *testing.T) {
	rr := httptest.NewRecorder()
	BadRequest(rr, "test-request-id", "Invalid input")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	var response ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if response.Code != "BAD_REQUEST" {
		t.Errorf("Expected code 'BAD_REQUEST', got '%s'", response.Code)
	}

	if response.Message != "Invalid input" {
		t.Errorf("Expected message 'Invalid input', got '%s'", response.Message)
	}

	if response.RequestID != "test-request-id" {
		t.Errorf("Expected request ID 'test-request-id', got '%s'", response.RequestID)
	}
}

func TestBadRequest_EmptyMessage(t *testing.T) {
	rr := httptest.NewRecorder()
	BadRequest(rr, "test-request-id", "")

	var response ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if response.Message != "Bad request" {
		t.Errorf("Expected default message 'Bad request', got '%s'", response.Message)
	}
}

func TestUnauthorized(t *testing.T) {
	rr := httptest.NewRecorder()
	Unauthorized(rr, "test-request-id")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if response.Code != "UNAUTHORIZED" {
		t.Errorf("Expected code 'UNAUTHORIZED', got '%s'", response.Code)
	}

	if response.Message != "Authentication required" {
		t.Errorf("Expected message 'Authentication required', got '%s'", response.Message)
	}

	if response.RequestID != "test-request-id" {
		t.Errorf("Expected request ID 'test-request-id', got '%s'", response.RequestID)
	}
}

func TestNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	NotFound(rr, "test-request-id", "User")

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if response.Code != "NOT_FOUND" {
		t.Errorf("Expected code 'NOT_FOUND', got '%s'", response.Code)
	}

	if response.Message != "User not found" {
		t.Errorf("Expected message 'User not found', got '%s'", response.Message)
	}
}

func TestNotFound_EmptyResource(t *testing.T) {
	rr := httptest.NewRecorder()
	NotFound(rr, "test-request-id", "")

	var response ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if response.Message != "Resource not found" {
		t.Errorf("Expected default message 'Resource not found', got '%s'", response.Message)
	}
}

func TestUnprocessableEntity(t *testing.T) {
	rr := httptest.NewRecorder()
	UnprocessableEntity(rr, "test-request-id", "Field 'email' is invalid")

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, rr.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if response.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected code 'VALIDATION_ERROR', got '%s'", response.Code)
	}

	if response.Message != "Field 'email' is invalid" {
		t.Errorf("Expected message 'Field 'email' is invalid', got '%s'", response.Message)
	}
}

func TestUnprocessableEntity_EmptyMessage(t *testing.T) {
	rr := httptest.NewRecorder()
	UnprocessableEntity(rr, "test-request-id", "")

	var response ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if response.Message != "Validation failed" {
		t.Errorf("Expected default message 'Validation failed', got '%s'", response.Message)
	}
}

func TestInternalServerError(t *testing.T) {
	rr := httptest.NewRecorder()
	InternalServerError(rr, "test-request-id")

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if response.Code != "INTERNAL_ERROR" {
		t.Errorf("Expected code 'INTERNAL_ERROR', got '%s'", response.Code)
	}

	if response.Message != "Internal server error" {
		t.Errorf("Expected message 'Internal server error', got '%s'", response.Message)
	}
}

func TestErrorResponse_JSONStructure(t *testing.T) {
	rr := httptest.NewRecorder()
	BadRequest(rr, "req-123", "Something went wrong")

	body := rr.Body.String()
	if !strings.Contains(body, `"code"`) {
		t.Error("Expected JSON to contain 'code' field")
	}
	if !strings.Contains(body, `"message"`) {
		t.Error("Expected JSON to contain 'message' field")
	}
	if !strings.Contains(body, `"request_id"`) {
		t.Error("Expected JSON to contain 'request_id' field")
	}
}

func TestErrorResponse_OmitsEmptyRequestID(t *testing.T) {
	rr := httptest.NewRecorder()
	BadRequest(rr, "", "Something went wrong")

	var response ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.RequestID != "" {
		t.Errorf("Expected empty request ID, got '%s'", response.RequestID)
	}
}
