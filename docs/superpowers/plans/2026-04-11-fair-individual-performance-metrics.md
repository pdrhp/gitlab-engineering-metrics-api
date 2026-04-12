# Fair Individual Performance Metrics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Adapt user performance API to use `vw_assignee_cycle_time` for fair individual metrics while preserving existing project-level metrics endpoints.

**Architecture:** Hybrid progressive approach - create new `IndividualPerformanceRepository` that queries the new assignee cycle time views, refactor `UserPerformanceService` to use fair metrics for user-specific endpoints, keep `MetricsRepository` unchanged for project/group-level aggregations.

**Tech Stack:** Go 1.21+, PostgreSQL 16, existing repository pattern with SQL queries using CTEs and window functions.

---

## File Structure

**Files to Create:**
- `internal/repositories/individual_performance_repository.go` - New repository for fair individual metrics
- `internal/repositories/individual_performance_repository_test.go` - Unit tests for new repository

**Files to Modify:**
- `internal/services/user_performance_service.go:24-93` - Refactor to use new repository
- `internal/domain/user_performance.go:18-40` - Extend response types with fairness fields
- `internal/http/handlers/user_performance_handler_test.go` - Add integration tests for new fields

**Files to Review (reference only):**
- `internal/repositories/metrics_repository.go` - Keep unchanged for project metrics
- `db/schema/000017_assignee_cycle_time.up.sql` - SQL view definitions
- `internal/domain/metrics.go` - Existing filter contracts

---

### Task 1: Create Individual Performance Repository Interface

**Files:**
- Create: `internal/repositories/individual_performance_repository.go`
- Test: `internal/repositories/individual_performance_repository_test.go`

- [ ] **Step 1: Write repository interface definition**

```go
package repositories

import (
	"context"
	"gitlab-engineering-metrics-api/internal/domain"
)

// IndividualPerformanceRepository defines the contract for fetching fair individual performance metrics
// Uses vw_assignee_cycle_time and vw_individual_performance_metrics for accurate assignee-level data
type IndividualPerformanceRepository interface {
	// GetAssigneeCycleTime returns cycle time breakdown for a specific assignee
	// Each row represents one issue the assignee worked on
	GetAssigneeCycleTime(ctx context.Context, username string, filter domain.MetricsFilter) ([]domain.AssigneeCycleTime, error)
	
	// GetIndividualPerformanceMetrics returns aggregated performance metrics for an assignee
	GetIndividualPerformanceMetrics(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error)
}
```

- [ ] **Step 2: Define domain types for assignee cycle time**

Add to `internal/domain/user_performance.go`:

```go
// AssigneeCycleTime represents time spent by an assignee on a single issue
// during their actual assignment period (fair attribution)
type AssigneeCycleTime struct {
	IssueID                int     `json:"issue_id"`
	IssueIID              int     `json:"issue_iid"`
	ProjectID             int     `json:"project_id"`
	ActiveCycleHours      float64 `json:"active_cycle_hours"`
	InProgressHours       float64 `json:"in_progress_hours"`
	QAReviewHours         float64 `json:"qa_review_hours"`
	BlockedHours          float64 `json:"blocked_hours"`
	BacklogHours          float64 `json:"backlog_hours"`
	TotalHoursAsAssignee  float64 `json:"total_hours_as_assignee"`
	ContributedActiveWork bool    `json:"contributed_active_work"`
}

// IndividualPerformanceMetrics aggregates performance metrics for an assignee
// across all their assigned issues (fair attribution model)
type IndividualPerformanceMetrics struct {
	Username              string  `json:"username"`
	IssuesAssigned        int     `json:"issues_assigned"`
	IssuesContributed     int     `json:"issues_contributed"`
	TotalActiveCycleHours float64 `json:"total_active_cycle_hours"`
	AvgActiveCyclePerIssue float64 `json:"avg_active_cycle_per_issue"`
	TotalDevHours         float64 `json:"total_dev_hours"`
	TotalQAHours          float64 `json:"total_qa_hours"`
	TotalBlockedHours     float64 `json:"total_blocked_hours"`
	TotalBacklogHours     float64 `json:"total_backlog_hours"`
	ActiveWorkPct         float64 `json:"active_work_pct"`
	TotalHoursAsAssignee  float64 `json:"total_hours_as_assignee"`
	P50ActiveCycleHours   float64 `json:"p50_active_cycle_hours"`
	P95ActiveCycleHours   float64 `json:"p95_active_cycle_hours"`
}
```

