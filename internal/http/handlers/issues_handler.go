package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/http/responses"
	"gitlab-engineering-metrics-api/internal/observability"
)

var issuesLogger = observability.GetLogger().With(slog.String("handler", "issues"))

// IssuesService defines the interface for issue operations
type IssuesService interface {
	ListIssues(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error)
	GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error)
}

// IssuesHandler handles issue-related HTTP requests
type IssuesHandler struct {
	service IssuesService
}

// NewIssuesHandler creates a new issues handler
func NewIssuesHandler(service IssuesService) *IssuesHandler {
	return &IssuesHandler{service: service}
}

// List handles GET /issues
func (h *IssuesHandler) List(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())

	issuesLogger.Debug("incoming request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("query", r.URL.RawQuery),
		slog.String("request_id", requestID),
	)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		issuesLogger.Warn("method not allowed",
			slog.String("method", r.Method),
			slog.String("request_id", requestID),
		)
		return
	}

	// Parse query parameters
	filter := domain.IssuesFilter{
		GroupPath: r.URL.Query().Get("group_path"),
		Assignee:  r.URL.Query().Get("assignee"),
		State:     r.URL.Query().Get("state"),
	}

	// Parse metric_flag if provided
	if metricFlag := r.URL.Query().Get("metric_flag"); metricFlag != "" {
		filter.MetricFlag = metricFlag
	}

	// Parse project_id if provided
	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil || projectID < 0 {
			issuesLogger.Warn("invalid project_id",
				slog.String("value", projectIDStr),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, "project_id must be a positive integer")
			return
		}
		filter.ProjectID = projectID
	}

	// Parse page parameter
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 0 {
			issuesLogger.Warn("invalid page",
				slog.String("value", pageStr),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, "page must be a non-negative integer")
			return
		}
		filter.Page = page
	}

	// Parse page_size parameter
	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 0 {
			issuesLogger.Warn("invalid page_size",
				slog.String("value", pageSizeStr),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, "page_size must be a non-negative integer")
			return
		}
		filter.PageSize = pageSize
	}

	// Validate metric_flag if provided
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
			issuesLogger.Warn("invalid metric_flag",
				slog.String("value", filter.MetricFlag),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, "metric_flag must be one of: bypass, rework, blocked")
			return
		}
	}

	// Parse issue_id if provided
	if issueIDStr := r.URL.Query().Get("issue_id"); issueIDStr != "" {
		issueID, err := strconv.Atoi(issueIDStr)
		if err != nil || issueID <= 0 {
			issuesLogger.Warn("invalid issue_id",
				slog.String("value", issueIDStr),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, "issue_id must be a positive integer")
			return
		}
		filter.IssueID = issueID
	}

	// Parse gitlab_issue_id if provided
	if gitlabIssueIDStr := r.URL.Query().Get("gitlab_issue_id"); gitlabIssueIDStr != "" {
		gitlabIssueID, err := strconv.Atoi(gitlabIssueIDStr)
		if err != nil || gitlabIssueID <= 0 {
			issuesLogger.Warn("invalid gitlab_issue_id",
				slog.String("value", gitlabIssueIDStr),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, "gitlab_issue_id must be a positive integer")
			return
		}
		filter.GitlabIssueID = gitlabIssueID
	}

	// Parse issue_iid if provided
	if issueIIDStr := r.URL.Query().Get("issue_iid"); issueIIDStr != "" {
		issueIID, err := strconv.Atoi(issueIIDStr)
		if err != nil || issueIID <= 0 {
			issuesLogger.Warn("invalid issue_iid",
				slog.String("value", issueIIDStr),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, "issue_iid must be a positive integer")
			return
		}
		filter.IssueIID = issueIID
	}

	issuesLogger.Debug("parsed filter",
		slog.Any("filter", filter),
		slog.String("request_id", requestID),
	)

	// Call service
	result, err := h.service.ListIssues(r.Context(), filter)
	if err != nil {
		// Check for validation errors
		errMsg := err.Error()
		if isValidationError(errMsg) {
			issuesLogger.Warn("validation error",
				slog.String("error", errMsg),
				slog.String("request_id", requestID),
			)
			responses.UnprocessableEntity(w, requestID, errMsg)
			return
		}
		issuesLogger.Error("failed to list issues",
			slog.String("error", err.Error()),
			slog.Any("filter", filter),
			slog.String("request_id", requestID),
		)
		responses.InternalServerError(w, requestID)
		return
	}

	issuesLogger.Info("issues listed successfully",
		slog.Int("count", len(result.Items)),
		slog.Int("total", result.Total),
		slog.Int("page", result.Page),
		slog.String("request_id", requestID),
	)

	// Debug: log first item metrics
	if len(result.Items) > 0 {
		first := result.Items[0]
		issuesLogger.Debug("first item metrics",
			slog.Int("issue_id", first.IssueID),
			slog.Bool("has_bypass", first.HasBypass),
			slog.Bool("has_rework", first.HasRework),
			slog.Bool("was_blocked", first.WasBlocked),
			slog.String("request_id", requestID),
		)
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		issuesLogger.Error("failed to encode response",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
		return
	}
}

// isValidationError checks if an error message is a validation error
func isValidationError(msg string) bool {
	validationPatterns := []string{
		"page must be",
		"page_size must be",
		"page_size cannot",
		"project_id must be",
		"issue_id must be",
		"gitlab_issue_id must be",
		"issue_iid must be",
	}
	for _, pattern := range validationPatterns {
		if containsString(msg, pattern) {
			return true
		}
	}
	return false
}

// containsString checks if a string contains a substring (case-sensitive)
func containsString(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
