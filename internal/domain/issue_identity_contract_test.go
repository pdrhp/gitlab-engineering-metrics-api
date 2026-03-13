package domain

import (
	"encoding/json"
	"testing"
)

// TestIssueIdentityContract_JSONTags verifies that all required identity fields
// exist in IssueListItem with the correct JSON tags
func TestIssueIdentityContract_JSONTags(t *testing.T) {
	// Create an IssueListItem with all identity fields populated
	item := IssueListItem{
		IssueID:       123,
		GitlabIssueID: 456,
		IssueIID:      789,
		ProjectID:     101,
		ProjectPath:   "my-group/my-project",
		Title:         "Test Issue",
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal IssueListItem: %v", err)
	}

	// Unmarshal into a map to check keys exist
	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Required JSON keys
	requiredKeys := []string{
		"issue_id",
		"gitlab_issue_id",
		"issue_iid",
		"project_id",
		"project_path",
	}

	// Verify each required key exists
	for _, key := range requiredKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("Required JSON key '%s' not found in IssueListItem JSON", key)
		}
	}

	// Verify issue_iid is present
	if _, exists := result["issue_iid"]; !exists {
		t.Errorf("Required key 'issue_iid' not found in IssueListItem JSON")
	}
}

// TestIssueIdentityContract_IssueSummary verifies IssueSummary has all identity fields
func TestIssueIdentityContract_IssueSummary(t *testing.T) {
	summary := IssueSummary{
		IssueID:       123,
		GitlabIssueID: 456,
		IssueIID:      789,
		ProjectID:     101,
		ProjectPath:   "my-group/my-project",
		Title:         "Test Issue",
	}

	jsonBytes, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Failed to marshal IssueSummary: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	requiredKeys := []string{
		"issue_id",
		"gitlab_issue_id",
		"issue_iid",
		"project_id",
		"project_path",
	}

	for _, key := range requiredKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("Required JSON key '%s' not found in IssueSummary JSON", key)
		}
	}
}

// TestIssueIdentityContract_GhostWorkIssue verifies GhostWorkIssue has all identity fields
func TestIssueIdentityContract_GhostWorkIssue(t *testing.T) {
	issue := GhostWorkIssue{
		IssueID:       123,
		GitlabIssueID: 456,
		IssueIID:      789,
		ProjectID:     101,
		ProjectPath:   "my-group/my-project",
		IssueTitle:    "Test Issue",
	}

	jsonBytes, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("Failed to marshal GhostWorkIssue: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	requiredKeys := []string{
		"issue_id",
		"gitlab_issue_id",
		"issue_iid",
		"project_id",
		"project_path",
	}

	for _, key := range requiredKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("Required JSON key '%s' not found in GhostWorkIssue JSON", key)
		}
	}
}

// TestIssueIdentityContract_AgingIssue verifies AgingIssue has all identity fields
func TestIssueIdentityContract_AgingIssue(t *testing.T) {
	issue := AgingIssue{
		IssueID:       123,
		GitlabIssueID: 456,
		IssueIID:      789,
		ProjectID:     101,
		ProjectPath:   "my-group/my-project",
		Title:         "Test Issue",
	}

	jsonBytes, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("Failed to marshal AgingIssue: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	requiredKeys := []string{
		"issue_id",
		"gitlab_issue_id",
		"issue_iid",
		"project_id",
		"project_path",
	}

	for _, key := range requiredKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("Required JSON key '%s' not found in AgingIssue JSON", key)
		}
	}
}
