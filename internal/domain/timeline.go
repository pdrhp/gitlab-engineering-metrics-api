package domain

import "time"

// IssueSummary represents a summary of an issue for timeline responses
type IssueSummary struct {
	Title                 string    `json:"title,omitempty"`
	MetadataLabels        []string  `json:"metadata_labels,omitempty"`
	Assignees             []string  `json:"assignees,omitempty"`
	CurrentCanonicalState string    `json:"current_canonical_state,omitempty"`
	GitlabCreatedAt       time.Time `json:"gitlab_created_at,omitempty"`
	// Unified identity fields
	IssueID       int    `json:"issue_id,omitempty"`
	GitlabIssueID int    `json:"gitlab_issue_id,omitempty"`
	IssueIID      int    `json:"issue_iid,omitempty"`
	ProjectID     int    `json:"project_id,omitempty"`
	ProjectPath   string `json:"project_path,omitempty"`
}

// TimelineEvent represents a single event in an issue's timeline
type TimelineEvent struct {
	Type                        string    `json:"type,omitempty"`
	Timestamp                   time.Time `json:"timestamp,omitempty"`
	Actor                       string    `json:"actor,omitempty"`
	FromState                   string    `json:"from_state,omitempty"`
	ToState                     string    `json:"to_state,omitempty"`
	DurationInPreviousStateMins int       `json:"duration_in_previous_state_mins,omitempty"`
	IsRework                    bool      `json:"is_rework,omitempty"`
	Body                        string    `json:"body,omitempty"`
}

// IssueTimelineResponse represents the response for issue timeline endpoint
type IssueTimelineResponse struct {
	Issue    IssueSummary    `json:"issue,omitempty"`
	Timeline []TimelineEvent `json:"timeline,omitempty"`
}

// TimelineFilter represents query parameters for timeline endpoint
// Note: issue ID is extracted from URL path parameter
type TimelineFilter struct {
	// Empty - issue ID comes from URL path
}
