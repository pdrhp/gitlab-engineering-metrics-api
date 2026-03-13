package domain

// GhostWorkIssue represents an issue that had ghost work (skipped IN_PROGRESS)
type GhostWorkIssue struct {
	IssueIID       int      `json:"issue_iid"`
	ProjectPath    string   `json:"project_path"`
	IssueTitle     string   `json:"issue_title"`
	Assignees      []string `json:"assignees"`
	FromState      string   `json:"from_state"`
	ToState        string   `json:"to_state"`
	TransitionTime string   `json:"transition_time"`
	DurationHours  float64  `json:"duration_hours"`
	CurrentState   string   `json:"current_state"`
	FinalDoneAt    string   `json:"final_done_at,omitempty"`
	// Unified identity fields
	IssueID       int `json:"issue_id,omitempty"`
	GitlabIssueID int `json:"gitlab_issue_id,omitempty"`
	ProjectID     int `json:"project_id,omitempty"`
}

// GhostWorkTransitionSummary represents aggregated ghost work by transition type
type GhostWorkTransitionSummary struct {
	FromState string `json:"from_state"`
	ToState   string `json:"to_state"`
	Count     int    `json:"count"`
}

// GhostWorkUserBreakdown represents ghost work aggregated by user
type GhostWorkUserBreakdown struct {
	Username       string `json:"username"`
	GhostWorkCount int    `json:"ghost_work_count"`
	IssueIIDs      []int  `json:"issue_iids"`
}

// GhostWorkMetricsResponse represents the complete ghost work deep dive response
type GhostWorkMetricsResponse struct {
	TotalIssues        int                          `json:"total_issues"`
	Period             Period                       `json:"period"`
	Issues             []GhostWorkIssue             `json:"issues"`
	TransitionAnalysis []GhostWorkTransitionSummary `json:"transition_analysis"`
	BreakdownByUser    []GhostWorkUserBreakdown     `json:"breakdown_by_user"`
	Page               int                          `json:"page"`
	PageSize           int                          `json:"page_size"`
	TotalPages         int                          `json:"total_pages"`
}

// GhostWorkFilter extends MetricsFilter with pagination and issue identifiers
type GhostWorkFilter struct {
	MetricsFilter
	Page          int `json:"page,omitempty"`
	PageSize      int `json:"page_size,omitempty"`
	IssueID       int `json:"issue_id,omitempty"`
	GitlabIssueID int `json:"gitlab_issue_id,omitempty"`
	IssueIID      int `json:"issue_iid,omitempty"`
}
