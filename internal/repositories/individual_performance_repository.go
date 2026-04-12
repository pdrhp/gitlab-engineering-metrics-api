package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/observability"
)

var individualPerformanceRepoLogger = observability.GetLogger().With(slog.String("repository", "individual_performance"))

// IndividualPerformanceRepository defines the contract for fetching fair individual performance metrics
// Uses vw_assignee_cycle_time and vw_individual_performance_metrics for accurate assignee-level data
type IndividualPerformanceRepository interface {
	// GetAssigneeCycleTime returns cycle time breakdown for a specific assignee
	// Each row represents one issue the assignee worked on
	GetAssigneeCycleTime(ctx context.Context, username string, filter domain.MetricsFilter) ([]domain.AssigneeCycleTime, error)

	// GetIndividualPerformanceMetrics returns aggregated performance metrics for an assignee
	GetIndividualPerformanceMetrics(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error)
}

// IndividualPerformanceRepositoryImpl implements IndividualPerformanceRepository
type IndividualPerformanceRepositoryImpl struct {
	db *sql.DB
}

// NewIndividualPerformanceRepository creates a new instance of the repository
func NewIndividualPerformanceRepository(db *sql.DB) *IndividualPerformanceRepositoryImpl {
	return &IndividualPerformanceRepositoryImpl{db: db}
}

// GetAssigneeCycleTime returns cycle time breakdown for a specific assignee
func (r *IndividualPerformanceRepositoryImpl) GetAssigneeCycleTime(ctx context.Context, username string, filter domain.MetricsFilter) ([]domain.AssigneeCycleTime, error) {
	individualPerformanceRepoLogger.Debug("getting assignee cycle time",
		slog.String("username", username),
		slog.String("start_date", filter.StartDate),
		slog.String("end_date", filter.EndDate),
		slog.Int("project_id", filter.ProjectID),
	)

	query := `
		SELECT
			act.issue_id,
			act.issue_iid,
			act.project_id,
			COALESCE(act.active_cycle_hours, 0) as active_cycle_hours,
			COALESCE(act.in_progress_hours, 0) as in_progress_hours,
			COALESCE(act.qa_review_hours, 0) as qa_review_hours,
			COALESCE(act.blocked_hours, 0) as blocked_hours,
			COALESCE(act.backlog_hours, 0) as backlog_hours,
			COALESCE(act.total_hours_as_assignee, 0) as total_hours_as_assignee,
			COALESCE(act.contributed_active_work, false) as contributed_active_work
		FROM vw_assignee_cycle_time act
		INNER JOIN vw_issue_lifecycle_metrics lcm ON lcm.issue_id = act.issue_id AND lcm.project_id = act.project_id
		WHERE act.assignee_username = $1
	`

	args := []interface{}{username}
	argIdx := 1

	if filter.ProjectID > 0 {
		argIdx++
		query += fmt.Sprintf(" AND act.project_id = $%d", argIdx)
		args = append(args, filter.ProjectID)
	}

	if filter.StartDate != "" {
		argIdx++
		query += fmt.Sprintf(" AND lcm.final_done_at >= $%d::date", argIdx)
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != "" {
		argIdx++
		query += fmt.Sprintf(" AND lcm.final_done_at < ($%d::date + INTERVAL '1 day')", argIdx)
		args = append(args, filter.EndDate)
	}

	query += " ORDER BY act.issue_id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		individualPerformanceRepoLogger.Error("assignee cycle time query failed",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get assignee cycle time: %w", err)
	}
	defer rows.Close()

	var cycleTimes []domain.AssigneeCycleTime
	for rows.Next() {
		var ct domain.AssigneeCycleTime
		if err := rows.Scan(
			&ct.IssueID,
			&ct.IssueIID,
			&ct.ProjectID,
			&ct.ActiveCycleHours,
			&ct.InProgressHours,
			&ct.QAReviewHours,
			&ct.BlockedHours,
			&ct.BacklogHours,
			&ct.TotalHoursAsAssignee,
			&ct.ContributedActiveWork,
		); err != nil {
			return nil, fmt.Errorf("failed to scan assignee cycle time: %w", err)
		}
		cycleTimes = append(cycleTimes, ct)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assignee cycle time rows: %w", err)
	}

	individualPerformanceRepoLogger.Debug("got assignee cycle time",
		slog.Int("issues_count", len(cycleTimes)),
	)

	return cycleTimes, nil
}

