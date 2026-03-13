package integration

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestIssues_List_Success(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.IssuesList = &domain.IssuesListResponse{
		Items: []domain.IssueListItem{
			{
				IssueID:               1,
				ProjectID:             101,
				ProjectPath:           "group/project",
				GitlabIssueID:         1001,
				IssueIID:              42,
				Title:                 "Fix critical bug",
				Assignees:             []string{"john.doe"},
				CurrentCanonicalState: "in_progress",
				LeadTimeDays:          5.5,
				CycleTimeDays:         3.2,
				BlockedTimeHours:      2.5,
				QAToDevReturnCount:    1,
			{
				IssueID:               2,
				ProjectID:             101,
				ProjectPath:           "group/project",
				GitlabIssueID:         1002,
				IssueIID:              43,
				Title:                 "Add new feature",
				Assignees:             []string{"jane.smith"},
				CurrentCanonicalState: "qa_review",
				LeadTimeDays:          3.0,
				CycleTimeDays:         2.0,
				BlockedTimeHours:      0,
				QAToDevReturnCount:    0,
			},
		},
		Page:     1,
		PageSize: 20,
		Total:    2,
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertContentType(t, resp, "application/json")

	var result domain.IssuesListResponse
	ParseResponse(t, resp, &result)

	if len(result.Items) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(result.Items))
	}

	if result.Items[0].Title != "Fix critical bug" {
		t.Errorf("Expected first issue title to be 'Fix critical bug', got %s", result.Items[0].Title)
	}

	if result.Total != 2 {
		t.Errorf("Expected total of 2, got %d", result.Total)
	}

	// Verify new identity fields are present
	if result.Items[0].ProjectPath != "group/project" {
		t.Errorf("Expected project_path to be 'group/project', got %s", result.Items[0].ProjectPath)
	}
	if result.Items[0].GitlabIssueID != 1001 {
		t.Errorf("Expected gitlab_issue_id to be 1001, got %d", result.Items[0].GitlabIssueID)
	}
	if result.Items[0].IssueID != 1 {
		t.Errorf("Expected issue_id to be 1, got %d", result.Items[0].IssueID)
	}
}

func TestIssues_List_WithPagination(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.IssuesList = &domain.IssuesListResponse{
		Items: []domain.IssueListItem{
			{
				IssueID:  3,
				Title:    "Issue on page 2",
				IssueIID: 45,
			},
		},
		Page:     2,
		PageSize: 10,
		Total:    15,
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?page=2&page_size=10", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)

	var result domain.IssuesListResponse
	ParseResponse(t, resp, &result)

	if result.Page != 2 {
		t.Errorf("Expected page 2, got %d", result.Page)
	}

	if result.PageSize != 10 {
		t.Errorf("Expected page size 10, got %d", result.PageSize)
	}
}

func TestIssues_List_WithProjectID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.IssuesList = &domain.IssuesListResponse{
		Items: []domain.IssueListItem{
			{
				IssueID:   1,
				ProjectID: 123,
				Title:     "Project-specific issue",
			},
		},
		Total: 1,
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?project_id=123", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)

	var result domain.IssuesListResponse
	ParseResponse(t, resp, &result)

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Items))
	}
}

func TestIssues_List_InvalidProjectID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?project_id=invalid", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestIssues_List_InvalidPage(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?page=invalid", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestIssues_List_InvalidPageSize(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?page_size=invalid", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestIssues_List_WithIssueID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.IssuesList = &domain.IssuesListResponse{
		Items: []domain.IssueListItem{
			{
				IssueID:   123,
				ProjectID: 101,
				Title:     "Issue by ID",
			},
		},
		Total: 1,
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?issue_id=123", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)

	var result domain.IssuesListResponse
	ParseResponse(t, resp, &result)

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Items))
	}
}

func TestIssues_List_WithGitlabIssueID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.IssuesList = &domain.IssuesListResponse{
		Items: []domain.IssueListItem{
			{
				IssueID:       1,
				GitlabIssueID: 456,
				ProjectID:     101,
				Title:         "Issue by GitLab ID",
			},
		},
		Total: 1,
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?gitlab_issue_id=456", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)

	var result domain.IssuesListResponse
	ParseResponse(t, resp, &result)

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Items))
	}
}

func TestIssues_List_WithIssueIID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.IssuesList = &domain.IssuesListResponse{
		Items: []domain.IssueListItem{
			{
				IssueID:   1,
				IssueIID:  789,
				ProjectID: 101,
				Title:     "Issue by IID",
			},
		},
		Total: 1,
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?issue_iid=789", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)

	var result domain.IssuesListResponse
	ParseResponse(t, resp, &result)

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Items))
	}
}

