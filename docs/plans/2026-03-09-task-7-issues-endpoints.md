# Task 7: Issues Endpoints Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Implement issues list endpoint with pagination/filters and timeline endpoint for issue events

**Architecture:** Follow existing patterns with repositories for data access, services for business logic/validation, and handlers for HTTP handling. Timeline combines data from multiple tables with UNION ALL.

**Tech Stack:** Go, PostgreSQL, standard library HTTP handlers

---

## Database Schema Reference

### Views Used:
- `vw_issue_lifecycle_metrics` - Main view for issues list with metrics
- `vw_issue_state_transitions` - State transitions timeline

### Tables Used:
- `issues` - Basic issue info
- `issue_events` - State change events
- `issue_comments` - Comments on issues

---

## Task 1: Create Issues Repository

**Files:**
- Create: `internal/repositories/issues_repository.go`
- Create: `internal/repositories/issues_repository_test.go`

**Step 1: Create repository struct and constructor**

```go
package repositories

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "gitlab-engineering-metrics-api/internal/domain"
)

type IssuesRepository struct {
    db *sql.DB
}

func NewIssuesRepository(db *sql.DB) *IssuesRepository {
    return &IssuesRepository{db: db}
}
```

**Step 2: Implement List method with filters and pagination**

```go
func (r *IssuesRepository) List(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error) {
    // Build base query from vw_issue_lifecycle_metrics
    // Support filters: project_id, group_path, assignee, state
    // Support pagination: page, page_size (default 20, max 100)
    // Return total count for pagination
}
```

**Step 3: Write failing tests**

Run: `go test ./internal/repositories -run TestIssuesRepository -v`
Expected: FAIL - undefined: IssuesRepository

**Step 4: Implement minimal code to pass tests**

**Step 5: Commit**

```bash
git add internal/repositories/issues_repository.go internal/repositories/issues_repository_test.go
git commit -m "feat: add issues repository with list and pagination"
```

---

## Task 2: Create Timeline Repository

**Files:**
- Create: `internal/repositories/timeline_repository.go`
- Create: `internal/repositories/timeline_repository_test.go`

**Step 1: Create repository struct and constructor**

```go
type TimelineRepository struct {
    db *sql.DB
}

func NewTimelineRepository(db *sql.DB) *TimelineRepository {
    return &TimelineRepository{db: db}
}
```

**Step 2: Implement GetTimeline method**

Query should combine:
- Issue basic info from `issues` table
- State transitions from `vw_issue_state_transitions`
- Comments from `issue_comments`
- Events from `issue_events` (if needed)
- Sort by timestamp ascending

```go
func (r *TimelineRepository) GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error) {
    // 1. Get issue summary from issues table
    // 2. Get state transitions as timeline events
    // 3. Get comments as timeline events
    // 4. Combine and sort by timestamp
    // 5. Return IssueTimelineResponse
}
```

**Step 3: Write tests for timeline repository**

**Step 4: Commit**

```bash
git add internal/repositories/timeline_repository.go internal/repositories/timeline_repository_test.go
git commit -m "feat: add timeline repository for issue events"
```

---

## Task 3: Create Issues Service

**Files:**
- Create: `internal/services/issues_service.go`
- Create: `internal/services/issues_service_test.go`

**Step 1: Define interfaces and create service struct**

```go
type IssuesRepository interface {
    List(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error)
}

type TimelineRepository interface {
    GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error)
}

type IssuesService struct {
    issuesRepo   IssuesRepository
    timelineRepo TimelineRepository
}

func NewIssuesService(issuesRepo IssuesRepository, timelineRepo TimelineRepository) *IssuesService {
    return &IssuesService{
        issuesRepo:   issuesRepo,
        timelineRepo: timelineRepo,
    }
}
```

**Step 2: Implement ListIssues method**

```go
func (s *IssuesService) ListIssues(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error) {
    // Validate: page >= 1
    // Validate: page_size between 1-100 (default 20)
    // Call issuesRepo.List
}
```

