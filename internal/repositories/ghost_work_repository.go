package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/lib/pq"
	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/observability"
)

var ghostWorkRepoLogger = observability.GetLogger().With(slog.String("repository", "ghost_work"))

// GhostWorkRepository handles database operations for ghost work metrics
type GhostWorkRepository struct {
	db *sql.DB
}

// NewGhostWorkRepository creates a new ghost work repository
func NewGhostWorkRepository(db *sql.DB) *GhostWorkRepository {
	return &GhostWorkRepository{db: db}
}

// GetGhostWorkIssues returns ghost work issues with their transitions and aggregates
func (r *GhostWorkRepository) GetGhostWorkIssues(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error) {
	ghostWorkRepoLogger.Debug("getting ghost work issues",
		slog.String("start_date", filter.StartDate),
		slog.String("end_date", filter.EndDate),
		slog.Int("page", filter.Page),
		slog.Int("page_size", filter.PageSize),
	)

	// Build base query conditions
	conditions, args := r.buildFilterConditions(filter, 0)

	// Get detailed issues with ghost work
	issues, err := r.queryGhostWorkIssues(ctx, conditions, args, filter.Page, filter.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to query ghost work issues: %w", err)
	}

	// Get transition analysis (no filters for aggregates)
	transitionAnalysis, err := r.queryTransitionAnalysis(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query transition analysis: %w", err)
	}

	// Get user breakdown (no filters for aggregates)
	userBreakdown, err := r.queryUserBreakdown(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query user breakdown: %w", err)
	}

	// Get total count
	totalIssues, err := r.queryTotalCount(ctx, conditions, args)
	if err != nil {
		return nil, fmt.Errorf("failed to query total count: %w", err)
	}

	totalPages := (totalIssues + filter.PageSize - 1) / filter.PageSize

	return &domain.GhostWorkMetricsResponse{
		TotalIssues:        totalIssues,
		Period:             domain.Period{StartDate: filter.StartDate, EndDate: filter.EndDate},
		Issues:             issues,
		TransitionAnalysis: transitionAnalysis,
		BreakdownByUser:    userBreakdown,
		Page:               filter.Page,
		PageSize:           filter.PageSize,
		TotalPages:         totalPages,
	}, nil
}

