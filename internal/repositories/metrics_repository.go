package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/lib/pq"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/observability"
)

var metricsRepoLogger = observability.GetLogger().With(slog.String("repository", "metrics"))

// MetricsRepository handles database operations for metrics
type MetricsRepository struct {
	db *sql.DB
}

// NewMetricsRepository creates a new metrics repository
func NewMetricsRepository(db *sql.DB) *MetricsRepository {
	return &MetricsRepository{db: db}
}

// GetDeliveryMetrics returns delivery metrics for the given filter
func (r *MetricsRepository) GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error) {
	metricsRepoLogger.Debug("getting delivery metrics",
		slog.String("start_date", filter.StartDate),
		slog.String("end_date", filter.EndDate),
		slog.String("group_path", filter.GroupPath),
		slog.Int("project_id", filter.ProjectID),
		slog.String("assignee", filter.Assignee),
	)

	// Build base query conditions
	conditions, args := r.buildFilterConditions(filter, 0)
	metricsRepoLogger.Debug("built filter conditions",
		slog.Int("condition_count", len(conditions)),
		slog.Int("arg_count", len(args)),
	)

	// Get throughput metrics
	throughput, err := r.getThroughputMetrics(ctx, conditions, args)
	if err != nil {
		metricsRepoLogger.Error("failed to get throughput metrics",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get throughput metrics: %w", err)
	}
	metricsRepoLogger.Debug("got throughput metrics",
		slog.Int("total_issues", throughput.TotalIssuesDone),
	)

	// Get speed metrics (lead time and cycle time with P85)
	speedMetrics, err := r.getSpeedMetrics(ctx, conditions, args)
	if err != nil {
		metricsRepoLogger.Error("failed to get speed metrics",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get speed metrics: %w", err)
	}

	// Get breakdown by assignee
	breakdown, err := r.getAssigneeBreakdown(ctx, conditions, args)
	if err != nil {
		metricsRepoLogger.Error("failed to get assignee breakdown",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get assignee breakdown: %w", err)
	}

	metricsRepoLogger.Info("delivery metrics retrieved successfully",
		slog.Int("throughput_total", throughput.TotalIssuesDone),
		slog.Int("assignee_breakdown_count", len(breakdown)),
	)

	return &domain.DeliveryMetricsResponse{
		Period: domain.Period{
			StartDate: filter.StartDate,
			EndDate:   filter.EndDate,
		},
		Throughput:          throughput,
		SpeedMetricsDays:    speedMetrics,
		BreakdownByAssignee: breakdown,
	}, nil
}

// GetQualityMetrics returns quality metrics for the given filter
func (r *MetricsRepository) GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error) {
	conditions, args := r.buildFilterConditions(filter, 0)

	// Get rework metrics
	rework, err := r.getReworkMetrics(ctx, conditions, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get rework metrics: %w", err)
	}

	// Get process health metrics
	processHealth, err := r.getProcessHealthMetrics(ctx, conditions, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get process health metrics: %w", err)
	}

	// Get bottleneck metrics
	bottlenecks, err := r.getBottleneckMetrics(ctx, conditions, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get bottleneck metrics: %w", err)
	}

	// Get defect metrics
	defects, err := r.getDefectMetrics(ctx, conditions, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get defect metrics: %w", err)
	}

	return &domain.QualityMetricsResponse{
		Rework:        rework,
		ProcessHealth: processHealth,
		Bottlenecks:   bottlenecks,
		Defects:       defects,
	}, nil
}

// GetWipMetrics returns WIP metrics for the given filter
func (r *MetricsRepository) GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error) {
	conditions, args := r.buildFilterConditions(filter, 0)

	// Get current WIP counts
	currentWIP, err := r.getCurrentWIP(ctx, conditions, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get current WIP: %w", err)
	}

	// Get aging WIP issues
	agingWIP, err := r.getAgingWIP(ctx, conditions, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get aging WIP: %w", err)
	}

	return &domain.WipMetricsResponse{
		CurrentWIP: currentWIP,
		AgingWIP:   agingWIP,
	}, nil
}

// GetDeliveryTrendMetrics returns delivery trend metrics with bucketing and correlation
func (r *MetricsRepository) GetDeliveryTrendMetrics(ctx context.Context, filter domain.DeliveryTrendFilter) (*domain.DeliveryTrendResponse, error) {
	// Defaults
	if filter.Bucket == "" {
		filter.Bucket = "week"
	}
	if filter.Timezone == "" {
		filter.Timezone = "UTC"
	}

	// Check project_id belongs to group_path if both provided
	if filter.ProjectID > 0 && filter.GroupPath != "" {
		var count int
		err := r.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM projects WHERE id = $1 AND regexp_replace(path, '/[^/]+$', '') = $2",
			filter.ProjectID, filter.GroupPath).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to validate project group membership: %w", err)
		}
		if count == 0 {
			return nil, fmt.Errorf("project_id %d does not belong to group_path %s", filter.ProjectID, filter.GroupPath)
		}
	}

	// Build filter conditions
	conditions, args := r.buildFilterConditions(filter.MetricsFilter, 0)

	// Add date range filter for final_done_at
	argIdx := len(args)
	if filter.StartDate != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("final_done_at >= $%d::date", argIdx))
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("final_done_at < ($%d::date + INTERVAL '1 day')", argIdx))
		args = append(args, filter.EndDate)
	}

	// Build bucketed query
	bucketExpr := fmt.Sprintf("date_trunc('%s', final_done_at AT TIME ZONE $%d)", filter.Bucket, argIdx+1)
	args = append(args, filter.Timezone)

	// Base query for bucketed data
	query := fmt.Sprintf(`
		WITH bucketed AS (
			SELECT
				%s::date AS bucket_start,
				COUNT(*)::float8 AS throughput,
				AVG(lead_time_hours) FILTER (WHERE is_completed AND lead_time_hours IS NOT NULL) AS lead_avg_hours,
				CASE
					WHEN COUNT(*) FILTER (WHERE is_completed AND lead_time_hours IS NOT NULL) >= 2
					THEN PERCENTILE_CONT(0.85) WITHIN GROUP (ORDER BY lead_time_hours)
						 FILTER (WHERE is_completed AND lead_time_hours IS NOT NULL)
					ELSE NULL
				END AS lead_p85_hours,
				AVG(cycle_time_hours) FILTER (WHERE is_completed AND cycle_time_hours IS NOT NULL) AS cycle_avg_hours,
				CASE
					WHEN COUNT(*) FILTER (WHERE is_completed AND cycle_time_hours IS NOT NULL) >= 2
					THEN PERCENTILE_CONT(0.85) WITHIN GROUP (ORDER BY cycle_time_hours)
						 FILTER (WHERE is_completed AND cycle_time_hours IS NOT NULL)
					ELSE NULL
				END AS cycle_p85_hours
			FROM vw_issue_lifecycle_metrics
			WHERE is_completed = true
			%s
			GROUP BY 1
		)
		SELECT
			bucket_start,
			throughput,
			lead_avg_hours,
			lead_p85_hours,
			cycle_avg_hours,
			cycle_p85_hours
		FROM bucketed
		ORDER BY bucket_start`, bucketExpr,
		func() string {
			if len(conditions) > 0 {
				return " AND " + strings.Join(conditions, " AND ")
			}
			return ""
		}())

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query trend metrics: %w", err)
	}
	defer rows.Close()

	var items []domain.DeliveryTrendPoint
	var throughputs []float64
	var leadAvgs []float64
	var cycleAvgs []float64

	for rows.Next() {
		var item domain.DeliveryTrendPoint
		var bucketStart time.Time
		var throughput float64
		var leadAvgHours, leadP85Hours, cycleAvgHours, cycleP85Hours sql.NullFloat64

		if err := rows.Scan(&bucketStart, &throughput, &leadAvgHours, &leadP85Hours, &cycleAvgHours, &cycleP85Hours); err != nil {
			return nil, fmt.Errorf("failed to scan trend row: %w", err)
		}

		// Calculate bucket end
		var bucketEnd time.Time
		switch filter.Bucket {
		case "day":
			bucketEnd = bucketStart
		case "week":
			bucketEnd = bucketStart.AddDate(0, 0, 6)
		case "month":
			bucketEnd = bucketStart.AddDate(0, 1, -1)
		}

		item.BucketStart = bucketStart.Format("2006-01-02")
		item.BucketEnd = bucketEnd.Format("2006-01-02")
		item.Throughput.TotalIssuesDone = int(throughput)

		if leadAvgHours.Valid {
			avg := leadAvgHours.Float64 / 24.0
			item.SpeedMetricsDays.LeadTime.Avg = &avg
			throughputs = append(throughputs, throughput)
			leadAvgs = append(leadAvgs, avg)
		}
		if leadP85Hours.Valid {
			p85 := leadP85Hours.Float64 / 24.0
			item.SpeedMetricsDays.LeadTime.P85 = &p85
		}
		if cycleAvgHours.Valid {
			avg := cycleAvgHours.Float64 / 24.0
			item.SpeedMetricsDays.CycleTime.Avg = &avg
			cycleAvgs = append(cycleAvgs, avg)
		}
		if cycleP85Hours.Valid {
			p85 := cycleP85Hours.Float64 / 24.0
			item.SpeedMetricsDays.CycleTime.P85 = &p85
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trend rows: %w", err)
	}

	// Build response
	resp := &domain.DeliveryTrendResponse{
		Period: domain.Period{
			StartDate: filter.StartDate,
			EndDate:   filter.EndDate,
		},
		Bucket:   filter.Bucket,
		Timezone: filter.Timezone,
		Items:    items,
	}

	// Set filters_applied
	if filter.GroupPath != "" {
		resp.FiltersApplied.GroupPath = &filter.GroupPath
	}
	if filter.ProjectID > 0 {
		resp.FiltersApplied.ProjectID = &filter.ProjectID
	}
	if filter.Assignee != "" {
		resp.FiltersApplied.Assignee = &filter.Assignee
	}

	// Calculate correlation if we have enough data points
	if len(throughputs) >= 2 {
		corr := calculateCorrelation(throughputs, leadAvgs)
		resp.Correlation = &domain.DeliveryTrendCorrelation{
			ThroughputVsLeadTimeR: corr,
		}
		if len(cycleAvgs) >= 2 {
			corrCycle := calculateCorrelation(throughputs, cycleAvgs)
			resp.Correlation.ThroughputVsCycleTimeR = corrCycle
		}
	}

	return resp, nil
}

