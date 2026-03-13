package integration

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestCatalog_Projects_Unauthorized(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeRequest(t, ts, http.MethodGet, "/api/v1/projects", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusUnauthorized)
}

func TestCatalog_Projects_Success(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.CatalogService.Projects = []domain.Project{
		{
			ID:           1,
			Name:         "Test Project",
			Path:         "group/test-project",
			TotalIssues:  42,
			LastSyncedAt: time.Now(),
		},
		{
			ID:           2,
			Name:         "Another Project",
			Path:         "group/another-project",
			TotalIssues:  10,
			LastSyncedAt: time.Now(),
		},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/projects", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertContentType(t, resp, "application/json")

	var projects []domain.Project
	ParseResponse(t, resp, &projects)

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	if projects[0].Name != "Test Project" {
		t.Errorf("Expected first project name to be 'Test Project', got %s", projects[0].Name)
	}
}

func TestCatalog_Projects_WithSearch(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.CatalogService.Projects = []domain.Project{
		{ID: 1, Name: "Test Project", Path: "group/test-project"},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/projects?search=test", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)

	var projects []domain.Project
	ParseResponse(t, resp, &projects)

	if len(projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(projects))
	}
}

func TestCatalog_Projects_InvalidSearch(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.CatalogService.Err = errors.New("search term must be at least 3 characters")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/projects?search=ab", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestCatalog_Projects_ServiceError(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.CatalogService.Err = errors.New("database connection failed")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/projects", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusInternalServerError)
}

func TestCatalog_Projects_MethodNotAllowed(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeAuthenticatedRequest(t, ts, http.MethodPost, "/api/v1/projects", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusMethodNotAllowed)
}

func TestCatalog_Groups_Success(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.CatalogService.Groups = []domain.Group{
		{
			GroupPath:    "company/engineering",
			ProjectCount: 10,
			TotalIssues:  100,
		},
		{
			GroupPath:    "company/product",
			ProjectCount: 5,
			TotalIssues:  50,
		},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/groups", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertContentType(t, resp, "application/json")

	var groups []domain.Group
	ParseResponse(t, resp, &groups)

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	if groups[0].GroupPath != "company/engineering" {
		t.Errorf("Expected first group path to be 'company/engineering', got %s", groups[0].GroupPath)
	}
}

func TestCatalog_Groups_Unauthorized(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeRequest(t, ts, http.MethodGet, "/api/v1/groups", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusUnauthorized)
}

func TestCatalog_Users_Success(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.CatalogService.Users = []domain.User{
		{
			Username:                  "john.doe",
			DisplayName:               "John Doe",
			ActiveIssues:              3,
			CompletedIssuesLast30Days: 10,
		},
		{
			Username:                  "jane.smith",
			DisplayName:               "Jane Smith",
			ActiveIssues:              2,
			CompletedIssuesLast30Days: 15,
		},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/users", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertContentType(t, resp, "application/json")

	var users []domain.User
	ParseResponse(t, resp, &users)

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	if users[0].Username != "john.doe" {
		t.Errorf("Expected first user username to be 'john.doe', got %s", users[0].Username)
	}
}

func TestCatalog_Users_Unauthorized(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeRequest(t, ts, http.MethodGet, "/api/v1/users", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusUnauthorized)
}

func TestCatalog_ResponseHasRequestID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.CatalogService.Projects = []domain.Project{
		{ID: 1, Name: "Test Project"},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/projects", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertHeaderExists(t, resp, "X-Request-ID")
}
