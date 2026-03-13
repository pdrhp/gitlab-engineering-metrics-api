-- Migration: 000009_create_metadata_mapping_table
-- Config Layer: Mapeamento de labels para metadados categóricos

CREATE TABLE IF NOT EXISTS metadata_mapping (
    id                  SERIAL PRIMARY KEY,
    gitlab_label_name   VARCHAR(255) NOT NULL UNIQUE,
    metadata_key        VARCHAR(100) NOT NULL,  -- tipo, prioridade, categoria
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_metadata_mapping_label ON metadata_mapping(gitlab_label_name);
CREATE INDEX idx_metadata_mapping_key ON metadata_mapping(metadata_key);

COMMENT ON TABLE metadata_mapping IS 'Config Layer: Mapeamento de labels que representam metadados (não-estados) como tipo, prioridade, etc.';
COMMENT ON COLUMN metadata_mapping.metadata_key IS 'Chave categórica: tipo, prioridade, severidade, etc.';
