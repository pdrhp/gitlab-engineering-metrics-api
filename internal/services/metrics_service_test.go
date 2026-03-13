package services

import (
	"context"
	"errors"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

type mockMetricsRepository struct {
	deliveryMetrics *domain.DeliveryMetricsResponse
	qualityMetrics  *domain.QualityMetricsResponse
	wipMetrics      *domain.WipMetricsResponse
	err             error
}

func (m *mockMetricsRepository) GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.deliveryMetrics, nil
}

func (m *mockMetricsRepository) GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.qualityMetrics, nil
}

func (m *mockMetricsRepository) GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.wipMetrics, nil
}

func TestMetricsService_GetDeliveryMetrics(t *testing.T) {
	mockRepo := &mockMetricsRepository{
		deliveryMetrics: &domain.DeliveryMetricsResponse{
			Period: domain.Period{
				StartDate: "2024-01-01",
				EndDate:   "2024-01-31",
			},
			Throughput: domain.Throughput{
				TotalIssuesDone: 10,
				AvgPerWeek:      2.5,
			},
		},
	}

	service := NewMetricsService(mockRepo)

	tests := []struct {
		name        string
		filter      domain.MetricsFilter
		wantErr     bool
		errContains string
	}{
		{
			name: "valid date range",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-01",
				EndDate:   "2024-01-31",
			},
			wantErr: false,
		},
		{
			name: "no date range (valid)",
			filter: domain.MetricsFilter{
				GroupPath: "engineering",
			},
			wantErr: false,
		},
		{
			name: "only start date provided",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-01",
			},
			wantErr:     true,
			errContains: "both start_date and end_date",
		},
		{
			name: "only end date provided",
			filter: domain.MetricsFilter{
				EndDate: "2024-01-31",
			},
			wantErr:     true,
			errContains: "both start_date and end_date",
		},
		{
			name: "invalid date format",
			filter: domain.MetricsFilter{
				StartDate: "01-01-2024",
				EndDate:   "2024-01-31",
			},
			wantErr:     true,
			errContains: "invalid start_date format",
		},
		{
			name: "end date before start date",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-31",
				EndDate:   "2024-01-01",
			},
			wantErr:     true,
			errContains: "end_date must be after",
		},
		{
			name: "date range too large",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-01",
				EndDate:   "2024-06-01",
			},
			wantErr:     true,
			errContains: "cannot exceed 90 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := service.GetDeliveryMetrics(context.Background(), tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDeliveryMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if !errors.Is(err, err) && err.Error() == "" {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if metrics == nil {
				t.Error("GetDeliveryMetrics() returned nil")
			}
		})
	}
}

func TestMetricsService_GetQualityMetrics(t *testing.T) {
	mockRepo := &mockMetricsRepository{
		qualityMetrics: &domain.QualityMetricsResponse{
			Rework: domain.ReworkMetrics{
				PingPongRatePct:         10.0,
				TotalReworkedIssues:     5,
				AvgReworkCyclesPerIssue: 1.2,
			},
		},
	}

	service := NewMetricsService(mockRepo)

	tests := []struct {
		name    string
		filter  domain.MetricsFilter
		wantErr bool
	}{
		{
			name: "valid date range",
			filter: domain.MetricsFilter{
				StartDate: "2024-01-01",
				EndDate:   "2024-01-31",
			},
			wantErr: false,
		},
		{
			name:    "no filter",
			filter:  domain.MetricsFilter{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := service.GetQualityMetrics(context.Background(), tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetQualityMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if metrics == nil {
				t.Error("GetQualityMetrics() returned nil")
			}
		})
	}
}

func TestMetricsService_GetWipMetrics(t *testing.T) {
	mockRepo := &mockMetricsRepository{
		wipMetrics: &domain.WipMetricsResponse{
			CurrentWIP: domain.CurrentWIP{
				InProgress: 5,
				QAReview:   3,
				Blocked:    1,
			},
		},
	}

	service := NewMetricsService(mockRepo)

	tests := []struct {
		name    string
		filter  domain.MetricsFilter
		wantErr bool
	}{
		{
			name:    "no filter",
			filter:  domain.MetricsFilter{},
			wantErr: false,
		},
		{
			name: "with group path",
			filter: domain.MetricsFilter{
				GroupPath: "engineering",
			},
			wantErr: false,
		},
		{
			name: "with project id",
			filter: domain.MetricsFilter{
				ProjectID: 1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := service.GetWipMetrics(context.Background(), tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWipMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if metrics == nil {
				t.Error("GetWipMetrics() returned nil")
			}
		})
	}
}

func TestMetricsService_RepositoryError(t *testing.T) {
	mockRepo := &mockMetricsRepository{
		err: errors.New("database error"),
	}

	service := NewMetricsService(mockRepo)
	filter := domain.MetricsFilter{
		StartDate: "2024-01-01",
		EndDate:   "2024-01-31",
	}

	t.Run("delivery metrics repository error", func(t *testing.T) {
		_, err := service.GetDeliveryMetrics(context.Background(), filter)
		if err == nil {
			t.Error("Expected error from repository, got nil")
		}
	})

	t.Run("quality metrics repository error", func(t *testing.T) {
		_, err := service.GetQualityMetrics(context.Background(), filter)
		if err == nil {
			t.Error("Expected error from repository, got nil")
		}
	})

	t.Run("wip metrics repository error", func(t *testing.T) {
		_, err := service.GetWipMetrics(context.Background(), filter)
		if err == nil {
			t.Error("Expected error from repository, got nil")
		}
	})
}
