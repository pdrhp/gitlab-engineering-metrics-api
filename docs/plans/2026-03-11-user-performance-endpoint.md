# User Performance Endpoint Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `GET /api/v1/users/{username}/performance` so a profile/performance screen can fetch delivery, quality, ghost work, and WIP data for one developer in a single request.

**Architecture:** Keep the HTTP contract user-oriented, but reuse the existing delivery/quality/WIP aggregation logic instead of inventing a second metrics stack. Add a thin user-performance service that validates the path username, confirms the user exists, calls the existing metrics service with `assignee=username`, and maps those responses into a front-end-friendly payload. Do not add new database migrations in v1; all required data already exists in `vw_issue_lifecycle_metrics`, `issues.assignees`, and the current users catalog query.

**Tech Stack:** Go, `net/http`, PostgreSQL, `database/sql`, existing repository/service/handler layers, OpenAPI docs, integration tests.

---

## Implementation Notes

- Use the current assignee semantics already present in `internal/repositories/jsonb_helpers.go:8-14` and `internal/repositories/metrics_repository.go:149-185`. In v1, the endpoint should mean "performance for issues currently attributed to this username" so it stays consistent with the rest of the API.
- Freeze the WIP contract now: `wip` remains a current snapshot and ignores `start_date` / `end_date`, exactly like `GET /api/v1/metrics/wip` does today.
- Treat ghost work as a user-facing alias for the existing `skipped_in_progress_flag` / bypass logic documented in `db/schema/000013_create_golden_engineering_views.up.sql:318` and `docs/database/README.md:214-254`.
- Reuse existing metrics queries before writing new SQL. The current code already supports assignee-filtered delivery, quality, and WIP in `internal/http/handlers/delivery_handler.go:56-62`, `internal/http/handlers/quality_handler.go:31-37`, `internal/http/handlers/wip_handler.go:31-35`, and `internal/repositories/metrics_repository.go:167-170`.
- Validate with both unit tests and one real request against local Docker Postgres before calling the work done.

---

### Task 1: Define the User Performance API contract

**Files:**
- Create: `internal/domain/user_performance.go`
- Modify: `internal/domain/user.go:1-9`
- Test: `internal/http/handlers/user_performance_handler_test.go`

**Step 1: Write the failing handler test for the JSON shape**

```go
func TestUserPerformanceHandler_Get_ReturnsAggregatedPayload(t *testing.T) {
    svc := &mockUserPerformanceService{
        response: &domain.UserPerformanceResponse{
            User: domain.UserPerformanceIdentity{
                Username:    "ianfelps",
                DisplayName: "ianfelps",
            },
            Period: domain.Period{StartDate: "2026-01-01", EndDate: "2026-01-31"},
            Delivery: domain.UserDeliveryMetrics{
                Throughput: domain.Throughput{TotalIssuesDone: 7, AvgPerWeek: 1.75},
            },
            Quality: domain.UserQualityMetrics{
                Rework: domain.ReworkMetrics{TotalReworkedIssues: 2},
                GhostWork: domain.GhostWorkMetrics{RatePct: 12.5},
            },
            WIP: domain.WipMetricsResponse{
                CurrentWIP: domain.CurrentWIP{QAReview: 1},
            },
        },
    }

    handler := NewUserPerformanceHandler(svc)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/users/ianfelps/performance?start_date=2026-01-01&end_date=2026-01-31", nil)
    rr := httptest.NewRecorder()

    handler.Get(rr, req)

    require.Equal(t, http.StatusOK, rr.Code)
    var got domain.UserPerformanceResponse
    require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
    require.Equal(t, "ianfelps", got.User.Username)
    require.Equal(t, 12.5, got.Quality.GhostWork.RatePct)
}
```

**Step 2: Run the test to verify it fails**

Run: `go test ./internal/http/handlers -run TestUserPerformanceHandler_Get_ReturnsAggregatedPayload -v`
Expected: FAIL with `undefined: domain.UserPerformanceResponse` and `undefined: NewUserPerformanceHandler`

**Step 3: Write the minimal domain contract**

