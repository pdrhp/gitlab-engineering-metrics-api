package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestWipHandler_Get(t *testing.T) {
	mockService := &mockMetricsService{
		wipMetrics: &domain.WipMetricsResponse{
			CurrentWIP: domain.CurrentWIP{
				InProgress: 5,
				QAReview:   3,
				Blocked:    1,
			},
			AgingWIP: []domain.AgingIssue{
				{
					IssueIID:     123,
					Title:        "Issue in progress",
					CurrentState: "IN_PROGRESS",
					DaysInState:  5,
					WarningFlag:  false,
				},
				{
					IssueIID:     124,
					Title:        "Issue in QA",
					CurrentState: "QA_REVIEW",
					DaysInState:  10,
					WarningFlag:  true,
				},
			},
		},
	}

	handler := NewWipHandler(mockService)

	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		expectedBody   bool
	}{
		{
			name:           "GET returns WIP metrics",
			method:         http.MethodGet,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody:   true,
		},
		{
			name:           "POST not allowed",
			method:         http.MethodPost,
			queryParams:    "",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   false,
		},
		{
			name:           "GET with group filter",
			method:         http.MethodGet,
			queryParams:    "?group_path=engineering",
			expectedStatus: http.StatusOK,
			expectedBody:   true,
		},
		{
			name:           "GET with project filter",
			method:         http.MethodGet,
			queryParams:    "?project_id=1",
			expectedStatus: http.StatusOK,
			expectedBody:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/metrics/wip"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.Get(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedBody {
				var response domain.WipMetricsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if response.CurrentWIP.InProgress != 5 {
					t.Errorf("Expected 5 in progress, got %d", response.CurrentWIP.InProgress)
				}
				if len(response.AgingWIP) != 2 {
					t.Errorf("Expected 2 aging issues, got %d", len(response.AgingWIP))
				}
			}
		})
	}
}

func TestWipHandler_Get_ServiceError(t *testing.T) {
	mockService := &mockMetricsService{
		err: errors.New("database error"),
	}

	handler := NewWipHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/wip", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestWipHandler_Get_AgingWIPIncludesIssueIdentityFields(t *testing.T) {
	mockService := &mockMetricsService{
		wipMetrics: &domain.WipMetricsResponse{
			CurrentWIP: domain.CurrentWIP{
				InProgress: 5,
				QAReview:   3,
				Blocked:    1,
			},
			AgingWIP: []domain.AgingIssue{
				{
					IssueID:       1,
					GitlabIssueID: 99123,
					IssueIID:      42,
					ProjectID:     101,
					ProjectPath:   "group/project",
					Title:         "Aging issue",
					CurrentState:  "IN_PROGRESS",
					DaysInState:   10,
					WarningFlag:   true,
				},
			},
		},
	}

	handler := NewWipHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/wip", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response domain.WipMetricsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return
	}

	if len(response.AgingWIP) != 1 {
		t.Errorf("Expected 1 aging issue, got %d", len(response.AgingWIP))
		return
	}

	issue := response.AgingWIP[0]

	// Verify unified identity fields
	if issue.IssueID != 1 {
		t.Errorf("Expected issue_id 1, got %d", issue.IssueID)
	}
	if issue.GitlabIssueID != 99123 {
		t.Errorf("Expected gitlab_issue_id 99123, got %d", issue.GitlabIssueID)
	}
	if issue.ProjectPath != "group/project" {
		t.Errorf("Expected project_path 'group/project', got %s", issue.ProjectPath)
	}
}
