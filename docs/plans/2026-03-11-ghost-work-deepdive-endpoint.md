# Ghost Work Deep Dive Endpoint Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a UX-driven endpoint `GET /api/v1/metrics/ghost-work` that returns a consolidated view of ghost work issues with their state transitions, enabling deep dive analysis.

**Architecture:** Follow the established pattern of consolidating data for front-end consumption. The endpoint will query `vw_issue_lifecycle_metrics` for issues with `skipped_in_progress_flag=true` and join with `vw_issue_state_transitions` to identify the specific transition that constitutes ghost work (BACKLOG→DONE or BACKLOG→QA_REVIEW). Reuse existing filter building patterns from metrics_repository.go.

**Tech Stack:** Go, PostgreSQL, database/sql, standard HTTP handlers, existing domain types.

---

## Database Schema Reference

### Views Used:
- `vw_issue_lifecycle_metrics` - Source for ghost work issues with `skipped_in_progress_flag`
- `vw_issue_state_transitions` - Source for state transitions to identify ghost work pattern

### Ghost Work Definition:
An issue has ghost work when it transitions from `BACKLOG` directly to `DONE` or `QA_REVIEW` without passing through `IN_PROGRESS`.

### Key Fields:
- `skipped_in_progress_flag` - Boolean flag in lifecycle metrics
- `canonical_state` → `next_canonical_state` - Transition pairs
- Focus transitions: `BACKLOG` → `DONE`, `BACKLOG` → `QA_REVIEW`

---

## Domain Model

### GhostWorkIssue
```go
type GhostWorkIssue struct {
    IssueIID        int       `json:"issue_iid"`
    ProjectPath     string    `json:"project_path"`
    IssueTitle      string    `json:"issue_title"`
    Assignees       []string  `json:"assignees"`
    FromState       string    `json:"from_state"`        // Always BACKLOG
    ToState         string    `json:"to_state"`          // DONE or QA_REVIEW
    TransitionTime  string    `json:"transition_time"`   // When the transition occurred
    DurationHours   float64   `json:"duration_hours"`    // Time spent in ghost transition
    CurrentState    string    `json:"current_state"`     // Current state of the issue
    FinalDoneAt     string    `json:"final_done_at,omitempty"`
}
```

### GhostWorkTransitionSummary
```go
type GhostWorkTransitionSummary struct {
    FromState string `json:"from_state"`
    ToState   string `json:"to_state"`
    Count     int    `json:"count"`
}
```

### GhostWorkUserBreakdown
```go
type GhostWorkUserBreakdown struct {
    Username          string `json:"username"`
    GhostWorkCount    int    `json:"ghost_work_count"`
    Issues            []int  `json:"issue_iids"`  // List of issue IIDs
}
```

### GhostWorkMetricsResponse
```go
type GhostWorkMetricsResponse struct {
    TotalIssues        int                        `json:"total_issues"`
    Period             domain.Period              `json:"period"`
    Issues             []GhostWorkIssue           `json:"issues"`
    TransitionAnalysis []GhostWorkTransitionSummary `json:"transition_analysis"`
    BreakdownByUser    []GhostWorkUserBreakdown   `json:"breakdown_by_user"`
    Pagination         domain.Pagination          `json:"pagination"`
}
```

---

## Task 1: Define Domain Types for Ghost Work Deep Dive

