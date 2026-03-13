package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestQualityHandler_Get(t *testing.T) {
	mockService := &mockMetricsService{
		qualityMetrics: &domain.QualityMetricsResponse{
			Rework: domain.ReworkMetrics{
				PingPongRatePct:         10.5,
				TotalReworkedIssues:     5,
				AvgReworkCyclesPerIssue: 1.2,
			},
			ProcessHealth: domain.ProcessHealthMetrics{
				BypassRatePct:        5.0,
				FirstTimePassRatePct: 85.0,
			},
			Bottlenecks: domain.BottleneckMetrics{
				TotalBlockedTimeHours:       48.0,
				AvgBlockedTimePerIssueHours: 8.0,
			},
			Defects: domain.DefectMetrics{
				BugRatioPct: 3.5,
			},
		},
	}

	handler := NewQualityHandler(mockService)

	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		expectedBody   bool
	}{
		{
			name:           "GET returns quality metrics",
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
			name:           "GET with group filter",
			method:         http.MethodGet,
			queryParams:    "?start_date=2024-01-01&end_date=2024-01-31&group_path=engineering",
			expectedStatus: http.StatusOK,
			expectedBody:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/metrics/quality"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.Get(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedBody {
				var response domain.QualityMetricsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}
				if response.Rework.TotalReworkedIssues != 5 {
					t.Errorf("Expected 5 reworked issues, got %d", response.Rework.TotalReworkedIssues)
				}
			}
		})
	}
}

func TestQualityHandler_Get_ServiceError(t *testing.T) {
	mockService := &mockMetricsService{
		err: errors.New("database error"),
	}

	handler := NewQualityHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/quality", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}