```go
package domain

type UserPerformanceIdentity struct {
    Username                  string `json:"username,omitempty"`
    DisplayName               string `json:"display_name,omitempty"`
    ActiveIssues              int    `json:"active_issues,omitempty"`
    CompletedIssuesLast30Days int    `json:"completed_issues_last_30_days,omitempty"`
}

type GhostWorkMetrics struct {
    RatePct float64 `json:"rate_pct,omitempty"`
}

type UserDeliveryMetrics struct {
    Throughput       Throughput   `json:"throughput,omitempty"`
    SpeedMetricsDays SpeedMetrics `json:"speed_metrics_days,omitempty"`
}

type UserQualityMetrics struct {
    Rework        ReworkMetrics        `json:"rework,omitempty"`
    GhostWork     GhostWorkMetrics     `json:"ghost_work,omitempty"`
    ProcessHealth ProcessHealthMetrics `json:"process_health,omitempty"`
    Bottlenecks   BottleneckMetrics    `json:"bottlenecks,omitempty"`
    Defects       DefectMetrics        `json:"defects,omitempty"`
}

type UserPerformanceResponse struct {
    User   UserPerformanceIdentity `json:"user,omitempty"`
    Period Period                  `json:"period,omitempty"`
    Delivery UserDeliveryMetrics   `json:"delivery,omitempty"`
    Quality  UserQualityMetrics    `json:"quality,omitempty"`
    WIP      WipMetricsResponse    `json:"wip,omitempty"`
}
```

**Step 4: Run the test again to verify only the missing handler symbols remain**

Run: `go test ./internal/http/handlers -run TestUserPerformanceHandler_Get_ReturnsAggregatedPayload -v`
Expected: FAIL with `undefined: NewUserPerformanceHandler`

**Step 5: Commit**

```bash
git add internal/domain/user.go internal/domain/user_performance.go internal/http/handlers/user_performance_handler_test.go
git commit -m "feat: define user performance response contract"
```

---

### Task 2: Add user lookup in the users repository

**Files:**
- Modify: `internal/repositories/users_repository.go:25-127`
- Modify: `internal/repositories/users_repository_test.go:1-55`
- Reference: `internal/repositories/jsonb_helpers.go:12-14`

**Step 1: Write the failing repository test for exact username lookup**

```go
func TestUsersRepository_GetByUsername(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := NewUsersRepository(db)

    user, err := repo.GetByUsername(context.Background(), "ianfelps", domain.CatalogFilter{})
    if err != nil {
        t.Fatalf("GetByUsername() error = %v", err)
    }
    if user == nil {
        t.Fatal("expected user, got nil")
    }
    if user.Username != "ianfelps" {
        t.Fatalf("expected ianfelps, got %s", user.Username)
    }
}
```

**Step 2: Run the test to verify it fails**

Run: `go test ./internal/repositories -run TestUsersRepository_GetByUsername -v`
Expected: FAIL with `repo.GetByUsername undefined`

**Step 3: Implement the minimal repository method by reusing the existing users CTE**

```go
func (r *UsersRepository) GetByUsername(ctx context.Context, username string, filter domain.CatalogFilter) (*domain.User, error) {
    query := `
        WITH normalized_assignees AS (
            SELECT
                i.id as issue_id,
                p.path as project_path,
                i.current_canonical_state,
                a.username
            FROM issues i
            JOIN projects p ON p.id = i.project_id
            CROSS JOIN LATERAL jsonb_array_elements_text(` + normalizedAssigneesJSONBExpr("i.assignees") + `) as a(username)
            WHERE 1=1
        ),
        user_stats AS (
            SELECT
                username,
                COUNT(*) FILTER (WHERE current_canonical_state NOT IN ('DONE', 'CANCELED')) as active_issues,
                COUNT(*) FILTER (
                    WHERE current_canonical_state = 'DONE'
                    AND EXISTS (
                        SELECT 1 FROM vw_issue_state_transitions t
                        WHERE t.issue_id = na.issue_id
                        AND t.canonical_state = 'DONE'
                        AND t.entered_at >= NOW() - INTERVAL '30 days'
                    )
                ) as completed_last_30_days
            FROM normalized_assignees na
            GROUP BY username
        )
        SELECT username, username as display_name, active_issues, completed_last_30_days
        FROM user_stats
        WHERE username = $1
        LIMIT 1
    `

    var u domain.User
    err := r.db.QueryRowContext(ctx, query, username).Scan(
        &u.Username,
        &u.DisplayName,
        &u.ActiveIssues,
        &u.CompletedIssuesLast30Days,
    )
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user by username: %w", err)
    }
    return &u, nil
}
```

