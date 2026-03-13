# Issue Identity and Project Context Standardization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Padronizar todos os endpoints que retornam issues para sempre incluir `project_id` + `project_path` e os dois identificadores de issue (`issue_id` interno + `gitlab_issue_id`), com filtros consistentes para navegação entre endpoints.

**Architecture:** Vamos introduzir um contrato único de identidade de issue nos DTOs de domínio e propagar esse contrato nos repositórios SQL, handlers HTTP e documentação OpenAPI/collection. A estratégia prioriza compatibilidade retroativa: manter campos já usados (`id`, `issue_iid`, `gitlab_iid`) enquanto adiciona nomes explícitos e consistentes. A implementação segue TDD por endpoint (falha -> implementação mínima -> passa), com commits pequenos por tarefa.

**Tech Stack:** Go (net/http), PostgreSQL (views `vw_issue_lifecycle_metrics`), testes `go test` (unit + integration), OpenAPI (`docs/openapi.yaml`), API collection YAML (`docs/GitLab Engineering Metrics API/**`).

---

### Task 0: Preparar Worktree Isolada

**Files:**
- Create: `../gitlab-engineering-metrics-api-issue-contract/` (worktree)
- Modify: none
- Test: none

**Step 1: Criar worktree dedicada com skill obrigatória**

Use `@superpowers:using-git-worktrees` para criar uma worktree nova a partir da branch atual.

**Step 2: Validar que a worktree está limpa**

Run: `git status`
Expected: `nothing to commit, working tree clean`

**Step 3: Validar bootstrap local**

Run: `go test ./internal/domain -run TestNewIssuesService -v`
Expected: PASS (sanity check do ambiente)

**Step 4: Commit checkpoint de setup (opcional)**

```bash
git add -A
git commit -m "chore: start isolated worktree for issue contract standardization"
```

**Step 5: Confirmar contexto para execução do plano**

Run: `pwd`
Expected: caminho da nova worktree

### Task 1: Definir Contrato Unificado de Identidade de Issue

**Files:**
- Create: `internal/domain/issue_identity_contract_test.go`
- Modify: `internal/domain/issue.go:6-43`
- Modify: `internal/domain/timeline.go:6-33`
- Modify: `internal/domain/ghost_work.go:4-48`
- Modify: `internal/domain/metrics.go:9-17`
- Test: `internal/domain/issue_identity_contract_test.go`

**Step 1: Write the failing test**

```go
func TestIssueIdentityContract_JSONTags(t *testing.T) {
	issue := domain.IssueListItem{}
	raw, _ := json.Marshal(issue)
	got := string(raw)

	requiredKeys := []string{
		"issue_id", "gitlab_issue_id", "issue_iid",
		"project_id", "project_path",
	}
	for _, key := range requiredKeys {
		if !strings.Contains(got, "\""+key+"\"") {
			t.Fatalf("missing key %s in IssueListItem", key)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain -run TestIssueIdentityContract_JSONTags -v`
Expected: FAIL com chave ausente (`project_path`, `gitlab_issue_id` ou `issue_id`)

**Step 3: Write minimal implementation**

```go
// internal/domain/issue.go
type IssueListItem struct {
	IssueID       int    `json:"issue_id,omitempty"`
	ID            int    `json:"id,omitempty"` // backward compatibility
	GitlabIssueID int    `json:"gitlab_issue_id,omitempty"`
	IssueIID      int    `json:"issue_iid,omitempty"`
	ProjectID     int    `json:"project_id,omitempty"`
	ProjectPath   string `json:"project_path,omitempty"`
	// ...campos atuais
}

// internal/domain/timeline.go
type IssueSummary struct {
	IssueID       int    `json:"issue_id,omitempty"`
	ID            int    `json:"id,omitempty"`
	GitlabIssueID int    `json:"gitlab_issue_id,omitempty"`
	GitlabIID     int    `json:"gitlab_iid,omitempty"`
	IssueIID      int    `json:"issue_iid,omitempty"`
	ProjectID     int    `json:"project_id,omitempty"`
	ProjectPath   string `json:"project_path,omitempty"`
	// ...campos atuais
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain -run TestIssueIdentityContract_JSONTags -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/issue.go internal/domain/timeline.go internal/domain/ghost_work.go internal/domain/metrics.go internal/domain/issue_identity_contract_test.go
git commit -m "feat: define unified issue identity and project context contract"
```

