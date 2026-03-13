-- Migration: 000004_create_projects_table
-- Silver Layer: Projetos normalizados (a partir de raw_projects)

CREATE TABLE IF NOT EXISTS projects (
    id                  INTEGER PRIMARY KEY,  -- ID do projeto no GitLab
    name                VARCHAR(255) NOT NULL,
    path                VARCHAR(255) NOT NULL,
    last_synced_at      TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_name ON projects(name);
CREATE INDEX idx_projects_path ON projects(path);

COMMENT ON TABLE projects IS 'Silver Layer: Projetos normalizados e estruturados, prontos para análise.';
