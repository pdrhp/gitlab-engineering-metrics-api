-- Migration: 000005_create_issues_table
-- Silver Layer: Issues normalizadas e estruturadas

CREATE TABLE IF NOT EXISTS issues (
    id                      SERIAL PRIMARY KEY,
    gitlab_issue_id         INTEGER NOT NULL,  -- ID global no GitLab
    project_id              INTEGER NOT NULL REFERENCES projects(id),
    iid                     INTEGER NOT NULL,  -- Número visível (#215)
    title                   VARCHAR(500),
    current_canonical_state VARCHAR(50),  -- Cache do último estado mapeado
    metadata_labels         JSONB DEFAULT '[]',  -- Labels categóricas (Bug, Correção, etc.)
    assignees               JSONB DEFAULT '[]',  -- Array de assignees
    gitlab_created_at       TIMESTAMPTZ NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(project_id, iid)
);

CREATE INDEX idx_issues_project_id ON issues(project_id);
CREATE INDEX idx_issues_gitlab_issue_id ON issues(gitlab_issue_id);
CREATE INDEX idx_issues_current_state ON issues(current_canonical_state);
CREATE INDEX idx_issues_gitlab_created_at ON issues(gitlab_created_at);

-- Partial unique index: only enforces uniqueness for real GitLab issues (gitlab_issue_id > 0)
-- Allows multiple stub issues with gitlab_issue_id = 0
CREATE UNIQUE INDEX idx_issues_gitlab_issue_id_unique ON issues(gitlab_issue_id) WHERE gitlab_issue_id > 0;

COMMENT ON TABLE issues IS 'Silver Layer: Issues normalizadas com estados canônicos mapeados e metadados estruturados.';
COMMENT ON COLUMN issues.metadata_labels IS 'Array de labels categóricas (não-estado) como Bug, Correção, etc.';
COMMENT ON COLUMN issues.current_canonical_state IS 'Cache do último estado canônico processado (BACKLOG, IN_PROGRESS, etc.).';
