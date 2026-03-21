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

type mockDeliveryTrendService struct {
	deliveryTrend *domain.DeliveryTrendResponse
	err           error
}

func (m *mockDeliveryTrendService) GetDeliveryTrendMetrics(ctx context.Context, filter domain.DeliveryTrendFilter) (*domain.DeliveryTrendResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.deliveryTrend, nil
}

func TestDeliveryTrendHandler_Get(t *testing.T) {
	mockService := &mockDeliveryTrendService{
		deliveryTrend: &domain.DeliveryTrendResponse{
			Bucket:   "week",
			Timezone: "UTC",
			Period: domain.Period{
				StartDate: "2026-02-01",
				EndDate:   "2026-03-01",
			},
			Items: []domain.DeliveryTrendPoint{
				{
					BucketStart: "2026-02-02",
					BucketEnd:   "2026-02-08",
					BucketLabel: "2026-W06",
					Throughput:  domain.DeliveryTrendThroughput{TotalIssuesDone: 14},
				},
			},
		},
	}

	handler := NewDeliveryTrendHandler(mockService)

	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		expectedBody   bool
	}{
		{
			name:           "GET returns delivery trend metrics",
			method:         http.MethodGet,
			queryParams:    "?start_date=2026-02-01&end_date=2026-03-01",
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
			queryParams:    "?start_date=2026-02-01&end_date=2026-03-01&group_path=engineering&project_id=1&assignee=user@example.com&bucket=week&timezone=UTC",
			expectedStatus: http.StatusOK,
			expectedBody:   true,
		},
		{
			name:           "GET with invalid project_id",
			method:         http.MethodGet,
			queryParams:    "?start_date=2026-02-01&end_date=2026-03-01&project_id=abc",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   false,
		},
		{
			name:           "GET with invalid include_empty_buckets",
			method:         http.MethodGet,
			queryParams:    "?start_date=2026-02-01&end_date=2026-03-01&include_empty_buckets=maybe",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/metrics/delivery/trend"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.Get(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedBody {
				var response domain.DeliveryTrendResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if response.Bucket != "week" {
					t.Errorf("Expected bucket 'week', got %s", response.Bucket)
				}
			}
		})
	}
}

func TestDeliveryTrendHandler_Get_ServiceError(t *testing.T) {
	mockService := &mockDeliveryTrendService{
		err: errors.New("database error"),
	}

	handler := NewDeliveryTrendHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/delivery/trend?start_date=2026-02-01&end_date=2026-03-01", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestDeliveryTrendHandler_Get_ValidationError(t *testing.T) {
	mockService := &mockDeliveryTrendService{
		err: errors.New("date range cannot exceed 366 days"),
	}

	handler := NewDeliveryTrendHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/delivery/trend?start_date=2025-01-01&end_date=2026-12-31", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestDeliveryTrendHandler_Get_UnprocessableEntity(t *testing.T) {
	mockService := &mockDeliveryTrendService{
		err: errors.New("project_id 275 does not belong to group_path web"),
	}

	handler := NewDeliveryTrendHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/delivery/trend?start_date=2026-02-01&end_date=2026-03-01&project_id=275&group_path=web", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", rr.Code)
	}
}
