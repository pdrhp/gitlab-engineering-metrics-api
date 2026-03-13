-- db/migrations/000011_create_dead_letter_queue.up.sql

CREATE TABLE IF NOT EXISTS dead_letter_queue (
    id BIGSERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL,
    issue_iid INTEGER,
    event_type VARCHAR(50),
    raw_payload JSONB,
    error_message TEXT NOT NULL,
    error_category VARCHAR(50) NOT NULL, -- 'api_error', 'parse_error', 'db_error', 'timeout'
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    last_retry_at TIMESTAMPTZ,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dlq_unresolved ON dead_letter_queue (resolved) WHERE resolved = FALSE;
CREATE INDEX idx_dlq_project ON dead_letter_queue (project_id);
CREATE INDEX idx_dlq_category ON dead_letter_queue (error_category);
