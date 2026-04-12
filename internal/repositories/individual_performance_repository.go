package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

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
			issue_id,
			issue_iid,
			project_id,
			COALESCE(active_cycle_hours, 0) as active_cycle_hours,
			COALESCE(in_progress_hours, 0) as in_progress_hours,
			COALESCE(qa_review_hours, 0) as qa_review_hours,
			COALESCE(blocked_hours, 0) as blocked_hours,
			COALESCE(backlog_hours, 0) as backlog_hours,
			COALESCE(total_hours_as_assignee, 0) as total_hours_as_assignee,
			COALESCE(contributed_active_work, false) as contributed_active_work
		FROM vw_assignee_cycle_time
		WHERE assignee_username = $1
	`

	args := []interface{}{username}
	argIdx := 1

	if filter.ProjectID > 0 {
		argIdx++
		query += fmt.Sprintf(" AND project_id = $%d", argIdx)
		args = append(args, filter.ProjectID)
	}

	query += " ORDER BY issue_id"

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
// Note: Date filtering is applied via join with vw_assignee_cycle_time which has issue-level data
func (r *IndividualPerformanceRepositoryImpl) GetIndividualPerformanceMetrics(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error) {
	individualPerformanceRepoLogger.Debug("getting individual performance metrics",
		slog.String("username", username),
		slog.String("start_date", filter.StartDate),
		slog.String("end_date", filter.EndDate),
		slog.Int("project_id", filter.ProjectID),
	)

	query := `
		SELECT
			$1 as assignee_username,
			COALESCE(SUM(issues_assigned), 0)::bigint as issues_assigned,
			COALESCE(SUM(issues_contributed), 0)::bigint as issues_contributed,
			COALESCE(SUM(total_active_cycle_hours), 0) as total_active_cycle_hours,
			COALESCE(AVG(avg_active_cycle_per_issue), 0) as avg_active_cycle_per_issue,
			COALESCE(SUM(total_dev_hours), 0) as total_dev_hours,
			COALESCE(SUM(total_qa_hours), 0) as total_qa_hours,
			COALESCE(SUM(total_blocked_hours), 0) as total_blocked_hours,
			COALESCE(SUM(total_backlog_hours), 0) as total_backlog_hours,
			CASE 
				WHEN SUM(total_hours_as_assignee) > 0 
				THEN ROUND((100.0 * SUM(total_active_cycle_hours) / SUM(total_hours_as_assignee))::numeric, 2)
				ELSE 0 
			END as active_work_pct,
			COALESCE(SUM(total_hours_as_assignee), 0) as total_hours_as_assignee,
			COALESCE(AVG(p50_active_cycle_hours), 0) as p50_active_cycle_hours,
			COALESCE(AVG(p95_active_cycle_hours), 0) as p95_active_cycle_hours
		FROM vw_individual_performance_metrics
		WHERE assignee_username = $1
	`

	args := []interface{}{username}
	argIdx := 1

	if filter.ProjectID > 0 {
		argIdx++
		query += fmt.Sprintf(" AND project_id = $%d", argIdx)
		args = append(args, filter.ProjectID)
	}

	query += " GROUP BY assignee_username"

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
