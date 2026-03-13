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

type mockGhostWorkService struct {
	response *domain.GhostWorkMetricsResponse
	err      error
}

func (m *mockGhostWorkService) GetGhostWorkMetrics(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestGhostWorkHandler_Get(t *testing.T) {
	mockService := &mockGhostWorkService{
		response: &domain.GhostWorkMetricsResponse{
			TotalIssues: 270,
			Period: domain.Period{
				StartDate: "2024-01-01",
				EndDate:   "2024-01-31",
			},
			Issues: []domain.GhostWorkIssue{
				{
					IssueIID:    371,
					ProjectPath: "apps-expo/dflegal-expo",
					IssueTitle:  "Test Issue",
					Assignees:   []string{"gabriel"},
					FromState:   "BACKLOG",
					ToState:     "QA_REVIEW",
				},
			},
			TransitionAnalysis: []domain.GhostWorkTransitionSummary{
				{FromState: "BACKLOG", ToState: "QA_REVIEW", Count: 432},
				{FromState: "BACKLOG", ToState: "DONE", Count: 26},
			},
			BreakdownByUser: []domain.GhostWorkUserBreakdown{
				{Username: "nevez", GhostWorkCount: 50, IssueIIDs: []int{371, 369}},
			},
			Page:       1,
			PageSize:   25,
			TotalPages: 11,
		},
	}

	handler := NewGhostWorkHandler(mockService)

	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		checkResponse  bool
	}{
		{
			name:           "GET returns ghost work metrics",
			method:         http.MethodGet,
			queryParams:    "?start_date=2024-01-01&end_date=2024-01-31",
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name:           "GET with pagination",
			method:         http.MethodGet,
			queryParams:    "?start_date=2024-01-01&end_date=2024-01-31&page=2&page_size=50",
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name:           "POST not allowed",
			method:         http.MethodPost,
			queryParams:    "",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
		},
		{
			name:           "GET with no filter",
			method:         http.MethodGet,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/metrics/ghost-work"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.Get(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkResponse && tt.expectedStatus == http.StatusOK {
				var response domain.GhostWorkMetricsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if response.TotalIssues != 270 {
					t.Errorf("Expected total_issues 270, got %d", response.TotalIssues)
				}
			}
		})
	}
}

func TestGhostWorkHandler_Get_ValidationErrors(t *testing.T) {
	mockService := &mockGhostWorkService{}
	handler := NewGhostWorkHandler(mockService)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "invalid date format",
			queryParams:    "?start_date=invalid&end_date=2024-01-31",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "only start_date provided",
			queryParams:    "?start_date=2024-01-01",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "only end_date provided",
			queryParams:    "?end_date=2024-01-31",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/ghost-work"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.Get(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestGhostWorkHandler_Get_ServiceError(t *testing.T) {
	mockService := &mockGhostWorkService{
		err: errors.New("database error"),
	}
	handler := NewGhostWorkHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/ghost-work?start_date=2024-01-01&end_date=2024-01-31", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestGhostWorkHandler_Get_InvalidProjectID(t *testing.T) {
	mockService := &mockGhostWorkService{
		response: &domain.GhostWorkMetricsResponse{TotalIssues: 0},
	}
	handler := NewGhostWorkHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/ghost-work?project_id=abc", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	// Invalid project_id should be ignored, not cause an error
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestGhostWorkHandler_Get_InvalidPagination(t *testing.T) {
	mockService := &mockGhostWorkService{
		response: &domain.GhostWorkMetricsResponse{TotalIssues: 0},
	}
	handler := NewGhostWorkHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/ghost-work?page=abc&page_size=def", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	// Invalid pagination values should be ignored, not cause an error
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestGhostWorkHandler_Get_ReturnsUnifiedIssueFields(t *testing.T) {
	mockService := &mockGhostWorkService{
		response: &domain.GhostWorkMetricsResponse{
			TotalIssues: 1,
			Issues: []domain.GhostWorkIssue{
				{
					IssueID:       1,
					GitlabIssueID: 99123,
					IssueIID:      42,
					ProjectID:     101,
					ProjectPath:   "group/project",
					IssueTitle:    "Ghost work issue",
					FromState:     "BACKLOG",
					ToState:       "DONE",
				},
			},
		},
	}
	handler := NewGhostWorkHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/ghost-work", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response domain.GhostWorkMetricsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return
	}

	if len(response.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(response.Issues))
		return
	}

	issue := response.Issues[0]

	// Verify unified identity fields
	if issue.IssueID != 1 {
		t.Errorf("Expected issue_id 1, got %d", issue.IssueID)
	}
	if issue.GitlabIssueID != 99123 {
		t.Errorf("Expected gitlab_issue_id 99123, got %d", issue.GitlabIssueID)
	}
	if issue.ProjectID != 101 {
		t.Errorf("Expected project_id 101, got %d", issue.ProjectID)
	}
	if issue.ProjectPath != "group/project" {
		t.Errorf("Expected project_path 'group/project', got %s", issue.ProjectPath)
	}
}