// queryGhostWorkIssues queries the detailed list of ghost work issues
func (r *GhostWorkRepository) queryGhostWorkIssues(ctx context.Context, conditions []string, args []interface{}, page, pageSize int) ([]domain.GhostWorkIssue, error) {
	query := `
		SELECT DISTINCT ON (m.issue_id)
			m.issue_id,
			m.gitlab_issue_id,
			m.issue_iid,
			m.project_id,
			m.project_path,
			m.issue_title,
			COALESCE(ARRAY(SELECT jsonb_array_elements_text(` + normalizedAssigneesJSONBExpr("m.assignees") + `)), ARRAY[]::text[]) as assignees,
			'BACKLOG' as from_state,
			CASE 
				WHEN t.next_canonical_state = 'DONE' THEN 'DONE'
				WHEN t.next_canonical_state = 'QA_REVIEW' THEN 'QA_REVIEW'
				ELSE t.next_canonical_state
			END as to_state,
			to_char(t.exited_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') as transition_time,
			t.duration_hours_to_next_state as duration_hours,
			m.current_canonical_state,
			to_char(m.final_done_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') as final_done_at
		FROM vw_issue_lifecycle_metrics m
		INNER JOIN vw_issue_state_transitions t ON t.issue_id = m.issue_id
		WHERE m.skipped_in_progress_flag = true
			AND t.canonical_state = 'BACKLOG'
			AND t.next_canonical_state IN ('DONE', 'QA_REVIEW')
	`

	if len(conditions) > 0 {
		query += " AND " + joinConditions(conditions)
	}

	query += ` ORDER BY m.issue_id, t.entered_at DESC`

	// Add pagination
	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	ghostWorkRepoLogger.Debug("executing ghost work issues query",
		slog.String("query", query),
		slog.Int("arg_count", len(args)),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		ghostWorkRepoLogger.Error("ghost work issues query failed",
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	defer rows.Close()

	var issues []domain.GhostWorkIssue
	for rows.Next() {
		var issue domain.GhostWorkIssue
		var assignees []string
		var finalDoneAt sql.NullString

		err := rows.Scan(
			&issue.IssueID,
			&issue.GitlabIssueID,
			&issue.IssueIID,
			&issue.ProjectID,
			&issue.ProjectPath,
			&issue.IssueTitle,
			pq.Array(&assignees),
			&issue.FromState,
			&issue.ToState,
			&issue.TransitionTime,
			&issue.DurationHours,
			&issue.CurrentState,
			&finalDoneAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ghost work issue: %w", err)
		}

		issue.Assignees = assignees
		if finalDoneAt.Valid {
			issue.FinalDoneAt = finalDoneAt.String
		}

		issues = append(issues, issue)
	}

	return issues, rows.Err()
}

// queryTransitionAnalysis aggregates ghost work by transition type
func (r *GhostWorkRepository) queryTransitionAnalysis(ctx context.Context) ([]domain.GhostWorkTransitionSummary, error) {
	query := `
		SELECT 
			'BACKLOG' as from_state,
			CASE 
				WHEN t.next_canonical_state = 'DONE' THEN 'DONE'
				WHEN t.next_canonical_state = 'QA_REVIEW' THEN 'QA_REVIEW'
				ELSE t.next_canonical_state
			END as to_state,
			COUNT(DISTINCT m.issue_id) as count
		FROM vw_issue_lifecycle_metrics m
		INNER JOIN vw_issue_state_transitions t ON t.issue_id = m.issue_id
		WHERE m.skipped_in_progress_flag = true
			AND t.canonical_state = 'BACKLOG'
			AND t.next_canonical_state IN ('DONE', 'QA_REVIEW')
		GROUP BY to_state ORDER BY count DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []domain.GhostWorkTransitionSummary
	for rows.Next() {
		var summary domain.GhostWorkTransitionSummary
		if err := rows.Scan(&summary.FromState, &summary.ToState, &summary.Count); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, rows.Err()
}

// queryUserBreakdown aggregates ghost work by user
func (r *GhostWorkRepository) queryUserBreakdown(ctx context.Context) ([]domain.GhostWorkUserBreakdown, error) {
	query := `
		SELECT 
			a.username,
			COUNT(DISTINCT m.issue_id) as ghost_work_count,
			ARRAY_AGG(DISTINCT m.issue_iid ORDER BY m.issue_iid) as issue_iids
		FROM vw_issue_lifecycle_metrics m
		INNER JOIN vw_issue_state_transitions t ON t.issue_id = m.issue_id
		CROSS JOIN LATERAL jsonb_array_elements_text(` + normalizedAssigneesJSONBExpr("m.assignees") + `) as a(username)
		WHERE m.skipped_in_progress_flag = true
			AND t.canonical_state = 'BACKLOG'
			AND t.next_canonical_state IN ('DONE', 'QA_REVIEW')
		GROUP BY a.username ORDER BY ghost_work_count DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdowns []domain.GhostWorkUserBreakdown
	for rows.Next() {
		var breakdown domain.GhostWorkUserBreakdown
		var issueIIDs []int64 // PostgreSQL returns int64 for ARRAY_AGG
		if err := rows.Scan(&breakdown.Username, &breakdown.GhostWorkCount, pq.Array(&issueIIDs)); err != nil {
			return nil, err
		}
		// Convert int64 to int
		for _, id := range issueIIDs {
			breakdown.IssueIIDs = append(breakdown.IssueIIDs, int(id))
		}
		breakdowns = append(breakdowns, breakdown)
	}

	return breakdowns, rows.Err()
}

// queryTotalCount returns the total number of ghost work issues
func (r *GhostWorkRepository) queryTotalCount(ctx context.Context, conditions []string, args []interface{}) (int, error) {
	query := `
		SELECT COUNT(DISTINCT m.issue_id)
		FROM vw_issue_lifecycle_metrics m
		INNER JOIN vw_issue_state_transitions t ON t.issue_id = m.issue_id
		WHERE m.skipped_in_progress_flag = true
			AND t.canonical_state = 'BACKLOG'
			AND t.next_canonical_state IN ('DONE', 'QA_REVIEW')
	`

	if len(conditions) > 0 {
		query += " AND " + joinConditions(conditions)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// buildFilterConditions builds SQL conditions for ghost work queries
func (r *GhostWorkRepository) buildFilterConditions(filter domain.GhostWorkFilter, startIdx int) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := startIdx

	if filter.GroupPath != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("m.project_path LIKE $%d || '%%'", argIdx))
		args = append(args, filter.GroupPath)
	}

	if filter.ProjectID > 0 {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("m.project_id = $%d", argIdx))
		args = append(args, filter.ProjectID)
	}

	if filter.Assignee != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("((jsonb_typeof(m.assignees) = 'array' AND m.assignees ? $%d) OR (jsonb_typeof(m.assignees) = 'object' AND m.assignees ? 'current' AND (m.assignees->'current') ? $%d))", argIdx, argIdx))
		args = append(args, filter.Assignee)
	}

	if filter.IssueID > 0 {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("m.issue_id = $%d", argIdx))
		args = append(args, filter.IssueID)
	}

	if filter.GitlabIssueID > 0 {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("m.gitlab_issue_id = $%d", argIdx))
		args = append(args, filter.GitlabIssueID)
	}

	if filter.IssueIID > 0 {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("m.issue_iid = $%d", argIdx))
		args = append(args, filter.IssueIID)
	}

	if filter.StartDate != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("m.final_done_at >= $%d::date", argIdx))
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("m.final_done_at < ($%d::date + INTERVAL '1 day')", argIdx))
		args = append(args, filter.EndDate)
	}

	return conditions, args
}

// joinConditions joins conditions with AND
func joinConditions(conditions []string) string {
	result := ""
	for i, c := range conditions {
		if i > 0 {
			result += " AND "
		}
		result += c
	}
	return result
}
