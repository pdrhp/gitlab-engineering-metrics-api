package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/http/responses"
	"gitlab-engineering-metrics-api/internal/observability"
)

var projectsLogger = observability.GetLogger().With(slog.String("handler", "projects"))

// CatalogService defines the interface for catalog operations
type CatalogService interface {
	ListProjects(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error)
	ListGroups(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error)
	ListUsers(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error)
}

// ProjectsHandler handles project-related HTTP requests
type ProjectsHandler struct {
	service CatalogService
}

// NewProjectsHandler creates a new projects handler
func NewProjectsHandler(service CatalogService) *ProjectsHandler {
	return &ProjectsHandler{service: service}
}

// List handles GET /projects
func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())

	projectsLogger.Debug("incoming request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("request_id", requestID),
	)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		projectsLogger.Warn("method not allowed",
			slog.String("method", r.Method),
			slog.String("request_id", requestID),
		)
		return
	}

	// Parse query parameters
	search := r.URL.Query().Get("search")

	// Validate search term
	if search != "" && len(search) < 3 {
		projectsLogger.Warn("validation error",
			slog.String("error", "search term must be at least 3 characters"),
			slog.String("request_id", requestID),
		)
		responses.BadRequest(w, requestID, "search term must be at least 3 characters")
		return
	}

	filter := domain.CatalogFilter{
		Search:    search,
		GroupPath: r.URL.Query().Get("group_path"),
	}

	projectsLogger.Debug("parsed filter",
		slog.Any("filter", filter),
		slog.String("request_id", requestID),
	)

	// Call service
	projects, err := h.service.ListProjects(r.Context(), filter)
	if err != nil {
		// Check if it's a validation error
		if err.Error() == "search term must be at least 3 characters" {
			projectsLogger.Warn("validation error",
				slog.String("error", err.Error()),
				slog.String("request_id", requestID),
			)
			responses.BadRequest(w, requestID, err.Error())
			return
		}
		projectsLogger.Error("failed to list projects",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
		responses.InternalServerError(w, requestID)
		return
	}

	projectsLogger.Info("projects listed successfully",
		slog.Int("count", len(projects)),
		slog.String("request_id", requestID),
	)

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(projects); err != nil {
		projectsLogger.Error("failed to encode response",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
		return
	}
}