**Step 4: Run repository tests**

Run: `go test ./internal/repositories -run 'TestUsersRepository_(List|GetByUsername)' -v`
Expected: PASS (or `SKIP` only if the local test database is unavailable)

**Step 5: Commit**

```bash
git add internal/repositories/users_repository.go internal/repositories/users_repository_test.go
git commit -m "feat: add exact user lookup for profile endpoints"
```

---

### Task 3: Create the user performance orchestration service

**Files:**
- Create: `internal/services/user_performance_service.go`
- Create: `internal/services/user_performance_service_test.go`
- Modify: `internal/services/metrics_service.go:12-109` (only if a shared interface extract makes the new service cleaner)

**Step 1: Write the failing service tests for orchestration and not-found behavior**

```go
func TestUserPerformanceService_Get(t *testing.T) {
    usersRepo := &mockUserLookupRepository{
        user: &domain.User{
            Username: "ianfelps",
            DisplayName: "ianfelps",
            ActiveIssues: 24,
            CompletedIssuesLast30Days: 103,
        },
    }
    metricsSvc := &mockUserPerformanceMetricsService{
        delivery: &domain.DeliveryMetricsResponse{Throughput: domain.Throughput{TotalIssuesDone: 20}},
        quality: &domain.QualityMetricsResponse{ProcessHealth: domain.ProcessHealthMetrics{BypassRatePct: 5}},
        wip: &domain.WipMetricsResponse{CurrentWIP: domain.CurrentWIP{QAReview: 2}},
    }

    svc := NewUserPerformanceService(usersRepo, metricsSvc)
    got, err := svc.Get(context.Background(), "ianfelps", domain.MetricsFilter{StartDate: "2026-01-01", EndDate: "2026-01-31"})

    require.NoError(t, err)
    require.Equal(t, "ianfelps", got.User.Username)
    require.Equal(t, 5.0, got.Quality.GhostWork.RatePct)
    require.Equal(t, 2, got.WIP.CurrentWIP.QAReview)
}

func TestUserPerformanceService_Get_UserNotFound(t *testing.T) {
    svc := NewUserPerformanceService(&mockUserLookupRepository{}, &mockUserPerformanceMetricsService{})

    _, err := svc.Get(context.Background(), "missing-user", domain.MetricsFilter{})

    require.Error(t, err)
    require.ErrorContains(t, err, "user not found")
}
```

**Step 2: Run the tests to verify they fail**

Run: `go test ./internal/services -run TestUserPerformanceService_Get -v`
Expected: FAIL with `undefined: NewUserPerformanceService`

**Step 3: Implement the minimal service that composes existing metrics endpoints**

```go
type UserLookupRepository interface {
    GetByUsername(ctx context.Context, username string, filter domain.CatalogFilter) (*domain.User, error)
}

type UserPerformanceMetricsService interface {
    GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error)
    GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error)
    GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error)
}

type UserPerformanceService struct {
    usersRepo   UserLookupRepository
    metricsSvc  UserPerformanceMetricsService
}

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

    return &domain.UserPerformanceResponse{
        User: domain.UserPerformanceIdentity{
            Username:                  user.Username,
            DisplayName:               user.DisplayName,
            ActiveIssues:              user.ActiveIssues,
            CompletedIssuesLast30Days: user.CompletedIssuesLast30Days,
        },
        Period: filterPeriod(filter),
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
        WIP: *wip,
    }, nil
}
```

**Step 4: Run the service tests**

Run: `go test ./internal/services -run 'TestUserPerformanceService_Get' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/user_performance_service.go internal/services/user_performance_service_test.go internal/services/metrics_service.go
git commit -m "feat: compose user performance from existing metrics services"
```

---

### Task 4: Add the HTTP handler and user path routing

**Files:**
- Create: `internal/http/handlers/user_performance_handler.go`
- Modify: `internal/http/handlers/user_performance_handler_test.go`
- Modify: `internal/app/routes.go:52-72`
- Modify: `test/integration/testapp.go:18-173`
- Test: `test/integration/catalog_test.go:161-221` or create `test/integration/user_performance_test.go`

**Step 1: Write the failing HTTP tests for path parsing and error handling**

