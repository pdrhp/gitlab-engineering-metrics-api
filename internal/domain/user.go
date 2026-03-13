package domain

// User represents a user with their issue statistics
type User struct {
	Username                  string `json:"username"`
	DisplayName               string `json:"display_name"`
	ActiveIssues              int    `json:"active_issues"`
	CompletedIssuesLast30Days int    `json:"completed_issues_last_30_days"`
}
