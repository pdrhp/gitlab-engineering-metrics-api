package observability

import (
	"testing"
	"time"
)

func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()
	if mc == nil {
		t.Fatal("NewMetricsCollector() returned nil")
	}

	if mc.requests == nil {
		t.Error("requests map not initialized")
	}
}

func TestRecordRequest(t *testing.T) {
	mc := NewMetricsCollector()

	// Record some requests
	mc.RecordRequest("GET", "/api/v1/users", 200, 100*time.Millisecond)
	mc.RecordRequest("GET", "/api/v1/users", 200, 150*time.Millisecond)
	mc.RecordRequest("GET", "/api/v1/users", 404, 50*time.Millisecond)
	mc.RecordRequest("POST", "/api/v1/users", 201, 200*time.Millisecond)
	mc.RecordRequest("POST", "/api/v1/users", 500, 300*time.Millisecond)

	// Check GET /api/v1/users metrics
	snapshot, ok := mc.GetEndpointMetrics("GET", "/api/v1/users")
	if !ok {
		t.Fatal("GET /api/v1/users metrics not found")
	}

	if snapshot.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", snapshot.TotalRequests)
	}

	if snapshot.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", snapshot.ErrorCount)
	}

	if snapshot.StatusCounts[200] != 2 {
		t.Errorf("StatusCounts[200] = %d, want 2", snapshot.StatusCounts[200])
	}

	if snapshot.StatusCounts[404] != 1 {
		t.Errorf("StatusCounts[404] = %d, want 1", snapshot.StatusCounts[404])
	}

	// Check POST /api/v1/users metrics
	snapshot, ok = mc.GetEndpointMetrics("POST", "/api/v1/users")
	if !ok {
		t.Fatal("POST /api/v1/users metrics not found")
	}

	if snapshot.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2", snapshot.TotalRequests)
	}

	if snapshot.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", snapshot.ErrorCount)
	}

	if snapshot.StatusCounts[201] != 1 {
		t.Errorf("StatusCounts[201] = %d, want 1", snapshot.StatusCounts[201])
	}

	if snapshot.StatusCounts[500] != 1 {
		t.Errorf("StatusCounts[500] = %d, want 1", snapshot.StatusCounts[500])
	}
}

func TestGetMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	// Record requests for different endpoints
	mc.RecordRequest("GET", "/api/v1/users", 200, 100*time.Millisecond)
	mc.RecordRequest("GET", "/api/v1/groups", 200, 200*time.Millisecond)
	mc.RecordRequest("POST", "/api/v1/users", 500, 300*time.Millisecond)

	snapshot := mc.GetMetrics()

	// Check totals
	if snapshot.Total != 3 {
		t.Errorf("Total = %d, want 3", snapshot.Total)
	}

	if snapshot.Errors != 1 {
		t.Errorf("Errors = %d, want 1", snapshot.Errors)
	}

	// Check that we have 3 endpoints
	if len(snapshot.Endpoints) != 3 {
		t.Errorf("Endpoints count = %d, want 3", len(snapshot.Endpoints))
	}

	// Verify each endpoint has required fields
	for _, ep := range snapshot.Endpoints {
		if ep.Method == "" {
			t.Error("Endpoint Method is empty")
		}
		if ep.Path == "" {
			t.Error("Endpoint Path is empty")
		}
		if ep.StatusCounts == nil {
			t.Error("Endpoint StatusCounts is nil")
		}
	}
}

func TestGetEndpointMetrics_NotFound(t *testing.T) {
	mc := NewMetricsCollector()

	_, ok := mc.GetEndpointMetrics("DELETE", "/api/v1/nonexistent")
	if ok {
		t.Error("GetEndpointMetrics() should return false for non-existent endpoint")
	}
}

func TestReset(t *testing.T) {
	mc := NewMetricsCollector()

	// Record some requests
	mc.RecordRequest("GET", "/api/v1/users", 200, 100*time.Millisecond)

	// Verify metrics exist
	snapshot := mc.GetMetrics()
	if snapshot.Total != 1 {
		t.Errorf("Total = %d, want 1 before reset", snapshot.Total)
	}

	// Reset
	mc.Reset()

	// Verify metrics are cleared
	snapshot = mc.GetMetrics()
	if snapshot.Total != 0 {
		t.Errorf("Total = %d, want 0 after reset", snapshot.Total)
	}

	if len(snapshot.Endpoints) != 0 {
		t.Errorf("Endpoints count = %d, want 0 after reset", len(snapshot.Endpoints))
	}
}

func TestCalculateP95(t *testing.T) {
	tests := []struct {
		name      string
		latencies []time.Duration
		want      time.Duration
	}{
		{
			name:      "empty slice",
			latencies: []time.Duration{},
			want:      0,
		},
		{
			name:      "single element",
			latencies: []time.Duration{100 * time.Millisecond},
			want:      100 * time.Millisecond,
		},
		{
			name:      "two elements",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			want:      200 * time.Millisecond,
		},
		{
			name:      "twenty elements",
			latencies: generateLatencies(20),
			want:      19 * time.Millisecond, // 95th percentile of 0-19ms is 19ms
		},
		{
			name:      "one hundred elements",
			latencies: generateLatencies(100),
			want:      95 * time.Millisecond, // 95th percentile of 0-99ms is 95ms
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateP95(tt.latencies)
			if got != tt.want {
				t.Errorf("calculateP95() = %v, want %v", got, tt.want)
			}
		})
	}
}

func generateLatencies(n int) []time.Duration {
	latencies := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		latencies[i] = time.Duration(i) * time.Millisecond
	}
	return latencies
}

func TestConcurrency(t *testing.T) {
	mc := NewMetricsCollector()

	// Run concurrent goroutines to record requests
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 10; j++ {
				mc.RecordRequest("GET", "/api/v1/users", 200, time.Duration(id+j)*time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Verify totals
	snapshot := mc.GetMetrics()
	if snapshot.Total != 1000 {
		t.Errorf("Total = %d, want 1000", snapshot.Total)
	}

	// Verify GET /api/v1/users endpoint
	epSnapshot, ok := mc.GetEndpointMetrics("GET", "/api/v1/users")
	if !ok {
		t.Fatal("GET /api/v1/users metrics not found")
	}

	if epSnapshot.TotalRequests != 1000 {
		t.Errorf("TotalRequests = %d, want 1000", epSnapshot.TotalRequests)
	}
}

func TestLatencyCalculations(t *testing.T) {
	mc := NewMetricsCollector()

	// Record requests with specific latencies
	mc.RecordRequest("GET", "/api/v1/test", 200, 100*time.Millisecond)
	mc.RecordRequest("GET", "/api/v1/test", 200, 200*time.Millisecond)
	mc.RecordRequest("GET", "/api/v1/test", 200, 300*time.Millisecond)

	snapshot, ok := mc.GetEndpointMetrics("GET", "/api/v1/test")
	if !ok {
		t.Fatal("GET /api/v1/test metrics not found")
	}

	// Average should be 200ms
	if snapshot.AvgLatency != "200ms" {
		t.Errorf("AvgLatency = %s, want 200ms", snapshot.AvgLatency)
	}

	// P95 should be 300ms (95th percentile of [100, 200, 300])
	if snapshot.P95Latency != "300ms" {
		t.Errorf("P95Latency = %s, want 300ms", snapshot.P95Latency)
	}
}
