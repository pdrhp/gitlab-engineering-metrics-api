package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	// Create a simple handler that checks for request ID in context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Expected request ID to be set in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RequestID middleware
	wrapped := RequestID(handler)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Execute
	wrapped.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Check that response has request ID header
	responseID := rr.Header().Get(RequestIDHeader)
	if responseID == "" {
		t.Error("Expected response to have X-Request-ID header")
	}
}

func TestRequestID_PreservesExistingID(t *testing.T) {
	existingID := "existing-request-id-123"

	// Create a handler that checks the request ID
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		if requestID != existingID {
			t.Errorf("Expected request ID %s, got %s", existingID, requestID)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RequestID middleware
	wrapped := RequestID(handler)

	// Create request with existing request ID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, existingID)
	rr := httptest.NewRecorder()

	// Execute
	wrapped.ServeHTTP(rr, req)

	// Check that response preserves the request ID
	responseID := rr.Header().Get(RequestIDHeader)
	if responseID != existingID {
		t.Errorf("Expected response to preserve request ID %s, got %s", existingID, responseID)
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	// Test with empty context (no request ID set)
	requestID := GetRequestID(nil)
	if requestID != "" {
		t.Errorf("Expected empty string for nil context, got %s", requestID)
	}
}