func calculateCorrelation(x, y []float64) *float64 {
	if len(x) != len(y) || len(x) < 2 {
		return nil
	}

	n := float64(len(x))
	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return nil
	}

	r := numerator / denominator
	return &r
}

// buildFilterConditions builds SQL conditions and args from filter
func (r *MetricsRepository) buildFilterConditions(filter domain.MetricsFilter, startIdx int) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := startIdx

	if filter.GroupPath != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("project_path LIKE $%d || '%%'", argIdx))
		args = append(args, filter.GroupPath)
	}

	if filter.ProjectID > 0 {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIdx))
		args = append(args, filter.ProjectID)
	}

	if filter.Assignee != "" {
		argIdx++
		conditions = append(conditions, assigneeContainsCondition(argIdx))
		args = append(args, filter.Assignee)
	}

	if filter.StartDate != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("final_done_at >= $%d::date", argIdx))
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("final_done_at < ($%d::date + INTERVAL '1 day')", argIdx))
		args = append(args, filter.EndDate)
	}

	return conditions, args
}

func (r *MetricsRepository) getThroughputMetrics(ctx context.Context, conditions []string, args []interface{}) (domain.Throughput, error) {
	query := `
		SELECT 
			COUNT(*) as total_issues_done,
			CASE 
				WHEN COUNT(*) > 0 AND (MAX(final_done_at) - MIN(final_done_at)) > INTERVAL '0 days'
				THEN ROUND(COUNT(*)::numeric / NULLIF(EXTRACT(EPOCH FROM (MAX(final_done_at) - MIN(final_done_at))) / 604800.0, 0), 2)
				ELSE 0
			END as avg_per_week
		FROM vw_issue_lifecycle_metrics
		WHERE is_completed = true
	`

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	metricsRepoLogger.Debug("executing throughput query",
		slog.String("query", query),
		slog.Int("arg_count", len(args)),
	)

	var throughput domain.Throughput
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&throughput.TotalIssuesDone,
		&throughput.AvgPerWeek,
	)
	if err != nil {
		metricsRepoLogger.Error("throughput query failed",
			slog.String("error", err.Error()),
			slog.String("query", query),
		)
	}

	return throughput, err
}

