# Delivery Trend Endpoint Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `GET /api/v1/metrics/delivery/trend` with bucketed throughput/speed metrics and Pearson correlation, including filter validation, OpenAPI update, and Bruno collection update.

**Architecture:** Add a dedicated trend flow in the existing metrics stack (handler -> service -> repository) while reusing shared filter helpers and response utilities. Compute bucketed aggregates directly from `vw_issue_lifecycle_metrics` using timezone-aware bucketing (`day|week|month`), optional gap-filling (`include_empty_buckets`), and SQL `corr(...)` for correlation. Keep existing `/metrics/delivery` behavior unchanged and isolate new validation rules to the trend endpoint.

**Tech Stack:** Go, net/http, database/sql, PostgreSQL (CTEs + `date_trunc` + `generate_series` + `corr`), OpenAPI YAML, Bruno YAML collection.

---

### Task 0: Setup Guardrails (Worktree + Existing Context)

**Files:**
- Read: `db/schema/000013_create_golden_engineering_views.up.sql`
- Read: `internal/repositories/metrics_repository.go`
- Read: `docs/openapi.yaml`

**Step 1: Create dedicated worktree**

Run: `git worktree add ../gitlab-engineering-metrics-api-delivery-trend -b feat/delivery-trend-endpoint`

Expected: new worktree created with branch `feat/delivery-trend-endpoint`.

**Step 2: Verify required data source fields exist**

Run:
```bash
docker exec gitlab-elt-postgres psql -U gitlab_elt -d gitlab_elt -c "SELECT COUNT(*) FILTER (WHERE is_completed) AS completed, COUNT(*) FILTER (WHERE is_completed AND lead_time_hours IS NOT NULL) AS lead_ok, COUNT(*) FILTER (WHERE is_completed AND cycle_time_hours IS NOT NULL) AS cycle_ok FROM vw_issue_lifecycle_metrics;"
```

Expected: `lead_ok` and `cycle_ok` are non-zero and close to `completed`.

**Step 3: Commit setup note (optional)**

```bash
git add docs/plans/2026-03-15-delivery-trend-endpoint.md
git commit -m "docs: add delivery trend implementation plan"
```

---

### Task 1: Add Domain Models for Delivery Trend

**Files:**
- Create: `internal/domain/delivery_trend.go`
- Modify: `internal/domain/metrics.go`
- Test: `internal/domain/delivery_trend_test.go`

**Step 1: Write the failing test**

```go
package domain

import (
    "encoding/json"
    "testing"
)

func TestDeliveryTrendResponse_JSONNullability(t *testing.T) {
    var leadAvg *float64
    resp := DeliveryTrendResponse{
        Bucket:   "week",
        Timezone: "UTC",
        Items: []DeliveryTrendPoint{{
            BucketStart: "2026-02-02",
            BucketEnd:   "2026-02-08",
            BucketLabel: "2026-W06",
            Throughput: DeliveryTrendThroughput{TotalIssuesDone: 0},
            SpeedMetricsDays: DeliveryTrendSpeedMetrics{
                LeadTime:  AvgP85MetricNullable{Avg: leadAvg, P85: nil},
                CycleTime: AvgP85MetricNullable{Avg: nil, P85: nil},
            },
        }},
    }

    raw, err := json.Marshal(resp)
    if err != nil {
        t.Fatalf("marshal error: %v", err)
    }
    s := string(raw)
    if !containsString(s, `"avg":null`) {
        t.Fatalf("expected null avg field, got: %s", s)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain -run TestDeliveryTrendResponse_JSONNullability -v`

Expected: FAIL with `undefined: DeliveryTrendResponse`.

**Step 3: Write minimal implementation**

