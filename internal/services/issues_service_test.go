package services

import (
	"context"
	"errors"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

// Mock repositories
type mockIssuesRepository struct {
	response *domain.IssuesListResponse
	err      error
}

func (m *mockIssuesRepository) List(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

type mockTimelineRepository struct {
	response *domain.IssueTimelineResponse
	err      error
}

func (m *mockTimelineRepository) GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestNewIssuesService(t *testing.T) {
	issuesRepo := &mockIssuesRepository{}
	timelineRepo := &mockTimelineRepository{}

	service := NewIssuesService(issuesRepo, timelineRepo)

	if service == nil {
		t.Error("Expected service to not be nil")
	}

	if service.issuesRepo != issuesRepo {
		t.Error("Expected service to hold the issues repository")
	}

	if service.timelineRepo != timelineRepo {
		t.Error("Expected service to hold the timeline repository")
	}
}

func TestIssuesService_ListIssues(t *testing.T) {
	mockIssuesRepo := &mockIssuesRepository{
		response: &domain.IssuesListResponse{
			Items: []domain.IssueListItem{
				{IssueID: 1, Title: "Test Issue"},
			},
			Page:     1,
			PageSize: 20,
			Total:    1,
		},
	}

	mockTimelineRepo := &mockTimelineRepository{}
	service := NewIssuesService(mockIssuesRepo, mockTimelineRepo)

	tests := []struct {
		name        string
		filter      domain.IssuesFilter
		wantErr     bool
		errContains string
	}{
		{
			name:   "valid filter with defaults",
			filter: domain.IssuesFilter{},
		},
		{
			name: "valid filter with custom pagination",
			filter: domain.IssuesFilter{
				Page:     2,
				PageSize: 50,
			},
		},
		{
			name: "valid filter with all filters",
			filter: domain.IssuesFilter{
				ProjectID: 123,
				GroupPath: "group/subgroup",
				Assignee:  "john.doe",
				State:     "DONE",
				Page:      1,
				PageSize:  25,
			},
		},
		{
			name: "page size at maximum",
			filter: domain.IssuesFilter{
				PageSize: 100,
			},
		},
		{
			name: "negative page number",
			filter: domain.IssuesFilter{
				Page: -1,
			},
			wantErr:     true,
			errContains: "page must be greater than or equal to 0",
		},
		{
			name: "negative page size",
			filter: domain.IssuesFilter{
				PageSize: -1,
			},
			wantErr:     true,
			errContains: "page_size must be greater than or equal to 0",
		},
		{
			name: "page size exceeds maximum",
			filter: domain.IssuesFilter{
				PageSize: 101,
			},
			wantErr:     true,
			errContains: "page_size cannot exceed 100",
		},
		{
			name: "negative project_id",
			filter: domain.IssuesFilter{
				ProjectID: -1,
			},
			wantErr:     true,
			errContains: "project_id must be a positive integer",
		},
		{
			name: "negative issue_id",
			filter: domain.IssuesFilter{
				IssueID: -1,
			},
			wantErr:     true,
			errContains: "issue_id must be a positive integer",
		},
		{
			name: "negative gitlab_issue_id",
			filter: domain.IssuesFilter{
				GitlabIssueID: -1,
			},
			wantErr:     true,
			errContains: "gitlab_issue_id must be a positive integer",
		},
		{
			name: "negative issue_iid",
			filter: domain.IssuesFilter{
				IssueIID: -1,
			},
			wantErr:     true,
			errContains: "issue_iid must be a positive integer",
		},
		{
			name: "valid filter with new identity fields",
			filter: domain.IssuesFilter{
				IssueID:       123,
				GitlabIssueID: 456,
				IssueIID:      789,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ListIssues(context.Background(), tt.filter)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected result, got nil")
				return
			}

			// Verify defaults were applied
			if tt.filter.Page <= 0 && result.Page != 1 {
				t.Errorf("Expected default page 1, got %d", result.Page)
			}
			if tt.filter.PageSize <= 0 && result.PageSize != 20 {
				t.Errorf("Expected default page_size 20, got %d", result.PageSize)
			}
		})
	}
}

func TestIssuesService_ListIssues_RepositoryError(t *testing.T) {
	mockIssuesRepo := &mockIssuesRepository{
		err: errors.New("database connection failed"),
	}
	mockTimelineRepo := &mockTimelineRepository{}
	service := NewIssuesService(mockIssuesRepo, mockTimelineRepo)

	filter := domain.IssuesFilter{}
	_, err := service.ListIssues(context.Background(), filter)

	if err == nil {
		t.Error("Expected error from repository, got nil")
	}
}

func TestIssuesService_GetTimeline(t *testing.T) {
	mockIssuesRepo := &mockIssuesRepository{}
	mockTimelineRepo := &mockTimelineRepository{
		response: &domain.IssueTimelineResponse{
			Issue: domain.IssueSummary{
				IssueID: 1,
				Title:   "Test Issue",
			},
			Timeline: []domain.TimelineEvent{
				{Type: "state_transition"},
			},
		},
	}
	service := NewIssuesService(mockIssuesRepo, mockTimelineRepo)

	tests := []struct {
		name        string
		issueID     int
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid issue ID",
			issueID: 1,
		},
		{
			name:        "zero issue ID",
			issueID:     0,
			wantErr:     true,
			errContains: "invalid issue ID",
		},
		{
			name:        "negative issue ID",
			issueID:     -1,
			wantErr:     true,
			errContains: "invalid issue ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetTimeline(context.Background(), tt.issueID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected result, got nil")
			}
		})
	}
}

func TestIssuesService_GetTimeline_NotFound(t *testing.T) {
	mockIssuesRepo := &mockIssuesRepository{}
	mockTimelineRepo := &mockTimelineRepository{
		err: errors.New("issue not found"),
	}
	service := NewIssuesService(mockIssuesRepo, mockTimelineRepo)

	_, err := service.GetTimeline(context.Background(), 999)

	if err == nil {
		t.Error("Expected error for not found issue, got nil")
	}

	if !containsString(err.Error(), "issue not found") {
		t.Errorf("Expected 'issue not found' error, got: %v", err)
	}
}

func TestIssuesService_GetTimeline_RepositoryError(t *testing.T) {
	mockIssuesRepo := &mockIssuesRepository{}
	mockTimelineRepo := &mockTimelineRepository{
		err: errors.New("database error"),
	}
	service := NewIssuesService(mockIssuesRepo, mockTimelineRepo)

	_, err := service.GetTimeline(context.Background(), 1)

	if err == nil {
		t.Error("Expected error from repository, got nil")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(s[:len(substr)] == substr) ||
		(s[len(s)-len(substr):] == substr) ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