### Task 2: Padronizar `/api/v1/issues` (lista + filtros por identificador)

**Files:**
- Modify: `internal/domain/issue.go:32-43`
- Modify: `internal/http/handlers/issues_handler.go:55-127`
- Modify: `internal/services/issues_service.go:78-99`
- Modify: `internal/repositories/issues_repository.go:50-190`
- Modify: `internal/repositories/issues_repository_test.go:23-145`
- Modify: `internal/http/handlers/issues_handler_test.go:47-310`
- Modify: `internal/services/issues_service_test.go:70-172`
- Modify: `test/integration/issues_test.go:12-190`
- Test: `internal/http/handlers/issues_handler_test.go`

**Step 1: Write the failing test**

```go
func TestIssuesHandler_List_ReturnsUnifiedIssueFields(t *testing.T) {
	mockService := &mockIssuesService{listResponse: &domain.IssuesListResponse{
		Items: []domain.IssueListItem{{
			IssueID: 10, GitlabIssueID: 99123, IssueIID: 42,
			ProjectID: 123, ProjectPath: "group/project",
		}},
	}}

	handler := NewIssuesHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/issues", nil)
	rr := httptest.NewRecorder()
	handler.List(rr, req)

	assertBodyContains(t, rr.Body.String(), "\"issue_id\":10")
	assertBodyContains(t, rr.Body.String(), "\"gitlab_issue_id\":99123")
	assertBodyContains(t, rr.Body.String(), "\"project_path\":\"group/project\"")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/http/handlers -run TestIssuesHandler_List_ReturnsUnifiedIssueFields -v`
Expected: FAIL (campos ausentes no JSON)

**Step 3: Write minimal implementation**

```go
// internal/domain/issue.go
type IssuesFilter struct {
	IssueID       int `json:"issue_id,omitempty"`
	GitlabIssueID int `json:"gitlab_issue_id,omitempty"`
	IssueIID      int `json:"issue_iid,omitempty"`
	ProjectID     int `json:"project_id,omitempty"`
	// ...
}

// internal/http/handlers/issues_handler.go
if v := r.URL.Query().Get("issue_id"); v != "" {
	issueID, err := strconv.Atoi(v)
	if err != nil || issueID <= 0 {
		responses.BadRequest(w, requestID, "issue_id must be a positive integer")
		return
	}
	filter.IssueID = issueID
}

// internal/repositories/issues_repository.go (SELECT)
SELECT
	issue_id,
	project_id,
	project_path,
	gitlab_issue_id,
	issue_iid,
	issue_title,
	...
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/http/handlers ./internal/services ./internal/repositories -run "Issues" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/issue.go internal/http/handlers/issues_handler.go internal/services/issues_service.go internal/repositories/issues_repository.go internal/repositories/issues_repository_test.go internal/http/handlers/issues_handler_test.go internal/services/issues_service_test.go test/integration/issues_test.go
git commit -m "feat: standardize issues list payload and identifier filters"
```

### Task 3: Padronizar `/api/v1/issues/:id/timeline` (campos de projeto + ids)

**Files:**
- Modify: `internal/domain/timeline.go:6-33`
- Modify: `internal/repositories/timeline_repository.go:46-91`
- Modify: `internal/http/handlers/timeline_handler_test.go:26-84`
- Modify: `test/integration/issues_test.go:213-279`
- Test: `internal/http/handlers/timeline_handler_test.go`

**Step 1: Write the failing test**

```go
func TestTimelineHandler_Get_ReturnsUnifiedIssueFields(t *testing.T) {
	mockService := &mockIssuesService{timelineResponse: &domain.IssueTimelineResponse{
		Issue: domain.IssueSummary{
			IssueID: 1, GitlabIssueID: 99123, IssueIID: 42,
			ProjectID: 101, ProjectPath: "group/project",
		},
	}}

	h := NewTimelineHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/1/timeline", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	assertBodyContains(t, rr.Body.String(), "\"project_path\":\"group/project\"")
	assertBodyContains(t, rr.Body.String(), "\"gitlab_issue_id\":99123")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/http/handlers -run TestTimelineHandler_Get_ReturnsUnifiedIssueFields -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
// internal/repositories/timeline_repository.go
query := `
	SELECT
		i.id,
		i.gitlab_issue_id,
		i.iid as issue_iid,
		i.iid as gitlab_iid,
		i.project_id,
		p.path as project_path,
		i.title,
		i.metadata_labels,
		i.assignees,
		COALESCE(i.current_canonical_state, 'UNKNOWN') as current_canonical_state,
		i.gitlab_created_at
	FROM issues i
	JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
