package domain

// UserPerformanceIdentity captures basic catalog information for a user along with
// recent delivery context needed by performance consumers.
type UserPerformanceIdentity struct {
	Username                  string `json:"username,omitempty"`
	DisplayName               string `json:"display_name,omitempty"`
	ActiveIssues              int    `json:"active_issues,omitempty"`
	CompletedIssuesLast30Days int    `json:"completed_issues_last_30_days,omitempty"`
}

// GhostWorkMetrics represents bypass or skipped-in-progress behavior surfaced to
// the frontend as ghost work.
type GhostWorkMetrics struct {
	RatePct float64 `json:"rate_pct,omitempty"`
}

// UserDeliveryMetrics aggregates delivery statistics for a single user.
type UserDeliveryMetrics struct {
	Throughput       Throughput   `json:"throughput,omitempty"`
	SpeedMetricsDays SpeedMetrics `json:"speed_metrics_days,omitempty"`
}

// UserQualityMetrics surfaces the quality posture for a single user.
type UserQualityMetrics struct {
	Rework        ReworkMetrics        `json:"rework,omitempty"`
	GhostWork     GhostWorkMetrics     `json:"ghost_work,omitempty"`
	ProcessHealth ProcessHealthMetrics `json:"process_health,omitempty"`
	Bottlenecks   BottleneckMetrics    `json:"bottlenecks,omitempty"`
	Defects       DefectMetrics        `json:"defects,omitempty"`
}

// UserPerformanceResponse is the contract returned by GET /api/v1/users/{username}/performance.
type UserPerformanceResponse struct {
	User     UserPerformanceIdentity `json:"user,omitempty"`
	Period   Period                  `json:"period,omitempty"`
	Delivery UserDeliveryMetrics     `json:"delivery,omitempty"`
	Quality  UserQualityMetrics      `json:"quality,omitempty"`
	WIP      WipMetricsResponse      `json:"wip,omitempty"`
}
