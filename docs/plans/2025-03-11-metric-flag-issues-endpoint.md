# Implementar metric_flag no endpoint /api/v1/issues

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implementar o parâmetro `metric_flag` no endpoint `/api/v1/issues` para filtrar issues por métricas específicas (bypass, rework, blocked)

**Architecture:** O campo `metric_flag` já existe no domínio `IssuesFilter` mas não é processado no handler. Precisamos adicionar o processamento no handler e filtrar no repositório baseado nas flags de métricas da view `vw_issue_lifecycle_metrics`.

**Tech Stack:** Go, PostgreSQL, JSONB

---

## Contexto

O endpoint `/api/v1/issues` deve aceitar um parâmetro `metric_flag` que filtra issues baseado em:
- `bypass` - issues que pularam IN_PROGRESS (ghost work)
- `rework` - issues com retrabalho (voltaram de QA para DEV)
- `blocked` - issues que estiveram bloqueadas

O campo já existe em `internal/domain/issue.go` no struct `IssuesFilter` mas não é usado.

---

## Task 1: Adicionar processamento do metric_flag no IssuesHandler

**Files:**
- Modify: `internal/http/handlers/issues_handler.go:55-70`

**Step 1: Adicionar parsing do metric_flag no handler**

```go
// After parsing state parameter (around line 60)
if metricFlag := r.URL.Query().Get("metric_flag"); metricFlag != "" {
    filter.MetricFlag = metricFlag
}
```

**Step 2: Run build to check for errors**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/http/handlers/issues_handler.go
git commit -m "feat: add metric_flag parsing in issues handler"
```

---

## Task 2: Implementar filtro de metric_flag no IssuesRepository

**Files:**
- Modify: `internal/repositories/issues_repository.go:50-80`

**Step 1: Adicionar condição para metric_flag no buildFilterConditions**

```go
if filter.MetricFlag != "" {
    switch filter.MetricFlag {
    case "bypass":
        conditions = append(conditions, "skipped_in_progress_flag = true")
    case "rework":
        argIdx++
        conditions = append(conditions, fmt.Sprintf("qa_to_dev_return_count > $%d", argIdx))
        args = append(args, 0)
    case "blocked":
        argIdx++
        conditions = append(conditions, fmt.Sprintf("blocked_time_hours > $%d", argIdx))
        args = append(args, 0)
    }
}
```

**Step 2: Run build to check for errors**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/repositories/issues_repository.go
git commit -m "feat: implement metric_flag filtering in issues repository"
```

---

## Task 3: Adicionar validação do metric_flag no handler

**Files:**
- Modify: `internal/http/handlers/issues_handler.go:150-165`

**Step 1: Adicionar validação do metric_flag**

Adicionar na função `isValidationError`:
```go
"invalid metric_flag",
```

E adicionar validação após parsing do filtro (antes da chamada ao service):
```go
if filter.MetricFlag != "" {
    validFlags := []string{"bypass", "rework", "blocked"}
    isValid := false
    for _, valid := range validFlags {
        if filter.MetricFlag == valid {
            isValid = true
            break
        }
    }
    if !isValid {
        responses.BadRequest(w, requestID, "metric_flag must be one of: bypass, rework, blocked")
        return
    }
}
```

**Step 2: Run build to check for errors**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/http/handlers/issues_handler.go
git commit -m "feat: add metric_flag validation"
```

---

## Task 4: Atualizar testes do handler

**Files:**
- Modify: `internal/http/handlers/issues_handler_test.go`

**Step 1: Adicionar teste para metric_flag válido**

```go
{
    name:       "GET with metric_flag bypass",
    method:     http.MethodGet,
    queryParams: "?metric_flag=bypass",
    wantStatus: http.StatusOK,
    setupMock: func(m *mockIssuesService) {
        m.On("ListIssues", mock.Anything, mock.MatchedBy(func(f domain.IssuesFilter) bool {
            return f.MetricFlag == "bypass"
        })).Return(&domain.IssuesListResponse{
            Items: []domain.IssueListItem{
                {ID: 1, Title: "Issue with bypass"},
            },
            Total:    1,
            Page:     1,
            PageSize: 25,
        }, nil)
    },
}
```

**Step 2: Adicionar teste para metric_flag inválido**

```go
{
    name:       "GET with invalid metric_flag",
    method:     http.MethodGet,
    queryParams: "?metric_flag=invalid",
    wantStatus: http.StatusBadRequest,
}
```

**Step 3: Run tests**

Run: `go test ./internal/http/handlers/... -v -run "TestIssuesHandler"`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/http/handlers/issues_handler_test.go
git commit -m "test: add tests for metric_flag parameter"
```

