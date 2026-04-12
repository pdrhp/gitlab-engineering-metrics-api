-- Migration: 000014_optimize_ghost_work_queries
-- Gold/Silver optimization: selective indexes for ghost-work and label/assignee filters

CREATE INDEX IF NOT EXISTS idx_issues_assignees_gin
ON issues USING GIN (assignees);

CREATE INDEX IF NOT EXISTS idx_issues_metadata_labels_gin
ON issues USING GIN (metadata_labels);

CREATE INDEX IF NOT EXISTS idx_issues_ghost_work_completed
ON issues (current_canonical_state, gitlab_created_at)
WHERE current_canonical_state = 'DONE';

CREATE INDEX IF NOT EXISTS idx_issue_events_ghost_transitions
ON issue_events (issue_id, event_timestamp, mapped_canonical_state)
WHERE is_noise = FALSE
  AND mapped_canonical_state IN ('BACKLOG', 'DONE', 'QA_REVIEW');

CREATE INDEX IF NOT EXISTS idx_issue_events_project_ghost
ON issue_events (project_id, mapped_canonical_state)
WHERE is_noise = FALSE;
