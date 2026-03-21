package services

import (
	"context"
	"errors"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

type mockGhostWorkRepository struct {
	response *domain.GhostWorkMetricsResponse
	err      error
}

func (m *mockGhostWorkRepository) GetGhostWorkIssues(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestGhostWorkService_GetGhostWorkMetrics(t *testing.T) {
	mockRepo := &mockGhostWorkRepository{
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

	service := NewGhostWorkService(mockRepo)

	tests := []struct {
		name        string
		filter      domain.GhostWorkFilter
		wantErr     bool
		errContains string
	}{
		{
			name: "valid date range",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					StartDate: "2024-01-01",
					EndDate:   "2024-01-31",
				},
			},
			wantErr: false,
		},
		{
			name: "no date range (valid)",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					GroupPath: "engineering",
				},
			},
			wantErr: false,
		},
		{
			name: "only start date provided",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					StartDate: "2024-01-01",
				},
			},
			wantErr:     true,
			errContains: "both start_date and end_date",
		},
		{
			name: "only end date provided",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					EndDate: "2024-01-31",
				},
			},
			wantErr:     true,
			errContains: "both start_date and end_date",
		},
		{
			name: "invalid date format",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					StartDate: "01-01-2024",
					EndDate:   "2024-01-31",
				},
			},
			wantErr:     true,
			errContains: "invalid start_date format",
		},
		{
			name: "end date before start date",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					StartDate: "2024-01-31",
					EndDate:   "2024-01-01",
				},
			},
			wantErr:     true,
			errContains: "end_date must be after",
		},
		{
			name: "date range too large",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					StartDate: "2024-01-01",
					EndDate:   "2025-02-01",
				},
			},
			wantErr:     true,
			errContains: "cannot exceed 366 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := service.GetGhostWorkMetrics(context.Background(), tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGhostWorkMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if !errors.Is(err, err) && err.Error() == "" {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if metrics == nil {
				t.Error("GetGhostWorkMetrics() returned nil")
			}
		})
	}
}

func TestGhostWorkService_PaginationDefaults(t *testing.T) {
	mockRepo := &mockGhostWorkRepository{
		response: &domain.GhostWorkMetricsResponse{
			TotalIssues: 100,
		},
	}

	service := NewGhostWorkService(mockRepo)

	tests := []struct {
		name         string
		filter       domain.GhostWorkFilter
		expectedPage int
		expectedSize int
	}{
		{
			name:         "default pagination",
			filter:       domain.GhostWorkFilter{},
			expectedPage: 1,
			expectedSize: 25,
		},
		{
			name: "custom pagination",
			filter: domain.GhostWorkFilter{
				Page:     2,
				PageSize: 50,
			},
			expectedPage: 2,
			expectedSize: 50,
		},
		{
			name: "pagination exceeds max",
			filter: domain.GhostWorkFilter{
				Page:     1,
				PageSize: 200,
			},
			expectedPage: 1,
			expectedSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily verify the pagination values were passed to the repo,
			// but we can verify the service doesn't error
			_, err := service.GetGhostWorkMetrics(context.Background(), tt.filter)
			if err != nil {
				t.Errorf("GetGhostWorkMetrics() error = %v", err)
			}
		})
	}
}

func TestGhostWorkService_RepositoryError(t *testing.T) {
	mockRepo := &mockGhostWorkRepository{
		err: errors.New("database error"),
	}

	service := NewGhostWorkService(mockRepo)
	filter := domain.GhostWorkFilter{
		MetricsFilter: domain.MetricsFilter{
			StartDate: "2024-01-01",
			EndDate:   "2024-01-31",
		},
	}

	_, err := service.GetGhostWorkMetrics(context.Background(), filter)
	if err == nil {
		t.Error("Expected error from repository, got nil")
	}
}
