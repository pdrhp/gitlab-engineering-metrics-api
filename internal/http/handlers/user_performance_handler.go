package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/http/responses"
	"gitlab-engineering-metrics-api/internal/observability"
)

var userPerformanceLogger = observability.GetLogger().With(slog.String("handler", "user_performance"))

// UserPerformanceService defines the minimal contract required by the handler.
type UserPerformanceService interface {
	Get(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.UserPerformanceResponse, error)
}

// UserPerformanceHandler wires HTTP requests to the user performance service.
type UserPerformanceHandler struct {
	service UserPerformanceService
}

// NewUserPerformanceHandler constructs a handler instance.
func NewUserPerformanceHandler(service UserPerformanceService) *UserPerformanceHandler {
	return &UserPerformanceHandler{service: service}
}

// Get currently handles GET /api/v1/users/{username}/performance requests.
func (h *UserPerformanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := extractUsernameFromPath(r.URL.Path)
	if !ok || username == "" {
		userPerformanceLogger.Warn("missing username in path",
			slog.String("path", r.URL.Path),
			slog.String("request_id", requestID),
		)
		responses.BadRequest(w, requestID, "username path parameter is required")
		return
	}

	filter := domain.MetricsFilter{
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
	}

	if projectID := r.URL.Query().Get("project_id"); projectID != "" {
		if id, err := strconv.Atoi(projectID); err == nil {
			filter.ProjectID = id
		}
	}

	resp, err := h.service.Get(r.Context(), username, filter)
	if err != nil {
		userPerformanceLogger.Error("service error",
			slog.String("error", err.Error()),
			slog.String("username", username),
			slog.String("request_id", requestID),
		)
		if strings.Contains(err.Error(), "user not found") {
			responses.NotFound(w, requestID, "User")
		} else if err.Error() == "both start_date and end_date are required when filtering by date" ||
			err.Error() == "invalid start_date format, expected YYYY-MM-DD" ||
			err.Error() == "invalid end_date format, expected YYYY-MM-DD" ||
			err.Error() == "end_date must be after start_date" ||
			err.Error() == "date range cannot exceed 366 days" {
			responses.BadRequest(w, requestID, err.Error())
		} else {
			responses.InternalServerError(w, requestID)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		userPerformanceLogger.Error("failed to encode response",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
	}
}

func extractUsernameFromPath(path string) (string, bool) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", false
	}
	segments := strings.Split(trimmed, "/")
	if len(segments) != 5 {
		return "", false
	}
	if segments[0] != "api" || segments[1] != "v1" || segments[2] != "users" || segments[4] != "performance" {
		return "", false
	}
	if segments[3] == "" {
		return "", false
	}
	return segments[3], true
}
