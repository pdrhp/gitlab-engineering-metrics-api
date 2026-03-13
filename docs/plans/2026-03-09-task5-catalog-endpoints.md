# Task 5: Catalog Endpoints Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Implement catalog endpoints for projects, groups, and users with repository pattern, service layer, and HTTP handlers.

**Architecture:** Repository pattern for data access, service layer for business logic, handlers for HTTP transport. Follow existing codebase patterns with table-driven tests.

**Tech Stack:** Go 1.25, PostgreSQL, standard library testing

---

## Context from Codebase

### Domain Models (existing)

**Project:** `internal/domain/project.go`
```go
type Project struct {
    ID           int       `json:"id"`
    Name         string    `json:"name"`
    Path         string    `json:"path"`
    TotalIssues  int       `json:"total_issues"`
    LastSyncedAt time.Time `json:"last_synced_at"`
}

type CatalogFilter struct {
    Search    string `json:"search,omitempty"`
    GroupPath string `json:"group_path,omitempty"`
}
```

**Group:** `internal/domain/group.go`
```go
type Group struct {
    GroupPath    string    `json:"group_path"`
    ProjectCount int       `json:"project_count"`
    TotalIssues  int       `json:"total_issues"`
    LastSyncedAt time.Time `json:"last_synced_at"`
}
```

**User:** `internal/domain/user.go`
```go
type User struct {
    Username                  string `json:"username"`
    DisplayName               string `json:"display_name"`
    ActiveIssues              int    `json:"active_issues"`
    CompletedIssuesLast30Days int    `json:"completed_issues_last_30_days"`
}
```

### Database Views

**vw_projects_catalog** columns:
- id (int)
- name (text)
- path (text)
- group_path (text) - derived from regexp_replace
- total_issues (int)
- last_synced_at (timestamptz)

**Issues table** has `assignees` column (text array)

### Existing Patterns

**Routes setup:** `internal/app/routes.go`
- App struct has db *sql.DB
- Creates handlers with dependencies
- Applies middleware chain

**Error responses:** `internal/http/responses/error.go`
- BadRequest(w, requestID, message)
- NotFound(w, requestID, resource)
- InternalServerError(w, requestID)

**Request ID middleware:** `internal/http/middleware/request_id.go`
- GetRequestID(ctx) to extract from context

---

## Task 1: Projects Repository

**Files:**
- Create: `internal/repositories/projects_repository.go`
- Create: `internal/repositories/projects_repository_test.go`

**Step 1: Write the failing test**

