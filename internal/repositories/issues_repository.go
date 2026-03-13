package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"gitlab-engineering-metrics-api/internal/domain"
)

// IssuesRepository handles database operations for issues
type IssuesRepository struct {
	db *sql.DB
}

// NewIssuesRepository creates a new issues repository
func NewIssuesRepository(db *sql.DB) *IssuesRepository {
	return &IssuesRepository{db: db}
}

// List returns a paginated list of issues with filters
func (r *IssuesRepository) List(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error) {
	// Build conditions and args
	conditions, args := r.buildFilterConditions(filter, 0)

	// Get total count
	total, err := r.getTotalCount(ctx, conditions, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get paginated issues
	items, err := r.getPaginatedIssues(ctx, filter, conditions, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get paginated issues: %w", err)
	}

	return &domain.IssuesListResponse{
		Items:    items,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Total:    total,
	}, nil
}

// buildFilterConditions builds SQL conditions and args from filter
func (r *IssuesRepository) buildFilterConditions(filter domain.IssuesFilter, startIdx int) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := startIdx

	if filter.ProjectID > 0 {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIdx))
		args = append(args, filter.ProjectID)
	}

	if filter.GroupPath != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("project_path LIKE $%d || '%%'", argIdx))
		args = append(args, filter.GroupPath)
	}

	if filter.Assignee != "" {
		argIdx++
		conditions = append(conditions, assigneeContainsCondition(argIdx))
		args = append(args, filter.Assignee)
	}

	if filter.State != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("current_canonical_state = $%d", argIdx))
		args = append(args, filter.State)
	}

	if filter.MetricFlag != "" {
		switch filter.MetricFlag {
		case "bypass":
			conditions = append(conditions, "skipped_in_progress_flag = true")
		case "rework":
			argIdx++
			conditions = append(conditions, fmt.Sprintf("qa_to_dev_return_count > $%d", argIdx))
			args = append(args, 0)
		case "blocked":
			argIdx++
			conditions = append(conditions, fmt.Sprintf("blocked_time_hours > $%d", argIdx))
			args = append(args, 0)
		}
	}

	if filter.IssueID > 0 {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("issue_id = $%d", argIdx))
		args = append(args, filter.IssueID)
	}

	if filter.GitlabIssueID > 0 {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("gitlab_issue_id = $%d", argIdx))
		args = append(args, filter.GitlabIssueID)
	}

	if filter.IssueIID > 0 {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("issue_iid = $%d", argIdx))
		args = append(args, filter.IssueIID)
	}

	return conditions, args
}

// getTotalCount returns the total count of issues matching the filter
func (r *IssuesRepository) getTotalCount(ctx context.Context, conditions []string, args []interface{}) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM vw_issue_lifecycle_metrics 
		WHERE 1=1
	`

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	var total int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

// getPaginatedIssues returns the paginated list of issues
func (r *IssuesRepository) getPaginatedIssues(ctx context.Context, filter domain.IssuesFilter, conditions []string, args []interface{}) ([]domain.IssueListItem, error) {
	query := `
		SELECT 
			issue_id,
			project_id,
			project_path,
			gitlab_issue_id,
			issue_iid,
			issue_title,
			COALESCE(ARRAY(SELECT jsonb_array_elements_text(` + normalizedAssigneesJSONBExpr("assignees") + `)), ARRAY[]::text[]) as assignees,
			COALESCE(current_canonical_state, 'UNKNOWN') as current_canonical_state,
			COALESCE(lead_time_hours / 24.0, 0) as lead_time_days,
			COALESCE(cycle_time_hours / 24.0, 0) as cycle_time_days,
			COALESCE(blocked_time_hours, 0) as blocked_time_hours,
			COALESCE(qa_to_dev_return_count, 0) as qa_to_dev_return_count,
			COALESCE(skipped_in_progress_flag, false) as has_bypass,
			CASE WHEN COALESCE(qa_to_dev_return_count, 0) > 0 THEN true ELSE false END as has_rework,
			CASE WHEN COALESCE(blocked_time_hours, 0) > 0 THEN true ELSE false END as was_blocked
		FROM vw_issue_lifecycle_metrics
		WHERE 1=1
	`

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY issue_id"

	// Add pagination
	if filter.PageSize > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, filter.PageSize)

		if filter.Page > 1 {
			offset := (filter.Page - 1) * filter.PageSize
			query += fmt.Sprintf(" OFFSET $%d", len(args)+1)
			args = append(args, offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.IssueListItem
	for rows.Next() {
		var item domain.IssueListItem
		var assignees []string
		var hasBypass, hasRework, wasBlocked bool
		err := rows.Scan(
			&item.IssueID,
			&item.ProjectID,
			&item.ProjectPath,
			&item.GitlabIssueID,
			&item.IssueIID,
			&item.Title,
			pq.Array(&assignees),
			&item.CurrentCanonicalState,
			&item.LeadTimeDays,
			&item.CycleTimeDays,
			&item.BlockedTimeHours,
			&item.QAToDevReturnCount,
			&hasBypass,
			&hasRework,
			&wasBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}
		item.Assignees = assignees
		item.HasBypass = hasBypass
		item.HasRework = hasRework
		item.WasBlocked = wasBlocked
		items = append(items, item)
	}

	return items, rows.Err()
}
