-- Migration: 000008_create_state_mapping_table
-- Config Layer: Mapeamento de labels para estados canônicos

CREATE TABLE IF NOT EXISTS state_mapping (
    id                  SERIAL PRIMARY KEY,
    gitlab_label_name   VARCHAR(255) NOT NULL UNIQUE,
    canonical_state     VARCHAR(50) NOT NULL,  -- BACKLOG, IN_PROGRESS, QA_REVIEW, BLOCKED, DONE, CANCELED, UNKNOWN
    description         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_canonical_state CHECK (canonical_state IN ('BACKLOG', 'IN_PROGRESS', 'QA_REVIEW', 'BLOCKED', 'DONE', 'CANCELED', 'UNKNOWN'))
);

CREATE INDEX idx_state_mapping_label ON state_mapping(gitlab_label_name);
CREATE INDEX idx_state_mapping_canonical ON state_mapping(canonical_state);

COMMENT ON TABLE state_mapping IS 'Config Layer: Mapeamento de labels do GitLab para estados canônicos.';
COMMENT ON COLUMN state_mapping.canonical_state IS 'Estado canônico: BACKLOG, IN_PROGRESS, QA_REVIEW, BLOCKED, DONE, CANCELED, UNKNOWN.';
