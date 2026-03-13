-- Migration: 000001_create_raw_projects_table
-- Bronze Layer: Cache de metadados de projetos do GitLab
-- Dados brutos, imutáveis, com payload JSONB completo

CREATE TABLE IF NOT EXISTS raw_projects (
    id                  INTEGER PRIMARY KEY,  -- ID do projeto no GitLab
    name                VARCHAR(255) NOT NULL,
    path                VARCHAR(255) NOT NULL,
    last_synced_at      TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z',
    raw_metadata        JSONB NOT NULL,  -- Payload completo da API GitLab
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_raw_projects_last_synced ON raw_projects(last_synced_at);

COMMENT ON TABLE raw_projects IS 'Bronze Layer: Cache de projetos do GitLab com metadados brutos preservados em JSONB.';
COMMENT ON COLUMN raw_projects.raw_metadata IS 'Payload completo retornado pela API GitLab para este projeto.';