func TestIssues_List_InvalidIssueID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?issue_id=invalid", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestIssues_List_InvalidGitlabIssueID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?gitlab_issue_id=invalid", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestIssues_List_InvalidIssueIID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?issue_iid=invalid", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestIssues_List_WithFilters(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.IssuesList = &domain.IssuesListResponse{
		Items: []domain.IssueListItem{
			{
				IssueID:               1,
				Title:                 "Filtered issue",
				Assignees:             []string{"john.doe"},
				CurrentCanonicalState: "done",
			},
		},
		Total: 1,
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues?group_path=engineering&assignee=john.doe&state=done", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)

	var result domain.IssuesListResponse
	ParseResponse(t, resp, &result)

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Items))
	}
}

func TestIssues_List_Unauthorized(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeRequest(t, ts, http.MethodGet, "/api/v1/issues", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusUnauthorized)
}

func TestIssues_List_ServiceError(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.ListErr = errors.New("database error")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusInternalServerError)
}

func TestIssues_Timeline_Success(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.Timeline = &domain.IssueTimelineResponse{
		Issue: domain.IssueSummary{
			IssueID:               1,
			GitlabIssueID:         99123,
			IssueIID:              42,
			ProjectID:             101,
			ProjectPath:           "group/project",
			Title:                 "Fix critical bug",
			MetadataLabels:        []string{"bug", "critical"},
			Assignees:             []string{"john.doe"},
			CurrentCanonicalState: "done",
			GitlabCreatedAt:       time.Now().Add(-7 * 24 * time.Hour),
		},
		Timeline: []domain.TimelineEvent{
			{
				Type:                        "state_change",
				Timestamp:                   time.Now().Add(-6 * 24 * time.Hour),
				Actor:                       "john.doe",
				FromState:                   "opened",
				ToState:                     "in_progress",
				DurationInPreviousStateMins: 1440,
				IsRework:                    false,
			},
			{
				Type:                        "state_change",
				Timestamp:                   time.Now().Add(-3 * 24 * time.Hour),
				Actor:                       "john.doe",
				FromState:                   "in_progress",
				ToState:                     "qa_review",
				DurationInPreviousStateMins: 4320,
				IsRework:                    false,
			},
			{
				Type:                        "state_change",
				Timestamp:                   time.Now().Add(-1 * 24 * time.Hour),
				Actor:                       "qa.team",
				FromState:                   "qa_review",
				ToState:                     "done",
				DurationInPreviousStateMins: 2880,
				IsRework:                    false,
			},
		},
	}

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues/42/timeline", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertContentType(t, resp, "application/json")

	var result domain.IssueTimelineResponse
	ParseResponse(t, resp, &result)

	if result.Issue.IssueIID != 42 {
		t.Errorf("Expected issue IID 42, got %d", result.Issue.IssueIID)
	}

	// Verify unified identity fields are present
	if result.Issue.IssueID != 1 {
		t.Errorf("Expected issue_id 1, got %d", result.Issue.IssueID)
	}
	if result.Issue.GitlabIssueID != 99123 {
		t.Errorf("Expected gitlab_issue_id 99123, got %d", result.Issue.GitlabIssueID)
	}
	if result.Issue.ProjectPath != "group/project" {
		t.Errorf("Expected project_path 'group/project', got %s", result.Issue.ProjectPath)
	}

	if len(result.Timeline) != 3 {
		t.Errorf("Expected 3 timeline events, got %d", len(result.Timeline))
	}

	if result.Timeline[0].FromState != "opened" {
		t.Errorf("Expected first event from_state to be 'opened', got %s", result.Timeline[0].FromState)
	}
}

func TestIssues_Timeline_NotFound(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.TimelineErr = errors.New("issue not found")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues/999/timeline", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusNotFound)
}

func TestIssues_Timeline_InvalidIssueID(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues/invalid/timeline", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusBadRequest)
}

func TestIssues_Timeline_InvalidPath(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues/42/invalid", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusNotFound)
}

func TestIssues_Timeline_Unauthorized(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	resp := MakeRequest(t, ts, http.MethodGet, "/api/v1/issues/42/timeline", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusUnauthorized)
}

func TestIssues_Timeline_ServiceError(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	ts.Builder.IssuesService.TimelineErr = errors.New("database connection failed")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues/42/timeline", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusInternalServerError)
}

func TestIssues_Timeline_InvalidIssueIDFromService(t *testing.T) {
	ts := SetupTestServer(t)
	defer TeardownTestServer(ts)

	// Use a valid positive ID in URL, but service returns invalid issue ID error
	ts.Builder.IssuesService.TimelineErr = errors.New("invalid issue ID: not found in database")

	resp := MakeAuthenticatedRequest(t, ts, http.MethodGet, "/api/v1/issues/42/timeline", nil)
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
}