```go
package domain

type DeliveryTrendFilter struct {
    MetricsFilter
    Bucket              string `json:"bucket,omitempty"`
    Timezone            string `json:"timezone,omitempty"`
    IncludeEmptyBuckets bool   `json:"include_empty_buckets"`
}

type AvgP85MetricNullable struct {
    Avg *float64 `json:"avg"`
    P85 *float64 `json:"p85"`
}

type DeliveryTrendThroughput struct {
    TotalIssuesDone int `json:"total_issues_done"`
}

type DeliveryTrendSpeedMetrics struct {
    LeadTime  AvgP85MetricNullable `json:"lead_time"`
    CycleTime AvgP85MetricNullable `json:"cycle_time"`
}

type DeliveryTrendPoint struct {
    BucketStart     string                    `json:"bucket_start"`
    BucketEnd       string                    `json:"bucket_end"`
    BucketLabel     string                    `json:"bucket_label"`
    Throughput      DeliveryTrendThroughput   `json:"throughput"`
    SpeedMetricsDays DeliveryTrendSpeedMetrics `json:"speed_metrics_days"`
}

type DeliveryTrendCorrelation struct {
    ThroughputVsLeadTimeR  *float64 `json:"throughput_vs_lead_time_r"`
    ThroughputVsCycleTimeR *float64 `json:"throughput_vs_cycle_time_r"`
}

type DeliveryTrendFiltersApplied struct {
    GroupPath *string `json:"group_path"`
    ProjectID *int    `json:"project_id"`
    Assignee  *string `json:"assignee"`
}

type DeliveryTrendResponse struct {
    Period         Period                      `json:"period"`
    Bucket         string                      `json:"bucket"`
    Timezone       string                      `json:"timezone"`
    FiltersApplied DeliveryTrendFiltersApplied `json:"filters_applied,omitempty"`
    Items          []DeliveryTrendPoint        `json:"items"`
    Correlation    *DeliveryTrendCorrelation   `json:"correlation,omitempty"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain -run TestDeliveryTrendResponse_JSONNullability -v`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/domain/delivery_trend.go internal/domain/metrics.go internal/domain/delivery_trend_test.go
git commit -m "feat: add domain contracts for delivery trend endpoint"
```

---

### Task 2: Extend Metrics Contracts (Interfaces) Without Breaking Existing Endpoints

**Files:**
- Modify: `internal/services/metrics_service.go`
- Modify: `internal/http/handlers/delivery_handler.go`
- Modify: `test/integration/testapp.go`
- Test: `internal/services/metrics_service_test.go`
- Test: `internal/http/handlers/delivery_handler_test.go`

**Step 1: Write the failing test**

```go
func TestMetricsService_GetDeliveryTrend_DelegatesToRepository(t *testing.T) {
    repo := &mockMetricsRepository{
        deliveryTrend: &domain.DeliveryTrendResponse{Bucket: "week", Timezone: "UTC"},
    }
    svc := NewMetricsService(repo)

    got, err := svc.GetDeliveryTrendMetrics(context.Background(), domain.DeliveryTrendFilter{Bucket: "week"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got == nil || got.Bucket != "week" {
        t.Fatalf("unexpected response: %#v", got)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services -run TestMetricsService_GetDeliveryTrend_DelegatesToRepository -v`

Expected: FAIL with missing method(s) in repository/service interfaces.

**Step 3: Write minimal implementation**

```go
// services/metrics_service.go
type MetricsRepository interface {
    GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error)
    GetDeliveryTrendMetrics(ctx context.Context, filter domain.DeliveryTrendFilter) (*domain.DeliveryTrendResponse, error)
    GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error)
    GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error)
}

func (s *MetricsService) GetDeliveryTrendMetrics(ctx context.Context, filter domain.DeliveryTrendFilter) (*domain.DeliveryTrendResponse, error) {
    if err := s.validateDeliveryTrendFilter(filter); err != nil {
        return nil, err
    }
    out, err := s.repo.GetDeliveryTrendMetrics(ctx, filter)
    if err != nil {
        return nil, fmt.Errorf("failed to get delivery trend metrics: %w", err)
    }
    return out, nil
}
```