**Files:**
- Create: `internal/domain/ghost_work.go`
- Test: `internal/domain/ghost_work_test.go` (optional - domain types typically don't need tests)

**Step 1: Create domain structs**

```go
package domain

// GhostWorkIssue represents an issue that had ghost work (skipped IN_PROGRESS)
type GhostWorkIssue struct {
    IssueIID       int      `json:"issue_iid"`
    ProjectPath    string   `json:"project_path"`
    IssueTitle     string   `json:"issue_title"`
    Assignees      []string `json:"assignees"`
    FromState      string   `json:"from_state"`
    ToState        string   `json:"to_state"`
    TransitionTime string   `json:"transition_time"`
    DurationHours  float64  `json:"duration_hours"`
    CurrentState   string   `json:"current_state"`
    FinalDoneAt    string   `json:"final_done_at,omitempty"`
}

// GhostWorkTransitionSummary represents aggregated ghost work by transition type
type GhostWorkTransitionSummary struct {
    FromState string `json:"from_state"`
    ToState   string `json:"to_state"`
    Count     int    `json:"count"`
}

// GhostWorkUserBreakdown represents ghost work aggregated by user
type GhostWorkUserBreakdown struct {
    Username       string `json:"username"`
    GhostWorkCount int    `json:"ghost_work_count"`
    IssueIIDs      []int  `json:"issue_iids"`
}

// GhostWorkMetricsResponse represents the complete ghost work deep dive response
type GhostWorkMetricsResponse struct {
    TotalIssues        int                          `json:"total_issues"`
    Period             Period                       `json:"period"`
    Issues             []GhostWorkIssue             `json:"issues"`
    TransitionAnalysis []GhostWorkTransitionSummary `json:"transition_analysis"`
    BreakdownByUser    []GhostWorkUserBreakdown     `json:"breakdown_by_user"`
    Pagination         Pagination                   `json:"pagination"`
}

// GhostWorkFilter extends MetricsFilter with pagination
type GhostWorkFilter struct {
    MetricsFilter
    Page     int `json:"page,omitempty"`
    PageSize int `json:"page_size,omitempty"`
}
```

**Step 2: Verify code compiles**

Run: `go build ./internal/domain`
Expected: PASS (no errors)

**Step 3: Commit**

Skip commit as per user instructions.

---

## Task 2: Create Ghost Work Repository

**Files:**
- Create: `internal/repositories/ghost_work_repository.go`
- Create: `internal/repositories/ghost_work_repository_test.go`

**Step 1: Create repository struct and constructor**

```go
package repositories

import (
    "context"
    "database/sql"
    "fmt"
    "log/slog"
    "strings"

    "github.com/lib/pq"
    "gitlab-engineering-metrics-api/internal/domain"
    "gitlab-engineering-metrics-api/internal/observability"
)

var ghostWorkRepoLogger = observability.GetLogger().With(slog.String("repository", "ghost_work"))

type GhostWorkRepository struct {
    db *sql.DB
}

func NewGhostWorkRepository(db *sql.DB) *GhostWorkRepository {
    return &GhostWorkRepository{db: db}
}
```

**Step 2: Implement GetGhostWorkIssues method**

Query must:
1. Filter issues where `skipped_in_progress_flag = true`
2. Join with transitions to find the ghost work transition (BACKLOG → DONE or BACKLOG → QA_REVIEW)
3. Support all standard filters (date range, project, group, assignee)
4. Support pagination
5. Return detailed issue info + transition details

```go
func (r *GhostWorkRepository) GetGhostWorkIssues(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error) {
    // Implementation here
    // Query joins vw_issue_lifecycle_metrics with vw_issue_state_transitions
    // Filters by skipped_in_progress_flag = true
    // Finds transitions where canonical_state = 'BACKLOG' AND next_canonical_state IN ('DONE', 'QA_REVIEW')
}
```

**Step 3: Write failing test**

```go
func TestGhostWorkRepository_GetGhostWorkIssues(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := NewGhostWorkRepository(db)
    ctx := context.Background()

    tests := []struct {
        name    string
        filter  domain.GhostWorkFilter
        wantErr bool
    }{
        {
            name:    "get ghost work issues with no filter",
            filter:  domain.GhostWorkFilter{},
            wantErr: false,
        },
        {
            name: "get ghost work issues with date range",
            filter: domain.GhostWorkFilter{
                MetricsFilter: domain.MetricsFilter{
                    StartDate: "2024-01-01",
                    EndDate:   "2024-12-31",
                },
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := repo.GetGhostWorkIssues(ctx, tt.filter)
            if (err != nil) != tt.wantErr {
                t.Errorf("GetGhostWorkIssues() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result == nil {
                t.Error("GetGhostWorkIssues() returned nil")
            }
        })
    }
}
```

**Step 4: Run test to verify it fails**

Run: `go test ./internal/repositories -run TestGhostWorkRepository_GetGhostWorkIssues -v`
Expected: FAIL with "undefined: GhostWorkRepository"

**Step 5: Implement minimal code**

Write the full query that:
- Uses the same filter building pattern as metrics_repository.go
- Joins lifecycle metrics with state transitions
- Identifies ghost work transitions
- Supports pagination with LIMIT/OFFSET
- Returns both detailed issues and aggregate summaries

**Step 6: Run test to verify it passes**

Run: `go test ./internal/repositories -run TestGhostWorkRepository_GetGhostWorkIssues -v`
Expected: PASS (or SKIP if DB unavailable)

**Step 7: Commit**

Skip commit as per user instructions.

---

## Task 3: Create Ghost Work Service

**Files:**
- Create: `internal/services/ghost_work_service.go`
- Create: `internal/services/ghost_work_service_test.go`

**Step 1: Define interface and create service**

```go
package services

import (
    "context"
    "errors"
    "fmt"

    "gitlab-engineering-metrics-api/internal/domain"
)

type GhostWorkRepository interface {
    GetGhostWorkIssues(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error)
}

type GhostWorkService struct {
    repo GhostWorkRepository
}

func NewGhostWorkService(repo GhostWorkRepository) *GhostWorkService {
    return &GhostWorkService{repo: repo}
}
```

**Step 2: Implement GetGhostWorkMetrics**

```go
func (s *GhostWorkService) GetGhostWorkMetrics(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error) {
    // Validate filter
    if err := s.validateFilter(filter); err != nil {
        return nil, err
    }

    // Set default pagination
    if filter.Page < 1 {
        filter.Page = 1
    }
    if filter.PageSize < 1 || filter.PageSize > 100 {
        filter.PageSize = 25
    }

    // Call repository
    result, err := s.repo.GetGhostWorkIssues(ctx, filter)
    if err != nil {
        return nil, fmt.Errorf("failed to get ghost work metrics: %w", err)
    }

    return result, nil
}

func (s *GhostWorkService) validateFilter(filter domain.GhostWorkFilter) error {
    // Same validation as metrics_service.go for date format
    if filter.StartDate != "" || filter.EndDate != "" {
        if filter.StartDate == "" || filter.EndDate == "" {
            return errors.New("both start_date and end_date are required when filtering by date")
        }
        // ... date format validation
    }
    return nil
}
```

**Step 3: Write tests**

Cover:
- Happy path
- Validation errors (invalid dates, pagination)
- Repository error propagation

**Step 4: Run tests**

Run: `go test ./internal/services -run TestGhostWorkService -v`
Expected: PASS

**Step 5: Commit**

Skip commit.

---

## Task 4: Create Ghost Work Handler

**Files:**
- Create: `internal/http/handlers/ghost_work_handler.go`
- Create: `internal/http/handlers/ghost_work_handler_test.go`

**Step 1: Create handler**

```go
package handlers

import (
    "context"
    "encoding/json"
    "net/http"
    "strconv"

    "gitlab-engineering-metrics-api/internal/domain"
    "gitlab-engineering-metrics-api/internal/http/middleware"
    "gitlab-engineering-metrics-api/internal/http/responses"
)

type GhostWorkService interface {
    GetGhostWorkMetrics(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error)
}

type GhostWorkHandler struct {
    service GhostWorkService
}

func NewGhostWorkHandler(service GhostWorkService) *GhostWorkHandler {
    return &GhostWorkHandler{service: service}
}

func (h *GhostWorkHandler) Get(w http.ResponseWriter, r *http.Request) {
    // Implementation follows same pattern as quality_handler.go
    // Parse query params: start_date, end_date, project_id, group_path, assignee
    // Parse pagination: page, page_size
    // Call service
    // Handle validation errors -> 400
    // Handle errors -> 500
    // Return JSON
}
```

**Step 2: Write tests**

Cover:
- GET returns ghost work data
- POST not allowed (405)
- Invalid date format (400)
- Invalid pagination (400)
- Service error (500)

**Step 3: Run tests**

Run: `go test ./internal/http/handlers -run TestGhostWorkHandler -v`
Expected: PASS

**Step 4: Commit**

Skip commit.

---

## Task 5: Register Route

**Files:**
- Modify: `internal/app/routes.go:74-92` (after registerMetricsRoutes)

**Step 1: Add ghost work routes registration**

```go
func (a *App) registerGhostWorkRoutes(mux *http.ServeMux) {
    // Create repository
    ghostWorkRepo := repositories.NewGhostWorkRepository(a.db)

    // Create service
    ghostWorkService := services.NewGhostWorkService(ghostWorkRepo)

    // Create handler
    ghostWorkHandler := handlers.NewGhostWorkHandler(ghostWorkService)

    // Register routes with auth middleware
    authMiddleware := middleware.Auth(a.validator)

    mux.Handle("/api/v1/metrics/ghost-work", authMiddleware(http.HandlerFunc(ghostWorkHandler.Get)))
}
```

**Step 2: Call registration from Routes()**

Add `a.registerGhostWorkRoutes(mux)` after registerMetricsRoutes.

**Step 3: Commit**

Skip commit.

---

## Task 6: Update Documentation

**Files:**
- Modify: `docs/openapi.yaml` - Add /metrics/ghost-work endpoint
- Modify: `README.md` - Add endpoint documentation
- Modify: `docs/api-architecture.md` - Add to endpoint table
- Create: `docs/GitLab Engineering Metrics API/Metrics/Retorna metricas de ghost work.yml`

**Step 1: Add OpenAPI path**

Add `/metrics/ghost-work` with all schema definitions (GhostWorkIssue, GhostWorkTransitionSummary, GhostWorkUserBreakdown, GhostWorkMetricsResponse).

**Step 2: Add to README**

Document the endpoint with curl example showing the consolidated response.

**Step 3: Update architecture docs**

Add endpoint to mapping table.

**Step 4: Create Bruno collection file**

Follow pattern of other metric endpoints.

**Step 5: Commit**

Skip commit.

---

## Task 7: Verification

**Step 1: Run tests**

```bash
go test ./internal/repositories -run TestGhostWorkRepository -v
go test ./internal/services -run TestGhostWorkService -v
go test ./internal/http/handlers -run TestGhostWorkHandler -v
go build ./cmd/api
```

**Step 2: Smoke test**

```bash
# Start server
go run ./cmd/api &

# Test endpoint
curl -s \
  -H "X-Client-ID: myclient" \
  -H "X-Client-Secret: mysecret" \
  "http://localhost:8080/api/v1/metrics/ghost-work?start_date=2026-01-01&end_date=2026-01-31&page=1&page_size=10"

# Verify response has: issues, transition_analysis, breakdown_by_user, pagination
```

**Step 3: Commit**

Skip commit.

---

## API Specification

### Request
```
GET /api/v1/metrics/ghost-work
```

**Query Parameters:**
- `start_date` (optional) - Filter by date range start (YYYY-MM-DD)
- `end_date` (optional) - Filter by date range end (YYYY-MM-DD)
- `project_id` (optional) - Filter by specific project
- `group_path` (optional) - Filter by group path (prefix match)
- `assignee` (optional) - Filter by assignee username
- `page` (optional, default: 1) - Page number for pagination
- `page_size` (optional, default: 25, max: 100) - Items per page

### Response 200
```json
{
  "total_issues": 270,
  "period": {
    "start_date": "2026-01-01",
    "end_date": "2026-01-31"
  },
  "issues": [
    {
      "issue_iid": 371,
      "project_path": "apps-expo/dflegal-expo",
      "issue_title": "[PASTA DE TRABALHO] - Filtragem de Ordens...",
      "assignees": ["gabriel"],
      "from_state": "BACKLOG",
      "to_state": "QA_REVIEW",
      "transition_time": "2026-03-06T14:17:15.282Z",
      "duration_hours": 0.01,
      "current_state": "DONE",
      "final_done_at": "2026-03-06T20:02:32.541Z"
    }
  ],
  "transition_analysis": [
    {
      "from_state": "BACKLOG",
      "to_state": "QA_REVIEW",
      "count": 432
    },
    {
      "from_state": "BACKLOG",
      "to_state": "DONE",
      "count": 26
    }
  ],
  "breakdown_by_user": [
    {
      "username": "nevez",
      "ghost_work_count": 50,
      "issue_iids": [371, 369, 364, ...]
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 25,
    "total": 270,
    "total_pages": 11
  }
}
```

This endpoint provides everything needed for a comprehensive ghost work analysis screen without requiring multiple API calls.
