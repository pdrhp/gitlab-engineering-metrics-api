package domain

import (
	"encoding/json"
	"testing"
)

func TestDeliveryTrendResponse_JSONNullability(t *testing.T) {
	var leadAvg *float64
	resp := DeliveryTrendResponse{
		Bucket:   "week",
		Timezone: "UTC",
		Items: []DeliveryTrendPoint{{
			BucketStart: "2026-02-02",
			BucketEnd:   "2026-02-08",
			BucketLabel: "2026-W06",
			Throughput:  DeliveryTrendThroughput{TotalIssuesDone: 0},
			SpeedMetricsDays: DeliveryTrendSpeedMetrics{
				LeadTime:  AvgP85MetricNullable{Avg: leadAvg, P85: nil},
				CycleTime: AvgP85MetricNullable{Avg: nil, P85: nil},
			},
		}},
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	s := string(raw)
	if !containsString(s, `"avg":null`) {
		t.Fatalf("expected null avg field, got: %s", s)
	}
}

func containsString(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