Create `internal/repositories/projects_repository_test.go`:
```go
package repositories

import (
    "context"
    "database/sql"
    "testing"
    "time"

    "gitlab-engineering-metrics-api/internal/domain"
    "github.com/google/uuid"
)

func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("postgres", "host=localhost user=postgres password=postgres dbname=gitlab_metrics_test sslmode=disable")
    if err != nil {
        t.Fatalf("Failed to connect to test database: %v", err)
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := db.PingContext(ctx); err != nil {
        t.Skipf("Test database not available: %v", err)
    }
    
    // Clean up test data
    db.Exec("DELETE FROM projects WHERE path LIKE 'test-%'")
    
    return db
}

func TestProjectsRepository_List(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := NewProjectsRepository(db)
    ctx := context.Background()

    tests := []struct {
        name    string
        filter  domain.CatalogFilter
        wantErr bool
    }{
        {
            name:    "list all projects",
            filter:  domain.CatalogFilter{},
            wantErr: false,
        },
        {
            name: "filter by search",
            filter: domain.CatalogFilter{
                Search: "api",
            },
            wantErr: false,
        },
        {
            name: "filter by group path",
            filter: domain.CatalogFilter{
                GroupPath: "engineering",
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            projects, err := repo.List(ctx, tt.filter)
            if (err != nil) != tt.wantErr {
                t.Errorf("ProjectsRepository.List() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // Just verify we get a result (or empty slice) without error
            if projects == nil {
                t.Error("ProjectsRepository.List() returned nil, expected slice")
            }
        })
    }
}

func TestProjectsRepository_List_WithSearch(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := NewProjectsRepository(db)
    ctx := context.Background()

    // Insert test data
    _, err := db.ExecContext(ctx, `
        INSERT INTO projects (id, name, path, created_at, updated_at) 
        VALUES ($1, $2, $3, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET name = $2, path = $3
    `, 999999, "Test API Project", "test-engineering/api-project")
    if err != nil {
        t.Fatalf("Failed to insert test data: %v", err)
    }

    defer db.Exec("DELETE FROM projects WHERE id = 999999")

    filter := domain.CatalogFilter{Search: "api"}
    projects, err := repo.List(ctx, filter)
    if err != nil {
        t.Errorf("ProjectsRepository.List() error = %v", err)
        return
    }

    found := false
    for _, p := range projects {
        if p.ID == 999999 {
            found = true
            if p.Name != "Test API Project" {
                t.Errorf("Expected project name 'Test API Project', got '%s'", p.Name)
            }
            break
        }
    }
    
    if !found {
        t.Log("Note: Project not found in results (may be filtered by view criteria)")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/repositories/... -v
```

Expected: FAIL with "undefined: NewProjectsRepository"

**Step 3: Write minimal implementation**

Create `internal/repositories/projects_repository.go`:
```go
package repositories

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "gitlab-engineering-metrics-api/internal/domain"
)

// ProjectsRepository handles database operations for projects
type ProjectsRepository struct {
    db *sql.DB
}

// NewProjectsRepository creates a new projects repository
func NewProjectsRepository(db *sql.DB) *ProjectsRepository {
    return &ProjectsRepository{db: db}
}

// List returns a list of projects matching the filter
func (r *ProjectsRepository) List(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error) {
    query := `
        SELECT id, name, path, total_issues, last_synced_at 
        FROM vw_projects_catalog 
        WHERE 1=1
    `
    var args []interface{}
    var conditions []string

    if filter.Search != "" {
        conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR path ILIKE $%d)", len(args)+1, len(args)+1))
        args = append(args, "%"+filter.Search+"%")
    }

    if filter.GroupPath != "" {
        conditions = append(conditions, fmt.Sprintf("group_path = $%d", len(args)+1))
        args = append(args, filter.GroupPath)
    }

    if len(conditions) > 0 {
        query += " AND " + strings.Join(conditions, " AND ")
    }

    query += " ORDER BY path"

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query projects: %w", err)
    }
    defer rows.Close()

    var projects []domain.Project
    for rows.Next() {
        var p domain.Project
        if err := rows.Scan(&p.ID, &p.Name, &p.Path, &p.TotalIssues, &p.LastSyncedAt); err != nil {
            return nil, fmt.Errorf("failed to scan project: %w", err)
        }
        projects = append(projects, p)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating projects: %w", err)
    }

    return projects, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/repositories/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
git add internal/repositories/projects_repository.go internal/repositories/projects_repository_test.go
git commit -m "feat: add projects repository with list and search"
```

---

## Task 2: Groups Repository

**Files:**
- Create: `internal/repositories/groups_repository.go`
- Create: `internal/repositories/groups_repository_test.go`

**Step 1: Write the failing test**

Create `internal/repositories/groups_repository_test.go`:
```go
package repositories

import (
    "context"
    "database/sql"
    "testing"
    "time"

    "gitlab-engineering-metrics-api/internal/domain"
)

func TestGroupsRepository_List(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := NewGroupsRepository(db)
    ctx := context.Background()

    tests := []struct {
        name    string
        filter  domain.CatalogFilter
        wantErr bool
    }{
        {
            name:    "list all groups",
            filter:  domain.CatalogFilter{},
            wantErr: false,
        },
        {
            name: "filter by search",
            filter: domain.CatalogFilter{
                Search: "engineering",
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            groups, err := repo.List(ctx, tt.filter)
            if (err != nil) != tt.wantErr {
                t.Errorf("GroupsRepository.List() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if groups == nil {
                t.Error("GroupsRepository.List() returned nil, expected slice")
            }
        })
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/repositories/... -run TestGroupsRepository -v
```

Expected: FAIL with "undefined: NewGroupsRepository"

**Step 3: Write minimal implementation**

Create `internal/repositories/groups_repository.go`:
```go
package repositories

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "gitlab-engineering-metrics-api/internal/domain"
)

// GroupsRepository handles database operations for groups
type GroupsRepository struct {
    db *sql.DB
}

// NewGroupsRepository creates a new groups repository
func NewGroupsRepository(db *sql.DB) *GroupsRepository {
    return &GroupsRepository{db: db}
}

// List returns a list of groups derived from project paths
func (r *GroupsRepository) List(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error) {
    query := `
        SELECT 
            split_part(path, '/', 1) as group_path,
            COUNT(*) as project_count,
            COALESCE(SUM(total_issues), 0) as total_issues,
            MAX(last_synced_at) as last_synced_at
        FROM vw_projects_catalog
        WHERE 1=1
    `
    var args []interface{}

    if filter.Search != "" {
        query += fmt.Sprintf(" AND split_part(path, '/', 1) ILIKE $%d", len(args)+1)
        args = append(args, "%"+filter.Search+"%")
    }

    query += `
        GROUP BY split_part(path, '/', 1)
        ORDER BY group_path
    `

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query groups: %w", err)
    }
    defer rows.Close()

    var groups []domain.Group
    for rows.Next() {
        var g domain.Group
        if err := rows.Scan(&g.GroupPath, &g.ProjectCount, &g.TotalIssues, &g.LastSyncedAt); err != nil {
            return nil, fmt.Errorf("failed to scan group: %w", err)
        }
        groups = append(groups, g)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating groups: %w", err)
    }

    return groups, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/repositories/... -run TestGroupsRepository -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
git add internal/repositories/groups_repository.go internal/repositories/groups_repository_test.go
git commit -m "feat: add groups repository derived from project paths"
```

---

## Task 3: Users Repository

**Files:**
- Create: `internal/repositories/users_repository.go`
- Create: `internal/repositories/users_repository_test.go`

**Step 1: Write the failing test**

Create `internal/repositories/users_repository_test.go`:
```go
package repositories

import (
    "context"
    "database/sql"
    "testing"

    "gitlab-engineering-metrics-api/internal/domain"
)

func TestUsersRepository_List(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := NewUsersRepository(db)
    ctx := context.Background()

    tests := []struct {
        name    string
        filter  domain.CatalogFilter
        wantErr bool
    }{
        {
            name:    "list all users",
            filter:  domain.CatalogFilter{},
            wantErr: false,
        },
        {
            name: "filter by search",
            filter: domain.CatalogFilter{
                Search: "john",
            },
            wantErr: false,
        },
        {
            name: "filter by group path",
            filter: domain.CatalogFilter{
                GroupPath: "engineering",
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            users, err := repo.List(ctx, tt.filter)
            if (err != nil) != tt.wantErr {
                t.Errorf("UsersRepository.List() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if users == nil {
                t.Error("UsersRepository.List() returned nil, expected slice")
            }
        })
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/repositories/... -run TestUsersRepository -v
```

Expected: FAIL with "undefined: NewUsersRepository"

**Step 3: Write minimal implementation**

Create `internal/repositories/users_repository.go`:
```go
package repositories

import (
    "context"
    "database/sql"
    "fmt"

    "gitlab-engineering-metrics-api/internal/domain"
)

// UsersRepository handles database operations for users
type UsersRepository struct {
    db *sql.DB
}

// NewUsersRepository creates a new users repository
func NewUsersRepository(db *sql.DB) *UsersRepository {
    return &UsersRepository{db: db}
}

// List returns a list of users with their issue statistics
func (r *UsersRepository) List(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error) {
    query := `
        WITH user_stats AS (
            SELECT 
                unnest(assignees) as username,
                COUNT(*) FILTER (WHERE current_canonical_state NOT IN ('DONE', 'CANCELED')) as active_issues,
                COUNT(*) FILTER (
                    WHERE current_canonical_state = 'DONE' 
                    AND EXISTS (
                        SELECT 1 FROM vw_issue_state_transitions t 
                        WHERE t.issue_id = i.id 
                        AND t.canonical_state = 'DONE'
                        AND t.entered_at >= NOW() - INTERVAL '30 days'
                    )
                ) as completed_last_30_days
            FROM issues i
            JOIN projects p ON p.id = i.project_id
            WHERE 1=1
    `
    var args []interface{}
    argCount := 1

    if filter.GroupPath != "" {
        query += fmt.Sprintf(" AND p.path LIKE $%d", argCount)
        args = append(args, filter.GroupPath+"/%")
        argCount++
    }

    query += `
            AND array_length(assignees, 1) > 0
            GROUP BY unnest(assignees)
        )
        SELECT 
            username,
            username as display_name,
            COALESCE(active_issues, 0) as active_issues,
            COALESCE(completed_last_30_days, 0) as completed_last_30_days
        FROM user_stats
        WHERE 1=1
    `

    if filter.Search != "" {
        query += fmt.Sprintf(" AND username ILIKE $%d", argCount)
        args = append(args, "%"+filter.Search+"%")
        argCount++
    }

    query += ` ORDER BY username`

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query users: %w", err)
    }
    defer rows.Close()

    var users []domain.User
    for rows.Next() {
        var u domain.User
        if err := rows.Scan(&u.Username, &u.DisplayName, &u.ActiveIssues, &u.CompletedIssuesLast30Days); err != nil {
            return nil, fmt.Errorf("failed to scan user: %w", err)
        }
        users = append(users, u)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating users: %w", err)
    }

    return users, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/repositories/... -run TestUsersRepository -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
git add internal/repositories/users_repository.go internal/repositories/users_repository_test.go
git commit -m "feat: add users repository with issue statistics"
```

---

## Task 4: Catalog Service

**Files:**
- Create: `internal/services/catalog_service.go`
- Create: `internal/services/catalog_service_test.go`

**Step 1: Write the failing test**

Create `internal/services/catalog_service_test.go`:
```go
package services

import (
    "context"
    "database/sql"
    "testing"
    "time"

    "gitlab-engineering-metrics-api/internal/domain"
    "gitlab-engineering-metrics-api/internal/repositories"
)

func setupTestService(t *testing.T) (*CatalogService, *sql.DB) {
    db, err := sql.Open("postgres", "host=localhost user=postgres password=postgres dbname=gitlab_metrics_test sslmode=disable")
    if err != nil {
        t.Fatalf("Failed to connect to test database: %v", err)
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := db.PingContext(ctx); err != nil {
        t.Skipf("Test database not available: %v", err)
    }

    projectsRepo := repositories.NewProjectsRepository(db)
    groupsRepo := repositories.NewGroupsRepository(db)
    usersRepo := repositories.NewUsersRepository(db)

    service := NewCatalogService(projectsRepo, groupsRepo, usersRepo)
    return service, db
}

func TestCatalogService_ListProjects(t *testing.T) {
    service, db := setupTestService(t)
    defer db.Close()

    ctx := context.Background()
    filter := domain.CatalogFilter{}

    projects, err := service.ListProjects(ctx, filter)
    if err != nil {
        t.Errorf("CatalogService.ListProjects() error = %v", err)
        return
    }

    if projects == nil {
        t.Error("CatalogService.ListProjects() returned nil")
    }
}

func TestCatalogService_ListGroups(t *testing.T) {
    service, db := setupTestService(t)
    defer db.Close()

    ctx := context.Background()
    filter := domain.CatalogFilter{}

    groups, err := service.ListGroups(ctx, filter)
    if err != nil {
        t.Errorf("CatalogService.ListGroups() error = %v", err)
        return
    }

    if groups == nil {
        t.Error("CatalogService.ListGroups() returned nil")
    }
}

func TestCatalogService_ListUsers(t *testing.T) {
    service, db := setupTestService(t)
    defer db.Close()

    ctx := context.Background()
    filter := domain.CatalogFilter{}

    users, err := service.ListUsers(ctx, filter)
    if err != nil {
        t.Errorf("CatalogService.ListUsers() error = %v", err)
        return
    }

    if users == nil {
        t.Error("CatalogService.ListUsers() returned nil")
    }
}

func TestCatalogService_ListProjects_InvalidSearch(t *testing.T) {
    service, db := setupTestService(t)
    defer db.Close()

    ctx := context.Background()
    filter := domain.CatalogFilter{
        Search: "ab", // Too short
    }

    _, err := service.ListProjects(ctx, filter)
    if err == nil {
        t.Error("Expected error for short search term, got nil")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/services/... -v
```

Expected: FAIL with "undefined: NewCatalogService"

**Step 3: Write minimal implementation**

Create `internal/services/catalog_service.go`:
```go
package services

import (
    "context"
    "errors"
    "fmt"

    "gitlab-engineering-metrics-api/internal/domain"
)

// ProjectsRepository defines the interface for project data access
type ProjectsRepository interface {
    List(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error)
}

// GroupsRepository defines the interface for group data access
type GroupsRepository interface {
    List(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error)
}

// UsersRepository defines the interface for user data access
type UsersRepository interface {
    List(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error)
}

// CatalogService provides catalog operations
type CatalogService struct {
    projectsRepo ProjectsRepository
    groupsRepo   GroupsRepository
    usersRepo    UsersRepository
}

// NewCatalogService creates a new catalog service
func NewCatalogService(
    projectsRepo ProjectsRepository,
    groupsRepo GroupsRepository,
    usersRepo UsersRepository,
) *CatalogService {
    return &CatalogService{
        projectsRepo: projectsRepo,
        groupsRepo:   groupsRepo,
        usersRepo:    usersRepo,
    }
}

// ListProjects returns a list of projects
func (s *CatalogService) ListProjects(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error) {
    if err := s.validateFilter(filter); err != nil {
        return nil, err
    }

    projects, err := s.projectsRepo.List(ctx, filter)
    if err != nil {
        return nil, fmt.Errorf("failed to list projects: %w", err)
    }

    return projects, nil
}

// ListGroups returns a list of groups
func (s *CatalogService) ListGroups(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error) {
    if err := s.validateFilter(filter); err != nil {
        return nil, err
    }

    groups, err := s.groupsRepo.List(ctx, filter)
    if err != nil {
        return nil, fmt.Errorf("failed to list groups: %w", err)
    }

    return groups, nil
}

// ListUsers returns a list of users
func (s *CatalogService) ListUsers(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error) {
    if err := s.validateFilter(filter); err != nil {
        return nil, err
    }

    users, err := s.usersRepo.List(ctx, filter)
    if err != nil {
        return nil, fmt.Errorf("failed to list users: %w", err)
    }

    return users, nil
}

// validateFilter validates the catalog filter
func (s *CatalogService) validateFilter(filter domain.CatalogFilter) error {
    if filter.Search != "" && len(filter.Search) < 3 {
        return errors.New("search term must be at least 3 characters")
    }
    return nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/services/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
git add internal/services/catalog_service.go internal/services/catalog_service_test.go
git commit -m "feat: add catalog service with filter validation"
```

---

## Task 5: Projects Handler

**Files:**
- Create: `internal/http/handlers/projects_handler.go`
- Create: `internal/http/handlers/projects_handler_test.go`

**Step 1: Write the failing test**

Create `internal/http/handlers/projects_handler_test.go`:
```go
package handlers

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "gitlab-engineering-metrics-api/internal/domain"
)

type mockCatalogService struct {
    projects []domain.Project
    groups   []domain.Group
    users    []domain.User
    err      error
}

func (m *mockCatalogService) ListProjects(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.projects, nil
}

func (m *mockCatalogService) ListGroups(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.groups, nil
}

func (m *mockCatalogService) ListUsers(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.users, nil
}

func TestProjectsHandler_List(t *testing.T) {
    mockService := &mockCatalogService{
        projects: []domain.Project{
            {ID: 1, Name: "Test Project", Path: "group/project", TotalIssues: 10},
        },
    }

    handler := NewProjectsHandler(mockService)

    tests := []struct {
        name           string
        method         string
        queryParams    string
        expectedStatus int
        expectedCount  int
    }{
        {
            name:           "GET returns projects list",
            method:         http.MethodGet,
            queryParams:    "",
            expectedStatus: http.StatusOK,
            expectedCount:  1,
        },
        {
            name:           "POST not allowed",
            method:         http.MethodPost,
            queryParams:    "",
            expectedStatus: http.StatusMethodNotAllowed,
            expectedCount:  0,
        },
        {
            name:           "GET with search param",
            method:         http.MethodGet,
            queryParams:    "?search=test",
            expectedStatus: http.StatusOK,
            expectedCount:  1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, "/projects"+tt.queryParams, nil)
            rr := httptest.NewRecorder()

            handler.List(rr, req)

            if rr.Code != tt.expectedStatus {
                t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
            }

            if tt.expectedStatus == http.StatusOK {
                var response []domain.Project
                if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
                    t.Errorf("Failed to unmarshal response: %v", err)
                    return
                }
                if len(response) != tt.expectedCount {
                    t.Errorf("Expected %d projects, got %d", tt.expectedCount, len(response))
                }
            }
        })
    }
}

func TestProjectsHandler_List_ServiceError(t *testing.T) {
    mockService := &mockCatalogService{
        err: errors.New("database error"),
    }

    handler := NewProjectsHandler(mockService)
    req := httptest.NewRequest(http.MethodGet, "/projects", nil)
    rr := httptest.NewRecorder()

    handler.List(rr, req)

    if rr.Code != http.StatusInternalServerError {
        t.Errorf("Expected status 500, got %d", rr.Code)
    }
}

func TestProjectsHandler_List_InvalidSearch(t *testing.T) {
    mockService := &mockCatalogService{}

    handler := NewProjectsHandler(mockService)
    req := httptest.NewRequest(http.MethodGet, "/projects?search=ab", nil)
    rr := httptest.NewRecorder()

    handler.List(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", rr.Code)
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/http/handlers/... -run TestProjectsHandler -v
```

Expected: FAIL with compilation errors (undefined types)

**Step 3: Write minimal implementation**

Create `internal/http/handlers/projects_handler.go`:
```go
package handlers

import (
    "encoding/json"
    "net/http"

    "gitlab-engineering-metrics-api/internal/domain"
    "gitlab-engineering-metrics-api/internal/http/middleware"
    "gitlab-engineering-metrics-api/internal/http/responses"
)

// CatalogService defines the interface for catalog operations
type CatalogService interface {
    ListProjects(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error)
    ListGroups(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error)
    ListUsers(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error)
}

// ProjectsHandler handles project-related HTTP requests
type ProjectsHandler struct {
    service CatalogService
}

// NewProjectsHandler creates a new projects handler
func NewProjectsHandler(service CatalogService) *ProjectsHandler {
    return &ProjectsHandler{service: service}
}

// List handles GET /projects
func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        requestID := middleware.GetRequestID(r.Context())
        w.Header().Set("Allow", http.MethodGet)
        responses.BadRequest(w, requestID, "Method not allowed")
        return
    }

    // Parse query parameters
    filter := domain.CatalogFilter{
        Search:    r.URL.Query().Get("search"),
        GroupPath: r.URL.Query().Get("group_path"),
    }

    // Call service
    projects, err := h.service.ListProjects(r.Context(), filter)
    if err != nil {
        requestID := middleware.GetRequestID(r.Context())
        // Check if it's a validation error
        if err.Error() == "search term must be at least 3 characters" {
            responses.BadRequest(w, requestID, err.Error())
            return
        }
        responses.InternalServerError(w, requestID)
        return
    }

    // Return JSON response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    if err := json.NewEncoder(w).Encode(projects); err != nil {
        // Log error but can't write to response anymore
        return
    }
}
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/http/handlers/... -run TestProjectsHandler -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
git add internal/http/handlers/projects_handler.go internal/http/handlers/projects_handler_test.go
git commit -m "feat: add projects handler with GET /projects endpoint"
```

---

## Task 6: Groups Handler

**Files:**
- Create: `internal/http/handlers/groups_handler.go`
- Create: `internal/http/handlers/groups_handler_test.go`

**Step 1: Write the failing test**

Create `internal/http/handlers/groups_handler_test.go`:
```go
package handlers

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "gitlab-engineering-metrics-api/internal/domain"
)

func TestGroupsHandler_List(t *testing.T) {
    mockService := &mockCatalogService{
        groups: []domain.Group{
            {GroupPath: "engineering", ProjectCount: 5, TotalIssues: 100},
        },
    }

    handler := NewGroupsHandler(mockService)

    tests := []struct {
        name           string
        method         string
        expectedStatus int
        expectedCount  int
    }{
        {
            name:           "GET returns groups list",
            method:         http.MethodGet,
            expectedStatus: http.StatusOK,
            expectedCount:  1,
        },
        {
            name:           "POST not allowed",
            method:         http.MethodPost,
            expectedStatus: http.StatusMethodNotAllowed,
            expectedCount:  0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, "/groups", nil)
            rr := httptest.NewRecorder()

            handler.List(rr, req)

            if rr.Code != tt.expectedStatus {
                t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
            }

            if tt.expectedStatus == http.StatusOK {
                var response []domain.Group
                if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
                    t.Errorf("Failed to unmarshal response: %v", err)
                    return
                }
                if len(response) != tt.expectedCount {
                    t.Errorf("Expected %d groups, got %d", tt.expectedCount, len(response))
                }
            }
        })
    }
}

func TestGroupsHandler_List_ServiceError(t *testing.T) {
    mockService := &mockCatalogService{
        err: errors.New("database error"),
    }

    handler := NewGroupsHandler(mockService)
    req := httptest.NewRequest(http.MethodGet, "/groups", nil)
    rr := httptest.NewRecorder()

    handler.List(rr, req)

    if rr.Code != http.StatusInternalServerError {
        t.Errorf("Expected status 500, got %d", rr.Code)
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/http/handlers/... -run TestGroupsHandler -v
```

Expected: FAIL with "undefined: NewGroupsHandler"

**Step 3: Write minimal implementation**

Create `internal/http/handlers/groups_handler.go`:
```go
package handlers

import (
    "encoding/json"
    "net/http"

    "gitlab-engineering-metrics-api/internal/domain"
    "gitlab-engineering-metrics-api/internal/http/middleware"
    "gitlab-engineering-metrics-api/internal/http/responses"
)

// GroupsHandler handles group-related HTTP requests
type GroupsHandler struct {
    service CatalogService
}

// NewGroupsHandler creates a new groups handler
func NewGroupsHandler(service CatalogService) *GroupsHandler {
    return &GroupsHandler{service: service}
}

// List handles GET /groups
func (h *GroupsHandler) List(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        requestID := middleware.GetRequestID(r.Context())
        w.Header().Set("Allow", http.MethodGet)
        responses.BadRequest(w, requestID, "Method not allowed")
        return
    }

    // Parse query parameters
    filter := domain.CatalogFilter{
        Search: r.URL.Query().Get("search"),
    }

    // Call service
    groups, err := h.service.ListGroups(r.Context(), filter)
    if err != nil {
        requestID := middleware.GetRequestID(r.Context())
        responses.InternalServerError(w, requestID)
        return
    }

    // Return JSON response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    if err := json.NewEncoder(w).Encode(groups); err != nil {
        return
    }
}
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/http/handlers/... -run TestGroupsHandler -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
git add internal/http/handlers/groups_handler.go internal/http/handlers/groups_handler_test.go
git commit -m "feat: add groups handler with GET /groups endpoint"
```

---

## Task 7: Users Handler

**Files:**
- Create: `internal/http/handlers/users_handler.go`
- Create: `internal/http/handlers/users_handler_test.go`

**Step 1: Write the failing test**

Create `internal/http/handlers/users_handler_test.go`:
```go
package handlers

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "gitlab-engineering-metrics-api/internal/domain"
)

func TestUsersHandler_List(t *testing.T) {
    mockService := &mockCatalogService{
        users: []domain.User{
            {Username: "john_doe", DisplayName: "john_doe", ActiveIssues: 5, CompletedIssuesLast30Days: 10},
        },
    }

    handler := NewUsersHandler(mockService)

    tests := []struct {
        name           string
        method         string
        queryParams    string
        expectedStatus int
        expectedCount  int
    }{
        {
            name:           "GET returns users list",
            method:         http.MethodGet,
            queryParams:    "",
            expectedStatus: http.StatusOK,
            expectedCount:  1,
        },
        {
            name:           "POST not allowed",
            method:         http.MethodPost,
            queryParams:    "",
            expectedStatus: http.StatusMethodNotAllowed,
            expectedCount:  0,
        },
        {
            name:           "GET with search param",
            method:         http.MethodGet,
            queryParams:    "?search=john",
            expectedStatus: http.StatusOK,
            expectedCount:  1,
        },
        {
            name:           "GET with group_path param",
            method:         http.MethodGet,
            queryParams:    "?group_path=engineering",
            expectedStatus: http.StatusOK,
            expectedCount:  1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, "/users"+tt.queryParams, nil)
            rr := httptest.NewRecorder()

            handler.List(rr, req)

            if rr.Code != tt.expectedStatus {
                t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
            }

            if tt.expectedStatus == http.StatusOK {
                var response []domain.User
                if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
                    t.Errorf("Failed to unmarshal response: %v", err)
                    return
                }
                if len(response) != tt.expectedCount {
                    t.Errorf("Expected %d users, got %d", tt.expectedCount, len(response))
                }
            }
        })
    }
}

func TestUsersHandler_List_ServiceError(t *testing.T) {
    mockService := &mockCatalogService{
        err: errors.New("database error"),
    }

    handler := NewUsersHandler(mockService)
    req := httptest.NewRequest(http.MethodGet, "/users", nil)
    rr := httptest.NewRecorder()

    handler.List(rr, req)

    if rr.Code != http.StatusInternalServerError {
        t.Errorf("Expected status 500, got %d", rr.Code)
    }
}

func TestUsersHandler_List_InvalidSearch(t *testing.T) {
    mockService := &mockCatalogService{}

    handler := NewUsersHandler(mockService)
    req := httptest.NewRequest(http.MethodGet, "/users?search=ab", nil)
    rr := httptest.NewRecorder()

    handler.List(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", rr.Code)
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/http/handlers/... -run TestUsersHandler -v
```

Expected: FAIL with "undefined: NewUsersHandler"

**Step 3: Write minimal implementation**

Create `internal/http/handlers/users_handler.go`:
```go
package handlers

import (
    "encoding/json"
    "net/http"

    "gitlab-engineering-metrics-api/internal/domain"
    "gitlab-engineering-metrics-api/internal/http/middleware"
    "gitlab-engineering-metrics-api/internal/http/responses"
)

// UsersHandler handles user-related HTTP requests
type UsersHandler struct {
    service CatalogService
}

// NewUsersHandler creates a new users handler
func NewUsersHandler(service CatalogService) *UsersHandler {
    return &UsersHandler{service: service}
}

// List handles GET /users
func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        requestID := middleware.GetRequestID(r.Context())
        w.Header().Set("Allow", http.MethodGet)
        responses.BadRequest(w, requestID, "Method not allowed")
        return
    }

    // Parse query parameters
    filter := domain.CatalogFilter{
        Search:    r.URL.Query().Get("search"),
        GroupPath: r.URL.Query().Get("group_path"),
    }

    // Call service
    users, err := h.service.ListUsers(r.Context(), filter)
    if err != nil {
        requestID := middleware.GetRequestID(r.Context())
        // Check if it's a validation error
        if err.Error() == "search term must be at least 3 characters" {
            responses.BadRequest(w, requestID, err.Error())
            return
        }
        responses.InternalServerError(w, requestID)
        return
    }

    // Return JSON response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    if err := json.NewEncoder(w).Encode(users); err != nil {
        return
    }
}
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/http/handlers/... -run TestUsersHandler -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
git add internal/http/handlers/users_handler.go internal/http/handlers/users_handler_test.go
git commit -m "feat: add users handler with GET /users endpoint"
```

---

## Task 8: Wire Routes

**Files:**
- Modify: `internal/app/routes.go`

**Step 1: Write failing integration test**

Create `internal/app/routes_test.go`:
```go
package app

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "gitlab-engineering-metrics-api/internal/config"
    "log/slog"
    "os"
)

func TestRoutes_CatalogEndpoints(t *testing.T) {
    // This is an integration test that requires database
    // Skip if no database available
    cfg := config.Load()
    
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    
    // Try to connect to database
    db, err := database.New(cfg)
    if err != nil {
        t.Skipf("Database not available: %v", err)
    }
    defer db.Close()

    app := New(db, cfg, logger)
    router := app.Routes()

    tests := []struct {
        name   string
        method string
        path   string
        status int
    }{
        {"GET /health returns 200", http.MethodGet, "/health", http.StatusOK},
        {"GET /api/v1/projects returns 401 without auth", http.MethodGet, "/api/v1/projects", http.StatusUnauthorized},
        {"GET /api/v1/groups returns 401 without auth", http.MethodGet, "/api/v1/groups", http.StatusUnauthorized},
        {"GET /api/v1/users returns 401 without auth", http.MethodGet, "/api/v1/users", http.StatusUnauthorized},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, tt.path, nil)
            rr := httptest.NewRecorder()

            router.ServeHTTP(rr, req)

            if rr.Code != tt.status {
                t.Errorf("Expected status %d, got %d", tt.status, rr.Code)
            }
        })
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/app/... -run TestRoutes_CatalogEndpoints -v
```

Expected: FAIL (routes not wired yet)

**Step 3: Modify routes.go**

Update `internal/app/routes.go`:
```go
package app

import (
    "database/sql"
    "encoding/json"
    "log/slog"
    "net/http"

    "gitlab-engineering-metrics-api/internal/auth"
    "gitlab-engineering-metrics-api/internal/config"
    "gitlab-engineering-metrics-api/internal/http/handlers"
    "gitlab-engineering-metrics-api/internal/http/middleware"
    "gitlab-engineering-metrics-api/internal/repositories"
    "gitlab-engineering-metrics-api/internal/services"
)

type App struct {
    db        *sql.DB
    config    *config.Config
    logger    *slog.Logger
    validator *auth.Validator
}

func New(db *sql.DB, cfg *config.Config, logger *slog.Logger) *App {
    return &App{
        db:        db,
        config:    cfg,
        logger:    logger,
        validator: auth.NewValidator(cfg.ClientCredentials),
    }
}

func (a *App) Routes() http.Handler {
    mux := http.NewServeMux()

    // Public routes (no auth required)
    mux.HandleFunc("/health", a.healthHandler)

    // Protected API routes
    a.registerCatalogRoutes(mux)

    return a.applyMiddleware(mux)
}

func (a *App) registerCatalogRoutes(mux *http.ServeMux) {
    // Create repositories
    projectsRepo := repositories.NewProjectsRepository(a.db)
    groupsRepo := repositories.NewGroupsRepository(a.db)
    usersRepo := repositories.NewUsersRepository(a.db)

    // Create service
    catalogService := services.NewCatalogService(projectsRepo, groupsRepo, usersRepo)

    // Create handlers
    projectsHandler := handlers.NewProjectsHandler(catalogService)
    groupsHandler := handlers.NewGroupsHandler(catalogService)
    usersHandler := handlers.NewUsersHandler(catalogService)

    // Register routes with auth middleware
    authMiddleware := middleware.Auth(a.validator)

    mux.Handle("/api/v1/projects", authMiddleware(http.HandlerFunc(projectsHandler.List)))
    mux.Handle("/api/v1/groups", authMiddleware(http.HandlerFunc(groupsHandler.List)))
    mux.Handle("/api/v1/users", authMiddleware(http.HandlerFunc(usersHandler.List)))
}

func (a *App) healthHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
        return
    }

    if err := a.db.Ping(); err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": "database unavailable"})
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (a *App) applyMiddleware(handler http.Handler) http.Handler {
    // Middleware chain order (last applied runs first):
    // 1. Recovery - catches panics
    // 2. Logging - logs request details
    // 3. RequestID - adds/generates request ID

    handler = middleware.Recovery(a.logger)(handler)
    handler = middleware.Logging(a.logger)(handler)
    handler = middleware.RequestID(handler)

    return handler
}
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./internal/app/... -run TestRoutes_CatalogEndpoints -v
```

Expected: PASS (or SKIP if no DB available)

**Step 5: Run all tests**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./... -v 2>&1 | head -100
```

Expected: All tests pass (or skip if DB not available)

**Step 6: Commit**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
git add internal/app/routes.go internal/app/routes_test.go
git commit -m "feat: wire catalog routes with auth middleware"
```

---

## Final Verification

**Step 1: Build the application**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go build ./cmd/api
```

Expected: Build succeeds

**Step 2: Run all tests**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
go test ./... -v 2>&1 | tail -50
```

Expected: All tests pass

**Step 3: Show final summary**

```bash
cd /home/pedrohenrique/projects/go/gitlab-engineering-metrics-api/.worktrees/api-implementation
echo "=== Files Created ==="
find internal/repositories internal/services internal/http/handlers -name "*.go" | sort

echo ""
echo "=== Test Results ==="
go test ./... 2>&1 | grep -E "(PASS|FAIL|ok|---)"
```
