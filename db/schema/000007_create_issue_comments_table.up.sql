-- Migration: 000007_create_issue_comments_table
-- Silver Layer: Comentários de issues estruturados

CREATE TABLE IF NOT EXISTS issue_comments (
    id                  BIGSERIAL PRIMARY KEY,
    gitlab_note_id      BIGINT NOT NULL UNIQUE,  -- ID do note no GitLab
    issue_id            INTEGER NOT NULL REFERENCES issues(id),
    author_name         VARCHAR(255),
    body                TEXT,
    comment_timestamp   TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_issue_comments_issue_id ON issue_comments(issue_id);
CREATE INDEX idx_issue_comments_gitlab_note_id ON issue_comments(gitlab_note_id);
CREATE INDEX idx_issue_comments_comment_timestamp ON issue_comments(comment_timestamp);

COMMENT ON TABLE issue_comments IS 'Silver Layer: Comentários de issues estruturados e normalizados.';
