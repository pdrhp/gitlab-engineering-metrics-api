-- Migration: 000013_create_golden_engineering_views
-- Gold Layer: Views analiticas para metricas de engenharia

DROP VIEW IF EXISTS vw_projects_catalog;
DROP VIEW IF EXISTS vw_project_engineering_metrics;
DROP VIEW IF EXISTS vw_issue_lifecycle_metrics;
DROP VIEW IF EXISTS vw_issue_state_intervals;
DROP VIEW IF EXISTS vw_issue_state_transitions;

UPDATE projects p
SET last_synced_at = rp.last_synced_at,
    updated_at = NOW()
FROM raw_projects rp
WHERE rp.id = p.id
  AND rp.last_synced_at > p.last_synced_at;

CREATE VIEW vw_issue_state_transitions AS
WITH ordered_events AS (
    SELECT
        ie.id AS issue_event_id,
        ie.gitlab_event_id,
        ie.issue_id,
        ie.project_id,
        p.path AS project_path,
        i.iid AS issue_iid,
        i.gitlab_issue_id,
        i.title AS issue_title,
        i.gitlab_created_at,
        i.current_canonical_state,
        ie.author_name,
        ie.raw_label_added,
        ie.raw_label_removed,
        ie.mapped_canonical_state AS canonical_state,
        ie.event_timestamp AS entered_at,
        ie.cycle_count,
        LAG(ie.mapped_canonical_state) OVER (
            PARTITION BY ie.issue_id
            ORDER BY ie.event_timestamp, ie.id
        ) AS previous_raw_state
    FROM issue_events ie
    JOIN issues i ON i.id = ie.issue_id
    JOIN projects p ON p.id = ie.project_id
    WHERE ie.is_noise = FALSE
),
deduplicated_events AS (
    SELECT *
    FROM ordered_events
    WHERE previous_raw_state IS DISTINCT FROM canonical_state
       OR previous_raw_state IS NULL
)
SELECT
    issue_event_id,
    gitlab_event_id,
    issue_id,
    project_id,
    project_path,
    issue_iid,
    gitlab_issue_id,
    issue_title,
    gitlab_created_at,
    current_canonical_state,
    ROW_NUMBER() OVER (
        PARTITION BY issue_id
        ORDER BY entered_at, issue_event_id
    ) AS transition_seq,
    LAG(canonical_state) OVER (
        PARTITION BY issue_id
        ORDER BY entered_at, issue_event_id
    ) AS previous_canonical_state,
    canonical_state,
    LEAD(canonical_state) OVER (
        PARTITION BY issue_id
        ORDER BY entered_at, issue_event_id
    ) AS next_canonical_state,
    entered_at,
    LEAD(entered_at) OVER (
        PARTITION BY issue_id
        ORDER BY entered_at, issue_event_id
    ) AS exited_at,
    ROUND((EXTRACT(EPOCH FROM (
        LEAD(entered_at) OVER (
            PARTITION BY issue_id
            ORDER BY entered_at, issue_event_id
        ) - entered_at
    )) / 3600.0)::numeric, 2) AS duration_hours_to_next_state,
    author_name,
    raw_label_added,
    raw_label_removed,
    cycle_count
FROM deduplicated_events;

COMMENT ON VIEW vw_issue_state_transitions IS 'Timeline limpo por issue, removendo ruido e estados canonicos consecutivos duplicados.';

CREATE VIEW vw_issue_state_intervals AS
SELECT
    t.issue_event_id,
    t.gitlab_event_id,
    t.issue_id,
    t.project_id,
    t.project_path,
    t.issue_iid,
    t.gitlab_issue_id,
    t.issue_title,
    t.gitlab_created_at,
    t.current_canonical_state,
    t.transition_seq,
    t.previous_canonical_state,
    t.canonical_state,
    t.next_canonical_state,
    t.entered_at,
    t.exited_at,
    (t.exited_at IS NULL) AS is_open_interval,
    ROUND((EXTRACT(EPOCH FROM (COALESCE(t.exited_at, NOW()) - t.entered_at)) / 3600.0)::numeric, 2) AS duration_hours,
    t.author_name,
    t.raw_label_added,
    t.raw_label_removed,
    t.cycle_count
FROM vw_issue_state_transitions t;

COMMENT ON VIEW vw_issue_state_intervals IS 'Intervalos por estado canonico com entered_at, exited_at e duracao em horas.';