// GetIndividualPerformanceMetrics returns aggregated performance metrics for an assignee
// Always aggregates across all projects unless ProjectID is specified in filter
// Date filtering is applied via JOIN with vw_issue_lifecycle_metrics using final_done_at
func (r *IndividualPerformanceRepositoryImpl) GetIndividualPerformanceMetrics(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error) {
	individualPerformanceRepoLogger.Debug("getting individual performance metrics",
		slog.String("username", username),
		slog.String("start_date", filter.StartDate),
		slog.String("end_date", filter.EndDate),
		slog.Int("project_id", filter.ProjectID),
	)

	baseQuery := `
		WITH filtered_assignee_data AS (
			SELECT
				act.assignee_username,
				act.project_id,
				act.issue_id,
				act.active_cycle_hours,
				act.in_progress_hours,
				act.qa_review_hours,
				act.blocked_hours,
				act.backlog_hours,
				act.total_hours_as_assignee,
				act.contributed_active_work
			FROM vw_assignee_cycle_time act
			INNER JOIN vw_issue_lifecycle_metrics lcm ON lcm.issue_id = act.issue_id AND lcm.project_id = act.project_id
			%s
		)
		SELECT
			assignee_username,
			COALESCE(COUNT(DISTINCT issue_id), 0)::bigint AS issues_assigned,
			COALESCE(COUNT(DISTINCT issue_id) FILTER (WHERE contributed_active_work), 0)::bigint AS issues_contributed,
			COALESCE(ROUND(SUM(active_cycle_hours)::numeric, 2), 0) AS total_active_cycle_hours,
			COALESCE(ROUND(AVG(active_cycle_hours)::numeric, 2), 0) AS avg_active_cycle_per_issue,
			COALESCE(ROUND(SUM(in_progress_hours)::numeric, 2), 0) AS total_dev_hours,
			COALESCE(ROUND(SUM(qa_review_hours)::numeric, 2), 0) AS total_qa_hours,
			COALESCE(ROUND(SUM(blocked_hours)::numeric, 2), 0) AS total_blocked_hours,
			COALESCE(ROUND(SUM(backlog_hours)::numeric, 2), 0) AS total_backlog_hours,
			CASE 
				WHEN SUM(total_hours_as_assignee) > 0 
				THEN ROUND((100.0 * SUM(active_cycle_hours) / SUM(total_hours_as_assignee))::numeric, 2)
				ELSE 0 
			END AS active_work_pct,
			COALESCE(ROUND(SUM(total_hours_as_assignee)::numeric, 2), 0) AS total_hours_as_assignee,
			COALESCE(ROUND(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY active_cycle_hours)::numeric, 2), 0) AS p50_active_cycle_hours,
			COALESCE(ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY active_cycle_hours)::numeric, 2), 0) AS p95_active_cycle_hours
		FROM filtered_assignee_data
		GROUP BY assignee_username
	`

	args := []interface{}{username}
	whereConditions := []string{"act.assignee_username = $1"}

	if filter.ProjectID > 0 {
		args = append(args, filter.ProjectID)
		whereConditions = append(whereConditions, fmt.Sprintf("act.project_id = $%d", len(args)))
	}

	if filter.StartDate != "" {
		args = append(args, filter.StartDate)
		whereConditions = append(whereConditions, fmt.Sprintf("lcm.final_done_at >= $%d::date", len(args)))
	}

	if filter.EndDate != "" {
		args = append(args, filter.EndDate)
		whereConditions = append(whereConditions, fmt.Sprintf("lcm.final_done_at < ($%d::date + INTERVAL '1 day')", len(args)))
	}

	whereClause := "WHERE " + strings.Join(whereConditions, " AND ")
	query := fmt.Sprintf(baseQuery, whereClause)

	var metrics domain.IndividualPerformanceMetrics
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&metrics.Username,
		&metrics.IssuesAssigned,
		&metrics.IssuesContributed,
		&metrics.TotalActiveCycleHours,
		&metrics.AvgActiveCyclePerIssue,
		&metrics.TotalDevHours,
		&metrics.TotalQAHours,
		&metrics.TotalBlockedHours,
		&metrics.TotalBacklogHours,
		&metrics.ActiveWorkPct,
		&metrics.TotalHoursAsAssignee,
		&metrics.P50ActiveCycleHours,
		&metrics.P95ActiveCycleHours,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			individualPerformanceRepoLogger.Debug("no individual performance metrics found",
				slog.String("username", username),
			)
			return nil, nil
		}
		individualPerformanceRepoLogger.Error("individual performance metrics query failed",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get individual performance metrics: %w", err)
	}

	individualPerformanceRepoLogger.Info("got individual performance metrics",
		slog.String("username", metrics.Username),
		slog.Int("issues_assigned", metrics.IssuesAssigned),
		slog.Int("issues_contributed", metrics.IssuesContributed),
	)

	return &metrics, nil
}
