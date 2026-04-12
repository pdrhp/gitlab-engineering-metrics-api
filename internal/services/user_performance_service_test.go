package services

import (
	"context"
	"errors"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

// Mock implementations for testing

type mockUserLookupRepository struct {
	user *domain.User
	err  error
}

func (m *mockUserLookupRepository) GetByUsername(ctx context.Context, username string, filter domain.CatalogFilter) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

type mockUserPerformanceMetricsService struct {
	delivery *domain.DeliveryMetricsResponse
	quality  *domain.QualityMetricsResponse
	wip      *domain.WipMetricsResponse
	err      error
}

func (m *mockUserPerformanceMetricsService) GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.delivery, nil
}

func (m *mockUserPerformanceMetricsService) GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.quality, nil
}

func (m *mockUserPerformanceMetricsService) GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.wip, nil
}

type mockIndividualPerformanceRepository struct {
	metrics *domain.IndividualPerformanceMetrics
	err     error
}

func (m *mockIndividualPerformanceRepository) GetIndividualPerformanceMetrics(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metrics, nil
}

func TestUserPerformanceService_Get_HappyPath(t *testing.T) {
	usersRepo := &mockUserLookupRepository{
		user: &domain.User{
			Username:                  "ianfelps",
			DisplayName:               "ianfelps",
			ActiveIssues:              24,
			CompletedIssuesLast30Days: 103,
		},
	}
	metricsSvc := &mockUserPerformanceMetricsService{
		delivery: &domain.DeliveryMetricsResponse{
			Throughput: domain.Throughput{TotalIssuesDone: 20, AvgPerWeek: 1.75},
			SpeedMetricsDays: domain.SpeedMetrics{
				LeadTime:  &domain.AvgP85Metric{Avg: 20.15, P85: 41.22},
				CycleTime: &domain.AvgP85Metric{Avg: 11.40, P85: 19.03},
			},
		},
		quality: &domain.QualityMetricsResponse{
			Rework:        domain.ReworkMetrics{PingPongRatePct: 35, TotalReworkedIssues: 7, AvgReworkCyclesPerIssue: 1.2},
			ProcessHealth: domain.ProcessHealthMetrics{BypassRatePct: 5, FirstTimePassRatePct: 65},
			Bottlenecks:   domain.BottleneckMetrics{TotalBlockedTimeHours: 48, AvgBlockedTimePerIssueHours: 6},
			Defects:       domain.DefectMetrics{BugRatioPct: 10},
		},
		wip: &domain.WipMetricsResponse{
			CurrentWIP: domain.CurrentWIP{InProgress: 3, QAReview: 2, Blocked: 0},
			AgingWIP:   []domain.AgingIssue{},
		},
	}
	individualPerfRepo := &mockIndividualPerformanceRepository{
		metrics: &domain.IndividualPerformanceMetrics{
			Username:               "ianfelps",
			IssuesAssigned:         15,
			IssuesContributed:      12,
			TotalActiveCycleHours:  120.5,
			AvgActiveCyclePerIssue: 10.04,
			TotalDevHours:          85.2,
			TotalQAHours:           20.3,
			TotalBlockedHours:      10.0,
			TotalBacklogHours:      5.0,
			ActiveWorkPct:          87.5,
			TotalHoursAsAssignee:   120.5,
			P50ActiveCycleHours:    9.5,
			P95ActiveCycleHours:    18.2,
		},
	}

	svc := NewUserPerformanceService(usersRepo, metricsSvc, individualPerfRepo)
	filter := domain.MetricsFilter{StartDate: "2026-01-01", EndDate: "2026-01-31"}
	got, err := svc.Get(context.Background(), "ianfelps", filter)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil {
		t.Fatal("expected response, got nil")
	}

	// Verify user identity
	if got.User.Username != "ianfelps" {
		t.Errorf("expected username 'ianfelps', got %s", got.User.Username)
	}
	if got.User.DisplayName != "ianfelps" {
		t.Errorf("expected display_name 'ianfelps', got %s", got.User.DisplayName)
	}
	if got.User.ActiveIssues != 24 {
		t.Errorf("expected active_issues 24, got %d", got.User.ActiveIssues)
	}
	if got.User.CompletedIssuesLast30Days != 103 {
		t.Errorf("expected completed_issues_last_30_days 103, got %d", got.User.CompletedIssuesLast30Days)
	}

	// Verify period
	if got.Period.StartDate != "2026-01-01" {
		t.Errorf("expected start_date '2026-01-01', got %s", got.Period.StartDate)
	}
	if got.Period.EndDate != "2026-01-31" {
		t.Errorf("expected end_date '2026-01-31', got %s", got.Period.EndDate)
	}

	// Verify delivery metrics
	if got.Delivery.Throughput.TotalIssuesDone != 20 {
		t.Errorf("expected total_issues_done 20, got %d", got.Delivery.Throughput.TotalIssuesDone)
	}

	// Verify ghost work rate is set from bypass rate
	if got.Quality.GhostWork.RatePct != 5 {
		t.Errorf("expected ghost_work.rate_pct 5, got %f", got.Quality.GhostWork.RatePct)
	}

	// Verify WIP metrics
	if got.WIP.CurrentWIP.QAReview != 2 {
		t.Errorf("expected wip.current_wip.qa_review 2, got %d", got.WIP.CurrentWIP.QAReview)
	}

	// Verify individual performance metrics
	if got.IndividualPerformance == nil {
		t.Fatal("expected IndividualPerformance, got nil")
	}
	if got.IndividualPerformance.Username != "ianfelps" {
		t.Errorf("expected username 'ianfelps', got %s", got.IndividualPerformance.Username)
	}
	if got.IndividualPerformance.IssuesAssigned != 15 {
		t.Errorf("expected issues_assigned 15, got %d", got.IndividualPerformance.IssuesAssigned)
	}
	if got.IndividualPerformance.IssuesContributed != 12 {
		t.Errorf("expected issues_contributed 12, got %d", got.IndividualPerformance.IssuesContributed)
	}
}

