-- Migration: 000003_create_sync_state_table
-- Controle de cursor de sincronização por projeto

CREATE TABLE IF NOT EXISTS sync_state (
    id              SERIAL PRIMARY KEY,
    project_id      INTEGER NOT NULL UNIQUE REFERENCES raw_projects(id),
    last_synced_at  TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sync_state_project_id ON sync_state(project_id);
CREATE INDEX idx_sync_state_last_synced ON sync_state(last_synced_at);

COMMENT ON TABLE sync_state IS 'Controla o cursor de sincronização incremental (updated_after) para cada projeto.';
