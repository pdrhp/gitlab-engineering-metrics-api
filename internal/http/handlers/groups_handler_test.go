package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestGroupsHandler_List(t *testing.T) {
	mockService := &mockCatalogService{
		groups: []domain.Group{
			{GroupPath: "engineering", ProjectCount: 5, TotalIssues: 100},
		},
	}

	handler := NewGroupsHandler(mockService)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "GET returns groups list",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "POST not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/groups", nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response []domain.Group
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if len(response) != tt.expectedCount {
					t.Errorf("Expected %d groups, got %d", tt.expectedCount, len(response))
				}
			}
		})
	}
}

func TestGroupsHandler_List_ServiceError(t *testing.T) {
	mockService := &mockCatalogService{
		err: errors.New("database error"),
	}

	handler := NewGroupsHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/groups", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}
