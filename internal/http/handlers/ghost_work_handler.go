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

// GhostWorkService defines the interface for ghost work operations
type GhostWorkService interface {
	GetGhostWorkMetrics(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error)
}

// GhostWorkHandler handles ghost work metrics HTTP requests
type GhostWorkHandler struct {
	service GhostWorkService
}

// NewGhostWorkHandler creates a new ghost work handler
func NewGhostWorkHandler(service GhostWorkService) *GhostWorkHandler {
	return &GhostWorkHandler{service: service}
}

// Get handles GET /api/v1/metrics/ghost-work
func (h *GhostWorkHandler) Get(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	filter := domain.GhostWorkFilter{
		MetricsFilter: domain.MetricsFilter{
			StartDate: r.URL.Query().Get("start_date"),
			EndDate:   r.URL.Query().Get("end_date"),
			GroupPath: r.URL.Query().Get("group_path"),
			Assignee:  r.URL.Query().Get("assignee"),
		},
	}

	// Parse project_id if provided
	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		if projectID, err := strconv.Atoi(projectIDStr); err == nil {
			filter.ProjectID = projectID
		}
	}

	// Parse issue_id if provided
	if issueIDStr := r.URL.Query().Get("issue_id"); issueIDStr != "" {
		if issueID, err := strconv.Atoi(issueIDStr); err == nil {
			filter.IssueID = issueID
		}
	}

	// Parse gitlab_issue_id if provided
	if gitlabIssueIDStr := r.URL.Query().Get("gitlab_issue_id"); gitlabIssueIDStr != "" {
		if gitlabIssueID, err := strconv.Atoi(gitlabIssueIDStr); err == nil {
			filter.GitlabIssueID = gitlabIssueID
		}
	}

	// Parse issue_iid if provided
	if issueIIDStr := r.URL.Query().Get("issue_iid"); issueIIDStr != "" {
		if issueIID, err := strconv.Atoi(issueIIDStr); err == nil {
			filter.IssueIID = issueIID
		}
	}

	// Parse pagination
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}
	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			filter.PageSize = pageSize
		}
	}

	// Call service
	metrics, err := h.service.GetGhostWorkMetrics(r.Context(), filter)
	if err != nil {
		// Check for validation errors
		if err.Error() == "both start_date and end_date are required when filtering by date" ||
			err.Error() == "invalid start_date format, expected YYYY-MM-DD" ||
			err.Error() == "invalid end_date format, expected YYYY-MM-DD" ||
			err.Error() == "end_date must be after start_date" ||
			err.Error() == "date range cannot exceed 90 days" {
			responses.BadRequest(w, requestID, err.Error())
			return
		}
		responses.InternalServerError(w, requestID)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		return
	}
}