```go
func TestUserPerformanceHandler_Get_InvalidProjectID(t *testing.T) {
    handler := NewUserPerformanceHandler(&mockUserPerformanceService{})
    req := httptest.NewRequest(http.MethodGet, "/api/v1/users/ianfelps/performance?project_id=abc", nil)
    rr := httptest.NewRecorder()

    handler.Get(rr, req)

    require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUserPerformanceHandler_Get_UserNotFound(t *testing.T) {
    handler := NewUserPerformanceHandler(&mockUserPerformanceService{err: errors.New("user not found")})
    req := httptest.NewRequest(http.MethodGet, "/api/v1/users/missing/performance", nil)
    rr := httptest.NewRecorder()

    handler.Get(rr, req)

    require.Equal(t, http.StatusNotFound, rr.Code)
}
```

**Step 2: Run the handler tests to verify they fail**

Run: `go test ./internal/http/handlers -run 'TestUserPerformanceHandler_Get' -v`
Expected: FAIL with missing handler/service symbols

**Step 3: Implement the handler and route dispatch**

```go
type UserPerformanceReader interface {
    Get(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.UserPerformanceResponse, error)
}

func (h *UserPerformanceHandler) Get(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.Header().Set("Allow", http.MethodGet)
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    username, ok := extractUserPerformanceUsername(r.URL.Path)
    if !ok {
        responses.NotFound(w, middleware.GetRequestID(r.Context()), "resource not found")
        return
    }

    filter := domain.MetricsFilter{
        StartDate: r.URL.Query().Get("start_date"),
        EndDate:   r.URL.Query().Get("end_date"),
    }

    if projectID := r.URL.Query().Get("project_id"); projectID != "" {
        parsed, err := strconv.Atoi(projectID)
        if err != nil {
            responses.BadRequest(w, middleware.GetRequestID(r.Context()), "invalid project_id")
            return
        }
        filter.ProjectID = parsed
    }

    payload, err := h.service.Get(r.Context(), username, filter)
    // map validation -> 400, not found -> 404, everything else -> 500
}

func isUserPerformancePath(path string) bool {
    parts := strings.Split(strings.Trim(path, "/"), "/")
    return len(parts) == 6 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "users" && parts[4] == "performance"
}
```

Update `internal/app/routes.go` so `/api/v1/users` keeps listing users, while `/api/v1/users/{username}/performance` is routed by the `/api/v1/users/` prefix handler, mirroring the current issues timeline approach.

**Step 4: Run unit and integration tests**

Run: `go test ./internal/http/handlers ./test/integration -run 'UserPerformance|Catalog_Users' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/http/handlers/user_performance_handler.go internal/http/handlers/user_performance_handler_test.go internal/app/routes.go test/integration/testapp.go test/integration/user_performance_test.go
git commit -m "feat: expose user performance endpoint"
```

---

### Task 5: Document the endpoint and keep the docs in sync

**Files:**
- Modify: `docs/openapi.yaml:127-171`
- Modify: `README.md:69-93`
- Modify: `docs/api-architecture.md:238-262`
- Modify: `docs/database/README.md:230-254`
- Optional: `docs/GitLab Engineering Metrics API/Users/Retorna performance consolidada por usuario.yml`

**Step 1: Write the failing docs-driven check by searching for the new path**

Run: `rg "/users/\{username\}/performance|/api/v1/users/.*/performance" README.md docs/openapi.yaml docs/api-architecture.md docs/database/README.md docs/GitLab\ Engineering\ Metrics\ API`
Expected: no matches

**Step 2: Add the OpenAPI path and response schema**

```yaml
/users/{username}/performance:
  get:
    tags: [Users]
    summary: Retorna metricas consolidadas de performance por usuario
    parameters:
      - in: path
        name: username
        required: true
        schema:
          type: string
      - $ref: '#/components/parameters/StartDateOptional'
      - $ref: '#/components/parameters/EndDateOptional'
      - $ref: '#/components/parameters/ProjectId'
    responses:
      '200':
        description: Performance consolidada do usuario
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UserPerformanceResponse'
      '400': { $ref: '#/components/responses/BadRequest' }
      '401': { $ref: '#/components/responses/Unauthorized' }
      '404': { $ref: '#/components/responses/NotFound' }
      '500': { $ref: '#/components/responses/InternalError' }
```

