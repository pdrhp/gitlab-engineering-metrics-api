package handlers

import (
	"encoding/json"
	"net/http"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/http/responses"
)

// UsersHandler handles user-related HTTP requests
type UsersHandler struct {
	service CatalogService
}

// NewUsersHandler creates a new users handler
func NewUsersHandler(service CatalogService) *UsersHandler {
	return &UsersHandler{service: service}
}

// List handles GET /users
func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	search := r.URL.Query().Get("search")

	// Validate search term
	if search != "" && len(search) < 3 {
		requestID := middleware.GetRequestID(r.Context())
		responses.BadRequest(w, requestID, "search term must be at least 3 characters")
		return
	}

	filter := domain.CatalogFilter{
		Search:    search,
		GroupPath: r.URL.Query().Get("group_path"),
	}

	// Call service
	users, err := h.service.ListUsers(r.Context(), filter)
	if err != nil {
		requestID := middleware.GetRequestID(r.Context())
		// Check if it's a validation error
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
	if err := json.NewEncoder(w).Encode(users); err != nil {
		return
	}
}
