package app

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"gitlab-engineering-metrics-api/internal/auth"
	"gitlab-engineering-metrics-api/internal/config"
	"gitlab-engineering-metrics-api/internal/http/handlers"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/observability"
	"gitlab-engineering-metrics-api/internal/repositories"
	"gitlab-engineering-metrics-api/internal/services"
)

// usernameRegex validates username format (alphanumeric, hyphens, underscores)
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type App struct {
	db        *sql.DB
	config    *config.Config
	logger    *slog.Logger
	validator *auth.Validator
	metrics   *observability.MetricsCollector
}

func New(db *sql.DB, cfg *config.Config, logger *slog.Logger) *App {
	return &App{
		db:        db,
		config:    cfg,
		logger:    logger,
		validator: auth.NewValidator(cfg.ClientCredentials),
		metrics:   observability.NewMetricsCollector(),
	}
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public routes (no auth required)
	mux.HandleFunc("/health", a.healthHandler)
	mux.HandleFunc("/metrics", a.metricsHandler)

	// Protected API routes
	a.registerCatalogRoutes(mux)
	a.registerMetricsRoutes(mux)
	a.registerIssuesRoutes(mux)

	return a.applyMiddleware(mux)
}

func (a *App) registerCatalogRoutes(mux *http.ServeMux) {
	// Create repositories
	projectsRepo := repositories.NewProjectsRepository(a.db)
	groupsRepo := repositories.NewGroupsRepository(a.db)
	usersRepo := repositories.NewUsersRepository(a.db)

	// Create service
	catalogService := services.NewCatalogService(projectsRepo, groupsRepo, usersRepo)

	// Create handlers
	projectsHandler := handlers.NewProjectsHandler(catalogService)
	groupsHandler := handlers.NewGroupsHandler(catalogService)
	usersHandler := handlers.NewUsersHandler(catalogService)

	// Create user performance handler
	metricsRepo := repositories.NewMetricsRepository(a.db)
	metricsService := services.NewMetricsService(metricsRepo)
	individualPerfRepo := repositories.NewIndividualPerformanceRepository(a.db)
	userPerformanceService := services.NewUserPerformanceService(usersRepo, metricsService, individualPerfRepo)
	userPerformanceHandler := handlers.NewUserPerformanceHandler(userPerformanceService)

	// Register routes with auth middleware
	authMiddleware := middleware.Auth(a.validator)

	mux.Handle("/api/v1/projects", authMiddleware(http.HandlerFunc(projectsHandler.List)))
	mux.Handle("/api/v1/groups", authMiddleware(http.HandlerFunc(groupsHandler.List)))
	mux.Handle("/api/v1/users", authMiddleware(http.HandlerFunc(usersHandler.List)))

	// User performance endpoint - need a custom handler to extract username from path
	mux.Handle("/api/v1/users/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a user performance request
		if isUserPerformancePath(r.URL.Path) {
			userPerformanceHandler.Get(w, r)
		} else {
			// Return 404 for other paths under /api/v1/users/
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
		}
	})))
}

// isUserPerformancePath checks if the path is a user performance endpoint
// Returns true only if path matches /api/v1/users/{username}/performance with valid username
func isUserPerformancePath(path string) bool {
	// Expected format: /api/v1/users/{username}/performance
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "users" || parts[4] != "performance" {
		return false
	}

	// Validate username format (parts[3])
	if !isValidUsername(parts[3]) {
		return false
	}

	return true
}

// isValidUsername validates that username contains only alphanumeric characters, hyphens, and underscores
func isValidUsername(username string) bool {
	return len(username) > 0 && len(username) <= 50 && usernameRegex.MatchString(username)
}

func (a *App) registerMetricsRoutes(mux *http.ServeMux) {
	// Create repository
	metricsRepo := repositories.NewMetricsRepository(a.db)

	// Create service
	metricsService := services.NewMetricsService(metricsRepo)

	// Create handlers
	deliveryHandler := handlers.NewDeliveryHandler(metricsService)
	qualityHandler := handlers.NewQualityHandler(metricsService)
	wipHandler := handlers.NewWipHandler(metricsService)

	// Create ghost work handler
	ghostWorkRepo := repositories.NewGhostWorkRepository(a.db)
	ghostWorkService := services.NewGhostWorkService(ghostWorkRepo)
	ghostWorkHandler := handlers.NewGhostWorkHandler(ghostWorkService)

	// Register routes with auth middleware
	authMiddleware := middleware.Auth(a.validator)

	mux.Handle("/api/v1/metrics/delivery", authMiddleware(http.HandlerFunc(deliveryHandler.Get)))
	mux.Handle("/api/v1/metrics/delivery/trend", authMiddleware(http.HandlerFunc(handlers.NewDeliveryTrendHandler(metricsService).Get)))
	mux.Handle("/api/v1/metrics/quality", authMiddleware(http.HandlerFunc(qualityHandler.Get)))
	mux.Handle("/api/v1/metrics/wip", authMiddleware(http.HandlerFunc(wipHandler.Get)))
	mux.Handle("/api/v1/metrics/ghost-work", authMiddleware(http.HandlerFunc(ghostWorkHandler.Get)))
}

func (a *App) registerIssuesRoutes(mux *http.ServeMux) {
	// Create repositories
	issuesRepo := repositories.NewIssuesRepository(a.db)
	timelineRepo := repositories.NewTimelineRepository(a.db)

	// Create service
	issuesService := services.NewIssuesService(issuesRepo, timelineRepo)

	// Create handlers
	issuesHandler := handlers.NewIssuesHandler(issuesService)
	timelineHandler := handlers.NewTimelineHandler(issuesService)

	// Register routes with auth middleware
	authMiddleware := middleware.Auth(a.validator)

	// Issues list endpoint
	mux.Handle("/api/v1/issues", authMiddleware(http.HandlerFunc(issuesHandler.List)))

	// Timeline endpoint - need a custom handler to extract issue ID from path
	mux.Handle("/api/v1/issues/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a timeline request
		if isTimelinePath(r.URL.Path) {
			timelineHandler.Get(w, r)
		} else {
			// Return 404 for other paths under /api/v1/issues/
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
		}
	})))
}

// isTimelinePath checks if the path is a timeline endpoint
func isTimelinePath(path string) bool {
	// Expected format: /api/v1/issues/:id/timeline
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 5 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "issues" && parts[4] == "timeline"
}

func (a *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	if err := a.db.Ping(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": "database unavailable"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (a *App) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	snapshot := a.metrics.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(snapshot)
}

func (a *App) applyMiddleware(handler http.Handler) http.Handler {
	// Middleware chain order (last applied runs first):
	// 1. Recovery - catches panics
	// 2. Metrics - records request metrics
	// 3. Logging - logs request details
	// 4. RequestID - adds/generates request ID

	handler = middleware.Recovery(a.logger)(handler)
	handler = middleware.Metrics(a.metrics)(handler)
	handler = middleware.Logging(a.logger)(handler)
	handler = middleware.RequestID(handler)

	return handler
}
