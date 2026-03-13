package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"gitlab-engineering-metrics-api/internal/auth"
	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/http/handlers"
	"gitlab-engineering-metrics-api/internal/http/middleware"
	"gitlab-engineering-metrics-api/internal/observability"
)

// MockCatalogService is a mock implementation of the catalog service
type MockCatalogService struct {
	Projects []domain.Project
	Groups   []domain.Group
	Users    []domain.User
	Err      error
}

func (m *MockCatalogService) ListProjects(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Projects, nil
}

func (m *MockCatalogService) ListGroups(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Groups, nil
}

func (m *MockCatalogService) ListUsers(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Users, nil
}

// MockMetricsService is a mock implementation of the metrics service
type MockMetricsService struct {
	DeliveryMetrics *domain.DeliveryMetricsResponse
	QualityMetrics  *domain.QualityMetricsResponse
	WipMetrics      *domain.WipMetricsResponse
	Err             error
}

func (m *MockMetricsService) GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.DeliveryMetrics, nil
}

func (m *MockMetricsService) GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.QualityMetrics, nil
}

func (m *MockMetricsService) GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.WipMetrics, nil
}

// MockIssuesService is a mock implementation of the issues service
type MockIssuesService struct {
	IssuesList  *domain.IssuesListResponse
	Timeline    *domain.IssueTimelineResponse
	TimelineErr error
	ListErr     error
}

func (m *MockIssuesService) ListIssues(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	return m.IssuesList, nil
}

func (m *MockIssuesService) GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error) {
	if m.TimelineErr != nil {
		return nil, m.TimelineErr
	}
	return m.Timeline, nil
}

// TestAppBuilder builds a test application with mock services
type TestAppBuilder struct {
	CatalogService *MockCatalogService
	MetricsService *MockMetricsService
	IssuesService  *MockIssuesService
	Validator      *auth.Validator
	Metrics        *observability.MetricsCollector
	Logger         *slog.Logger
}

// NewTestAppBuilder creates a new test app builder with defaults
func NewTestAppBuilder() *TestAppBuilder {
	return &TestAppBuilder{
		CatalogService: &MockCatalogService{},
		MetricsService: &MockMetricsService{},
		IssuesService:  &MockIssuesService{},
		Validator:      auth.NewValidator(map[string]string{TestClientID: TestClientSecret}),
		Metrics:        observability.NewMetricsCollector(),
		Logger:         slog.New(slog.NewJSONHandler(os.NewFile(0, os.DevNull), nil)),
	}
}

// Build creates an HTTP handler with all routes configured
func (b *TestAppBuilder) Build() http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", b.healthHandler)
	mux.HandleFunc("/metrics", b.metricsHandler)

	// Protected API routes with auth middleware
	authMiddleware := middleware.Auth(b.Validator)

	// Catalog routes
	projectsHandler := handlers.NewProjectsHandler(b.CatalogService)
	groupsHandler := handlers.NewGroupsHandler(b.CatalogService)
	usersHandler := handlers.NewUsersHandler(b.CatalogService)

	mux.Handle("/api/v1/projects", authMiddleware(http.HandlerFunc(projectsHandler.List)))
	mux.Handle("/api/v1/groups", authMiddleware(http.HandlerFunc(groupsHandler.List)))
	mux.Handle("/api/v1/users", authMiddleware(http.HandlerFunc(usersHandler.List)))

	// Metrics routes
	deliveryHandler := handlers.NewDeliveryHandler(b.MetricsService)
	qualityHandler := handlers.NewQualityHandler(b.MetricsService)
	wipHandler := handlers.NewWipHandler(b.MetricsService)

	mux.Handle("/api/v1/metrics/delivery", authMiddleware(http.HandlerFunc(deliveryHandler.Get)))
	mux.Handle("/api/v1/metrics/quality", authMiddleware(http.HandlerFunc(qualityHandler.Get)))
	mux.Handle("/api/v1/metrics/wip", authMiddleware(http.HandlerFunc(wipHandler.Get)))

	// Issues routes
	issuesHandler := handlers.NewIssuesHandler(b.IssuesService)
	timelineHandler := handlers.NewTimelineHandler(b.IssuesService)

	mux.Handle("/api/v1/issues", authMiddleware(http.HandlerFunc(issuesHandler.List)))
	mux.Handle("/api/v1/issues/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := splitPath(path)
		if len(parts) == 5 && parts[4] == "timeline" {
			timelineHandler.Get(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
		}
	})))

	// Apply middleware chain
	handler := middleware.Recovery(b.Logger)(mux)
	handler = middleware.Metrics(b.Metrics)(handler)
	handler = middleware.Logging(b.Logger)(handler)
	handler = middleware.RequestID(handler)

	return handler
}

func (b *TestAppBuilder) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (b *TestAppBuilder) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	snapshot := b.Metrics.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(snapshot)
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

// SetupTestServerWithBuilder creates a test server with a custom builder
func SetupTestServerWithBuilder(t TestingT, builder *TestAppBuilder) *httptest.Server {
	return httptest.NewServer(builder.Build())
}

// TestingT is an interface for testing.T to allow flexibility
type TestingT interface {
	Helper()
	Fatalf(format string, args ...interface{})
}
