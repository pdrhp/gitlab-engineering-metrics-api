-- Migration: 000002_create_raw_events_table
-- Bronze Layer: Eventos brutos extraídos da API GitLab
-- Imutável, append-only, preserva payload completo em JSONB

CREATE TABLE IF NOT EXISTS raw_events (
    id                  BIGSERIAL PRIMARY KEY,
    gitlab_event_id     BIGINT,  -- ID do evento no GitLab (pode ser NULL para events antigos)
    project_id          INTEGER NOT NULL,
    issue_iid           INTEGER NOT NULL,
    event_type          VARCHAR(50) NOT NULL,  -- 'label_event', 'note', 'issue_update'
    raw_payload         JSONB NOT NULL,  -- Payload COMPLETO da API GitLab
    fetched_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed           BOOLEAN NOT NULL DEFAULT FALSE,  -- Flag para transformação Silver
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_raw_events_project_id ON raw_events(project_id);
CREATE INDEX idx_raw_events_issue_iid ON raw_events(project_id, issue_iid);
CREATE INDEX idx_raw_events_fetched_at ON raw_events(fetched_at);
CREATE INDEX idx_raw_events_processed ON raw_events(processed) WHERE processed = FALSE;
CREATE INDEX idx_raw_events_event_type ON raw_events(event_type);

COMMENT ON TABLE raw_events IS 'Bronze Layer: Eventos brutos do GitLab preservados exatamente como recebidos da API. Imutável e append-only.';
COMMENT ON COLUMN raw_events.raw_payload IS 'Payload completo da API GitLab (resource_label_events, notes, etc.) preservado em JSONB.';
COMMENT ON COLUMN raw_events.processed IS 'Flag indicando se o evento já foi transformado para a camada Silver.';
