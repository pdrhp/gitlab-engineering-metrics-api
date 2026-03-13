package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/http/responses"
)

// QualityHandler handles quality metrics HTTP requests
type QualityHandler struct {
	service MetricsService
}

// NewQualityHandler creates a new quality handler
func NewQualityHandler(service MetricsService) *QualityHandler {
	return &QualityHandler{service: service}
}

// Get handles GET /api/v1/metrics/quality
func (h *QualityHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	filter := domain.MetricsFilter{
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
		GroupPath: r.URL.Query().Get("group_path"),
		Assignee:  r.URL.Query().Get("assignee"),
	}

	// Parse project_id if provided
	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		if projectID, err := strconv.Atoi(projectIDStr); err == nil {
			filter.ProjectID = projectID
		}
	}

	// Call service
	metrics, err := h.service.GetQualityMetrics(r.Context(), filter)
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
