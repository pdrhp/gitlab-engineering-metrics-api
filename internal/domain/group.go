package domain

import "time"

// Group represents a GitLab group with aggregated project statistics
type Group struct {
	GroupPath    string    `json:"group_path"`
	ProjectCount int       `json:"project_count"`
	TotalIssues  int       `json:"total_issues"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}
