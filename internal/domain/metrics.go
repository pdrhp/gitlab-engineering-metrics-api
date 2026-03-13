package domain

// AvgP85Metric represents a metric with average and P85 values
type AvgP85Metric struct {
	Avg float64 `json:"avg,omitempty"`
	P85 float64 `json:"p85,omitempty"`
}

// AgingIssue represents an issue that has been in a state for a while
type AgingIssue struct {
	IssueIID     int      `json:"issue_iid,omitempty"`
	Title        string   `json:"title,omitempty"`
	Assignees    []string `json:"assignees,omitempty"`
	CurrentState string   `json:"current_state,omitempty"`
	DaysInState  int      `json:"days_in_state,omitempty"`
	WarningFlag  bool     `json:"warning_flag,omitempty"`
	// Unified identity fields
	IssueID       int    `json:"issue_id,omitempty"`
	GitlabIssueID int    `json:"gitlab_issue_id,omitempty"`
	ProjectID     int    `json:"project_id,omitempty"`
	ProjectPath   string `json:"project_path,omitempty"`
}

// Period represents a date range
type Period struct {
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}

// Throughput represents delivery throughput metrics
type Throughput struct {
	TotalIssuesDone int     `json:"total_issues_done,omitempty"`
	AvgPerWeek      float64 `json:"avg_per_week,omitempty"`
}

// SpeedMetrics represents speed-related metrics (lead time and cycle time)
type SpeedMetrics struct {
	LeadTime  *AvgP85Metric `json:"lead_time,omitempty"`
	CycleTime *AvgP85Metric `json:"cycle_time,omitempty"`
}

// AssigneeBreakdown represents metrics broken down by assignee
type AssigneeBreakdown struct {
	Assignee        string  `json:"assignee,omitempty"`
	IssuesDelivered int     `json:"issues_delivered,omitempty"`
	AvgCycleTime    float64 `json:"avg_cycle_time,omitempty"`
}

// ReworkMetrics represents rework-related quality metrics
type ReworkMetrics struct {
	PingPongRatePct         float64 `json:"ping_pong_rate_pct,omitempty"`
	TotalReworkedIssues     int     `json:"total_reworked_issues,omitempty"`
	AvgReworkCyclesPerIssue float64 `json:"avg_rework_cycles_per_issue,omitempty"`
}

// ProcessHealthMetrics represents process health indicators
type ProcessHealthMetrics struct {
	BypassRatePct        float64 `json:"bypass_rate_pct,omitempty"`
	FirstTimePassRatePct float64 `json:"first_time_pass_rate_pct,omitempty"`
}

// BottleneckMetrics represents bottleneck-related metrics
type BottleneckMetrics struct {
	TotalBlockedTimeHours       float64 `json:"total_blocked_time_hours,omitempty"`
	AvgBlockedTimePerIssueHours float64 `json:"avg_blocked_time_per_issue_hours,omitempty"`
}

// DefectMetrics represents defect-related metrics
type DefectMetrics struct {
	BugRatioPct float64 `json:"bug_ratio_pct,omitempty"`
}

// CurrentWIP represents current work in progress breakdown
type CurrentWIP struct {
	InProgress int `json:"in_progress,omitempty"`
	QAReview   int `json:"qa_review,omitempty"`
	Blocked    int `json:"blocked,omitempty"`
}

// DeliveryMetricsResponse represents the response for delivery metrics endpoint
type DeliveryMetricsResponse struct {
	Period              Period              `json:"period,omitempty"`
	Throughput          Throughput          `json:"throughput,omitempty"`
	SpeedMetricsDays    SpeedMetrics        `json:"speed_metrics_days,omitempty"`
	BreakdownByAssignee []AssigneeBreakdown `json:"breakdown_by_assignee,omitempty"`
}

// QualityMetricsResponse represents the response for quality metrics endpoint
type QualityMetricsResponse struct {
	Rework        ReworkMetrics        `json:"rework,omitempty"`
	ProcessHealth ProcessHealthMetrics `json:"process_health,omitempty"`
	Bottlenecks   BottleneckMetrics    `json:"bottlenecks,omitempty"`
	Defects       DefectMetrics        `json:"defects,omitempty"`
}

// WipMetricsResponse represents the response for WIP metrics endpoint
type WipMetricsResponse struct {
	CurrentWIP CurrentWIP   `json:"current_wip,omitempty"`
	AgingWIP   []AgingIssue `json:"aging_wip,omitempty"`
}

// MetricsFilter represents query parameters for metrics endpoints
type MetricsFilter struct {
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
	GroupPath string `json:"group_path,omitempty"`
	ProjectID int    `json:"project_id,omitempty"`
	Assignee  string `json:"assignee,omitempty"`
}