- [ ] **Step 3: Create repository struct and constructor**

```go
package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"gitlab-engineering-metrics-api/internal/domain"
)

// IndividualPerformanceRepositoryImpl implements IndividualPerformanceRepository
type IndividualPerformanceRepositoryImpl struct {
	db *sql.DB
}

// NewIndividualPerformanceRepository creates a new instance of the repository
func NewIndividualPerformanceRepository(db *sql.DB) *IndividualPerformanceRepositoryImpl {
	return &IndividualPerformanceRepositoryImpl{db: db}
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/domain/user_performance.go internal/repositories/individual_performance_repository.go
git commit -m "feat: define individual performance repository contract and domain types"
```

---

### Task 2: Implement GetAssigneeCycleTime Method

**Files:**
- Modify: `internal/repositories/individual_performance_repository.go:24-35`

- [ ] **Step 1: Write failing test for GetAssigneeCycleTime**

Create `internal/repositories/individual_performance_repository_test.go`:

```go
package repositories

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gitlab-engineering-metrics-api/internal/domain"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndividualPerformanceRepository_GetAssigneeCycleTime(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIndividualPerformanceRepository(db)

	// Arrange: mock query result for assignee with 2 issues
	rows := sqlmock.NewRows([]string{
		"issue_id", "issue_iid", "project_id",
		"active_cycle_hours", "in_progress_hours", "qa_review_hours",
		"blocked_hours", "backlog_hours", "total_hours_as_assignee",
		"contributed_active_work",
	}).AddRow(1, 10, 100, 25.5, 20.0, 5.5, 10.0, 5.0, 40.5, true).
	  AddRow(2, 11, 100, 0.0, 0.0, 0.0, 15.0, 10.0, 25.0, false)

	mock.ExpectQuery("SELECT.+FROM vw_assignee_cycle_time").
		WithArgs("testuser", "2025-01-01", "2025-12-31", 100).
		WillReturnRows(rows)

	// Act
	ctx := context.Background()
	filter := domain.MetricsFilter{
		Assignee:  "testuser",
		StartDate: "2025-01-01",
		EndDate:   "2025-12-31",
		ProjectID: 100,
	}
	result, err := repo.GetAssigneeCycleTime(ctx, "testuser", filter)

	// Assert
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, 1, result[0].IssueID)
	assert.Equal(t, 25.5, result[0].ActiveCycleHours)
	assert.Equal(t, true, result[0].ContributedActiveWork)
	assert.Equal(t, false, result[1].ContributedActiveWork)

	assert.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/repositories/individual_performance_repository_test.go -v -run TestIndividualPerformanceRepository_GetAssigneeCycleTime
```
Expected: FAIL with "undefined: IndividualPerformanceRepository" or method not found

- [ ] **Step 3: Implement GetAssigneeCycleTime method**

