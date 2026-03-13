package integration

import (
	"errors"
	"net/http"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestMetrics_Delivery_Success(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.DeliveryMetrics = &domain.DeliveryMetricsResponse{
		Period: domain.Period{
			StartDate: "2024-01-01",
			EndDate:   "2024-01-31",
		},
		Throughput: domain.Throughput{
			TotalIssuesDone: 42,
			AvgPerWeek:      10.5,
		},
		SpeedMetricsDays: domain.SpeedMetrics{
			LeadTime: &domain.AvgP85Metric{
				Avg: 5.2,
				P85: 8.5,
			},
			CycleTime: &domain.AvgP85Metric{
				Avg: 3.1,
				P85: 5.0,
			},
		},
		BreakdownByAssignee: []domain.AssigneeBreakdown{
			{
				Assignee:        "john.doe",
				IssuesDelivered: 15,
				AvgCycleTime:    2.8,
			},
		},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertContentType(t, resp, "application/json")

	var metrics domain.DeliveryMetricsResponse
	ParseResponse(t, resp, &metrics)

	if metrics.Throughput.TotalIssuesDone != 42 {
		t.Errorf("Expected 42 issues done, got %d", metrics.Throughput.TotalIssuesDone)
	}

	if metrics.SpeedMetricsDays.LeadTime.Avg != 5.2 {
		t.Errorf("Expected lead time avg of 5.2, got %f", metrics.SpeedMetricsDays.LeadTime.Avg)
	}
}

func TestMetrics_Delivery_WithDateRange(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.DeliveryMetrics = &domain.DeliveryMetricsResponse{
		Period: domain.Period{
			StartDate: "2024-01-01",
			EndDate:   "2024-01-31",
		},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery?start_date=2024-01-01&end_date=2024-01-31", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
}

func TestMetrics_Delivery_InvalidDateFormat(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.Err = errors.New("invalid start_date format, expected YYYY-MM-DD")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery?start_date=invalid&end_date=2024-01-31", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestMetrics_Delivery_MissingDateRange(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.Err = errors.New("both start_date and end_date are required when filtering by date")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery?start_date=2024-01-01", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestMetrics_Delivery_InvalidDateRange(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.Err = errors.New("end_date must be after start_date")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery?start_date=2024-02-01&end_date=2024-01-01", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestMetrics_Delivery_DateRangeTooLarge(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.Err = errors.New("date range cannot exceed 90 days")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery?start_date=2024-01-01&end_date=2024-12-31", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestMetrics_Delivery_Unauthorized(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusUnauthorized)
}

func TestMetrics_Delivery_ServiceError(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.Err = errors.New("database error")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusInternalServerError)
}

func TestMetrics_Quality_Success(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.QualityMetrics = &domain.QualityMetricsResponse{
		Rework: domain.ReworkMetrics{
			PingPongRatePct:         15.5,
			TotalReworkedIssues:     8,
			AvgReworkCyclesPerIssue: 1.2,
		},
		ProcessHealth: domain.ProcessHealthMetrics{
			BypassRatePct:        5.0,
			FirstTimePassRatePct: 85.0,
		},
		Bottlenecks: domain.BottleneckMetrics{
			TotalBlockedTimeHours:       120.5,
			AvgBlockedTimePerIssueHours: 2.8,
		},
		Defects: domain.DefectMetrics{
			BugRatioPct: 10.5,
		},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/quality", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertContentType(t, resp, "application/json")

	var metrics domain.QualityMetricsResponse
	ParseResponse(t, resp, &metrics)

	if metrics.Rework.PingPongRatePct != 15.5 {
		t.Errorf("Expected ping pong rate of 15.5%%, got %f", metrics.Rework.PingPongRatePct)
	}

	if metrics.ProcessHealth.FirstTimePassRatePct != 85.0 {
		t.Errorf("Expected first time pass rate of 85%%, got %f", metrics.ProcessHealth.FirstTimePassRatePct)
	}
}

func TestMetrics_Quality_Unauthorized(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeRequest(t, ts, http.MethodGet, "/api/v1/metrics/quality", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusUnauthorized)
}

func TestMetrics_WIP_Success(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.WipMetrics = &domain.WipMetricsResponse{
		CurrentWIP: domain.CurrentWIP{
			InProgress: 12,
			QAReview:   5,
			Blocked:    2,
		},
		AgingWIP: []domain.AgingIssue{
			{
				IssueID:       1,
				GitlabIssueID: 99123,
				IssueIID:      123,
				ProjectID:     101,
				ProjectPath:   "group/project",
				Title:         "Fix critical bug",
				Assignees:     []string{"john.doe"},
				CurrentState:  "in_progress",
				DaysInState:   5,
				WarningFlag:   true,
			},
		},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/wip", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertContentType(t, resp, "application/json")

	var metrics domain.WipMetricsResponse
	ParseResponse(t, resp, &metrics)

	if metrics.CurrentWIP.InProgress != 12 {
		t.Errorf("Expected 12 issues in progress, got %d", metrics.CurrentWIP.InProgress)
	}

	if len(metrics.AgingWIP) != 1 {
		t.Errorf("Expected 1 aging issue, got %d", len(metrics.AgingWIP))
	}

	// Verify unified identity fields are present
	if metrics.AgingWIP[0].IssueID != 1 {
		t.Errorf("Expected issue_id 1, got %d", metrics.AgingWIP[0].IssueID)
	}
	if metrics.AgingWIP[0].GitlabIssueID != 99123 {
		t.Errorf("Expected gitlab_issue_id 99123, got %d", metrics.AgingWIP[0].GitlabIssueID)
	}
	if metrics.AgingWIP[0].ProjectPath != "group/project" {
		t.Errorf("Expected project_path 'group/project', got %s", metrics.AgingWIP[0].ProjectPath)
	}
}

func TestMetrics_WIP_Unauthorized(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeRequest(t, ts, http.MethodGet, "/api/v1/metrics/wip", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusUnauthorized)
}

func TestMetrics_WithProjectID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.DeliveryMetrics = &domain.DeliveryMetricsResponse{
		Period: domain.Period{
			StartDate: "2024-01-01",
			EndDate:   "2024-01-31",
		},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery?project_id=123", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
}

func TestMetrics_WithGroupPath(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.QualityMetrics = &domain.QualityMetricsResponse{}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/quality?group_path=engineering", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
}

func TestMetrics_WithAssignee(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.MetricsService.WipMetrics = &domain.WipMetricsResponse{}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/wip?assignee=john.doe", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
}