func (r *MetricsRepository) getSpeedMetrics(ctx context.Context, conditions []string, args []interface{}) (domain.SpeedMetrics, error) {
	query := `
		SELECT 
			AVG(lead_time_hours) FILTER (WHERE is_completed AND lead_time_hours IS NOT NULL) as lead_time_avg,
			PERCENTILE_CONT(0.85) WITHIN GROUP (ORDER BY lead_time_hours) 
				FILTER (WHERE is_completed AND lead_time_hours IS NOT NULL) as lead_time_p85,
			AVG(cycle_time_hours) FILTER (WHERE is_completed AND cycle_time_hours IS NOT NULL) as cycle_time_avg,
			PERCENTILE_CONT(0.85) WITHIN GROUP (ORDER BY cycle_time_hours) 
				FILTER (WHERE is_completed AND cycle_time_hours IS NOT NULL) as cycle_time_p85
		FROM vw_issue_lifecycle_metrics
		WHERE 1=1
	`

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	metricsRepoLogger.Debug("executing speed metrics query",
		slog.String("query", query),
		slog.Int("arg_count", len(args)),
	)

	var leadTimeAvg, leadTimeP85, cycleTimeAvg, cycleTimeP85 sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&leadTimeAvg,
		&leadTimeP85,
		&cycleTimeAvg,
		&cycleTimeP85,
	)
	if err != nil {
		metricsRepoLogger.Error("speed metrics query failed",
			slog.String("error", err.Error()),
			slog.String("query", query),
		)
		return domain.SpeedMetrics{}, err
	}

	speedMetrics := domain.SpeedMetrics{}
	if leadTimeAvg.Valid || leadTimeP85.Valid {
		speedMetrics.LeadTime = &domain.AvgP85Metric{}
		if leadTimeAvg.Valid {
			speedMetrics.LeadTime.Avg = leadTimeAvg.Float64 / 24.0 // Convert to days
		}
		if leadTimeP85.Valid {
			speedMetrics.LeadTime.P85 = leadTimeP85.Float64 / 24.0 // Convert to days
		}
	}
	if cycleTimeAvg.Valid || cycleTimeP85.Valid {
		speedMetrics.CycleTime = &domain.AvgP85Metric{}
		if cycleTimeAvg.Valid {
			speedMetrics.CycleTime.Avg = cycleTimeAvg.Float64 / 24.0 // Convert to days
		}
		if cycleTimeP85.Valid {
			speedMetrics.CycleTime.P85 = cycleTimeP85.Float64 / 24.0 // Convert to days
		}
	}

	return speedMetrics, nil
}

