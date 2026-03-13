-- Migration: 000012_create_raw_issues_table
-- Purpose: Store raw issue metadata from GitLab for title/description extraction

CREATE TABLE IF NOT EXISTS raw_issues (
    id BIGSERIAL PRIMARY KEY,
    gitlab_issue_id BIGINT NOT NULL,
    project_id INTEGER NOT NULL,
    iid INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    state TEXT NOT NULL, -- 'opened', 'closed'
    raw_payload JSONB NOT NULL, -- Full issue payload from GitLab API
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, iid)
);

CREATE INDEX idx_raw_issues_project_id ON raw_issues(project_id);
CREATE INDEX idx_raw_issues_gitlab_id ON raw_issues(gitlab_issue_id);

COMMENT ON TABLE raw_issues IS 'Bronze Layer: Raw issue metadata from GitLab API';
COMMENT ON COLUMN raw_issues.raw_payload IS 'Complete issue JSON payload from GitLab API';
