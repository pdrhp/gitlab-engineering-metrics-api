# Refactor UserPerformanceService to Use IndividualPerformanceRepository Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor `UserPerformanceService` to integrate `IndividualPerformanceRepository` and populate the `IndividualPerformance` field in the response.

**Architecture:** Add `IndividualPerformanceRepository` as a dependency to `UserPerformanceService`, update the constructor to accept it, and modify the `Get` method to fetch individual performance metrics using the fair attribution model.

**Tech Stack:** Go 1.21+, sqlx, existing domain types (`IndividualPerformanceMetrics`, `UserPerformanceResponse`)

---

### Task 1: Add IndividualPerformanceRepository Interface to Service File

**Files:**
- Modify: `internal/services/user_performance_service.go:12-22`

- [ ] **Step 1: Add interface definition after line 22**

Add the interface definition for `IndividualPerformanceRepository`:

```go
// IndividualPerformanceRepository defines the contract for fair individual metrics
type IndividualPerformanceRepository interface {
	GetIndividualPerformanceMetrics(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error)
}
```

- [ ] **Step 2: Run go fmt to format the file**

```bash
go fmt ./internal/services/user_performance_service.go
```

Expected: File formatted successfully

- [ ] **Step 3: Run go vet to verify syntax**

```bash
go vet ./internal/services/user_performance_service.go
```

Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/services/user_performance_service.go
git commit -m "refactor: add IndividualPerformanceRepository interface to UserPerformanceService"
```

---

### Task 2: Update UserPerformanceService Struct and Constructor

**Files:**
- Modify: `internal/services/user_performance_service.go:24-36`

- [ ] **Step 1: Update struct to add individualPerfRepo field**

Change lines 25-28:

```go
type UserPerformanceService struct {
	usersRepo          UserLookupRepository
	metricsSvc         UserPerformanceMetricsService
	individualPerfRepo IndividualPerformanceRepository
}
```

- [ ] **Step 2: Update constructor to accept IndividualPerformanceRepository**

Change lines 31-36:

```go
func NewUserPerformanceService(
	usersRepo UserLookupRepository,
	metricsSvc UserPerformanceMetricsService,
	individualPerfRepo IndividualPerformanceRepository,
) *UserPerformanceService {
	return &UserPerformanceService{
		usersRepo:          usersRepo,
		metricsSvc:         metricsSvc,
		individualPerfRepo: individualPerfRepo,
	}
}
```

- [ ] **Step 3: Run go fmt to format the file**

```bash
go fmt ./internal/services/
```

Expected: File formatted successfully

- [ ] **Step 4: Run go vet to verify syntax**

```bash
go vet ./internal/services/
```

Expected: No errors (existing tests will fail due to missing argument - that's expected)

- [ ] **Step 5: Commit**

```bash
git add internal/services/user_performance_service.go
git commit -m "refactor: add individualPerfRepo field to UserPerformanceService struct and constructor"
```

---

### Task 3: Update Get Method to Fetch Individual Performance Metrics

**Files:**
- Modify: `internal/services/user_performance_service.go:38-93`

- [ ] **Step 1: Add individual performance fetch before building response**

After line 67 (after the `wip` metrics fetch), add:

```go
// Get fair individual performance metrics
individualPerf, err := s.individualPerfRepo.GetIndividualPerformanceMetrics(ctx, username, filter)
if err != nil {
	return nil, fmt.Errorf("failed to load individual performance metrics: %w", err)
}
```

- [ ] **Step 2: Add IndividualPerformance field to response**

Update the response struct (lines 69-92) to include the new field:

```go
return &domain.UserPerformanceResponse{
	User: domain.UserPerformanceIdentity{
		Username:                  user.Username,
		DisplayName:               user.DisplayName,
		ActiveIssues:              user.ActiveIssues,
		CompletedIssuesLast30Days: user.CompletedIssuesLast30Days,
	},
	Period: domain.Period{
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	},
	Delivery: domain.UserDeliveryMetrics{
		Throughput:       delivery.Throughput,
		SpeedMetricsDays: delivery.SpeedMetricsDays,
	},
	Quality: domain.UserQualityMetrics{
		Rework:        quality.Rework,
		GhostWork:     domain.GhostWorkMetrics{RatePct: quality.ProcessHealth.BypassRatePct},
		ProcessHealth: quality.ProcessHealth,
		Bottlenecks:   quality.Bottlenecks,
		Defects:       quality.Defects,
	},
	WIP:                   *wip,
	IndividualPerformance: individualPerf,
}, nil
```

- [ ] **Step 3: Run go fmt to format the file**

```bash
go fmt ./internal/services/
```

Expected: File formatted successfully

- [ ] **Step 4: Run go vet to verify syntax**

```bash
go vet ./internal/services/
```

Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/services/user_performance_service.go
git commit -m "refactor: populate IndividualPerformance field in Get method response"
```