CREATE VIEW vw_issue_lifecycle_metrics AS
WITH last_transition AS (
    SELECT DISTINCT ON (t.issue_id)
        t.issue_id,
        t.canonical_state AS derived_current_canonical_state,
        t.entered_at AS last_transition_at
    FROM vw_issue_state_transitions t
    ORDER BY t.issue_id, t.entered_at DESC, t.issue_event_id DESC
),
lifecycle_points AS (
    SELECT
        i.id AS issue_id,
        i.project_id,
        p.path AS project_path,
        i.iid AS issue_iid,
        i.gitlab_issue_id,
        i.title AS issue_title,
        COALESCE(lt.derived_current_canonical_state, i.current_canonical_state) AS current_canonical_state,
        i.current_canonical_state AS cached_current_canonical_state,
        i.metadata_labels,
        i.assignees,
        i.gitlab_created_at,
        MIN(t.entered_at) FILTER (WHERE t.canonical_state = 'BACKLOG') AS first_backlog_at,
        MIN(t.entered_at) FILTER (WHERE t.canonical_state = 'IN_PROGRESS') AS first_in_progress_at,
        MIN(t.entered_at) FILTER (WHERE t.canonical_state = 'QA_REVIEW') AS first_qa_review_at,
        MIN(t.entered_at) FILTER (WHERE t.canonical_state = 'BLOCKED') AS first_blocked_at,
        MIN(t.entered_at) FILTER (WHERE t.canonical_state = 'DONE') AS first_done_at,
        MAX(t.entered_at) FILTER (WHERE t.canonical_state = 'DONE') AS last_done_at,
        COUNT(*) FILTER (WHERE t.canonical_state = 'IN_PROGRESS') AS in_progress_entry_count,
        COUNT(*) FILTER (WHERE t.canonical_state = 'QA_REVIEW') AS qa_review_entry_count,
        COUNT(*) FILTER (WHERE t.canonical_state = 'BLOCKED') AS blocked_entry_count,
        COUNT(*) FILTER (WHERE t.canonical_state = 'DONE') AS done_entry_count,
        COUNT(*) FILTER (
            WHERE t.previous_canonical_state = 'QA_REVIEW'
              AND t.canonical_state = 'IN_PROGRESS'
        ) AS qa_to_dev_return_count,
        COUNT(*) FILTER (
            WHERE t.previous_canonical_state = 'DONE'
              AND t.canonical_state <> 'DONE'
        ) AS reopened_after_done_count,
        MAX(t.cycle_count) AS max_cycle_count_recorded
    FROM issues i
    JOIN projects p ON p.id = i.project_id
    LEFT JOIN last_transition lt ON lt.issue_id = i.id
    LEFT JOIN vw_issue_state_transitions t ON t.issue_id = i.id
    GROUP BY
        i.id,
        i.project_id,
        p.path,
        i.iid,
        i.gitlab_issue_id,
        i.title,
        lt.derived_current_canonical_state,
        i.current_canonical_state,
        i.metadata_labels,
        i.assignees,
        i.gitlab_created_at
),
interval_rollup AS (
    SELECT
        lp.issue_id,
        ROUND(SUM(
            CASE
                WHEN si.canonical_state IN ('IN_PROGRESS', 'QA_REVIEW') THEN
                    GREATEST(
                        EXTRACT(EPOCH FROM (
                            LEAST(
                                COALESCE(
                                    si.exited_at,
                                    CASE
                                        WHEN lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL THEN lp.last_done_at
                                        ELSE NOW()
                                    END
                                ),
                                CASE
                                    WHEN lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL THEN lp.last_done_at
                                    ELSE NOW()
                                END
                            ) - si.entered_at
                        )) / 3600.0,
                        0
                    )
                ELSE 0
            END
        )::numeric, 2) AS cycle_time_hours,
        ROUND(SUM(
            CASE
                WHEN si.canonical_state = 'IN_PROGRESS' THEN
                    GREATEST(
                        EXTRACT(EPOCH FROM (
                            LEAST(
                                COALESCE(
                                    si.exited_at,
                                    CASE
                                        WHEN lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL THEN lp.last_done_at
                                        ELSE NOW()
                                    END
                                ),
                                CASE
                                    WHEN lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL THEN lp.last_done_at
                                    ELSE NOW()
                                END
                            ) - si.entered_at
                        )) / 3600.0,
                        0
                    )
                ELSE 0
            END
        )::numeric, 2) AS in_progress_time_hours,
        ROUND(SUM(
            CASE
                WHEN si.canonical_state = 'QA_REVIEW' THEN
                    GREATEST(
                        EXTRACT(EPOCH FROM (
                            LEAST(
                                COALESCE(
                                    si.exited_at,
                                    CASE
                                        WHEN lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL THEN lp.last_done_at
                                        ELSE NOW()
                                    END
                                ),
                                CASE
                                    WHEN lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL THEN lp.last_done_at
                                    ELSE NOW()
                                END
                            ) - si.entered_at
                        )) / 3600.0,
                        0
                    )
                ELSE 0
            END
        )::numeric, 2) AS qa_review_time_hours,
        ROUND(SUM(
            CASE
                WHEN si.canonical_state = 'BLOCKED' THEN
                    GREATEST(
                        EXTRACT(EPOCH FROM (
                            LEAST(
                                COALESCE(
                                    si.exited_at,
                                    CASE
                                        WHEN lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL THEN lp.last_done_at
                                        ELSE NOW()
                                    END
                                ),
                                CASE
                                    WHEN lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL THEN lp.last_done_at
                                    ELSE NOW()
                                END
                            ) - si.entered_at
                        )) / 3600.0,
                        0
                    )
                ELSE 0
            END
        )::numeric, 2) AS blocked_time_hours
    FROM lifecycle_points lp
    LEFT JOIN vw_issue_state_intervals si ON si.issue_id = lp.issue_id
    GROUP BY lp.issue_id
),
first_touch AS (
    SELECT
        lp.issue_id,
        MIN(t.entered_at) FILTER (
            WHERE t.canonical_state IN ('IN_PROGRESS', 'QA_REVIEW', 'BLOCKED', 'DONE')
        ) AS first_touch_at
    FROM lifecycle_points lp
    LEFT JOIN vw_issue_state_transitions t ON t.issue_id = lp.issue_id
    GROUP BY lp.issue_id
)
SELECT
    lp.issue_id,
    lp.project_id,
    lp.project_path,
    lp.issue_iid,
    lp.gitlab_issue_id,
    lp.issue_title,
    lp.current_canonical_state,
    lp.cached_current_canonical_state,
    lp.metadata_labels,
    lp.assignees,
    lp.gitlab_created_at,
    COALESCE(lp.first_backlog_at, lp.gitlab_created_at) AS lifecycle_start_at,
    CASE
        WHEN lp.first_backlog_at IS NOT NULL THEN 'BACKLOG_EVENT'
        ELSE 'ISSUE_CREATED_AT_FALLBACK'
    END AS lifecycle_start_source,
    lp.first_backlog_at,
    lp.first_in_progress_at,
    lp.first_qa_review_at,
    lp.first_blocked_at,
    ft.first_touch_at,
    lp.first_done_at,
    lp.last_done_at AS final_done_at,
    (lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL) AS is_completed,
    (lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL AND lp.in_progress_entry_count = 0) AS skipped_in_progress_flag,
    lp.in_progress_entry_count,
    lp.qa_review_entry_count,
    lp.blocked_entry_count,
    lp.done_entry_count,
    lp.qa_to_dev_return_count,
    lp.reopened_after_done_count,
    lp.max_cycle_count_recorded,
    ir.in_progress_time_hours,
    ir.qa_review_time_hours,
    ir.cycle_time_hours,
    ir.blocked_time_hours,
    ROUND((EXTRACT(EPOCH FROM (
        COALESCE(lp.last_done_at, NOW()) - COALESCE(lp.first_backlog_at, lp.gitlab_created_at)
    )) / 3600.0)::numeric, 2) AS elapsed_lead_time_hours,
    CASE
        WHEN lp.current_canonical_state = 'DONE' AND lp.last_done_at IS NOT NULL THEN
            ROUND((EXTRACT(EPOCH FROM (
                lp.last_done_at - COALESCE(lp.first_backlog_at, lp.gitlab_created_at)
            )) / 3600.0)::numeric, 2)
        ELSE NULL
    END AS lead_time_hours,
    CASE
        WHEN ft.first_touch_at IS NOT NULL THEN
            ROUND((EXTRACT(EPOCH FROM (
                ft.first_touch_at - COALESCE(lp.first_backlog_at, lp.gitlab_created_at)
            )) / 3600.0)::numeric, 2)
        ELSE NULL
    END AS backlog_wait_hours,
    CASE
        WHEN lp.current_canonical_state = 'DONE'
         AND lp.last_done_at IS NOT NULL
         AND (EXTRACT(EPOCH FROM (
                lp.last_done_at - COALESCE(lp.first_backlog_at, lp.gitlab_created_at)
             )) / 3600.0) > 0 THEN
            ROUND((100.0 * COALESCE(ir.cycle_time_hours, 0) / (
                EXTRACT(EPOCH FROM (
                    lp.last_done_at - COALESCE(lp.first_backlog_at, lp.gitlab_created_at)
                )) / 3600.0
            ))::numeric, 2)
        ELSE NULL
    END AS flow_efficiency_pct
