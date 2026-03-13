package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab-engineering-metrics-api/internal/auth"
)

func TestAuth_ValidCredentials(t *testing.T) {
	// Setup validator with test credentials
	creds := map[string]string{
		"test-client": "test-secret",
	}
	validator := auth.NewValidator(creds)

	// Create handler that checks client info
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientInfo := GetClientInfo(r.Context())
		if clientInfo == nil {
			t.Error("Expected client info to be set in context")
		}
		if clientInfo.ClientID != "test-client" {
			t.Errorf("Expected client ID 'test-client', got '%s'", clientInfo.ClientID)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RequestID and Auth middleware
	wrapped := RequestID(Auth(validator)(handler))

	// Create request with valid credentials
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(ClientIDHeader, "test-client")
	req.Header.Set(ClientSecretHeader, "test-secret")
	rr := httptest.NewRecorder()

	// Execute
	wrapped.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestAuth_InvalidCredentials(t *testing.T) {
	// Setup validator with test credentials
	creds := map[string]string{
		"test-client": "test-secret",
	}
	validator := auth.NewValidator(creds)

	// Create simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for invalid credentials")
	})

	// Wrap with RequestID and Auth middleware
	wrapped := RequestID(Auth(validator)(handler))

	// Create request with invalid credentials
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(ClientIDHeader, "test-client")
	req.Header.Set(ClientSecretHeader, "wrong-secret")
	rr := httptest.NewRecorder()

	// Execute
	wrapped.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	// Check response body contains error
	body := rr.Body.String()
	if body == "" {
		t.Error("Expected error response body")
	}
}

func TestAuth_MissingCredentials(t *testing.T) {
	// Setup validator
	creds := map[string]string{
		"test-client": "test-secret",
	}
	validator := auth.NewValidator(creds)

	// Create simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for missing credentials")
	})

	// Wrap with RequestID and Auth middleware
	wrapped := RequestID(Auth(validator)(handler))

	// Create request without credentials
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Execute
	wrapped.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestGetClientInfo_EmptyContext(t *testing.T) {
	// Test with empty context (no client info set)
	clientInfo := GetClientInfo(nil)
	if clientInfo != nil {
		t.Errorf("Expected nil for nil context, got %v", clientInfo)
	}
}
