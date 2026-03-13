package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestUsersHandler_List(t *testing.T) {
	mockService := &mockCatalogService{
		users: []domain.User{
			{Username: "john_doe", DisplayName: "john_doe", ActiveIssues: 5, CompletedIssuesLast30Days: 10},
		},
	}

	handler := NewUsersHandler(mockService)

	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "GET returns users list",
			method:         http.MethodGet,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "POST not allowed",
			method:         http.MethodPost,
			queryParams:    "",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedCount:  0,
		},
		{
			name:           "GET with search param",
			method:         http.MethodGet,
			queryParams:    "?search=john",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with group_path param",
			method:         http.MethodGet,
			queryParams:    "?group_path=engineering",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/users"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response []domain.User
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if len(response) != tt.expectedCount {
					t.Errorf("Expected %d users, got %d", tt.expectedCount, len(response))
				}
			}
		})
	}
}

func TestUsersHandler_List_ServiceError(t *testing.T) {
	mockService := &mockCatalogService{
		err: errors.New("database error"),
	}

	handler := NewUsersHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestUsersHandler_List_InvalidSearch(t *testing.T) {
	mockService := &mockCatalogService{}

	handler := NewUsersHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/users?search=ab", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}
