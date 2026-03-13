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

var deliveryLogger = observability.GetLogger().With(slog.String("handler", "delivery"))

// MetricsService defines the interface for metrics operations
type MetricsService interface {
	GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error)
	GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error)
	GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error)
}

// DeliveryHandler handles delivery metrics HTTP requests
type DeliveryHandler struct {
	service MetricsService
}

// NewDeliveryHandler creates a new delivery handler
func NewDeliveryHandler(service MetricsService) *DeliveryHandler {
	return &DeliveryHandler{service: service}
}

// Get handles GET /api/v1/metrics/delivery
func (h *DeliveryHandler) Get(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())

	deliveryLogger.Debug("incoming request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("query", r.URL.RawQuery),
		slog.String("request_id", requestID),
	)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		deliveryLogger.Warn("method not allowed",
			slog.String("method", r.Method),
			slog.String("request_id", requestID),
		)
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
			deliveryLogger.Debug("parsed project_id",
				slog.Int("project_id", projectID),
				slog.String("request_id", requestID),
			)
		} else {
			deliveryLogger.Warn("invalid project_id",
				slog.String("value", projectIDStr),
				slog.String("error", err.Error()),
				slog.String("request_id", requestID),
			)
		}
	}

	deliveryLogger.Debug("parsed filter",
		slog.String("start_date", filter.StartDate),
		slog.String("end_date", filter.EndDate),
		slog.String("group_path", filter.GroupPath),
		slog.String("assignee", filter.Assignee),
		slog.Int("project_id", filter.ProjectID),
		slog.String("request_id", requestID),
	)

	// Call service
	metrics, err := h.service.GetDeliveryMetrics(r.Context(), filter)
	if err != nil {
		// Check for validation errors
		if err.Error() == "both start_date and end_date are required when filtering by date" ||
			err.Error() == "invalid start_date format, expected YYYY-MM-DD" ||
			err.Error() == "invalid end_date format, expected YYYY-MM-DD" ||
			err.Error() == "end_date must be after start_date" ||
			err.Error() == "date range cannot exceed 90 days" {
			deliveryLogger.Warn("validation error",
				slog.String("error", err.Error()),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, err.Error())
			return
		}
		deliveryLogger.Error("failed to get delivery metrics",
			slog.String("error", err.Error()),
			slog.Any("filter", filter),
			slog.String("request_id", requestID),
		)
		responses.InternalServerError(w, requestID)
		return
	}

	deliveryLogger.Info("delivery metrics retrieved",
		slog.Int("assignee_breakdown_count", len(metrics.BreakdownByAssignee)),
		slog.Int("throughput_total", metrics.Throughput.TotalIssuesDone),
		slog.String("period_start", metrics.Period.StartDate),
		slog.String("period_end", metrics.Period.EndDate),
		slog.String("request_id", requestID),
	)

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		deliveryLogger.Error("failed to encode response",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
		return
	}
}
