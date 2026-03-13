package repositories

import (
	"context"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestGhostWorkRepository_GetGhostWorkIssues(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewGhostWorkRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		filter  domain.GhostWorkFilter
		wantErr bool
	}{
		{
			name:    "get ghost work issues with no filter",
			filter:  domain.GhostWorkFilter{},
			wantErr: false,
		},
		{
			name: "get ghost work issues with date range",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					StartDate: "2024-01-01",
					EndDate:   "2024-12-31",
				},
			},
			wantErr: false,
		},
		{
			name: "get ghost work issues with assignee filter",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					Assignee:  "test-user",
					StartDate: "2024-01-01",
					EndDate:   "2024-12-31",
				},
			},
			wantErr: false,
		},
		{
			name: "get ghost work issues with pagination",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					StartDate: "2024-01-01",
					EndDate:   "2024-12-31",
				},
				Page:     1,
				PageSize: 10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetGhostWorkIssues(ctx, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGhostWorkIssues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result == nil {
				t.Error("GetGhostWorkIssues() returned nil")
			}
		})
	}
}

func TestGhostWorkRepository_BuildFilterConditions(t *testing.T) {
	repo := &GhostWorkRepository{}

	tests := []struct {
		name             string
		filter           domain.GhostWorkFilter
		expectedCondsLen int
		expectedArgsLen  int
	}{
		{
			name:             "empty filter",
			filter:           domain.GhostWorkFilter{},
			expectedCondsLen: 0,
			expectedArgsLen:  0,
		},
		{
			name: "all filters set",
			filter: domain.GhostWorkFilter{
				MetricsFilter: domain.MetricsFilter{
					StartDate: "2024-01-01",
					EndDate:   "2024-12-31",
					GroupPath: "engineering",
					ProjectID: 1,
					Assignee:  "user@example.com",
				},
			},
			expectedCondsLen: 5,
			expectedArgsLen:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions, args := repo.buildFilterConditions(tt.filter, 0)
			if len(conditions) != tt.expectedCondsLen {
				t.Errorf("expected %d conditions, got %d", tt.expectedCondsLen, len(conditions))
			}
			if len(args) != tt.expectedArgsLen {
				t.Errorf("expected %d args, got %d", tt.expectedArgsLen, len(args))
			}
		})
	}
}
