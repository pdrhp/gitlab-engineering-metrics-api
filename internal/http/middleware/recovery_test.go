package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRecovery_CatchesPanic(t *testing.T) {
	// Create a logger that writes to a buffer for testing
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	// Wrap with RequestID and Recovery middleware
	wrapped := RequestID(Recovery(logger)(panicHandler))

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Execute - should not panic
	wrapped.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}

	// Check response body contains error
	body := rr.Body.String()
	if !strings.Contains(body, "INTERNAL_ERROR") {
		t.Errorf("Expected error response to contain INTERNAL_ERROR, got: %s", body)
	}
}

func TestRecovery_NormalRequest(t *testing.T) {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a normal handler
	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with Recovery middleware
	wrapped := Recovery(logger)(normalHandler)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Execute
	wrapped.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Check response body
	body := rr.Body.String()
	if body != "OK" {
		t.Errorf("Expected response body 'OK', got: %s", body)
	}
}
