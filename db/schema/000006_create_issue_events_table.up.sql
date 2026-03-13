-- Migration: 000006_create_issue_events_table
-- Silver Layer: Eventos de issues normalizados e mapeados

CREATE TABLE IF NOT EXISTS issue_events (
    id                      BIGSERIAL PRIMARY KEY,
    gitlab_event_id         BIGINT,  -- ID do evento no GitLab
    issue_id                INTEGER NOT NULL REFERENCES issues(id),
    project_id              INTEGER NOT NULL REFERENCES projects(id),
    issue_iid               INTEGER NOT NULL,
    author_name             VARCHAR(255),
    raw_label_added         VARCHAR(255),  -- Label adicionada (texto original)
    raw_label_removed       VARCHAR(255),  -- Label removida (texto original)
    mapped_canonical_state  VARCHAR(50) NOT NULL,  -- Estado canônico mapeado
    event_timestamp         TIMESTAMPTZ NOT NULL,
    is_noise                BOOLEAN NOT NULL DEFAULT FALSE,  -- Dedo nervoso
    cycle_count             INTEGER DEFAULT 0,  -- Contador de ciclos de retrabalho
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(gitlab_event_id, project_id)
);

CREATE INDEX idx_issue_events_issue_id ON issue_events(issue_id);
CREATE INDEX idx_issue_events_project_id ON issue_events(project_id);
CREATE INDEX idx_issue_events_issue_iid ON issue_events(project_id, issue_iid);
CREATE INDEX idx_issue_events_canonical_state ON issue_events(mapped_canonical_state);
CREATE INDEX idx_issue_events_event_timestamp ON issue_events(event_timestamp);
CREATE INDEX idx_issue_events_is_noise ON issue_events(is_noise) WHERE is_noise = FALSE;

COMMENT ON TABLE issue_events IS 'Silver Layer: Eventos de issues normalizados com estados canônicos mapeados. Dados limpos e estruturados.';
COMMENT ON COLUMN issue_events.mapped_canonical_state IS 'Estado canônico mapeado (BACKLOG, IN_PROGRESS, QA_REVIEW, BLOCKED, DONE, CANCELED, UNKNOWN).';
COMMENT ON COLUMN issue_events.is_noise IS 'Flag indicando evento de "dedo nervoso" (transição < 15min) para ser ignorado em métricas.';
COMMENT ON COLUMN issue_events.cycle_count IS 'Número de ciclos de retrabalho (transições QA_REVIEW → IN_PROGRESS).';