**Step 3: Implement GetTimeline method**

```go
func (s *IssuesService) GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error) {
    // Validate issueID > 0
    // Call timelineRepo.GetTimeline
}
```

**Step 4: Write tests for service**

**Step 5: Commit**

```bash
git add internal/services/issues_service.go internal/services/issues_service_test.go
git commit -m "feat: add issues service with validation"
```

---

## Task 4: Create Issues Handler

**Files:**
- Create: `internal/http/handlers/issues_handler.go`
- Create: `internal/http/handlers/issues_handler_test.go`

**Step 1: Create handler with interface and constructor**

```go
type IssuesService interface {
    ListIssues(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error)
}

type IssuesHandler struct {
    service IssuesService
}

func NewIssuesHandler(service IssuesService) *IssuesHandler {
    return &IssuesHandler{service: service}
}
```

**Step 2: Implement List handler**

```go
func (h *IssuesHandler) List(w http.ResponseWriter, r *http.Request) {
    // Handle GET only
    // Parse query params: project_id, group_path, assignee, state, page, page_size
    // Convert to IssuesFilter
    // Call service.ListIssues
    // Handle errors: 400 for validation, 422 for unprocessable, 500 for internal
    // Return 200 with IssuesListResponse
}
```

**Step 3: Write comprehensive tests**

Tests must cover:
- GET returns issues list
- POST not allowed (405)
- Invalid page param (400)
- Invalid page_size param (422)
- Service error (500)

**Step 4: Commit**

```bash
git add internal/http/handlers/issues_handler.go internal/http/handlers/issues_handler_test.go
git commit -m "feat: add issues handler with list endpoint"
```

---

## Task 5: Create Timeline Handler

**Files:**
- Create: `internal/http/handlers/timeline_handler.go`
- Create: `internal/http/handlers/timeline_handler_test.go`

**Step 1: Create handler with interface and constructor**

```go
type TimelineService interface {
    GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error)
}

type TimelineHandler struct {
    service TimelineService
}

func NewTimelineHandler(service TimelineService) *TimelineHandler {
    return &TimelineHandler{service: service}
}
```

**Step 2: Implement Get handler**

```go
func (h *TimelineHandler) Get(w http.ResponseWriter, r *http.Request) {
    // Handle GET only
    // Parse issue ID from URL path: /api/v1/issues/:id/timeline
    // Call service.GetTimeline
    // Return 404 if issue not found
    // Return 200 with IssueTimelineResponse
}
```

**Step 3: Write comprehensive tests**

Tests must cover:
- GET returns timeline
- POST not allowed (405)
- Invalid issue ID format (400)
- Issue not found (404)
- Service error (500)

**Step 4: Commit**

```bash
git add internal/http/handlers/timeline_handler.go internal/http/handlers/timeline_handler_test.go
git commit -m "feat: add timeline handler with get endpoint"
```

---

## Task 6: Wire Routes

**Files:**
- Modify: `internal/app/routes.go`

**Step 1: Add registerIssuesRoutes method**

Add to App struct section:
```go
func (a *App) registerIssuesRoutes(mux *http.ServeMux) {
    // Create repositories
    issuesRepo := repositories.NewIssuesRepository(a.db)
    timelineRepo := repositories.NewTimelineRepository(a.db)
    
    // Create service
    issuesService := services.NewIssuesService(issuesRepo, timelineRepo)
    
    // Create handlers
    issuesHandler := handlers.NewIssuesHandler(issuesService)
    timelineHandler := handlers.NewTimelineHandler(issuesService)
    
    // Register routes with auth middleware
    authMiddleware := middleware.Auth(a.validator)
    
    mux.Handle("/api/v1/issues", authMiddleware(http.HandlerFunc(issuesHandler.List)))
    mux.Handle("/api/v1/issues/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Route /api/v1/issues/:id/timeline
    })))
}
```

**Step 2: Call registerIssuesRoutes from Routes()**

