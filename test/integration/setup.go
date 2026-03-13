package integration

import (
	"net/http/httptest"
	"testing"

	"gitlab-engineering-metrics-api/internal/auth"
	"gitlab-engineering-metrics-api/internal/observability"
)

const (
	TestClientID     = "test-client"
	TestClientSecret = "test-secret"
)

// TestServer wraps a test HTTP server with its dependencies
type TestServer struct {
	Server    *httptest.Server
	Validator *auth.Validator
	Metrics   *observability.MetricsCollector
	Builder   *TestAppBuilder
}

// SetupTestServer creates a test HTTP server with mock dependencies
func SetupTestServer(t *testing.T) *TestServer {
	t.Helper()

	builder := NewTestAppBuilder()
	server := httptest.NewServer(builder.Build())

	return &TestServer{
		Server:    server,
		Validator: builder.Validator,
		Metrics:   builder.Metrics,
		Builder:   builder,
	}
}

// TeardownTestServer cleans up the test server and resources
func TeardownTestServer(ts *TestServer) {
	if ts.Server != nil {
		ts.Server.Close()
	}
}
