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

type mockIssuesService struct {
	listResponse     *domain.IssuesListResponse
	timelineResponse *domain.IssueTimelineResponse
	err              error
}

func (m *mockIssuesService) ListIssues(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.listResponse, nil
}

func (m *mockIssuesService) GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.timelineResponse, nil
}

func TestNewIssuesHandler(t *testing.T) {
	mockService := &mockIssuesService{}
	handler := NewIssuesHandler(mockService)

	if handler == nil {
		t.Error("Expected handler to not be nil")
	}

	if handler.service != mockService {
		t.Error("Expected handler to hold the service")
	}
}

func TestIssuesHandler_List(t *testing.T) {
	mockService := &mockIssuesService{
		listResponse: &domain.IssuesListResponse{
			Items: []domain.IssueListItem{
				{IssueID: 1, Title: "Test Issue", ProjectID: 123},
			},
			Page:     1,
			PageSize: 20,
			Total:    1,
		},
	}

	handler := NewIssuesHandler(mockService)

	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "GET returns issues list",
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
			name:           "GET with project_id filter",
			method:         http.MethodGet,
			queryParams:    "?project_id=123",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with group_path filter",
			method:         http.MethodGet,
			queryParams:    "?group_path=group/subgroup",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with assignee filter",
			method:         http.MethodGet,
			queryParams:    "?assignee=john.doe",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with state filter",
			method:         http.MethodGet,
			queryParams:    "?state=DONE",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with page parameter",
			method:         http.MethodGet,
			queryParams:    "?page=2",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with page_size parameter",
			method:         http.MethodGet,
			queryParams:    "?page_size=50",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with all parameters",
			method:         http.MethodGet,
			queryParams:    "?project_id=123&group_path=group&assignee=john&state=DONE&page=1&page_size=25",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with metric_flag bypass",
			method:         http.MethodGet,
			queryParams:    "?metric_flag=bypass",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with invalid metric_flag",
			method:         http.MethodGet,
			queryParams:    "?metric_flag=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:           "GET with issue_id filter",
			method:         http.MethodGet,
			queryParams:    "?issue_id=123",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with gitlab_issue_id filter",
			method:         http.MethodGet,
			queryParams:    "?gitlab_issue_id=456",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with issue_iid filter",
			method:         http.MethodGet,
			queryParams:    "?issue_iid=789",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "GET with all identity filters",
			method:         http.MethodGet,
			queryParams:    "?issue_id=1&gitlab_issue_id=100&issue_iid=10",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/issues"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response domain.IssuesListResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if len(response.Items) != tt.expectedCount {
					t.Errorf("Expected %d issues, got %d", tt.expectedCount, len(response.Items))
				}
			}
		})
	}
}

func TestIssuesHandler_List_InvalidProjectID(t *testing.T) {
	mockService := &mockIssuesService{}
	handler := NewIssuesHandler(mockService)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "invalid project_id format",
			queryParams:    "?project_id=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative project_id",
			queryParams:    "?project_id=-1",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/issues"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestIssuesHandler_List_InvalidIssueID(t *testing.T) {
	mockService := &mockIssuesService{}
	handler := NewIssuesHandler(mockService)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "invalid issue_id format",
			queryParams:    "?issue_id=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zero issue_id",
			queryParams:    "?issue_id=0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative issue_id",
			queryParams:    "?issue_id=-1",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/issues"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestIssuesHandler_List_InvalidGitlabIssueID(t *testing.T) {
	mockService := &mockIssuesService{}
	handler := NewIssuesHandler(mockService)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "invalid gitlab_issue_id format",
			queryParams:    "?gitlab_issue_id=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zero gitlab_issue_id",
			queryParams:    "?gitlab_issue_id=0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative gitlab_issue_id",
			queryParams:    "?gitlab_issue_id=-1",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/issues"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestIssuesHandler_List_InvalidIssueIID(t *testing.T) {
	mockService := &mockIssuesService{}
	handler := NewIssuesHandler(mockService)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "invalid issue_iid format",
			queryParams:    "?issue_iid=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zero issue_iid",
			queryParams:    "?issue_iid=0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative issue_iid",
			queryParams:    "?issue_iid=-1",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/issues"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestIssuesHandler_List_InvalidPageParams(t *testing.T) {
	mockService := &mockIssuesService{}
	handler := NewIssuesHandler(mockService)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "invalid page format",
			queryParams:    "?page=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative page",
			queryParams:    "?page=-1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid page_size format",
			queryParams:    "?page_size=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative page_size",
			queryParams:    "?page_size=-1",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/issues"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestIssuesHandler_List_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		serviceErr     error
		expectedStatus int
		errContains    string
	}{
		{
			name:           "page_size exceeds maximum",
			serviceErr:     errors.New("page_size cannot exceed 100"),
			expectedStatus: http.StatusUnprocessableEntity,
			errContains:    "page_size cannot exceed 100",
		},
		{
			name:           "invalid page number",
			serviceErr:     errors.New("page must be greater than or equal to 0"),
			expectedStatus: http.StatusUnprocessableEntity,
			errContains:    "page must be greater than or equal to 0",
		},
		{
			name:           "invalid project_id",
			serviceErr:     errors.New("project_id must be a positive integer"),
			expectedStatus: http.StatusUnprocessableEntity,
			errContains:    "project_id must be a positive integer",
		},
		{
			name:           "invalid issue_id",
			serviceErr:     errors.New("issue_id must be a positive integer"),
			expectedStatus: http.StatusUnprocessableEntity,
			errContains:    "issue_id must be a positive integer",
		},
		{
			name:           "invalid gitlab_issue_id",
			serviceErr:     errors.New("gitlab_issue_id must be a positive integer"),
			expectedStatus: http.StatusUnprocessableEntity,
			errContains:    "gitlab_issue_id must be a positive integer",
		},
		{
			name:           "invalid issue_iid",
			serviceErr:     errors.New("issue_iid must be a positive integer"),
			expectedStatus: http.StatusUnprocessableEntity,
			errContains:    "issue_iid must be a positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockIssuesService{err: tt.serviceErr}
			handler := NewIssuesHandler(mockService)

			req := httptest.NewRequest(http.MethodGet, "/issues", nil)
			rr := httptest.NewRecorder()

			handler.List(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestIssuesHandler_List_ServiceError(t *testing.T) {
	mockService := &mockIssuesService{
		err: errors.New("database connection failed"),
	}

	handler := NewIssuesHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/issues", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}