---

## Task 5: Atualizar testes do repositório

**Files:**
- Modify: `internal/repositories/issues_repository_test.go`

**Step 1: Adicionar teste para filtro bypass**

```go
{
    name: "filter by bypass metric_flag",
    filter: domain.IssuesFilter{
        MetricFlag: "bypass",
        Page:       1,
        PageSize:   10,
    },
    setupMock: func(mock sqlmock.Sqlmock) {
        mock.ExpectQuery("SELECT COUNT\\(\\*\\).*skipped_in_progress_flag = true").
            WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
        
        mock.ExpectQuery("SELECT.*skipped_in_progress_flag = true").
            WillReturnRows(sqlmock.NewRows([]string{"issue_id", "project_id", "issue_iid", "issue_title", "assignees", "current_canonical_state", "lead_time_days", "cycle_time_days", "blocked_time_hours", "qa_to_dev_return_count"}).
                AddRow(1, 1, 1, "Bypass Issue", []byte("[\"user1\"]"), "DONE", 10.0, 5.0, 0.0, 0))
    },
    wantTotal: 5,
    wantErr:   false,
}
```

**Step 2: Run tests**

Run: `go test ./internal/repositories/... -v -run "TestIssuesRepository"`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/repositories/issues_repository_test.go
git commit -m "test: add repository tests for metric_flag filtering"
```

---

## Task 6: Testar manualmente os endpoints

**Step 1: Testar sem metric_flag**

Run:
```bash
curl -s -H "X-Client-ID:myclient" -H "X-Client-Secret:mysecret" \
  "http://localhost:8080/api/v1/issues?page=1&page_size=5" | jq '.total'
```
Expected: Número total de issues

**Step 2: Testar com metric_flag=bypass**

Run:
```bash
curl -s -H "X-Client-ID:myclient" -H "X-Client-Secret:mysecret" \
  "http://localhost:8080/api/v1/issues?metric_flag=bypass&page=1&page_size=5" | jq '.total'
```
Expected: Número de issues com bypass (ghost work) - deve ser menor que o total

**Step 3: Testar com metric_flag=rework**

Run:
```bash
curl -s -H "X-Client-ID:myclient" -H "X-Client-Secret:mysecret" \
  "http://localhost:8080/api/v1/issues?metric_flag=rework&page=1&page_size=5" | jq '.total'
```
Expected: Número de issues com retrabalho

**Step 4: Testar com metric_flag=blocked**

Run:
```bash
curl -s -H "X-Client-ID:myclient" -H "X-Client-Secret:mysecret" \
  "http://localhost:8080/api/v1/issues?metric_flag=blocked&page=1&page_size=5" | jq '.total'
```
Expected: Número de issues que estiveram bloqueadas

**Step 5: Testar com metric_flag inválido**

Run:
```bash
curl -s -H "X-Client-ID:myclient" -H "X-Client-Secret:mysecret" \
  "http://localhost:8080/api/v1/issues?metric_flag=invalid" | jq '.code, .message'
```
Expected: BAD_REQUEST com mensagem de erro

**Step 6: Commit final**

```bash
git commit -m "feat: complete metric_flag implementation for issues endpoint"
```

---

## Documentação Bruno

**Step 1: Atualizar documentação da collection**

Adicionar à documentação Bruno em `docs/GitLab Engineering Metrics API/Issues/Lista issues para drill-down operacional.yml`:

```yaml
params:
  - name: metric_flag
    value: bypass
    type: query
    disabled: true
    description: Filtra issues por métrica - bypass (ghost work), rework (retrabalho), blocked (bloqueadas)
```

---

## Summary

After implementation:
- `/api/v1/issues` retorna todas as issues (comportamento atual)
- `/api/v1/issues?metric_flag=bypass` retorna apenas issues com ghost work (pularam IN_PROGRESS)
- `/api/v1/issues?metric_flag=rework` retorna apenas issues com retrabalho (voltaram de QA)
- `/api/v1/issues?metric_flag=blocked` retorna apenas issues que estiveram bloqueadas

Estes filtros podem ser combinados com outros parâmetros como `group_path`, `assignee`, `project_id`, etc.
