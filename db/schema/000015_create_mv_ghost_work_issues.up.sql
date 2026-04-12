-- Migration: 000015_create_mv_ghost_work_issues
-- Optional materialized view for heavy workloads (Task 5 fallback)
-- Only apply if vw_issue_lifecycle_metrics query shows p95 > 2s

CREATE MATERIALIZED VIEW mv_ghost_work_issues AS
SELECT
    m.issue_id,
    m.project_id,
    m.project_path,
    m.issue_iid,
    m.gitlab_issue_id,
    m.issue_title,
    m.assignees,
    m.final_done_at,
    m.skipped_in_progress_flag,
    t.canonical_state AS from_state,
    t.next_canonical_state AS to_state,
    t.entered_at AS transition_time,
    t.duration_hours_to_next_state AS duration_hours
FROM vw_issue_lifecycle_metrics m
INNER JOIN vw_issue_state_transitions t ON t.issue_id = m.issue_id
WHERE m.skipped_in_progress_flag = true
  AND t.canonical_state = 'BACKLOG'
  AND t.next_canonical_state IN ('DONE', 'QA_REVIEW');

CREATE INDEX idx_mv_ghost_work_project ON mv_ghost_work_issues (project_id);
CREATE INDEX idx_mv_ghost_work_done_at ON mv_ghost_work_issues (final_done_at);
CREATE INDEX idx_mv_ghost_work_flag ON mv_ghost_work_issues (skipped_in_progress_flag);

COMMENT ON MATERIALIZED VIEW mv_ghost_work_issues IS 'Materialized view for ghost-work queries - pre-computed join of lifecycle metrics with state transitions for issues that skipped IN_PROGRESS.';
