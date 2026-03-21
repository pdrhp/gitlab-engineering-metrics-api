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

var deliveryTrendLogger = observability.GetLogger().With(slog.String("handler", "delivery_trend"))

type DeliveryTrendService interface {
	GetDeliveryTrendMetrics(ctx context.Context, filter domain.DeliveryTrendFilter) (*domain.DeliveryTrendResponse, error)
}

type DeliveryTrendHandler struct {
	service DeliveryTrendService
}

func NewDeliveryTrendHandler(service DeliveryTrendService) *DeliveryTrendHandler {
	return &DeliveryTrendHandler{service: service}
}

func (h *DeliveryTrendHandler) Get(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())

	deliveryTrendLogger.Debug("incoming request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("query", r.URL.RawQuery),
		slog.String("request_id", requestID),
	)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		deliveryTrendLogger.Warn("method not allowed",
			slog.String("method", r.Method),
			slog.String("request_id", requestID),
		)
		return
	}

	// Parse query parameters
	filter := domain.DeliveryTrendFilter{
		MetricsFilter: domain.MetricsFilter{
			StartDate: r.URL.Query().Get("start_date"),
			EndDate:   r.URL.Query().Get("end_date"),
			GroupPath: r.URL.Query().Get("group_path"),
			Assignee:  r.URL.Query().Get("assignee"),
		},
		Bucket:              r.URL.Query().Get("bucket"),
		Timezone:            r.URL.Query().Get("timezone"),
		IncludeEmptyBuckets: true,
	}

	// Parse include_empty_buckets
	if v := r.URL.Query().Get("include_empty_buckets"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			deliveryTrendLogger.Warn("invalid include_empty_buckets",
				slog.String("value", v),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, "include_empty_buckets must be a boolean")
			return
		}
		filter.IncludeEmptyBuckets = b
	}

	// Parse project_id if provided
	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil || projectID <= 0 {
			deliveryTrendLogger.Warn("invalid project_id",
				slog.String("value", projectIDStr),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, "project_id must be a positive integer")
			return
		}
		filter.ProjectID = projectID
	}

	deliveryTrendLogger.Debug("parsed filter",
		slog.String("start_date", filter.StartDate),
		slog.String("end_date", filter.EndDate),
		slog.String("group_path", filter.GroupPath),
		slog.String("assignee", filter.Assignee),
		slog.Int("project_id", filter.ProjectID),
		slog.String("bucket", filter.Bucket),
		slog.String("timezone", filter.Timezone),
		slog.Bool("include_empty_buckets", filter.IncludeEmptyBuckets),
		slog.String("request_id", requestID),
	)

	// Call service
	metrics, err := h.service.GetDeliveryTrendMetrics(r.Context(), filter)
	if err != nil {
		msg := err.Error()
		// Check for validation errors
		if containsString(msg, "invalid") || containsString(msg, "must") || containsString(msg, "exceed") || containsString(msg, "required") {
			deliveryTrendLogger.Warn("validation error",
				slog.String("error", msg),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, msg)
			return
		}
		// Check for semantic validation errors (422)
		if containsString(msg, "does not belong to") {
			deliveryTrendLogger.Warn("semantic validation error",
				slog.String("error", msg),
				slog.String("request_id", requestID),
			)
			responses.UnprocessableEntity(w, requestID, msg)
			return
		}
		deliveryTrendLogger.Error("failed to get delivery trend metrics",
			slog.String("error", err.Error()),
			slog.Any("filter", filter),
			slog.String("request_id", requestID),
		)
		responses.InternalServerError(w, requestID)
		return
	}

	deliveryTrendLogger.Info("delivery trend metrics retrieved",
		slog.Int("items_count", len(metrics.Items)),
		slog.String("bucket", metrics.Bucket),
		slog.String("timezone", metrics.Timezone),
		slog.String("request_id", requestID),
	)

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		deliveryTrendLogger.Error("failed to encode response",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
		return
	}
}
