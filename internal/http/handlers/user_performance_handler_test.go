package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/responses"
)

type mockUserPerformanceService struct {
	response *domain.UserPerformanceResponse
	err      error
}

func (m *mockUserPerformanceService) Get(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.UserPerformanceResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestUserPerformanceHandler_Get_ReturnsAggregatedPayload(t *testing.T) {
	mockResponse := &domain.UserPerformanceResponse{
		User: domain.UserPerformanceIdentity{
			Username:    "ianfelps",
			DisplayName: "ianfelps",
		},
		Period: domain.Period{StartDate: "2026-01-01", EndDate: "2026-01-31"},
		Delivery: domain.UserDeliveryMetrics{
			Throughput: domain.Throughput{TotalIssuesDone: 7, AvgPerWeek: 1.75},
		},
		Quality: domain.UserQualityMetrics{
			Rework:    domain.ReworkMetrics{TotalReworkedIssues: 2},
			GhostWork: domain.GhostWorkMetrics{RatePct: 12.5},
		},
		WIP: domain.WipMetricsResponse{
			CurrentWIP: domain.CurrentWIP{QAReview: 1},
		},
	}

	svc := &mockUserPerformanceService{response: mockResponse}
	handler := NewUserPerformanceHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/ianfelps/performance?start_date=2026-01-01&end_date=2026-01-31", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var got domain.UserPerformanceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if got.User.Username != "ianfelps" {
		t.Fatalf("expected username ianfelps, got %s", got.User.Username)
	}
	if got.Delivery.Throughput.TotalIssuesDone != 7 {
		t.Fatalf("expected throughput total 7, got %d", got.Delivery.Throughput.TotalIssuesDone)
	}
	if got.Quality.GhostWork.RatePct != 12.5 {
		t.Fatalf("expected ghost work rate 12.5, got %f", got.Quality.GhostWork.RatePct)
	}
}

func TestUserPerformanceHandler_Get_MethodNotAllowed(t *testing.T) {
	svc := &mockUserPerformanceService{}
	handler := NewUserPerformanceHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/ianfelps/performance", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rr.Code)
	}
}

func TestUserPerformanceHandler_Get_MissingUsername(t *testing.T) {
	svc := &mockUserPerformanceService{}
	handler := NewUserPerformanceHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/performance", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var errResp responses.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Code != "BAD_REQUEST" {
		t.Fatalf("expected error code BAD_REQUEST, got %s", errResp.Code)
	}
	if errResp.Message != "username path parameter is required" {
		t.Fatalf("expected error message for missing username, got %s", errResp.Message)
	}
}

func TestUserPerformanceHandler_Get_ServiceError(t *testing.T) {
	svc := &mockUserPerformanceService{err: errors.New("service failure")}
	handler := NewUserPerformanceHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/ianfelps/performance", nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}

	var errResp responses.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Code != "INTERNAL_ERROR" {
		t.Fatalf("expected error code INTERNAL_ERROR, got %s", errResp.Code)
	}
}
