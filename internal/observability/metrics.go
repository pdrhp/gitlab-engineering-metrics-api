package observability

import (
	"sync"
	"time"
)

// MetricsCollector tracks HTTP request metrics
type MetricsCollector struct {
	mu       sync.RWMutex
	requests map[string]*EndpointMetrics
}

// EndpointMetrics holds metrics for a specific endpoint
type EndpointMetrics struct {
	Method        string          `json:"method"`
	Path          string          `json:"path"`
	TotalRequests int64           `json:"total_requests"`
	StatusCounts  map[int]int64   `json:"status_counts"`
	ErrorCount    int64           `json:"error_count"`
	Latencies     []time.Duration `json:"latencies"`
	TotalLatency  time.Duration   `json:"total_latency"`
}

// Snapshot represents a point-in-time view of all metrics
type Snapshot struct {
	Endpoints []EndpointSnapshot `json:"endpoints"`
	Total     int64              `json:"total_requests"`
	Errors    int64              `json:"total_errors"`
}

// EndpointSnapshot is a thread-safe copy of endpoint metrics
type EndpointSnapshot struct {
	Method        string        `json:"method"`
	Path          string        `json:"path"`
	TotalRequests int64         `json:"total_requests"`
	StatusCounts  map[int]int64 `json:"status_counts"`
	ErrorCount    int64         `json:"error_count"`
	AvgLatency    string        `json:"avg_latency"`
	P95Latency    string        `json:"p95_latency"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requests: make(map[string]*EndpointMetrics),
	}
}

// RecordRequest records metrics for an HTTP request
func (mc *MetricsCollector) RecordRequest(method, path string, status int, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := method + " " + path

	endpoint, exists := mc.requests[key]
	if !exists {
		endpoint = &EndpointMetrics{
			Method:       method,
			Path:         path,
			StatusCounts: make(map[int]int64),
		}
		mc.requests[key] = endpoint
	}

	endpoint.TotalRequests++
	endpoint.StatusCounts[status]++
	endpoint.TotalLatency += duration
	endpoint.Latencies = append(endpoint.Latencies, duration)

	// Count as error if status >= 400
	if status >= 400 {
		endpoint.ErrorCount++
	}
}

// GetMetrics returns a snapshot of current metrics
func (mc *MetricsCollector) GetMetrics() Snapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	snapshot := Snapshot{
		Endpoints: make([]EndpointSnapshot, 0, len(mc.requests)),
	}

	for _, endpoint := range mc.requests {
		snapshot.Total += endpoint.TotalRequests
		snapshot.Errors += endpoint.ErrorCount

		endpointSnapshot := EndpointSnapshot{
			Method:        endpoint.Method,
			Path:          endpoint.Path,
			TotalRequests: endpoint.TotalRequests,
			StatusCounts:  make(map[int]int64),
			ErrorCount:    endpoint.ErrorCount,
		}

		// Copy status counts
		for status, count := range endpoint.StatusCounts {
			endpointSnapshot.StatusCounts[status] = count
		}

		// Calculate average latency
		if endpoint.TotalRequests > 0 {
			avg := endpoint.TotalLatency / time.Duration(endpoint.TotalRequests)
			endpointSnapshot.AvgLatency = avg.String()
		}

		// Calculate P95 latency
		if len(endpoint.Latencies) > 0 {
			p95 := calculateP95(endpoint.Latencies)
			endpointSnapshot.P95Latency = p95.String()
		}

		snapshot.Endpoints = append(snapshot.Endpoints, endpointSnapshot)
	}

	return snapshot
}

// GetEndpointMetrics returns metrics for a specific endpoint
func (mc *MetricsCollector) GetEndpointMetrics(method, path string) (EndpointSnapshot, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := method + " " + path
	endpoint, exists := mc.requests[key]
	if !exists {
		return EndpointSnapshot{}, false
	}

	snapshot := EndpointSnapshot{
		Method:        endpoint.Method,
		Path:          endpoint.Path,
		TotalRequests: endpoint.TotalRequests,
		StatusCounts:  make(map[int]int64),
		ErrorCount:    endpoint.ErrorCount,
	}

	for status, count := range endpoint.StatusCounts {
		snapshot.StatusCounts[status] = count
	}

	if endpoint.TotalRequests > 0 {
		avg := endpoint.TotalLatency / time.Duration(endpoint.TotalRequests)
		snapshot.AvgLatency = avg.String()
	}

	if len(endpoint.Latencies) > 0 {
		p95 := calculateP95(endpoint.Latencies)
		snapshot.P95Latency = p95.String()
	}

	return snapshot, true
}

// Reset clears all collected metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.requests = make(map[string]*EndpointMetrics)
}

// calculateP95 calculates the 95th percentile of a slice of durations
func calculateP95(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Make a copy to avoid modifying original
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple insertion sort for small slices
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	// Calculate P95 index
	index := int(float64(len(sorted)) * 0.95)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}