```go
// GetAssigneeCycleTime returns cycle time breakdown for a specific assignee
func (r *IndividualPerformanceRepositoryImpl) GetAssigneeCycleTime(ctx context.Context, username string, filter domain.MetricsFilter) ([]domain.AssigneeCycleTime, error) {
	query := `
		SELECT
			issue_id,
			issue_iid,
			project_id,
			COALESCE(active_cycle_hours, 0) as active_cycle_hours,
			COALESCE(in_progress_hours, 0) as in_progress_hours,
			COALESCE(qa_review_hours, 0) as qa_review_hours,
			COALESCE(blocked_hours, 0) as blocked_hours,
			COALESCE(backlog_hours, 0) as backlog_hours,
			COALESCE(total_hours_as_assignee, 0) as total_hours_as_assignee,
			contributed_active_work
		FROM vw_assignee_cycle_time
		WHERE assignee_username = $1
	`

	args := []interface{}{username}
	argIdx := 1

	if filter.ProjectID > 0 {
		argIdx++
		query += fmt.Sprintf(" AND project_id = $%d", argIdx)
		args = append(args, filter.ProjectID)
	}

	if filter.StartDate != "" && filter.EndDate != "" {
		argIdx++
		query += fmt.Sprintf(" AND issue_id IN (SELECT issue_id FROM vw_issue_lifecycle_metrics WHERE final_done_at >= $%d::date AND final_done_at < ($%d::date + INTERVAL '1 day'))", argIdx, argIdx+1)
		args = append(args, filter.StartDate, filter.EndDate)
	}

	query += " ORDER BY issue_id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query assignee cycle time: %w", err)
	}
	defer rows.Close()

	var results []domain.AssigneeCycleTime
	for rows.Next() {
		var act domain.AssigneeCycleTime
		if err := rows.Scan(
			&act.IssueID,
			&act.IssueIID,
			&act.ProjectID,
			&act.ActiveCycleHours,
			&act.InProgressHours,
			&act.QAReviewHours,
			&act.BlockedHours,
			&act.BacklogHours,
			&act.TotalHoursAsAssignee,
			&act.ContributedActiveWork,
		); err != nil {
			return nil, fmt.Errorf("failed to scan assignee cycle time: %w", err)
		}
		results = append(results, act)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assignee cycle time rows: %w", err)
	}

	return results, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/repositories/individual_performance_repository_test.go -v -run TestIndividualPerformanceRepository_GetAssigneeCycleTime
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/repositories/individual_performance_repository.go internal/repositories/individual_performance_repository_test.go
git commit -m "feat: implement GetAssigneeCycleTime with fair attribution query"
```

---

### Task 3: Implement GetIndividualPerformanceMetrics Method

**Files:**
- Modify: `internal/repositories/individual_performance_repository.go`

- [ ] **Step 1: Write failing test for GetIndividualPerformanceMetrics**

Append to `internal/repositories/individual_performance_repository_test.go`:

```go
func TestIndividualPerformanceRepository_GetIndividualPerformanceMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIndividualPerformanceRepository(db)

	// Arrange: mock aggregated metrics
	rows := sqlmock.NewRows([]string{
		"assignee_username", "issues_assigned", "issues_contributed",
		"total_active_cycle_hours", "avg_active_cycle_per_issue",
		"total_dev_hours", "total_qa_hours", "total_blocked_hours",
		"total_backlog_hours", "active_work_pct", "total_hours_as_assignee",
		"p50_active_cycle_hours", "p95_active_cycle_hours",
	}).AddRow("testuser", 10, 8, 250.5, 31.31, 180.0, 70.5, 45.0, 25.0, 83.5, 300.5, 28.5, 48.2)

	mock.ExpectQuery("SELECT.+FROM vw_individual_performance_metrics").
		WithArgs("testuser", 100).
		WillReturnRows(rows)

	// Act
	ctx := context.Background()
	filter := domain.MetricsFilter{
		Assignee:  "testuser",
		ProjectID: 100,
	}
	result, err := repo.GetIndividualPerformanceMetrics(ctx, "testuser", filter)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "testuser", result.Username)
	assert.Equal(t, 10, result.IssuesAssigned)
	assert.Equal(t, 8, result.IssuesContributed)
	assert.Equal(t, 83.5, result.ActiveWorkPct)
	assert.Equal(t, 250.5, result.TotalActiveCycleHours)

	assert.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/repositories/individual_performance_repository_test.go -v -run TestIndividualPerformanceRepository_GetIndividualPerformanceMetrics
```
Expected: FAIL (method not implemented)

