package repositories

import (
	"database/sql"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestNewIssuesRepository(t *testing.T) {
	db := &sql.DB{}
	repo := NewIssuesRepository(db)

	if repo == nil {
		t.Error("Expected repository to not be nil")
	}

	if repo.db != db {
		t.Error("Expected repository to hold the database connection")
	}
}

func TestIssuesRepository_buildFilterConditions(t *testing.T) {
	repo := &IssuesRepository{}

	tests := []struct {
		name      string
		filter    domain.IssuesFilter
		wantCond  int // expected number of conditions
		wantArgs  int // expected number of args
		checkCond func([]string) bool
		checkArgs func([]interface{}) bool
	}{
		{
			name:     "empty filter",
			filter:   domain.IssuesFilter{},
			wantCond: 0,
			wantArgs: 0,
		},
		{
			name:     "project_id filter",
			filter:   domain.IssuesFilter{ProjectID: 123},
			wantCond: 1,
			wantArgs: 1,
			checkCond: func(conds []string) bool {
				return len(conds) > 0 && conds[0] == "project_id = $1"
			},
			checkArgs: func(args []interface{}) bool {
				return len(args) > 0 && args[0] == 123
			},
		},
		{
			name:     "group_path filter",
			filter:   domain.IssuesFilter{GroupPath: "group/subgroup"},
			wantCond: 1,
			wantArgs: 1,
			checkCond: func(conds []string) bool {
				return len(conds) > 0 && conds[0] == "project_path LIKE $1 || '%'"
			},
			checkArgs: func(args []interface{}) bool {
				return len(args) > 0 && args[0] == "group/subgroup"
			},
		},
		{
			name:     "assignee filter",
			filter:   domain.IssuesFilter{Assignee: "john.doe"},
			wantCond: 1,
			wantArgs: 1,
			checkCond: func(conds []string) bool {
				return len(conds) > 0 && conds[0] == "((jsonb_typeof(assignees) = 'array' AND assignees ? $1) OR (jsonb_typeof(assignees) = 'object' AND assignees ? 'current' AND (assignees->'current') ? $1))"
			},
			checkArgs: func(args []interface{}) bool {
				return len(args) > 0 && args[0] == "john.doe"
			},
		},
		{
			name:     "state filter",
			filter:   domain.IssuesFilter{State: "DONE"},
			wantCond: 1,
			wantArgs: 1,
			checkCond: func(conds []string) bool {
				return len(conds) > 0 && conds[0] == "current_canonical_state = $1"
			},
			checkArgs: func(args []interface{}) bool {
				return len(args) > 0 && args[0] == "DONE"
			},
		},
		{
			name:     "multiple filters",
			filter:   domain.IssuesFilter{ProjectID: 123, State: "IN_PROGRESS"},
			wantCond: 2,
			wantArgs: 2,
		},
		{
			name:     "issue_id filter",
			filter:   domain.IssuesFilter{IssueID: 456},
			wantCond: 1,
			wantArgs: 1,
			checkCond: func(conds []string) bool {
				return len(conds) > 0 && conds[0] == "issue_id = $1"
			},
			checkArgs: func(args []interface{}) bool {
				return len(args) > 0 && args[0] == 456
			},
		},
		{
			name:     "gitlab_issue_id filter",
			filter:   domain.IssuesFilter{GitlabIssueID: 789},
			wantCond: 1,
			wantArgs: 1,
			checkCond: func(conds []string) bool {
				return len(conds) > 0 && conds[0] == "gitlab_issue_id = $1"
			},
			checkArgs: func(args []interface{}) bool {
				return len(args) > 0 && args[0] == 789
			},
		},
		{
			name:     "issue_iid filter",
			filter:   domain.IssuesFilter{IssueIID: 42},
			wantCond: 1,
			wantArgs: 1,
			checkCond: func(conds []string) bool {
				return len(conds) > 0 && conds[0] == "issue_iid = $1"
			},
			checkArgs: func(args []interface{}) bool {
				return len(args) > 0 && args[0] == 42
			},
		},
		{
			name:     "all identity filters together",
			filter:   domain.IssuesFilter{IssueID: 1, GitlabIssueID: 100, IssueIID: 10},
			wantCond: 3,
			wantArgs: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions, args := repo.buildFilterConditions(tt.filter, 0)

			if len(conditions) != tt.wantCond {
				t.Errorf("Expected %d conditions, got %d", tt.wantCond, len(conditions))
			}

			if len(args) != tt.wantArgs {
				t.Errorf("Expected %d args, got %d", tt.wantArgs, len(args))
			}

			if tt.checkCond != nil && !tt.checkCond(conditions) {
				t.Errorf("Condition check failed. Got: %v", conditions)
			}

			if tt.checkArgs != nil && !tt.checkArgs(args) {
				t.Errorf("Args check failed. Got: %v", args)
			}
		})
	}
}

func TestIssuesRepository_buildFilterConditions_WithStartIdx(t *testing.T) {
	repo := &IssuesRepository{}

	filter := domain.IssuesFilter{
		ProjectID: 123,
		State:     "DONE",
	}

	conditions, args := repo.buildFilterConditions(filter, 2)

	// Should use $3 and $4 as placeholders (startIdx + 1, startIdx + 2)
	if len(conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(conditions))
	}

	if conditions[0] != "project_id = $3" {
		t.Errorf("Expected first condition to use $3, got: %s", conditions[0])
	}

	if conditions[1] != "current_canonical_state = $4" {
		t.Errorf("Expected second condition to use $4, got: %s", conditions[1])
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}