FROM lifecycle_points lp
LEFT JOIN interval_rollup ir ON ir.issue_id = lp.issue_id
LEFT JOIN first_touch ft ON ft.issue_id = lp.issue_id;

COMMENT ON VIEW vw_issue_lifecycle_metrics IS 'Metricas por issue: lead time, cycle time, blocked time, rework, ghost work e timestamps de ciclo.';

CREATE VIEW vw_project_engineering_metrics AS
SELECT
    p.id AS project_id,
    p.path AS project_path,
    COUNT(m.issue_id) AS total_issues,
    COUNT(m.issue_id) FILTER (WHERE m.is_completed) AS completed_issues,
    COUNT(m.issue_id) FILTER (WHERE NOT m.is_completed) AS open_issues,
    COUNT(m.issue_id) FILTER (WHERE m.current_canonical_state = 'BACKLOG') AS backlog_issues,
    COUNT(m.issue_id) FILTER (WHERE m.current_canonical_state = 'IN_PROGRESS') AS in_progress_issues,
    COUNT(m.issue_id) FILTER (WHERE m.current_canonical_state = 'QA_REVIEW') AS qa_review_issues,
    COUNT(m.issue_id) FILTER (WHERE m.current_canonical_state = 'BLOCKED') AS blocked_issues,
    COUNT(m.issue_id) FILTER (WHERE m.current_canonical_state = 'CANCELED') AS canceled_issues,
    COUNT(m.issue_id) FILTER (WHERE m.is_completed AND m.final_done_at >= NOW() - INTERVAL '30 days') AS completed_last_30_days,
    ROUND((AVG(m.lead_time_hours) FILTER (WHERE m.is_completed))::numeric, 2) AS avg_lead_time_hours,
    ROUND((PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY m.lead_time_hours) FILTER (WHERE m.is_completed))::numeric, 2) AS p50_lead_time_hours,
    ROUND((AVG(m.cycle_time_hours) FILTER (WHERE m.is_completed))::numeric, 2) AS avg_cycle_time_hours,
    ROUND((PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY m.cycle_time_hours) FILTER (WHERE m.is_completed))::numeric, 2) AS p50_cycle_time_hours,
    ROUND((AVG(m.blocked_time_hours) FILTER (WHERE m.is_completed))::numeric, 2) AS avg_blocked_time_hours,
    ROUND((AVG(m.backlog_wait_hours) FILTER (WHERE m.is_completed))::numeric, 2) AS avg_backlog_wait_hours,
    ROUND((AVG(m.flow_efficiency_pct) FILTER (WHERE m.is_completed))::numeric, 2) AS avg_flow_efficiency_pct,
    COUNT(m.issue_id) FILTER (WHERE m.is_completed AND m.skipped_in_progress_flag) AS ghost_work_completed_issues,
    ROUND((100.0 * COUNT(m.issue_id) FILTER (WHERE m.is_completed AND m.skipped_in_progress_flag) / NULLIF(COUNT(m.issue_id) FILTER (WHERE m.is_completed), 0))::numeric, 2) AS ghost_work_pct,
    COUNT(m.issue_id) FILTER (WHERE m.qa_to_dev_return_count > 0) AS rework_issues,
    ROUND(AVG(m.qa_to_dev_return_count)::numeric, 2) AS avg_rework_count,
    ROUND((100.0 * COUNT(m.issue_id) FILTER (WHERE m.qa_to_dev_return_count > 0) / NULLIF(COUNT(m.issue_id), 0))::numeric, 2) AS rework_issue_pct,
    COUNT(m.issue_id) FILTER (WHERE m.blocked_time_hours > 0) AS blocked_issues_with_time,
    ROUND((100.0 * COUNT(m.issue_id) FILTER (WHERE m.blocked_time_hours > 0) / NULLIF(COUNT(m.issue_id), 0))::numeric, 2) AS blocked_issue_pct
FROM projects p
LEFT JOIN vw_issue_lifecycle_metrics m ON m.project_id = p.id
GROUP BY p.id, p.path;

COMMENT ON VIEW vw_project_engineering_metrics IS 'Resumo por projeto com throughput, tempos medios, ghost work, retrabalho e bloqueios.';

CREATE VIEW vw_projects_catalog AS
SELECT
    p.id,
    p.name,
    p.path,
    regexp_replace(p.path, '/[^/]+$', '') AS group_path,
    COALESCE(m.total_issues, 0) AS total_issues,
    GREATEST(p.last_synced_at, COALESCE(rp.last_synced_at, p.last_synced_at))::timestamptz AS last_synced_at,
    p.created_at,
    p.updated_at
FROM projects p
LEFT JOIN raw_projects rp ON rp.id = p.id
LEFT JOIN vw_project_engineering_metrics m ON m.project_id = p.id;

COMMENT ON VIEW vw_projects_catalog IS 'Catalogo de projetos para APIs e seletores, com volumetria de issues e ultimo sync consolidado.';