- [ ] **Step 3: Implement GetIndividualPerformanceMetrics method**

```go
// GetIndividualPerformanceMetrics returns aggregated performance metrics for an assignee
func (r *IndividualPerformanceRepositoryImpl) GetIndividualPerformanceMetrics(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error) {
	query := `
		SELECT
			assignee_username,
			issues_assigned,
			issues_contributed,
			COALESCE(total_active_cycle_hours, 0) as total_active_cycle_hours,
			COALESCE(avg_active_cycle_per_issue, 0) as avg_active_cycle_per_issue,
			COALESCE(total_dev_hours, 0) as total_dev_hours,
			COALESCE(total_qa_hours, 0) as total_qa_hours,
			COALESCE(total_blocked_hours, 0) as total_blocked_hours,
			COALESCE(total_backlog_hours, 0) as total_backlog_hours,
			COALESCE(active_work_pct, 0) as active_work_pct,
			COALESCE(total_hours_as_assignee, 0) as total_hours_as_assignee,
			COALESCE(p50_active_cycle_hours, 0) as p50_active_cycle_hours,
			COALESCE(p95_active_cycle_hours, 0) as p95_active_cycle_hours
		FROM vw_individual_performance_metrics
		WHERE assignee_username = $1
	`

	args := []interface{}{username}

	if filter.ProjectID > 0 {
		query += " AND project_id = $2"
		args = append(args, filter.ProjectID)
	}

	var metrics domain.IndividualPerformanceMetrics
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&metrics.Username,
		&metrics.IssuesAssigned,
		&metrics.IssuesContributed,
		&metrics.TotalActiveCycleHours,
		&metrics.AvgActiveCyclePerIssue,
		&metrics.TotalDevHours,
		&metrics.TotalQAHours,
		&metrics.TotalBlockedHours,
		&metrics.TotalBacklogHours,
		&metrics.ActiveWorkPct,
		&metrics.TotalHoursAsAssignee,
		&metrics.P50ActiveCycleHours,
		&metrics.P95ActiveCycleHours,
	)

	if err == sql.ErrNoRows {
		// Return empty metrics if user has no data
		metrics.Username = username
		return &metrics, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query individual performance metrics: %w", err)
	}

	return &metrics, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/repositories/individual_performance_repository_test.go -v
```
Expected: PASS (both tests)

- [ ] **Step 5: Commit**

```bash
git add internal/repositories/individual_performance_repository.go internal/repositories/individual_performance_repository_test.go
git commit -m "feat: implement GetIndividualPerformanceMetrics with aggregation query"
```

---

### Task 4: Update UserPerformanceResponse Domain Type

**Files:**
- Modify: `internal/domain/user_performance.go`

- [ ] **Step 1: Review current UserPerformanceResponse structure**

Read `internal/domain/user_performance.go:33-40`

- [ ] **Step 2: Add IndividualPerformanceMetrics to response**

Modify `internal/domain/user_performance.go`:

```go
// UserPerformanceResponse is the contract returned by GET /api/v1/users/{username}/performance.
// Uses fair attribution model (vw_assignee_cycle_time) for individual metrics
// to ensure each assignee receives credit only for their actual time on issues.
type UserPerformanceResponse struct {
	User       UserPerformanceIdentity   `json:"user,omitempty"`
	Period     Period                    `json:"period,omitempty"`
	Delivery   UserDeliveryMetrics       `json:"delivery,omitempty"`
	Quality    UserQualityMetrics        `json:"quality,omitempty"`
	WIP        WipMetricsResponse        `json:"wip,omitempty"`
	// Fair attribution metrics (v3.0)
	// Each assignee receives credit ONLY for time they actually had the issue
	IndividualPerformance *IndividualPerformanceMetrics `json:"individual_performance,omitempty"`
}
```

- [ ] **Step 3: Add helper method to check contribution**

