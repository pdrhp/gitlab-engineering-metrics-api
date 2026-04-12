package repositories

import (
	"context"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestIndividualPerformanceRepository_GetAssigneeCycleTime(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIndividualPerformanceRepository(db)
	ctx := context.Background()

	tests := []struct {
		name     string
		username string
		filter   domain.MetricsFilter
		wantErr  bool
	}{
		{
			name:     "get assignee cycle time with no filter",
			username: "root",
			filter:   domain.MetricsFilter{},
			wantErr:  false,
		},
		{
			name:     "get assignee cycle time with date range",
			username: "root",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-01",
				EndDate:   "2024-12-31",
			},
			wantErr: false,
		},
		{
			name:     "get assignee cycle time with project filter",
			username: "root",
			filter: domain.MetricsFilter{
				ProjectID: 1,
			},
			wantErr: false,
		},
		{
			name:     "get assignee cycle time for non-existent user",
			username: "nonexistent-user-xyz123",
			filter:   domain.MetricsFilter{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetAssigneeCycleTime(ctx, tt.username, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAssigneeCycleTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result == nil {
				t.Error("GetAssigneeCycleTime() returned nil, expected slice")
			}
		})
	}
}

func TestIndividualPerformanceRepository_GetIndividualPerformanceMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIndividualPerformanceRepository(db)
	ctx := context.Background()

	tests := []struct {
		name     string
		username string
		filter   domain.MetricsFilter
		wantErr  bool
		wantNil  bool
	}{
		{
			name:     "get individual performance metrics - happy path",
			username: "root",
			filter:   domain.MetricsFilter{},
			wantErr:  false,
			wantNil:  false,
		},
		{
			name:     "get individual performance metrics with date range",
			username: "root",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-01",
				EndDate:   "2024-12-31",
			},
			wantErr: false,
			wantNil: false,
		},
		{
			name:     "get individual performance metrics with project filter",
			username: "root",
			filter: domain.MetricsFilter{
				ProjectID: 1,
			},
			wantErr: false,
			wantNil: false,
		},
		{
			name:     "get individual performance metrics for non-existent user",
			username: "nonexistent-user-xyz123",
			filter:   domain.MetricsFilter{},
			wantErr:  false,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetIndividualPerformanceMetrics(ctx, tt.username, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIndividualPerformanceMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil && result != nil {
				t.Errorf("GetIndividualPerformanceMetrics() expected nil, got %+v", result)
				return
			}
			if !tt.wantNil && result == nil && !tt.wantErr {
				t.Errorf("GetIndividualPerformanceMetrics() expected non-nil result, got nil")
				return
			}

			// Validate data structure for non-nil results
			if result != nil {
				if result.Username != tt.username {
					t.Errorf("Expected username %s, got %s", tt.username, result.Username)
				}
				if result.IssuesAssigned < result.IssuesContributed {
					t.Errorf("Expected IssuesAssigned (%d) >= IssuesContributed (%d) for fair attribution",
						result.IssuesAssigned, result.IssuesContributed)
				}
				if result.ActiveWorkPct < 0 || result.ActiveWorkPct > 100 {
					t.Errorf("Expected ActiveWorkPct between 0-100, got %f", result.ActiveWorkPct)
				}
				if result.TotalHoursAsAssignee < 0 {
					t.Errorf("Expected TotalHoursAsAssignee >= 0, got %f", result.TotalHoursAsAssignee)
				}
			}
		})
	}
}