func (r *MetricsRepository) getAssigneeBreakdown(ctx context.Context, conditions []string, args []interface{}) ([]domain.AssigneeBreakdown, error) {
	query := `
		SELECT 
			a.assignee,
			COUNT(*) as issues_delivered,
			AVG(cycle_time_hours) as avg_cycle_time
		FROM vw_issue_lifecycle_metrics m
		CROSS JOIN LATERAL jsonb_array_elements_text(` + normalizedAssigneesJSONBExpr("m.assignees") + `) as a(assignee)
		WHERE is_completed = true
	`

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += ` 
		GROUP BY a.assignee
		ORDER BY issues_delivered DESC
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdown []domain.AssigneeBreakdown
	for rows.Next() {
		var b domain.AssigneeBreakdown
		var avgCycleTime sql.NullFloat64
		if err := rows.Scan(&b.Assignee, &b.IssuesDelivered, &avgCycleTime); err != nil {
			return nil, err
		}
		if avgCycleTime.Valid {
			b.AvgCycleTime = avgCycleTime.Float64 / 24.0 // Convert to days
		}
		breakdown = append(breakdown, b)
	}

	return breakdown, rows.Err()
}

func (r *MetricsRepository) getReworkMetrics(ctx context.Context, conditions []string, args []interface{}) (domain.ReworkMetrics, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE qa_to_dev_return_count > 0) as total_reworked_issues,
			AVG(qa_to_dev_return_count) as avg_rework_cycles,
			COUNT(*) as total_issues
		FROM vw_issue_lifecycle_metrics
		WHERE 1=1
	`

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	var rework domain.ReworkMetrics
	var totalReworked, totalIssues int
	var avgCycles sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&totalReworked,
		&avgCycles,
		&totalIssues,
	)
	if err != nil {
		return rework, err
	}

	rework.TotalReworkedIssues = totalReworked
	if avgCycles.Valid {
		rework.AvgReworkCyclesPerIssue = avgCycles.Float64
	}
	if totalIssues > 0 {
		rework.PingPongRatePct = float64(totalReworked) / float64(totalIssues) * 100.0
	}

	return rework, nil
}

