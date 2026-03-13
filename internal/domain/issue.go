package domain

import "time"

// IssueListItem represents a single issue in a list response
type IssueListItem struct {
	ProjectID             int      `json:"project_id,omitempty"`
	Title                 string   `json:"title,omitempty"`
	Assignees             []string `json:"assignees,omitempty"`
	CurrentCanonicalState string   `json:"current_canonical_state,omitempty"`
	LeadTimeDays          float64  `json:"lead_time_days,omitempty"`
	CycleTimeDays         float64  `json:"cycle_time_days,omitempty"`
	BlockedTimeHours      float64  `json:"blocked_time_hours,omitempty"`
	QAToDevReturnCount    int      `json:"qa_to_dev_return_count,omitempty"`
	// Metric flags for observability - always include for frontend filtering
	// Note: no omitempty so false values are included in JSON
	HasBypass  bool `json:"has_bypass"`
	HasRework  bool `json:"has_rework"`
	WasBlocked bool `json:"was_blocked"`
	// Unified identity fields
	IssueID       int    `json:"issue_id,omitempty"`
	GitlabIssueID int    `json:"gitlab_issue_id,omitempty"`
	IssueIID      int    `json:"issue_iid,omitempty"`
	ProjectPath   string `json:"project_path,omitempty"`
}

// IssuesListResponse represents the paginated response for issues list endpoint
type IssuesListResponse struct {
	Items    []IssueListItem `json:"items,omitempty"`
	Page     int             `json:"page,omitempty"`
	PageSize int             `json:"page_size,omitempty"`
	Total    int             `json:"total,omitempty"`
}

// IssuesFilter represents query parameters for issues list endpoint
type IssuesFilter struct {
	ProjectID     int       `json:"project_id,omitempty"`
	GroupPath     string    `json:"group_path,omitempty"`
	Assignee      string    `json:"assignee,omitempty"`
	State         string    `json:"state,omitempty"`
	MetricFlag    string    `json:"metric_flag,omitempty"`
	StartDate     time.Time `json:"start_date,omitempty"`
	EndDate       time.Time `json:"end_date,omitempty"`
	Page          int       `json:"page,omitempty"`
	PageSize      int       `json:"page_size,omitempty"`
	IssueID       int       `json:"issue_id,omitempty"`
	GitlabIssueID int       `json:"gitlab_issue_id,omitempty"`
	IssueIID      int       `json:"issue_iid,omitempty"`
}
