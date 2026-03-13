package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/http/responses"
)

// WipHandler handles WIP metrics HTTP requests
type WipHandler struct {
	service MetricsService
}

// NewWipHandler creates a new WIP handler
func NewWipHandler(service MetricsService) *WipHandler {
	return &WipHandler{service: service}
}

// Get handles GET /api/v1/metrics/wip
func (h *WipHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	filter := domain.MetricsFilter{
		GroupPath: r.URL.Query().Get("group_path"),
		Assignee:  r.URL.Query().Get("assignee"),
	}

	// Parse project_id if provided
	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		if projectID, err := strconv.Atoi(projectIDStr); err == nil {
			filter.ProjectID = projectID
		}
	}

	// Note: WIP metrics don't use date range - they show current state
	// Date parameters are ignored for WIP endpoint

	// Call service
	metrics, err := h.service.GetWipMetrics(r.Context(), filter)
	if err != nil {
		requestID := middleware.GetRequestID(r.Context())
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