Also add the new schema objects (`GhostWorkMetrics`, `UserDeliveryMetrics`, `UserQualityMetrics`, `UserPerformanceResponse`) and a real `curl` example in `README.md`.

**Step 3: Update architecture and database docs**

Add one line to the endpoint-to-layer table and one line to the API consumption map stating that `/api/v1/users/{username}/performance` reads from `issues` + `vw_issue_lifecycle_metrics` via user lookup plus existing metrics aggregation.

**Step 4: Run a docs sanity check**

Run: `rg "users/\{username\}/performance|UserPerformanceResponse|ghost_work" README.md docs/openapi.yaml docs/api-architecture.md docs/database/README.md`
Expected: matches in all edited docs

**Step 5: Commit**

```bash
git add README.md docs/openapi.yaml docs/api-architecture.md docs/database/README.md docs/GitLab\ Engineering\ Metrics\ API/Users/Retorna\ performance\ consolidada\ por\ usuario.yml
git commit -m "docs: describe consolidated user performance endpoint"
```

---

### Task 6: Verify against the real database and smoke-test the endpoint

**Files:**
- Modify: none
- Test: local running app + Docker Postgres
- Reference: `db/schema/000013_create_golden_engineering_views.up.sql:293-397`

**Step 1: Confirm the local database has a real user to test**

Run:

```bash
docker exec gitlab-elt-postgres psql -U gitlab_elt -d gitlab_elt -c "WITH normalized AS (SELECT DISTINCT jsonb_array_elements_text(CASE WHEN jsonb_typeof(assignees) = 'array' THEN assignees WHEN jsonb_typeof(assignees) = 'object' AND assignees ? 'current' THEN assignees->'current' ELSE '[]'::jsonb END) AS username FROM issues) SELECT username FROM normalized ORDER BY 1 LIMIT 10;"
```

Expected: at least one username such as `ianfelps`, `danilo`, or `nevez`

**Step 2: Run the focused Go test suites**

Run: `go test ./internal/repositories ./internal/services ./internal/http/handlers ./test/integration -run 'UserPerformance|UsersRepository_GetByUsername|Catalog_Users' -v`
Expected: PASS

**Step 3: Start the API locally**

Run: `go run ./cmd/api`
Expected: the server starts without route registration errors

**Step 4: Smoke-test the new endpoint with one real user**

Run:

```bash
curl -s \
  -H "X-Client-ID: myclient" \
  -H "X-Client-Secret: mysecret" \
  "http://localhost:8080/api/v1/users/ianfelps/performance?start_date=2026-01-01&end_date=2026-01-31&project_id=225"
```

Expected: `200 OK` with `user`, `delivery`, `quality`, and `wip` sections in one JSON object

**Step 5: Commit**

```bash
git status
git add -A
git commit -m "test: verify user performance endpoint end-to-end"
```

---

## Suggested Response Shape

Use this as the target JSON contract for the front-end:

```json
{
  "user": {
    "username": "ianfelps",
    "display_name": "ianfelps",
    "active_issues": 24,
    "completed_issues_last_30_days": 103
  },
  "period": {
    "start_date": "2026-01-01",
    "end_date": "2026-01-31"
  },
  "delivery": {
    "throughput": {
      "total_issues_done": 20,
      "avg_per_week": 5
    },
    "speed_metrics_days": {
      "lead_time": { "avg": 20.15, "p85": 41.22 },
      "cycle_time": { "avg": 11.40, "p85": 19.03 }
    }
  },
  "quality": {
    "rework": {
      "ping_pong_rate_pct": 35,
      "total_reworked_issues": 7,
      "avg_rework_cycles_per_issue": 1.2
    },
    "ghost_work": {
      "rate_pct": 5
    },
    "process_health": {
      "bypass_rate_pct": 5,
      "first_time_pass_rate_pct": 65
    },
    "bottlenecks": {
      "total_blocked_time_hours": 48,
      "avg_blocked_time_per_issue_hours": 6
    },
    "defects": {
      "bug_ratio_pct": 10
    }
  },
  "wip": {
    "current_wip": {
      "in_progress": 3,
      "qa_review": 2,
      "blocked": 0
    },
    "aging_wip": []
  }
}
```

This keeps the contract experience-oriented without duplicating the `breakdown_by_assignee` slice that only makes sense for team-wide endpoints.