Also extend `handlers.MetricsService` interface in `internal/http/handlers/delivery_handler.go` to include the new method, then update all mocks (`mockMetricsService`, integration `MockMetricsService`, etc.) with a stubbed implementation.

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/services -run TestMetricsService_GetDeliveryTrend_DelegatesToRepository -v
go test ./internal/http/handlers -run TestDeliveryHandler_Get -v
go test ./test/integration -run TestMetrics_Delivery_Success -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/services/metrics_service.go internal/services/metrics_service_test.go internal/http/handlers/delivery_handler.go internal/http/handlers/delivery_handler_test.go test/integration/testapp.go
git commit -m "refactor: extend metrics interfaces for delivery trend support"
```

---

### Task 3: Implement Service-Level Validation Rules for Trend Endpoint

**Files:**
- Modify: `internal/services/metrics_service.go`
- Modify: `internal/services/metrics_service_test.go`

**Step 1: Write the failing tests**

```go
func TestMetricsService_validateDeliveryTrendFilter(t *testing.T) {
    svc := NewMetricsService(&mockMetricsRepository{})

    tests := []struct {
        name    string
        filter  domain.DeliveryTrendFilter
        wantErr string
    }{
        {"invalid bucket", domain.DeliveryTrendFilter{MetricsFilter: domain.MetricsFilter{StartDate: "2026-01-01", EndDate: "2026-01-31"}, Bucket: "quarter"}, "bucket must be one of: day, week, month"},
        {"bad timezone", domain.DeliveryTrendFilter{MetricsFilter: domain.MetricsFilter{StartDate: "2026-01-01", EndDate: "2026-01-31"}, Bucket: "week", Timezone: "Mars/Olympus"}, "invalid timezone"},
        {"range too large", domain.DeliveryTrendFilter{MetricsFilter: domain.MetricsFilter{StartDate: "2025-01-01", EndDate: "2026-12-31"}, Bucket: "week", Timezone: "UTC"}, "date range cannot exceed 366 days"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := svc.GetDeliveryTrendMetrics(context.Background(), tt.filter)
            if err == nil || !containsString(err.Error(), tt.wantErr) {
                t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
            }
        })
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services -run TestMetricsService_validateDeliveryTrendFilter -v`

Expected: FAIL (validation function not implemented or wrong messages).

**Step 3: Write minimal implementation**

```go
func (s *MetricsService) validateDeliveryTrendFilter(filter domain.DeliveryTrendFilter) error {
    if filter.StartDate == "" || filter.EndDate == "" {
        return errors.New("both start_date and end_date are required")
    }

    startDate, err := time.Parse("2006-01-02", filter.StartDate)
    if err != nil {
        return errors.New("invalid start_date format, expected YYYY-MM-DD")
    }
    endDate, err := time.Parse("2006-01-02", filter.EndDate)
    if err != nil {
        return errors.New("invalid end_date format, expected YYYY-MM-DD")
    }
    if endDate.Before(startDate) {
        return errors.New("end_date must be after start_date")
    }
    if endDate.Sub(startDate) > (366 * 24 * time.Hour) {
        return errors.New("date range cannot exceed 366 days")
    }

    if filter.Bucket == "" {
        filter.Bucket = "week"
    }
    if filter.Bucket != "day" && filter.Bucket != "week" && filter.Bucket != "month" {
        return errors.New("bucket must be one of: day, week, month")
    }

    tz := filter.Timezone
    if tz == "" {
        tz = "UTC"
    }
    if _, err := time.LoadLocation(tz); err != nil {
        return errors.New("invalid timezone")
    }

    return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services -run TestMetricsService_validateDeliveryTrendFilter -v`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/services/metrics_service.go internal/services/metrics_service_test.go
git commit -m "feat: add delivery trend validation rules"
```

---

### Task 4: Implement Repository Query for Bucketed Trend + Correlation

**Files:**
- Modify: `internal/repositories/metrics_repository.go`
- Modify: `internal/repositories/metrics_repository_test.go`

**Step 1: Write the failing repository test**

```go
func TestMetricsRepository_GetDeliveryTrendMetrics(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := NewMetricsRepository(db)
    filter := domain.DeliveryTrendFilter{
        MetricsFilter: domain.MetricsFilter{StartDate: "2026-02-01", EndDate: "2026-03-01"},
        Bucket:              "week",
        Timezone:            "America/Sao_Paulo",
        IncludeEmptyBuckets: true,
    }

    got, err := repo.GetDeliveryTrendMetrics(context.Background(), filter)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got == nil || len(got.Items) == 0 {
        t.Fatalf("expected non-empty trend response, got %#v", got)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/repositories -run TestMetricsRepository_GetDeliveryTrendMetrics -v`

Expected: FAIL with `undefined: GetDeliveryTrendMetrics`.

**Step 3: Write minimal implementation**

```go
func (r *MetricsRepository) GetDeliveryTrendMetrics(ctx context.Context, filter domain.DeliveryTrendFilter) (*domain.DeliveryTrendResponse, error) {
    // 1) defaults
    if filter.Bucket == "" { filter.Bucket = "week" }
    if filter.Timezone == "" { filter.Timezone = "UTC" }

    // 2) semantic validation in DB: if both project_id and group_path, ensure project belongs to group_path
    // SELECT 1 FROM projects WHERE id=$1 AND regexp_replace(path, '/[^/]+$', '')=$2

    // 3) build filtered completed issues based on done_at localized date
    // local_done_date := (final_done_at AT TIME ZONE $tz)::date

    // 4) aggregate by bucket with timezone-aware date_trunc
    // throughput := count(*)
    // lead/cycle avg days := avg(hours)/24
    // p85 days := CASE WHEN count(*) >= 2 THEN percentile_cont(0.85)/24 ELSE NULL END

    // 5) optionally fill empty buckets with generate_series
    // include_empty_buckets=true => left join generated bucket timeline

    // 6) compute correlation from non-null avg points:
    // corr(throughput::float8, lead_avg_days)
    // corr(throughput::float8, cycle_avg_days)

    // 7) map rows to []domain.DeliveryTrendPoint and return
    return &domain.DeliveryTrendResponse{/*...*/}, nil
}
```

Use these SQL conventions in final query builder:
- `week` bucket: `date_trunc('week', final_done_at AT TIME ZONE $1)` (ISO Monday-Sunday in Postgres).
- Labels:
  - day: `to_char(bucket_start, 'YYYY-MM-DD')`
  - week: `to_char(bucket_start, 'IYYY-"W"IW')`
  - month: `to_char(bucket_start, 'YYYY-MM')`

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/repositories -run TestMetricsRepository_GetDeliveryTrendMetrics -v
go test ./internal/repositories -run TestMetricsRepository_BuildFilterConditions -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/repositories/metrics_repository.go internal/repositories/metrics_repository_test.go
git commit -m "feat: add delivery trend repository query with bucketing and correlation"
```

---

### Task 5: Add Dedicated HTTP Handler for `/metrics/delivery/trend`

**Files:**
- Create: `internal/http/handlers/delivery_trend_handler.go`
- Create: `internal/http/handlers/delivery_trend_handler_test.go`
- Modify: `internal/app/routes.go`

**Step 1: Write the failing handler test**

```go
func TestDeliveryTrendHandler_Get_Success(t *testing.T) {
    svc := &mockMetricsService{deliveryTrend: &domain.DeliveryTrendResponse{Bucket: "week", Timezone: "UTC", Items: []domain.DeliveryTrendPoint{}}}
    h := NewDeliveryTrendHandler(svc)

    req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/delivery/trend?start_date=2026-02-01&end_date=2026-03-01", nil)
    rr := httptest.NewRecorder()
    h.Get(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/http/handlers -run TestDeliveryTrendHandler_Get_Success -v`

Expected: FAIL with `undefined: NewDeliveryTrendHandler`.

**Step 3: Write minimal implementation**

```go
package handlers

type DeliveryTrendHandler struct { service MetricsService }

func NewDeliveryTrendHandler(service MetricsService) *DeliveryTrendHandler {
    return &DeliveryTrendHandler{service: service}
}

func (h *DeliveryTrendHandler) Get(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.Header().Set("Allow", http.MethodGet)
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    requestID := middleware.GetRequestID(r.Context())
    filter := domain.DeliveryTrendFilter{
        MetricsFilter: domain.MetricsFilter{
            StartDate: r.URL.Query().Get("start_date"),
            EndDate:   r.URL.Query().Get("end_date"),
            GroupPath: r.URL.Query().Get("group_path"),
            Assignee:  r.URL.Query().Get("assignee"),
        },
        Bucket:              r.URL.Query().Get("bucket"),
        Timezone:            r.URL.Query().Get("timezone"),
        IncludeEmptyBuckets: true,
    }

    if v := r.URL.Query().Get("include_empty_buckets"); v != "" {
        b, err := strconv.ParseBool(v)
        if err != nil {
            responses.BadRequest(w, requestID, "include_empty_buckets must be boolean")
            return
        }
        filter.IncludeEmptyBuckets = b
    }

    if pid := r.URL.Query().Get("project_id"); pid != "" {
        n, err := strconv.Atoi(pid)
        if err != nil || n <= 0 {
            responses.BadRequest(w, requestID, "project_id must be a positive integer")
            return
        }
        filter.ProjectID = n
    }

    out, err := h.service.GetDeliveryTrendMetrics(r.Context(), filter)
    if err != nil {
        msg := err.Error()
        if containsString(msg, "invalid") || containsString(msg, "must") || containsString(msg, "exceed") {
            responses.BadRequest(w, requestID, msg)
            return
        }
        if containsString(msg, "does not belong to group_path") {
            responses.UnprocessableEntity(w, requestID, msg)
            return
        }
        responses.InternalServerError(w, requestID)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(out)
}
```

Register route in `internal/app/routes.go`:

```go
deliveryTrendHandler := handlers.NewDeliveryTrendHandler(metricsService)
mux.Handle("/api/v1/metrics/delivery/trend", authMiddleware(http.HandlerFunc(deliveryTrendHandler.Get)))
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/http/handlers -run TestDeliveryTrendHandler -v
go test ./internal/app -run Test.*Routes -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/http/handlers/delivery_trend_handler.go internal/http/handlers/delivery_trend_handler_test.go internal/app/routes.go
git commit -m "feat: expose delivery trend metrics endpoint"
```

---

### Task 6: Add Integration Coverage for New Endpoint

**Files:**
- Modify: `test/integration/testapp.go`
- Modify: `test/integration/metrics_test.go`

**Step 1: Write failing integration tests**

```go
func TestMetrics_DeliveryTrend_Success(t *testing.T) {
    ts := SetupTestServer(t)
    defer TeardownTestServer(ts)

    ts.Builder.MetricsService.DeliveryTrend = &domain.DeliveryTrendResponse{
        Bucket: "week",
        Timezone: "UTC",
        Items: []domain.DeliveryTrendPoint{{
            BucketStart: "2026-02-02",
            BucketEnd:   "2026-02-08",
            BucketLabel: "2026-W06",
            Throughput: domain.DeliveryTrendThroughput{TotalIssuesDone: 14},
            SpeedMetricsDays: domain.DeliveryTrendSpeedMetrics{},
        }},
    }

    resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/metrics/delivery/trend?start_date=2026-02-01&end_date=2026-03-01", nil)
    defer resp.Body.Close()
    AssertStatusCode(t, resp, http.StatusOK)
}
```

Also add tests for:
- `400` invalid bucket
- `422` project/group mismatch
- `401` unauthorized.

**Step 2: Run test to verify it fails**

Run: `go test ./test/integration -run TestMetrics_DeliveryTrend -v`

Expected: FAIL (missing mock field, route, or handler).

**Step 3: Write minimal implementation updates**

In `test/integration/testapp.go`:
- Add `DeliveryTrend *domain.DeliveryTrendResponse` field to mock.
- Implement `GetDeliveryTrendMetrics` method.
- Register `/api/v1/metrics/delivery/trend` route in test mux.

**Step 4: Run test to verify it passes**

Run: `go test ./test/integration -run TestMetrics_DeliveryTrend -v`

Expected: PASS.

**Step 5: Commit**

```bash
git add test/integration/testapp.go test/integration/metrics_test.go
git commit -m "test: add integration coverage for delivery trend endpoint"
```

---

### Task 7: Update OpenAPI Spec (`docs/openapi.yaml`)

**Files:**
- Modify: `docs/openapi.yaml`

**Step 1: Write failing contract-check test (lightweight)**

Create a grep-based assertion in CI/local script style (if no parser test exists):

```bash
grep -n "/metrics/delivery/trend" docs/openapi.yaml
```

Expected before change: no match / non-zero exit.

**Step 2: Add path + schemas (minimal full contract)**

Add:
- Path `GET /metrics/delivery/trend`
- Query params: `start_date`, `end_date`, `project_id`, `group_path`, `assignee`, `bucket`, `timezone`, `include_empty_buckets`
- Responses: `200`, `400`, `401`, `422`, `500`
- Schemas:
  - `DeliveryTrendPoint`
  - `AvgP85MetricNullable`
  - `DeliveryTrendCorrelation`
  - `DeliveryTrendResponse`

Use this `bucket` block:

```yaml
- in: query
  name: bucket
  required: false
  schema:
    type: string
    enum: [day, week, month]
    default: week
```

**Step 3: Validate spec structure**

Run:
```bash
go test ./... -run OpenAPI -v
```

If no OpenAPI tests exist, run:
```bash
grep -n "DeliveryTrendResponse" docs/openapi.yaml
```

Expected: new path and schemas are present.

**Step 4: Commit**

```bash
git add docs/openapi.yaml
git commit -m "docs: add openapi contract for delivery trend endpoint"
```

---

### Task 8: Update Bruno Collection in `docs/`

**Files:**
- Create: `docs/GitLab Engineering Metrics API/Metrics/Retorna serie temporal de delivery para correlacao.yml`
- Modify: `docs/GitLab Engineering Metrics API/Metrics/Retorna metricas de qualidade e gargalos.yml`
- Modify: `docs/GitLab Engineering Metrics API/Metrics/Retorna snapshot de WIP atual e aging.yml`
- Modify: `docs/GitLab Engineering Metrics API/Metrics/Retorna deep dive de ghost work.yml`
- (Optional ordering check) `docs/GitLab Engineering Metrics API/opencollection.yml`

**Step 1: Write the new Bruno request file with examples**

```yaml
info:
  name: Retorna serie temporal de delivery para correlacao
  type: http
  seq: 2
  tags:
    - Metrics

http:
  method: GET
  url: "{{baseUrl}}/metrics/delivery/trend"
  params:
    - name: start_date
      value: "2026-02-01"
      type: query
    - name: end_date
      value: "2026-03-01"
      type: query
    - name: bucket
      value: week
      type: query
    - name: timezone
      value: America/Sao_Paulo
      type: query
    - name: include_empty_buckets
      value: "true"
      type: query
  auth: inherit
```

Include examples for `200`, `400`, `401`, `422`, `500`.

**Step 2: Re-sequence existing Metrics requests**

Set:
- delivery -> `seq: 1`
- delivery trend -> `seq: 2`
- quality -> `seq: 3`
- wip -> `seq: 4`
- ghost-work -> `seq: 5`

**Step 3: Validate collection readability**

Run: `grep -n "seq:" "docs/GitLab Engineering Metrics API/Metrics"/*.yml`

Expected: sequential order without duplicates.

**Step 4: Commit**

```bash
git add "docs/GitLab Engineering Metrics API/Metrics/Retorna serie temporal de delivery para correlacao.yml" "docs/GitLab Engineering Metrics API/Metrics/Retorna metricas de qualidade e gargalos.yml" "docs/GitLab Engineering Metrics API/Metrics/Retorna snapshot de WIP atual e aging.yml" "docs/GitLab Engineering Metrics API/Metrics/Retorna deep dive de ghost work.yml"
git commit -m "docs: add bruno request for delivery trend endpoint"
```

---

### Task 9: Final Verification and Smoke Tests

**Files:**
- Verify only (no new files)

**Step 1: Run focused test suite**

```bash
go test ./internal/domain -run DeliveryTrend -v
go test ./internal/services -run DeliveryTrend -v
go test ./internal/repositories -run DeliveryTrend -v
go test ./internal/http/handlers -run DeliveryTrend -v
go test ./test/integration -run DeliveryTrend -v
```

Expected: all PASS.

**Step 2: Run broad safety suite**

```bash
go test ./internal/http/handlers/... ./internal/services/... ./internal/repositories/...
go test ./test/integration/...
go build ./cmd/api
```

Expected: PASS.

**Step 3: Manual API smoke test (real server)**

```bash
curl -s -H "X-Client-ID: myclient" -H "X-Client-Secret: mysecret" "http://localhost:8080/api/v1/metrics/delivery/trend?start_date=2026-02-01&end_date=2026-03-01&bucket=week&timezone=America/Sao_Paulo&include_empty_buckets=true"
```

Expected: JSON with `period`, `bucket`, `timezone`, `items[]`, `correlation`.

**Step 4: Validate error semantics**

Run:
```bash
curl -i -H "X-Client-ID: myclient" -H "X-Client-Secret: mysecret" "http://localhost:8080/api/v1/metrics/delivery/trend?start_date=2026-03-01&end_date=2026-02-01"
curl -i -H "X-Client-ID: myclient" -H "X-Client-Secret: mysecret" "http://localhost:8080/api/v1/metrics/delivery/trend?start_date=2026-02-01&end_date=2026-03-01&project_id=275&group_path=web"
```

Expected:
- first request -> `400`
- second request -> `422`.

**Step 5: Final commit**

```bash
git add .
git commit -m "feat: implement delivery trend endpoint with docs and bruno collection"
```

---

## Notes for Execution Quality

- Use `@superpowers:test-driven-development` before each coding task (write failing test first).
- Use `@superpowers:verification-before-completion` before declaring done.
- Keep DRY: reuse `buildFilterConditions` and JSONB assignee helpers; do not duplicate filter SQL.
- Keep YAGNI: no materialized views or schema migrations for this endpoint.
- Prefer small commits per task to simplify rollback and code review.