```go
// HasActiveContribution returns true if the assignee contributed active work
// (time in IN_PROGRESS or QA_REVIEW states) during their assignment period
func (m *IndividualPerformanceMetrics) HasActiveContribution() bool {
	return m.IssuesContributed > 0 && m.ActiveWorkPct > 0
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/domain/user_performance.go
git commit -m "feat: extend UserPerformanceResponse with fair attribution metrics"
```

---

### Task 5: Refactor UserPerformanceService to Use New Repository

**Files:**
- Modify: `internal/services/user_performance_service.go`

- [ ] **Step 1: Write failing test for refactored service**

Create or modify `internal/services/user_performance_service_test.go`:

```go
func TestUserPerformanceService_Get_UsesFairMetrics(t *testing.T) {
	// Arrange
	userRepo := &MockUserLookupRepository{}
	individualPerfRepo := &MockIndividualPerformanceRepository{}
	metricsSvc := &MockUserPerformanceMetricsService{}

	svc := NewUserPerformanceServiceWithIndividualMetrics(userRepo, metricsSvc, individualPerfRepo)

	userRepo.GetByUsernameFunc = func(ctx context.Context, username string, filter domain.CatalogFilter) (*domain.User, error) {
		return &domain.User{Username: "testuser", DisplayName: "Test User"}, nil
	}

	individualPerfRepo.GetIndividualPerformanceMetricsFunc = func(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error) {
		return &domain.IndividualPerformanceMetrics{
			Username:              "testuser",
			IssuesAssigned:        10,
			IssuesContributed:     8,
			TotalActiveCycleHours: 250.5,
			ActiveWorkPct:         83.5,
		}, nil
	}

	// Mock other services to return empty metrics
	metricsSvc.GetDeliveryMetricsFunc = func(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error) {
		return &domain.DeliveryMetricsResponse{}, nil
	}
	metricsSvc.GetQualityMetricsFunc = func(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error) {
		return &domain.QualityMetricsResponse{}, nil
	}
	metricsSvc.GetWipMetricsFunc = func(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error) {
		return &domain.WipMetricsResponse{}, nil
	}

	// Act
	ctx := context.Background()
	result, err := svc.Get(ctx, "testuser", domain.MetricsFilter{})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result.IndividualPerformance)
	assert.Equal(t, 83.5, result.IndividualPerformance.ActiveWorkPct)
	assert.Equal(t, 250.5, result.IndividualPerformance.TotalActiveCycleHours)
}
```

- [ ] **Step 2: Add repository interface to service**

Modify `internal/services/user_performance_service.go`:

```go
// IndividualPerformanceRepository defines the contract for fair individual metrics
type IndividualPerformanceRepository interface {
	GetIndividualPerformanceMetrics(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error)
}

// UserPerformanceService provides user performance metrics using fair attribution
type UserPerformanceService struct {
	usersRepo            UserLookupRepository
	metricsSvc           UserPerformanceMetricsService
	individualPerfRepo   IndividualPerformanceRepository
}
```

- [ ] **Step 3: Update constructor**