func TestUserPerformanceService_Get_UserNotFound(t *testing.T) {
	usersRepo := &mockUserLookupRepository{
		user: nil, // User not found
	}
	metricsSvc := &mockUserPerformanceMetricsService{}
	individualPerfRepo := &mockIndividualPerformanceRepository{}

	svc := NewUserPerformanceService(usersRepo, metricsSvc, individualPerfRepo)
	_, err := svc.Get(context.Background(), "missing-user", domain.MetricsFilter{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "user not found" {
		t.Errorf("expected 'user not found' error, got %v", err)
	}
}

func TestUserPerformanceService_Get_EmptyUsername(t *testing.T) {
	usersRepo := &mockUserLookupRepository{}
	metricsSvc := &mockUserPerformanceMetricsService{}
	individualPerfRepo := &mockIndividualPerformanceRepository{}

	svc := NewUserPerformanceService(usersRepo, metricsSvc, individualPerfRepo)
	_, err := svc.Get(context.Background(), "", domain.MetricsFilter{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "username is required" {
		t.Errorf("expected 'username is required' error, got %v", err)
	}

	// Test with whitespace only
	_, err = svc.Get(context.Background(), "   ", domain.MetricsFilter{})
	if err == nil {
		t.Fatal("expected error for whitespace username, got nil")
	}
}

func TestUserPerformanceService_Get_ServiceError(t *testing.T) {
	usersRepo := &mockUserLookupRepository{
		user: &domain.User{
			Username:    "ianfelps",
			DisplayName: "ianfelps",
		},
	}
	metricsSvc := &mockUserPerformanceMetricsService{
		err: errors.New("database connection failed"),
	}
	individualPerfRepo := &mockIndividualPerformanceRepository{}

	svc := NewUserPerformanceService(usersRepo, metricsSvc, individualPerfRepo)
	_, err := svc.Get(context.Background(), "ianfelps", domain.MetricsFilter{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "database connection failed" {
		t.Errorf("expected service error to propagate, got %v", err)
	}
}

func TestUserPerformanceService_Get_RepositoryError(t *testing.T) {
	usersRepo := &mockUserLookupRepository{
		err: errors.New("repository error"),
	}
	metricsSvc := &mockUserPerformanceMetricsService{}
	individualPerfRepo := &mockIndividualPerformanceRepository{}

	svc := NewUserPerformanceService(usersRepo, metricsSvc, individualPerfRepo)
	_, err := svc.Get(context.Background(), "ianfelps", domain.MetricsFilter{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "failed to load user: repository error"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got %v", expectedMsg, err)
	}
}

func TestUserPerformanceService_Get_FilterAssigneeSet(t *testing.T) {
	usersRepo := &mockUserLookupRepository{
		user: &domain.User{
			Username: "ianfelps",
		},
	}

	var capturedFilter domain.MetricsFilter
	metricsSvc := &mockUserPerformanceMetricsService{
		delivery: &domain.DeliveryMetricsResponse{},
		quality:  &domain.QualityMetricsResponse{},
		wip:      &domain.WipMetricsResponse{},
	}
	individualPerfRepo := &mockIndividualPerformanceRepository{
		metrics: &domain.IndividualPerformanceMetrics{},
	}

	// Override to capture the filter
	metricsSvcWithCapture := &mockMetricsSvcWithCapture{
		mockUserPerformanceMetricsService: metricsSvc,
		captureFilter: func(f domain.MetricsFilter) {
			capturedFilter = f
		},
	}

	svc := NewUserPerformanceService(usersRepo, metricsSvcWithCapture, individualPerfRepo)
	filter := domain.MetricsFilter{StartDate: "2026-01-01", EndDate: "2026-01-31"}
	svc.Get(context.Background(), "ianfelps", filter)

	if capturedFilter.Assignee != "ianfelps" {
		t.Errorf("expected assignee filter to be set to 'ianfelps', got %s", capturedFilter.Assignee)
	}
}

// Helper type to capture the filter
type mockMetricsSvcWithCapture struct {
	*mockUserPerformanceMetricsService
	captureFilter func(domain.MetricsFilter)
}

func (m *mockMetricsSvcWithCapture) GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error) {
	m.captureFilter(filter)
	return m.mockUserPerformanceMetricsService.GetDeliveryMetrics(ctx, filter)
}

func TestUserPerformanceService_Get_NoIndividualPerformance(t *testing.T) {
	usersRepo := &mockUserLookupRepository{
		user: &domain.User{
			Username:    "new-user",
			DisplayName: "New User",
		},
	}
	metricsSvc := &mockUserPerformanceMetricsService{
		delivery: &domain.DeliveryMetricsResponse{
			Throughput:       domain.Throughput{},
			SpeedMetricsDays: domain.SpeedMetrics{},
		},
		quality: &domain.QualityMetricsResponse{},
		wip:     &domain.WipMetricsResponse{},
	}
	individualPerfRepo := &mockIndividualPerformanceRepository{
		metrics: nil, // User has no performance metrics yet
	}

	svc := NewUserPerformanceService(usersRepo, metricsSvc, individualPerfRepo)
	filter := domain.MetricsFilter{StartDate: "2026-01-01", EndDate: "2026-01-31"}
	got, err := svc.Get(context.Background(), "new-user", filter)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil {
		t.Fatal("expected response, got nil")
	}
	// IndividualPerformance should be nil when user has no metrics
	if got.IndividualPerformance != nil {
		t.Errorf("expected IndividualPerformance to be nil, got %v", got.IndividualPerformance)
	}
}
