package repositories

import (
	"context"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestMetricsRepository_GetDeliveryMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		filter  domain.MetricsFilter
		wantErr bool
	}{
		{
			name:    "get delivery metrics with no filter",
			filter:  domain.MetricsFilter{},
			wantErr: false,
		},
		{
			name: "get delivery metrics with date range",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-01",
				EndDate:   "2024-12-31",
			},
			wantErr: false,
		},
		{
			name: "get delivery metrics with group path",
			filter: domain.MetricsFilter{
				GroupPath: "engineering",
				StartDate: "2024-01-01",
				EndDate:   "2024-12-31",
			},
			wantErr: false,
		},
		{
			name: "get delivery metrics with project id",
			filter: domain.MetricsFilter{
				ProjectID: 1,
				StartDate: "2024-01-01",
				EndDate:   "2024-12-31",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := repo.GetDeliveryMetrics(ctx, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("MetricsRepository.GetDeliveryMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if metrics == nil {
				t.Error("MetricsRepository.GetDeliveryMetrics() returned nil")
			}
		})
	}
}

func TestMetricsRepository_GetQualityMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		filter  domain.MetricsFilter
		wantErr bool
	}{
		{
			name:    "get quality metrics with no filter",
			filter:  domain.MetricsFilter{},
			wantErr: false,
		},
		{
			name: "get quality metrics with date range",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-01",
				EndDate:   "2024-12-31",
			},
			wantErr: false,
		},
		{
			name: "get quality metrics with assignee filter",
			filter: domain.MetricsFilter{
				Assignee:  "user@example.com",
				StartDate: "2024-01-01",
				EndDate:   "2024-12-31",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := repo.GetQualityMetrics(ctx, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("MetricsRepository.GetQualityMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if metrics == nil {
				t.Error("MetricsRepository.GetQualityMetrics() returned nil")
			}
		})
	}
}

func TestMetricsRepository_GetWipMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		filter  domain.MetricsFilter
		wantErr bool
	}{
		{
			name:    "get WIP metrics with no filter",
			filter:  domain.MetricsFilter{},
			wantErr: false,
		},
		{
			name: "get WIP metrics with group path",
			filter: domain.MetricsFilter{
				GroupPath: "engineering",
			},
			wantErr: false,
		},
		{
			name: "get WIP metrics with project id",
			filter: domain.MetricsFilter{
				ProjectID: 1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := repo.GetWipMetrics(ctx, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("MetricsRepository.GetWipMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if metrics == nil {
				t.Error("MetricsRepository.GetWipMetrics() returned nil")
			}
		})
	}
}

func TestMetricsRepository_BuildFilterConditions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)

	tests := []struct {
		name             string
		filter           domain.MetricsFilter
		expectedCondsLen int
		expectedArgsLen  int
	}{
		{
			name:             "empty filter",
			filter:           domain.MetricsFilter{},
			expectedCondsLen: 0,
			expectedArgsLen:  0,
		},
		{
			name: "all filters set",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-01",
				EndDate:   "2024-12-31",
				GroupPath: "engineering",
				ProjectID: 1,
				Assignee:  "user@example.com",
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

func TestMetricsRepository_BuildFilterConditions_AssigneeUsesJSONB(t *testing.T) {
	repo := &MetricsRepository{}

	conditions, args := repo.buildFilterConditions(domain.MetricsFilter{Assignee: "john.doe"}, 0)

	if len(conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(conditions))
	}

	want := "((jsonb_typeof(assignees) = 'array' AND assignees ? $1) OR (jsonb_typeof(assignees) = 'object' AND assignees ? 'current' AND (assignees->'current') ? $1))"
	if conditions[0] != want {
		t.Fatalf("unexpected condition. got=%q want=%q", conditions[0], want)
	}

	if len(args) != 1 || args[0] != "john.doe" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