`
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/http/handlers ./test/integration -run "Timeline" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/timeline.go internal/repositories/timeline_repository.go internal/http/handlers/timeline_handler_test.go test/integration/issues_test.go
git commit -m "feat: add project path and gitlab issue id to timeline response"
```

### Task 4: Padronizar issues embutidas em `/api/v1/metrics/wip`

**Files:**
- Modify: `internal/domain/metrics.go:9-17`
- Modify: `internal/domain/metrics.go:97-104`
- Modify: `internal/http/handlers/wip_handler.go:31-46`
- Modify: `internal/repositories/metrics_repository.go:150-186`
- Modify: `internal/repositories/metrics_repository.go:488-541`
- Modify: `internal/http/handlers/wip_handler_test.go:13-121`
- Modify: `test/integration/metrics_test.go:199-237`
- Test: `internal/http/handlers/wip_handler_test.go`

**Step 1: Write the failing test**

```go
func TestWipHandler_Get_AgingWIPIncludesIssueIdentityFields(t *testing.T) {
	mockService := &mockMetricsService{wipMetrics: &domain.WipMetricsResponse{
		AgingWIP: []domain.AgingIssue{{
			IssueID: 1, GitlabIssueID: 99123, IssueIID: 42,
			ProjectID: 101, ProjectPath: "group/project",
		}},
	}}

	h := NewWipHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/wip", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	assertBodyContains(t, rr.Body.String(), "\"project_path\":\"group/project\"")
	assertBodyContains(t, rr.Body.String(), "\"issue_id\":1")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/http/handlers -run TestWipHandler_Get_AgingWIPIncludesIssueIdentityFields -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
// internal/repositories/metrics_repository.go (getAgingWIP)
SELECT
	issue_id,
	gitlab_issue_id,
	issue_iid,
	project_id,
	project_path,
	issue_title,
	COALESCE(ARRAY(SELECT jsonb_array_elements_text(...)), ARRAY[]::text[]) as assignees,
	current_canonical_state,
	COALESCE(EXTRACT(DAY FROM (NOW() - first_in_progress_at)), 0)::int as days_in_state
FROM vw_issue_lifecycle_metrics
...
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/http/handlers ./internal/repositories ./test/integration -run "WIP|Aging" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/metrics.go internal/http/handlers/wip_handler.go internal/repositories/metrics_repository.go internal/http/handlers/wip_handler_test.go test/integration/metrics_test.go
git commit -m "feat: standardize aging wip issue identity and project context"
```

### Task 5: Padronizar `/api/v1/metrics/ghost-work` e filtros cruzados

**Files:**
- Modify: `internal/domain/ghost_work.go:4-48`
- Modify: `internal/domain/metrics.go:97-104`
- Modify: `internal/http/handlers/ghost_work_handler.go:39-66`
- Modify: `internal/repositories/ghost_work_repository.go:77-149`
- Modify: `internal/repositories/ghost_work_repository.go:254-291`
- Modify: `internal/repositories/ghost_work_repository_test.go:76-116`
- Modify: `internal/http/handlers/ghost_work_handler_test.go:26-209`
- Test: `internal/http/handlers/ghost_work_handler_test.go`

**Step 1: Write the failing test**