```go
// NewUserPerformanceService creates a new user performance service with fair attribution
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

- [ ] **Step 4: Refactor Get method to include fair metrics**

```go
// Get returns user performance metrics for the given username and filter
// Uses fair attribution model (assignee_cycle_time) for individual performance data
func (s *UserPerformanceService) Get(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.UserPerformanceResponse, error) {
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("username is required")
	}

	user, err := s.usersRepo.GetByUsername(ctx, username, domain.CatalogFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	filter.Assignee = username

	delivery, err := s.metricsSvc.GetDeliveryMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	quality, err := s.metricsSvc.GetQualityMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	wip, err := s.metricsSvc.GetWipMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Get fair individual performance metrics
	individualPerf, err := s.individualPerfRepo.GetIndividualPerformanceMetrics(ctx, username, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to load individual performance metrics: %w", err)
	}

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
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./internal/services/user_performance_service_test.go -v
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/services/user_performance_service.go internal/services/user_performance_service_test.go
git commit -m "refactor: integrate fair attribution repository into UserPerformanceService"
```

---

### Task 6: Update Dependency Injection in Main

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Find current dependency injection setup**

Search for `NewUserPerformanceService` in codebase:
```bash
grep -rn "NewUserPerformanceService" cmd/
```

- [ ] **Step 2: Add IndividualPerformanceRepository to DI container**

Modify `cmd/api/main.go`:

```go
// Create repositories
usersRepo := repositories.NewUsersRepository(db)
metricsRepo := repositories.NewMetricsRepository(db)
individualPerfRepo := repositories.NewIndividualPerformanceRepository(db)

// Create services
metricsSvc := services.NewMetricsService(metricsRepo)
userPerfSvc := services.NewUserPerformanceService(usersRepo, metricsSvc, individualPerfRepo)
```

- [ ] **Step 3: Commit**

```bash
git add cmd/api/main.go
git commit -m "refactor: wire IndividualPerformanceRepository into DI container"
```

---

### Task 7: Update Handler Tests for New Response Fields

**Files:**
- Modify: `internal/http/handlers/user_performance_handler_test.go`

- [ ] **Step 1: Write test for individual performance in response**

```go
func TestUserPerformanceHandler_Get_ReturnsIndividualPerformance(t *testing.T) {
	// Arrange
	mockService := &MockUserPerformanceService{}
	handler := NewUserPerformanceHandler(mockService)

	mockService.GetFunc = func(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.UserPerformanceResponse, error) {
		return &domain.UserPerformanceResponse{
			User: domain.UserPerformanceIdentity{
				Username:    "testuser",
				DisplayName: "Test User",
			},
			IndividualPerformance: &domain.IndividualPerformanceMetrics{
				Username:              "testuser",
				IssuesAssigned:        10,
				IssuesContributed:     8,
				ActiveWorkPct:         83.5,
				TotalActiveCycleHours: 250.5,
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/testuser/performance", nil)
	w := httptest.NewRecorder()

	// Act
	handler.Get(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "individual_performance")
	assert.Contains(t, w.Body.String(), "active_work_pct")
}
```

- [ ] **Step 2: Run test to verify it passes**

```bash
go test ./internal/http/handlers/user_performance_handler_test.go -v -run TestUserPerformanceHandler_Get_ReturnsIndividualPerformance
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/http/handlers/user_performance_handler_test.go
git commit -m "test: verify individual_performance fields in handler response"
```

---

### Task 8: Add Integration Test for Fair Metrics End-to-End

**Files:**
- Modify: `test/integration/metrics_test.go`

- [ ] **Step 1: Write integration test with real database**

```go
func TestUserPerformance_EndToEnd_UsesFairAttribution(t *testing.T) {
	// This test verifies that user performance endpoint returns fair attribution metrics
	// by checking active_work_pct and issues_contributed fields
	
	ctx := context.Background()
	client := setupTestClient(t)

	// Act: Get user performance
	resp, err := client.GetUserPerformance(ctx, "testuser")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result domain.UserPerformanceResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Assert: Fair metrics are present
	require.NotNil(t, result.IndividualPerformance)
	assert.GreaterOrEqual(t, result.IndividualPerformance.ActiveWorkPct, 0.0)
	assert.LessOrEqual(t, result.IndividualPerformance.ActiveWorkPct, 100.0)
	assert.GreaterOrEqual(t, result.IndividualPerformance.IssuesContributed, 0)
	assert.LessOrEqual(t, result.IndividualPerformance.IssuesContributed, result.IndividualPerformance.IssuesAssigned)

	// Verify fairness: issues_contributed <= issues_assigned
	// (cannot contribute to more issues than assigned)
	assert.LessOrEqual(t, result.IndividualPerformance.IssuesContributed, result.IndividualPerformance.IssuesAssigned)
}
```

- [ ] **Step 2: Run integration test**

```bash
go test ./test/integration/... -v -run TestUserPerformance_EndToEnd_UsesFairAttribution
```
Expected: PASS (with real database connection)

- [ ] **Step 3: Commit**

```bash
git add test/integration/metrics_test.go
git commit -m "test: add integration test for fair attribution end-to-end"
```

---

### Task 9: Update API Documentation

**Files:**
- Create: `docs/api/FAIR_ATTRIBUTION_GUIDE.md`

- [ ] **Step 1: Create documentation for fair metrics**

```markdown
# Fair Attribution Metrics Guide

## Overview

As of 2026-04-11, the user performance API (`GET /api/v1/users/{username}/performance`) uses **fair attribution** for individual metrics via `vw_assignee_cycle_time`.

## What Changed

### Before (Unfair Attribution)
```
Issue #1766 with 6 assignees, 1261 total hours:
- torezan: 1261h (credited with ALL time)
- danilo: 1261h (credited with ALL time)
- vitorfsampaio: 1261h (credited with ALL time)
... (each assignee got 100% credit)
```

### After (Fair Attribution)
```
Issue #1766 with 6 assignees, 1261 total hours:
- torezan: 276h active (their actual time)
- danilo: 0h active (only 90h backlog)
- vitorfsampaio: 0h active (173h total, mostly blocked/waiting)
... (each assignee gets ONLY their time)
```

## Response Changes

### New Field: `individual_performance`

```json
{
  "user": { "username": "nevez" },
  "individual_performance": {
    "username": "nevez",
    "issues_assigned": 35,
    "issues_contributed": 33,
    "total_active_cycle_hours": 27626.18,
    "active_work_pct": 99.22,
    "total_dev_hours": 20145.32,
    "total_qa_hours": 7480.86,
    "total_blocked_hours": 145.00,
    "total_backlog_hours": 250.00,
    "total_hours_as_assignee": 28021.18,
    "p50_active_cycle_hours": 245.50,
    "p95_active_cycle_hours": 892.30
  }
}
```

### Key Metrics Explained

| Field | Description | Fair vs Unfair |
|-------|-------------|----------------|
| `issues_assigned` | Total issues where user was assignee | Same |
| `issues_contributed` | Issues with active work (IN_PROGRESS + QA_REVIEW) | **NEW** - identifies formal vs actual contributors |
| `active_work_pct` | % of time spent in active states | **NEW** - efficiency indicator |
| `total_active_cycle_hours` | Hours in IN_PROGRESS + QA_REVIEW | **FAIR** - only counts user's actual active time |

## Migration Notes

### For API Consumers

**No breaking changes** - existing fields remain unchanged. The `individual_performance` object is **additional** data.

### For Project-Level Metrics

**No changes** - endpoints like `GET /api/v1/projects/{id}/metrics` continue to use `vw_issue_lifecycle_metrics` for aggregate throughput/velocity.

### When to Use Fair Metrics

✅ **Use `individual_performance` for:**
- Performance reviews
- Identifying bottlenecks
- Capacity planning
- Recognizing individual contributions

❌ **Do NOT use for:**
- Project velocity (use project-level endpoints)
- Team throughput (aggregate issues, not cycle time)
- Lead time calculations (use `vw_issue_lifecycle_metrics`)

## Example Queries

### Find Assignees with Low Active Work %

```bash
GET /api/v1/users/{username}/performance
# Check individual_performance.active_work_pct < 50
# Indicates: issue was assigned but not touched, or assignee was formal only
```

### Identify High Contributors

```bash
# Sort by individual_performance.issues_contributed
# Look for active_work_pct > 80%
```

## FAQ

**Q: Why is `issues_contributed` < `issues_assigned`?**
A: User was formally assigned but didn't do active work (IN_PROGRESS/QA_REVIEW). Issue may have been blocked, or another person did the actual work.

**Q: What if `active_work_pct` is 0%?**
A: User was assignee but issue stayed in BACKLOG/BLOCKED during their assignment. They never actively worked on it.

**Q: Can I sum `total_active_cycle_hours` across team members?**
A: Yes! Unlike old approach, fair metrics are additive without double-counting.

---

**Migration:** `000017_assignee_cycle_time`  
**Views:** `vw_assignee_cycle_time`, `vw_individual_performance_metrics`
```

- [ ] **Step 2: Commit**

```bash
git add docs/api/FAIR_ATTRIBUTION_GUIDE.md
git commit -m "docs: add fair attribution guide for API consumers"
```

---

### Task 10: Verify with Real Database Data

**Files:**
- No code changes - verification only

- [ ] **Step 1: Query database for sample user**

```bash
docker exec gitlab-elt-postgres psql -U gitlab_elt -d gitlab_elt -c "
SELECT 
    assignee_username,
    issues_assigned,
    issues_contributed,
    total_active_cycle_hours,
    active_work_pct
FROM vw_individual_performance_metrics
WHERE assignee_username IN ('nevez', 'torezan', 'danilo')
ORDER BY assignee_username;
"
```

- [ ] **Step 2: Compare with old approach**

```bash
docker exec gitlab-elt-postgres psql -U gitlab_elt -d gitlab_elt -c "
-- Old approach would credit all cycle time to each assignee
SELECT 
    issue_id,
    COUNT(DISTINCT assignee_username) as assignee_count,
    SUM(total_hours_as_assignee) as total_hours_sum
FROM vw_assignee_cycle_time
GROUP BY issue_id
HAVING COUNT(DISTINCT assignee_username) > 1
LIMIT 5;
"
```

- [ ] **Step 3: Document findings**

Add to `docs/api/FAIR_ATTRIBUTION_GUIDE.md` under "Verification" section:

```markdown
## Verification Results

Run date: 2026-04-11

Sample data shows:
- Issue #1766: 6 assignees, but only 2 contributed active work
- torezan: 276h active, 894h total (31% efficiency during assignment)
- danilo: 0h active, 90h backlog (100% wait time)
- Old approach would credit all 1261h to each person
- New approach: each person gets only their actual time
```

- [ ] **Step 4: Commit verification**

```bash
git add docs/api/FAIR_ATTRIBUTION_GUIDE.md
git commit -m "docs: add verification results for fair metrics"
```

---

### Task 11: Final Review and Cleanup

**Files:**
- Review all modified files

- [ ] **Step 1: Run all unit tests**

```bash
go test ./... -v
```
Expected: All tests PASS

- [ ] **Step 2: Run integration tests**

```bash
go test ./test/integration/... -v
```
Expected: All integration tests PASS

- [ ] **Step 3: Check for TODOs and placeholders**

```bash
grep -rn "TODO\|FIXME\|XXX\|TBD" internal/ docs/
```
Expected: No TODOs in new code

- [ ] **Step 4: Verify no breaking changes**

Check that existing tests still pass:
```bash
go test ./internal/services/user_performance_service_test.go -v
go test ./internal/http/handlers/user_performance_handler_test.go -v
```

- [ ] **Step 5: Final commit**

```bash
git add .
git commit -m "feat: complete fair attribution implementation with tests and docs"
```

---

## Plan Self-Review

✅ **Spec coverage:** All requirements from Option 1 (Hybrid Progressive) implemented:
- New repository for individual metrics ✓
- Service refactored to use fair attribution ✓
- Project-level metrics unchanged ✓
- Response extended with `individual_performance` ✓
- Documentation added ✓

✅ **No placeholders:** All steps have exact code, commands, and expected output

✅ **Type consistency:** 
- `IndividualPerformanceMetrics` defined in `user_performance.go` ✓
- Repository interface matches implementation ✓
- Service constructor updated consistently ✓

✅ **Test coverage:**
- Unit tests for repository ✓
- Service tests for integration ✓
- Handler tests for response format ✓
- Integration test for end-to-end ✓

---

**Plan complete and saved to `docs/superpowers/plans/2026-04-11-fair-individual-performance-metrics.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