func (r *MetricsRepository) getProcessHealthMetrics(ctx context.Context, conditions []string, args []interface{}) (domain.ProcessHealthMetrics, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE skipped_in_progress_flag = true) as bypass_count,
			COUNT(*) FILTER (WHERE is_completed = true AND skipped_in_progress_flag = false) as first_time_pass_count,
			COUNT(*) FILTER (WHERE is_completed = true) as total_completed
		FROM vw_issue_lifecycle_metrics
		WHERE 1=1
	`

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	var bypassCount, firstTimePassCount, totalCompleted int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&bypassCount,
		&firstTimePassCount,
		&totalCompleted,
	)
	if err != nil {
		return domain.ProcessHealthMetrics{}, err
	}

	var health domain.ProcessHealthMetrics
	if totalCompleted > 0 {
		health.BypassRatePct = float64(bypassCount) / float64(totalCompleted) * 100.0
		health.FirstTimePassRatePct = float64(firstTimePassCount) / float64(totalCompleted) * 100.0
	}

	return health, nil
}

func (r *MetricsRepository) getBottleneckMetrics(ctx context.Context, conditions []string, args []interface{}) (domain.BottleneckMetrics, error) {
	query := `
		SELECT 
			COALESCE(SUM(blocked_time_hours), 0) as total_blocked_hours,
			AVG(blocked_time_hours) FILTER (WHERE blocked_time_hours > 0) as avg_blocked_per_issue
		FROM vw_issue_lifecycle_metrics
		WHERE 1=1
	`

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	var metrics domain.BottleneckMetrics
	var avgBlocked sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&metrics.TotalBlockedTimeHours,
		&avgBlocked,
	)
	if err != nil {
		return metrics, err
	}

	if avgBlocked.Valid {
		metrics.AvgBlockedTimePerIssueHours = avgBlocked.Float64
	}

	return metrics, nil
}

func (r *MetricsRepository) getDefectMetrics(ctx context.Context, conditions []string, args []interface{}) (domain.DefectMetrics, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE metadata_labels IS NOT NULL AND LOWER(metadata_labels::text) LIKE '%bug%') as bug_count,
			COUNT(*) as total_issues
		FROM vw_issue_lifecycle_metrics
		WHERE 1=1
	`

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	var bugCount, totalIssues int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&bugCount, &totalIssues)
	if err != nil {
		return domain.DefectMetrics{}, err
	}

	var defects domain.DefectMetrics
	if totalIssues > 0 {
		defects.BugRatioPct = float64(bugCount) / float64(totalIssues) * 100.0
	}

	return defects, nil
}

