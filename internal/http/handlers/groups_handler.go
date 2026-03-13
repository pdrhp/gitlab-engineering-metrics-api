package handlers

import (
	"encoding/json"
	"net/http"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/http/responses"
)

// GroupsHandler handles group-related HTTP requests
type GroupsHandler struct {
	service CatalogService
}

// NewGroupsHandler creates a new groups handler
func NewGroupsHandler(service CatalogService) *GroupsHandler {
	return &GroupsHandler{service: service}
}

// List handles GET /groups
func (h *GroupsHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	filter := domain.CatalogFilter{
		Search:    r.URL.Query().Get("search"),
		GroupPath: r.URL.Query().Get("group_path"),
	}

	// Call service
	groups, err := h.service.ListGroups(r.Context(), filter)
	if err != nil {
		requestID := middleware.GetRequestID(r.Context())
		if err.Error() == "search term must be at least 3 characters" {
			responses.BadRequest(w, requestID, err.Error())
			return
		}
		responses.InternalServerError(w, requestID)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(groups); err != nil {
		return
	}
}