```go
func TestGhostWorkHandler_Get_ReturnsUnifiedIssueFields(t *testing.T) {
	mockService := &mockGhostWorkService{response: &domain.GhostWorkMetricsResponse{
		Issues: []domain.GhostWorkIssue{{
			IssueID: 1, GitlabIssueID: 99123, IssueIID: 42,
			ProjectID: 101, ProjectPath: "group/project",
		}},
	}}

	h := NewGhostWorkHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/ghost-work", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	assertBodyContains(t, rr.Body.String(), "\"gitlab_issue_id\":99123")
	assertBodyContains(t, rr.Body.String(), "\"project_id\":101")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/http/handlers -run TestGhostWorkHandler_Get_ReturnsUnifiedIssueFields -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
// internal/repositories/ghost_work_repository.go
SELECT DISTINCT ON (m.issue_id)
	m.issue_id,
	m.gitlab_issue_id,
	m.issue_iid,
	m.project_id,
	m.project_path,
	m.issue_title,
	...
FROM vw_issue_lifecycle_metrics m
...

// filtros novos (buildFilterConditions)
if filter.IssueID > 0 {
	argIdx++
	conditions = append(conditions, fmt.Sprintf("m.issue_id = $%d", argIdx))
	args = append(args, filter.IssueID)
}
if filter.GitlabIssueID > 0 {
	argIdx++
	conditions = append(conditions, fmt.Sprintf("m.gitlab_issue_id = $%d", argIdx))
	args = append(args, filter.GitlabIssueID)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/http/handlers ./internal/repositories -run "GhostWork" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/ghost_work.go internal/domain/metrics.go internal/http/handlers/ghost_work_handler.go internal/repositories/ghost_work_repository.go internal/repositories/ghost_work_repository_test.go internal/http/handlers/ghost_work_handler_test.go
git commit -m "feat: standardize ghost-work issue contract and identifier filters"
```

### Task 6: Atualizar Documentacao de Contrato e Rodar Regressao Completa

**Files:**
- Modify: `docs/openapi.yaml:611-707`
- Modify: `docs/GitLab Engineering Metrics API/Issues/Lista issues para drill-down operacional.yml:128-147`
- Modify: `docs/GitLab Engineering Metrics API/Issues/Retorna timeline detalhada de uma issue.yml:42-65`
- Modify: `docs/GitLab Engineering Metrics API/Metrics/Retorna snapshot de WIP atual e aging.yml`
- Modify: `docs/GitLab Engineering Metrics API/Metrics/Retorna deep dive de ghost work.yml`
- Test: `test/integration/issues_test.go`
- Test: `test/integration/metrics_test.go`

**Step 1: Write the failing test**

```go
func TestIssues_List_Success_ContainsStandardizedIdentityFields(t *testing.T) {
	// após parse da resposta
	if result.Items[0].ProjectPath == "" || result.Items[0].GitlabIssueID == 0 || result.Items[0].IssueID == 0 {
		t.Fatal("missing standardized identity fields")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./test/integration -run "Issues_List_Success_ContainsStandardizedIdentityFields|Metrics_WIP_Success" -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```yaml
# docs/openapi.yaml (IssueListItem)
IssueListItem:
  type: object
  properties:
    issue_id:
      type: integer
    id:
      type: integer
      description: "Deprecated alias of issue_id"
    gitlab_issue_id:
      type: integer
    issue_iid:
      type: integer
    project_id:
      type: integer
    project_path:
      type: string
```

**Step 4: Run test to verify it passes + full regression**

Run: `go test ./...`
Expected: PASS (todos os pacotes)

**Step 5: Commit**

```bash
git add docs/openapi.yaml "docs/GitLab Engineering Metrics API/Issues/Lista issues para drill-down operacional.yml" "docs/GitLab Engineering Metrics API/Issues/Retorna timeline detalhada de uma issue.yml" "docs/GitLab Engineering Metrics API/Metrics/Retorna snapshot de WIP atual e aging.yml" "docs/GitLab Engineering Metrics API/Metrics/Retorna deep dive de ghost work.yml" test/integration/issues_test.go test/integration/metrics_test.go
git commit -m "docs: align issue contracts and filters across API endpoints"
```

## Definition of Done

- Todos os endpoints com payload de issue incluem: `issue_id`, `gitlab_issue_id`, `issue_iid`, `project_id`, `project_path`.
- `/api/v1/issues` e `/api/v1/metrics/ghost-work` aceitam filtro por identificador (`issue_id`, `gitlab_issue_id`, opcionalmente `issue_iid`).
- Endpoint `/api/v1/issues/:id/timeline` expõe os mesmos campos no objeto `issue`.
- `docs/openapi.yaml` e collection YAML refletem o contrato final.
- Suite completa `go test ./...` passa sem regressões.

## Notes for Execution

- Use `@superpowers:test-driven-development` em cada tarefa (red -> green -> refactor mínimo).
- Mantenha DRY/YAGNI: não criar abstrações novas sem repetição concreta.
- Não remover campos antigos imediatamente (`id`, `gitlab_iid`) para evitar quebra; marcar como deprecated na documentação.
- Commits frequentes: um por tarefa.
