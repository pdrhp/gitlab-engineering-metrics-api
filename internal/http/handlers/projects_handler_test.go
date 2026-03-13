package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

type mockCatalogService struct {
	projects []domain.Project
	groups   []domain.Group
	users    []domain.User
	err      error
}

func (m *mockCatalogService) ListProjects(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.projects, nil
}

func (m *mockCatalogService) ListGroups(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.groups, nil
}

func (m *mockCatalogService) ListUsers(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users, nil
}

func TestProjectsHandler_List(t *testing.T) {
	mockService := &mockCatalogService{
		projects: []domain.Project{
			{ID: 1, Name: "Test Project", Path: "group/project", TotalIssues: 10},
		},
	}

	handler := NewProjectsHandler(mockService)

	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "GET returns projects list",
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
			queryParams:    "?search=test",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/projects"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response []domain.Project
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if len(response) != tt.expectedCount {
					t.Errorf("Expected %d projects, got %d", tt.expectedCount, len(response))
				}
			}
		})
	}
}

func TestProjectsHandler_List_ServiceError(t *testing.T) {
	mockService := &mockCatalogService{
		err: errors.New("database error"),
	}

	handler := NewProjectsHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestProjectsHandler_List_InvalidSearch(t *testing.T) {
	mockService := &mockCatalogService{}

	handler := NewProjectsHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/projects?search=ab", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}