---

### Task 4: Update Tests to Mock IndividualPerformanceRepository

**Files:**
- Modify: `internal/services/user_performance_service_test.go:11-52`
- Modify: `internal/services/user_performance_service_test.go:53-130`

- [ ] **Step 1: Add mock implementation for IndividualPerformanceRepository**

Add after line 50 (after the `mockUserPerformanceMetricsService` methods):

```go
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
```

- [ ] **Step 2: Update TestUserPerformanceService_Get_HappyPath to include mock**

Update the test setup (lines 53-80):

```go
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
```

- [ ] **Step 3: Add assertion for IndividualPerformance in happy path test**

After line 127 (after the WIP verification), add:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/services/ -v -run TestUserPerformanceService_Get_HappyPath
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/services/user_performance_service_test.go
git commit -m "test: add mock IndividualPerformanceRepository and update happy path test"
```

---

### Task 5: Add Test for Nil IndividualPerformance Case

**Files:**
- Modify: `internal/services/user_performance_service_test.go`

- [ ] **Step 1: Add test for when user has no individual performance metrics**

Add after line 249 (end of file):

```go
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
```

- [ ] **Step 2: Run the new test to verify it passes**

```bash
go test ./internal/services/ -v -run TestUserPerformanceService_Get_NoIndividualPerformance
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/services/user_performance_service_test.go
git commit -m "test: add test for nil IndividualPerformance case"
```

---

### Task 6: Update All Existing Tests to Pass New Constructor

**Files:**
- Modify: `internal/services/user_performance_service_test.go:131-249`

- [ ] **Step 1: Update TestUserPerformanceService_Get_UserNotFound**

Update the test (lines 131-147):

```go
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
```

- [ ] **Step 2: Update TestUserPerformanceService_Get_EmptyUsername**

Update the test (lines 148-168):

```go
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
```

- [ ] **Step 3: Update TestUserPerformanceService_Get_ServiceError**

Update the test (lines 169-190):

```go
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
```

- [ ] **Step 4: Update TestUserPerformanceService_Get_RepositoryError**

Update the test (lines 191-209):

```go
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
```

- [ ] **Step 5: Update TestUserPerformanceService_Get_FilterAssigneeSet**

Update the test (lines 210-249):

```go
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
```

- [ ] **Step 6: Run all tests to verify they pass**

```bash
go test ./internal/services/ -v -run TestUserPerformanceService
```

Expected: All tests PASS

- [ ] **Step 7: Run go fmt and go vet**

```bash
go fmt ./internal/services/
go vet ./internal/services/
```

Expected: No errors

- [ ] **Step 8: Commit**

```bash
git add internal/services/user_performance_service_test.go
git commit -m "test: update all existing tests to use new constructor signature"
```

---

### Task 7: Run Full Test Suite and Verify No Regressions

**Files:**
- None (verification only)

- [ ] **Step 1: Run all service tests**

```bash
go test ./internal/services/ -v
```

Expected: All tests PASS

- [ ] **Step 2: Run full build to verify compilation**

```bash
go build ./...
```

Expected: Build succeeds

- [ ] **Step 3: Check for any compilation errors in main.go or DI container**

```bash
go build -o /dev/null ./cmd/...
```

Expected: Build succeeds (there may be errors in DI/wiring code that needs updating - this is expected and will be fixed in a follow-up task)