func (r *MetricsRepository) getCurrentWIP(ctx context.Context, conditions []string, args []interface{}) (domain.CurrentWIP, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE current_canonical_state = 'IN_PROGRESS') as in_progress,
			COUNT(*) FILTER (WHERE current_canonical_state = 'QA_REVIEW') as qa_review,
			COUNT(*) FILTER (WHERE current_canonical_state = 'BLOCKED') as blocked
		FROM vw_issue_lifecycle_metrics
		WHERE current_canonical_state IN ('IN_PROGRESS', 'QA_REVIEW', 'BLOCKED')
	`

	// Add additional conditions but exclude date range for WIP
	wipConditions := r.excludeDateConditions(conditions)
	if len(wipConditions) > 0 {
		// Build args without date range params
		_, wipArgs := r.buildWIPFilterConditions(args, conditions)
		query += " AND " + strings.Join(wipConditions, " AND ")
		return r.queryCurrentWIP(ctx, query, wipArgs)
	}

	return r.queryCurrentWIP(ctx, query, args)
}

func (r *MetricsRepository) queryCurrentWIP(ctx context.Context, query string, args []interface{}) (domain.CurrentWIP, error) {
	var wip domain.CurrentWIP
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&wip.InProgress,
		&wip.QAReview,
		&wip.Blocked,
	)
	return wip, err
}

func (r *MetricsRepository) getAgingWIP(ctx context.Context, conditions []string, args []interface{}) ([]domain.AgingIssue, error) {
	query := `
		SELECT 
			issue_id,
			gitlab_issue_id,
			issue_iid,
			project_id,
			project_path,
			issue_title,
			COALESCE(ARRAY(SELECT jsonb_array_elements_text(` + normalizedAssigneesJSONBExpr("assignees") + `)), ARRAY[]::text[]) as assignees,
			current_canonical_state,
			COALESCE(EXTRACT(DAY FROM (NOW() - first_in_progress_at)), 0)::int as days_in_state
		FROM vw_issue_lifecycle_metrics
		WHERE current_canonical_state IN ('IN_PROGRESS', 'QA_REVIEW', 'BLOCKED')
	`

	// Add additional conditions but exclude date range for WIP
	wipConditions := r.excludeDateConditions(conditions)
	if len(wipConditions) > 0 {
		// Build args without date range params
		_, wipArgs := r.buildWIPFilterConditions(args, conditions)
		query += " AND " + strings.Join(wipConditions, " AND ")
		return r.queryAgingWIP(ctx, query, wipArgs)
	}

	return r.queryAgingWIP(ctx, query, args)
}

func (r *MetricsRepository) queryAgingWIP(ctx context.Context, query string, args []interface{}) ([]domain.AgingIssue, error) {
	query += " ORDER BY days_in_state DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agingIssues []domain.AgingIssue
	for rows.Next() {
		var issue domain.AgingIssue
		var assignees []string
		if err := rows.Scan(
			&issue.IssueID,
			&issue.GitlabIssueID,
			&issue.IssueIID,
			&issue.ProjectID,
			&issue.ProjectPath,
			&issue.Title,
			pq.Array(&assignees),
			&issue.CurrentState,
			&issue.DaysInState,
		); err != nil {
			return nil, err
		}
		issue.Assignees = assignees
		// Flag issues older than 7 days
		issue.WarningFlag = issue.DaysInState > 7
		agingIssues = append(agingIssues, issue)
	}

	return agingIssues, rows.Err()
}

// excludeDateConditions removes date-related conditions from the list
func (r *MetricsRepository) excludeDateConditions(conditions []string) []string {
	var filtered []string
	for _, c := range conditions {
		if !strings.Contains(c, "final_done_at") {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// buildWIPFilterConditions builds args excluding date range params
func (r *MetricsRepository) buildWIPFilterConditions(originalArgs []interface{}, conditions []string) ([]string, []interface{}) {
	var filteredConditions []string
	var filteredArgs []interface{}

	for i, c := range conditions {
		if !strings.Contains(c, "final_done_at") && i < len(originalArgs) {
			filteredConditions = append(filteredConditions, c)
			filteredArgs = append(filteredArgs, originalArgs[i])
		}
	}

	return filteredConditions, filteredArgs
}

// timeToDays converts hours to days, handling NULL values
func timeToDays(hours sql.NullFloat64) float64 {
	if hours.Valid {
		return hours.Float64 / 24.0
	}
	return 0
}
