package domain

import "time"

// Project represents a GitLab project with its metadata
type Project struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	TotalIssues  int       `json:"total_issues"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}

// CatalogFilter represents query parameters for catalog endpoints
type CatalogFilter struct {
	Search    string `json:"search,omitempty"`
	GroupPath string `json:"group_path,omitempty"`
}
