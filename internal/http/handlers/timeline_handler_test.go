package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestNewTimelineHandler(t *testing.T) {
	mockService := &mockIssuesService{}
	handler := NewTimelineHandler(mockService)

	if handler == nil {
		t.Error("Expected handler to not be nil")
	}

	if handler.service != mockService {
		t.Error("Expected handler to hold the service")
	}
}

func TestTimelineHandler_Get(t *testing.T) {
	mockService := &mockIssuesService{
		timelineResponse: &domain.IssueTimelineResponse{
			Issue: domain.IssueSummary{
				IssueID: 1,
				Title:   "Test Issue",
			},
			Timeline: []domain.TimelineEvent{
				{Type: "state_transition"},
			},
		},
	}

	handler := NewTimelineHandler(mockService)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "GET returns timeline",
			method:         http.MethodGet,
			path:           "/api/v1/issues/1/timeline",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST not allowed",
			method:         http.MethodPost,
			path:           "/api/v1/issues/1/timeline",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			handler.Get(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response domain.IssueTimelineResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if response.Issue.IssueID != 1 {
					t.Errorf("Expected issue_id 1, got %d", response.Issue.IssueID)
				}
			}
		})
	}
}

func TestTimelineHandler_Get_InvalidIssueID(t *testing.T) {
	mockService := &mockIssuesService{}
	handler := NewTimelineHandler(mockService)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "invalid issue_id format",
			path:           "/api/v1/issues/abc/timeline",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zero issue_id",
			path:           "/api/v1/issues/0/timeline",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative issue_id",
			path:           "/api/v1/issues/-1/timeline",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid path - missing timeline",
			path:           "/api/v1/issues/1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid path - wrong structure",
			path:           "/api/v1/issues",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			handler.Get(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestTimelineHandler_Get_NotFound(t *testing.T) {
	mockService := &mockIssuesService{
		err: errors.New("issue not found"),
	}

	handler := NewTimelineHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/999/timeline", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

func TestTimelineHandler_Get_InvalidIssueIDValidation(t *testing.T) {
	mockService := &mockIssuesService{
		err: errors.New("invalid issue ID"),
	}

	handler := NewTimelineHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/1/timeline", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", rr.Code)
	}
}

func TestTimelineHandler_Get_ServiceError(t *testing.T) {
	mockService := &mockIssuesService{
		err: errors.New("database connection failed"),
	}

	handler := NewTimelineHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/1/timeline", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestTimelineHandler_Get_ReturnsUnifiedIssueFields(t *testing.T) {
	mockService := &mockIssuesService{
		timelineResponse: &domain.IssueTimelineResponse{
			Issue: domain.IssueSummary{
				IssueID:       1,
				GitlabIssueID: 99123,
				IssueIID:      42,
				ProjectID:     101,
				ProjectPath:   "group/project",
				Title:         "Test Issue",
			},
			Timeline: []domain.TimelineEvent{
				{Type: "state_transition"},
			},
		},
	}

	handler := NewTimelineHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/1/timeline", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response domain.IssueTimelineResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return
	}

	// Verify unified identity fields
	if response.Issue.IssueID != 1 {
		t.Errorf("Expected issue_id 1, got %d", response.Issue.IssueID)
	}
	if response.Issue.GitlabIssueID != 99123 {
		t.Errorf("Expected gitlab_issue_id 99123, got %d", response.Issue.GitlabIssueID)
	}
	if response.Issue.IssueIID != 42 {
		t.Errorf("Expected issue_iid 42, got %d", response.Issue.IssueIID)
	}
	if response.Issue.ProjectID != 101 {
		t.Errorf("Expected project_id 101, got %d", response.Issue.ProjectID)
	}
	if response.Issue.ProjectPath != "group/project" {
		t.Errorf("Expected project_path 'group/project', got %s", response.Issue.ProjectPath)
	}
}
