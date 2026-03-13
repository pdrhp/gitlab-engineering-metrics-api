package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/http/responses"
)

// TimelineService defines the interface for timeline operations
type TimelineService interface {
	GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error)
}

// TimelineHandler handles timeline-related HTTP requests
type TimelineHandler struct {
	service TimelineService
}

// NewTimelineHandler creates a new timeline handler
func NewTimelineHandler(service TimelineService) *TimelineHandler {
	return &TimelineHandler{service: service}
}

// Get handles GET /issues/:id/timeline
func (h *TimelineHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := middleware.GetRequestID(r.Context())

	// Parse issue ID from URL path
	// Expected path: /api/v1/issues/:id/timeline
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Path should be: ["api", "v1", "issues", ":id", "timeline"]
	if len(parts) < 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "issues" || parts[4] != "timeline" {
		responses.BadRequest(w, requestID, "Invalid URL path. Expected: /api/v1/issues/:id/timeline")
		return
	}

	issueIDStr := parts[3]
	issueID, err := strconv.Atoi(issueIDStr)
	if err != nil || issueID <= 0 {
		responses.BadRequest(w, requestID, "issue_id must be a positive integer")
		return
	}

	// Call service
	timeline, err := h.service.GetTimeline(r.Context(), issueID)
	if err != nil {
		errMsg := err.Error()
		if containsSubstring(errMsg, "issue not found") {
			responses.NotFound(w, requestID, "Issue")
			return
		}
		if containsSubstring(errMsg, "invalid issue ID") {
			responses.UnprocessableEntity(w, requestID, errMsg)
			return
		}
		responses.InternalServerError(w, requestID)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(timeline); err != nil {
		// Log error but can't write to response anymore
		return
	}
}

// containsSubstring checks if a string contains a substring
func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}
