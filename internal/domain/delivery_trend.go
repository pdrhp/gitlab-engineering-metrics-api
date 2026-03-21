package domain

type DeliveryTrendFilter struct {
	MetricsFilter
	Bucket              string `json:"bucket,omitempty"`
	Timezone            string `json:"timezone,omitempty"`
	IncludeEmptyBuckets bool   `json:"include_empty_buckets"`
}

type AvgP85MetricNullable struct {
	Avg *float64 `json:"avg"`
	P85 *float64 `json:"p85"`
}

type DeliveryTrendThroughput struct {
	TotalIssuesDone int `json:"total_issues_done"`
}

type DeliveryTrendSpeedMetrics struct {
	LeadTime  AvgP85MetricNullable `json:"lead_time"`
	CycleTime AvgP85MetricNullable `json:"cycle_time"`
}

type DeliveryTrendPoint struct {
	BucketStart      string                    `json:"bucket_start"`
	BucketEnd        string                    `json:"bucket_end"`
	BucketLabel      string                    `json:"bucket_label"`
	Throughput       DeliveryTrendThroughput   `json:"throughput"`
	SpeedMetricsDays DeliveryTrendSpeedMetrics `json:"speed_metrics_days"`
}

type DeliveryTrendCorrelation struct {
	ThroughputVsLeadTimeR  *float64 `json:"throughput_vs_lead_time_r"`
	ThroughputVsCycleTimeR *float64 `json:"throughput_vs_cycle_time_r"`
}

type DeliveryTrendFiltersApplied struct {
	GroupPath *string `json:"group_path"`
	ProjectID *int    `json:"project_id"`
	Assignee  *string `json:"assignee"`
}

type DeliveryTrendResponse struct {
	Period         Period                      `json:"period"`
	Bucket         string                      `json:"bucket"`
	Timezone       string                      `json:"timezone"`
	FiltersApplied DeliveryTrendFiltersApplied `json:"filters_applied,omitempty"`
	Items          []DeliveryTrendPoint        `json:"items"`
	Correlation    *DeliveryTrendCorrelation   `json:"correlation,omitempty"`
}
