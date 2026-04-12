-- Migration: 000017_assignee_cycle_time
-- Purpose: Track cycle time per assignee during their actual assignment periods
-- This provides FAIR individual metrics (each assignee gets credit for their actual time)

-- DEPENDS ON: vw_issue_lifecycle_metrics (from migration 000016)
-- DEPENDS ON: vw_issue_state_transitions (from migration 000016)

-- Verify dependencies exist before proceeding
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_views WHERE viewname = 'vw_issue_lifecycle_metrics') THEN
        RAISE EXCEPTION 'Required view vw_issue_lifecycle_metrics does not exist. Run migration 000016 first.';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_views WHERE viewname = 'vw_issue_state_transitions') THEN
        RAISE EXCEPTION 'Required view vw_issue_state_transitions does not exist. Run migration 000016 first.';
    END IF;
END $$;

-- Indexes for JSONB operations on assignees (performance optimization)
CREATE INDEX IF NOT EXISTS idx_issues_assignees_history_gin
ON issues USING GIN (assignees)
WHERE assignees IS NOT NULL AND assignees != 'null'::jsonb;

-- Index for issue_events filtering by noise and timestamp
CREATE INDEX IF NOT EXISTS idx_issue_events_assignee_lookup
ON issue_events (issue_id, event_timestamp)
WHERE is_noise = FALSE;

-- Create assignee cycle time view
CREATE VIEW vw_assignee_cycle_time AS
WITH assignee_periods AS (
    -- Expand assignee history JSONB array into individual assignment periods
    -- Validates assigned_at field to prevent malformed data from inflating cycle times
    SELECT
        i.id AS issue_id,
        i.project_id,
        i.iid AS issue_iid,
        (ae->>'username') AS assignee_username,
        (ae->>'assigned_at')::timestamptz AS assigned_at,
        COALESCE(
            NULLIF(ae->>'unassigned_at', '')::timestamptz,
            (SELECT MIN(ie.event_timestamp) 
             FROM issue_events ie 
             WHERE ie.issue_id = i.id 
               AND ie.event_timestamp > (ae->>'assigned_at')::timestamptz
               AND ie.mapped_canonical_state = 'DONE'
            ),
            NOW()
        ) AS unassigned_at
    FROM issues i
    CROSS JOIN LATERAL jsonb_array_elements(
        COALESCE(i.assignees->'history', '[]'::jsonb)
    ) ae
    WHERE i.assignees IS NOT NULL 
      AND i.assignees != 'null'
      AND i.assignees->'history' IS NOT NULL
      -- Validate assigned_at to prevent malformed records
      AND ae->>'assigned_at' IS NOT NULL 
      AND ae->>'assigned_at' != ''
),
state_during_assignee AS (
    -- For each assignee period, find all state changes that occurred
    SELECT
        ap.issue_id,
        ap.issue_iid,
        ap.project_id,
        ap.assignee_username,
        ap.assigned_at,
        ap.unassigned_at,
        ie.mapped_canonical_state,
        ie.event_timestamp AS state_changed_at,
        LEAD(ie.event_timestamp, 1, ap.unassigned_at) OVER (
            PARTITION BY ap.issue_id, ap.assignee_username 
            ORDER BY ie.event_timestamp
        ) AS next_state_change
    FROM assignee_periods ap
    LEFT JOIN issue_events ie ON ie.issue_id = ap.issue_id
        AND ie.event_timestamp >= ap.assigned_at
        AND ie.event_timestamp < ap.unassigned_at
        AND ie.is_noise = FALSE
),
time_by_state AS (
    -- Calculate time spent in each state during assignee period
    SELECT
        issue_id,
        issue_iid,
        project_id,
        assignee_username,
        mapped_canonical_state,
        SUM(
            EXTRACT(EPOCH FROM (
                LEAST(next_state_change, unassigned_at) - GREATEST(state_changed_at, assigned_at)
            )) / 3600.0
        ) AS hours_in_state
    FROM state_during_assignee
    WHERE next_state_change IS NOT NULL OR unassigned_at IS NOT NULL
    GROUP BY issue_id, issue_iid, project_id, assignee_username, mapped_canonical_state
)
-- Aggregate per assignee per issue
SELECT
    issue_id,
    issue_iid,
    project_id,
    assignee_username,
    ROUND(SUM(hours_in_state) FILTER (WHERE mapped_canonical_state IN ('IN_PROGRESS', 'QA_REVIEW'))::numeric, 2) AS active_cycle_hours,
    ROUND(SUM(hours_in_state) FILTER (WHERE mapped_canonical_state = 'IN_PROGRESS')::numeric, 2) AS in_progress_hours,
    ROUND(SUM(hours_in_state) FILTER (WHERE mapped_canonical_state = 'QA_REVIEW')::numeric, 2) AS qa_review_hours,
    ROUND(SUM(hours_in_state) FILTER (WHERE mapped_canonical_state = 'BLOCKED')::numeric, 2) AS blocked_hours,
    ROUND(SUM(hours_in_state) FILTER (WHERE mapped_canonical_state = 'BACKLOG')::numeric, 2) AS backlog_hours,
    ROUND(SUM(hours_in_state)::numeric, 2) AS total_hours_as_assignee,
    CASE 
        WHEN SUM(hours_in_state) FILTER (WHERE mapped_canonical_state IN ('IN_PROGRESS', 'QA_REVIEW')) > 0 THEN TRUE 
        ELSE FALSE 
    END AS contributed_active_work
FROM time_by_state
GROUP BY issue_id, issue_iid, project_id, assignee_username;

COMMENT ON VIEW vw_assignee_cycle_time IS 'Tempo de cycle time atribuído a CADA assignee durante seus períodos de responsabilidade. Separa active work (IN_PROGRESS + QA_REVIEW) de wait time (BACKLOG, BLOCKED).';

-- Create individual performance metrics view
CREATE VIEW vw_individual_performance_metrics AS
SELECT
    assignee_username,
    project_id,
    COUNT(DISTINCT issue_id) AS issues_assigned,
    COUNT(DISTINCT issue_id) FILTER (WHERE contributed_active_work) AS issues_contributed,
    ROUND(SUM(active_cycle_hours)::numeric, 2) AS total_active_cycle_hours,
    ROUND(AVG(active_cycle_hours)::numeric, 2) AS avg_active_cycle_per_issue,
    ROUND(SUM(in_progress_hours)::numeric, 2) AS total_dev_hours,
    ROUND(SUM(qa_review_hours)::numeric, 2) AS total_qa_hours,
    ROUND(SUM(blocked_hours)::numeric, 2) AS total_blocked_hours,
    ROUND(SUM(backlog_hours)::numeric, 2) AS total_backlog_hours,
    ROUND((100.0 * SUM(active_cycle_hours) / NULLIF(SUM(total_hours_as_assignee), 0))::numeric, 2) AS active_work_pct,
    ROUND(SUM(total_hours_as_assignee)::numeric, 2) AS total_hours_as_assignee,
    COUNT(*) FILTER (WHERE active_cycle_hours > 100) AS high_cycle_time_issues,
    ROUND(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY active_cycle_hours)::numeric, 2) AS p50_active_cycle_hours,
    ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY active_cycle_hours)::numeric, 2) AS p95_active_cycle_hours
FROM vw_assignee_cycle_time
GROUP BY assignee_username, project_id;

COMMENT ON VIEW vw_individual_performance_metrics IS 'Métricas de performance individual por assignee: tempo ativo (trabalho real) vs tempo total (incluindo espera). Inclui percentis para identificar outliers.';
