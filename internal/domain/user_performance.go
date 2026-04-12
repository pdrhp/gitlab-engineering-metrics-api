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

// AssigneeCycleTime represents time spent by an assignee on a single issue
// during their actual assignment period (fair attribution)
type AssigneeCycleTime struct {
	IssueID               int     `json:"issue_id"`
	IssueIID              int     `json:"issue_iid"`
	ProjectID             int     `json:"project_id"`
	ActiveCycleHours      float64 `json:"active_cycle_hours"`
	InProgressHours       float64 `json:"in_progress_hours"`
	QAReviewHours         float64 `json:"qa_review_hours"`
	BlockedHours          float64 `json:"blocked_hours"`
	BacklogHours          float64 `json:"backlog_hours"`
	TotalHoursAsAssignee  float64 `json:"total_hours_as_assignee"`
	ContributedActiveWork bool    `json:"contributed_active_work"`
}

// IndividualPerformanceMetrics aggregates performance metrics for an assignee
// across all their assigned issues (fair attribution model)
type IndividualPerformanceMetrics struct {
	Username               string  `json:"username"`
	IssuesAssigned         int     `json:"issues_assigned"`
	IssuesContributed      int     `json:"issues_contributed"`
	TotalActiveCycleHours  float64 `json:"total_active_cycle_hours"`
	AvgActiveCyclePerIssue float64 `json:"avg_active_cycle_per_issue"`
	TotalDevHours          float64 `json:"total_dev_hours"`
	TotalQAHours           float64 `json:"total_qa_hours"`
	TotalBlockedHours      float64 `json:"total_blocked_hours"`
	TotalBacklogHours      float64 `json:"total_backlog_hours"`
	ActiveWorkPct          float64 `json:"active_work_pct"`
	TotalHoursAsAssignee   float64 `json:"total_hours_as_assignee"`
	P50ActiveCycleHours    float64 `json:"p50_active_cycle_hours"`
	P95ActiveCycleHours    float64 `json:"p95_active_cycle_hours"`
}

// HasActiveContribution returns true if the assignee has contributed active work
func (m *IndividualPerformanceMetrics) HasActiveContribution() bool {
	return m.IssuesContributed > 0
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
// Uses fair attribution model (vw_assignee_cycle_time) for individual metrics
// to ensure each assignee receives credit only for their actual time on issues.
type UserPerformanceResponse struct {
	User                  UserPerformanceIdentity       `json:"user,omitempty"`
	Period                Period                        `json:"period,omitempty"`
	Delivery              UserDeliveryMetrics           `json:"delivery,omitempty"`
	Quality               UserQualityMetrics            `json:"quality,omitempty"`
	WIP                   WipMetricsResponse            `json:"wip,omitempty"`
	IndividualPerformance *IndividualPerformanceMetrics `json:"individual_performance,omitempty"`
}
