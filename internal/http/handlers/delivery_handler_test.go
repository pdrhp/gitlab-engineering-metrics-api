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

type mockMetricsService struct {
	deliveryMetrics *domain.DeliveryMetricsResponse
	qualityMetrics  *domain.QualityMetricsResponse
	wipMetrics      *domain.WipMetricsResponse
	deliveryTrend   *domain.DeliveryTrendResponse
	err             error
}

func (m *mockMetricsService) GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.deliveryMetrics, nil
}

func (m *mockMetricsService) GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.qualityMetrics, nil
}

func (m *mockMetricsService) GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.wipMetrics, nil
}

func (m *mockMetricsService) GetDeliveryTrendMetrics(ctx context.Context, filter domain.DeliveryTrendFilter) (*domain.DeliveryTrendResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.deliveryTrend, nil
}

func TestDeliveryHandler_Get(t *testing.T) {
	mockService := &mockMetricsService{
		deliveryMetrics: &domain.DeliveryMetricsResponse{
			Period: domain.Period{
				StartDate: "2024-01-01",
				EndDate:   "2024-01-31",
			},
			Throughput: domain.Throughput{
				TotalIssuesDone: 10,
				AvgPerWeek:      2.5,
			},
			SpeedMetricsDays: domain.SpeedMetrics{
				LeadTime: &domain.AvgP85Metric{
					Avg: 5.2,
					P85: 8.5,
				},
				CycleTime: &domain.AvgP85Metric{
					Avg: 3.1,
					P85: 5.2,
				},
			},
		},
	}

	handler := NewDeliveryHandler(mockService)

	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		expectedBody   bool
	}{
		{
			name:           "GET returns delivery metrics",
			method:         http.MethodGet,
			queryParams:    "?start_date=2024-01-01&end_date=2024-01-31",
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
			name:           "GET with all filters",
			method:         http.MethodGet,
			queryParams:    "?start_date=2024-01-01&end_date=2024-01-31&group_path=engineering&project_id=1&assignee=user@example.com",
			expectedStatus: http.StatusOK,
			expectedBody:   true,
		},
		{
			name:           "GET with no filters",
			method:         http.MethodGet,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/metrics/delivery"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.Get(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedBody {
				var response domain.DeliveryMetricsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if response.Throughput.TotalIssuesDone != 10 {
					t.Errorf("Expected 10 total issues, got %d", response.Throughput.TotalIssuesDone)
				}
			}
		})
	}
}

func TestDeliveryHandler_Get_ServiceError(t *testing.T) {
	mockService := &mockMetricsService{
		err: errors.New("database error"),
	}

	handler := NewDeliveryHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/delivery", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestDeliveryHandler_Get_ValidationError(t *testing.T) {
	// Service returns validation error
	mockService := &mockMetricsService{
		err: errors.New("date range cannot exceed 366 days"),
	}

	handler := NewDeliveryHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/delivery?start_date=2024-01-01&end_date=2024-12-31", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}