```go
func (a *App) Routes() http.Handler {
    // ... existing code ...
    a.registerCatalogRoutes(mux)
    a.registerMetricsRoutes(mux)
    a.registerIssuesRoutes(mux)  // Add this line
    // ...
}
```

**Step 3: Write test for routes**

**Step 4: Commit**

```bash
git add internal/app/routes.go
git commit -m "feat: wire issues and timeline routes"
```

---

## Task 7: Run All Tests

**Step 1: Run full test suite**

```bash
go test ./... -v
```

**Expected:** All tests pass including new ones

**Step 2: Verify specific scenarios**

```bash
go test ./internal/repositories -v
go test ./internal/services -v
go test ./internal/http/handlers -v
```

**Step 3: Final commit if all pass**

```bash
git add -A
git commit -m "test: add comprehensive tests for issues endpoints (404, 400, 422)"
```

---

## Error Handling Requirements

### HTTP Status Codes:
- **200 OK** - Success
- **400 Bad Request** - Invalid query parameters (bad format)
- **401 Unauthorized** - Missing/invalid auth (handled by middleware)
- **404 Not Found** - Issue ID doesn't exist
- **422 Unprocessable Entity** - Validation errors (page < 1, page_size > 100)
- **500 Internal Server Error** - Database or unexpected errors

### Validation Rules:
- `page`: >= 1, default 1
- `page_size`: 1-100, default 20
- `issue_id`: positive integer
- `project_id`: positive integer (if provided)

---

## Database Query Patterns

### Issues List Query:
```sql
SELECT 
    issue_id, project_id, issue_iid, issue_title, 
    assignees, current_canonical_state, 
    lead_time_hours/24 as lead_time_days,
    cycle_time_hours/24 as cycle_time_days,
    blocked_time_hours,
    qa_to_dev_return_count
FROM vw_issue_lifecycle_metrics
WHERE 1=1
  AND ($1 = 0 OR project_id = $1)
  AND ($2 = '' OR project_path LIKE $2 || '%')
  AND ($3 = '' OR $3 = ANY(assignees))
  AND ($4 = '' OR current_canonical_state = $4)
ORDER BY issue_id
LIMIT $5 OFFSET $6
```

### Timeline Query (UNION ALL pattern):
```sql
-- Issue summary
SELECT id, title, iid as gitlab_iid, project_id, 
       metadata_labels, assignees, current_canonical_state, gitlab_created_at
FROM issues WHERE id = $1;

-- State transitions
SELECT 
    'state_transition' as type,
    entered_at as timestamp,
    author_name as actor,
    previous_canonical_state as from_state,
    canonical_state as to_state,
    COALESCE(duration_hours_to_next_state * 60, 0) as duration_mins,
    CASE WHEN previous_canonical_state = 'QA_REVIEW' AND canonical_state = 'IN_PROGRESS' 
         THEN true ELSE false END as is_rework
FROM vw_issue_state_transitions
WHERE issue_id = $1

UNION ALL

-- Comments
SELECT 
    'comment' as type,
    comment_timestamp as timestamp,
    author_name as actor,
    NULL as from_state,
    NULL as to_state,
    0 as duration_mins,
    false as is_rework,
    body
FROM issue_comments
WHERE issue_id = $1

ORDER BY timestamp ASC
```

---

## Implementation Notes

1. **Pagination**: Use LIMIT/OFFSET with total count from separate COUNT(*) query
2. **URL Path Parsing**: Use `strings.Split(r.URL.Path, "/")` or regex to extract issue ID
3. **Error Messages**: Match existing patterns in responses package
4. **Null Handling**: Use `sql.Null*` types for nullable database columns
5. **JSON Tags**: Ensure domain structs have proper `json:"field,omitempty"` tags

---

## Success Criteria

- [ ] `GET /api/v1/issues` returns paginated list with filters
- [ ] `GET /api/v1/issues/:id/timeline` returns timeline events
- [ ] 404 returned when issue not found
- [ ] 400 returned for invalid parameters
- [ ] 422 returned for validation errors
- [ ] All tests pass: `go test ./...`
