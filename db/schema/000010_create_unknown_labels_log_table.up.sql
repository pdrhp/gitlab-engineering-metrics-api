-- Migration: 000010_create_unknown_labels_log_table
-- Config Layer: Log de labels não mapeadas para auditoria

CREATE TABLE IF NOT EXISTS unknown_labels_log (
    id                  SERIAL PRIMARY KEY,
    label_name          VARCHAR(255) NOT NULL UNIQUE,
    occurrence_count    INTEGER NOT NULL DEFAULT 1,
    first_seen_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_unknown_labels_count ON unknown_labels_log(occurrence_count DESC);
CREATE INDEX idx_unknown_labels_last_seen ON unknown_labels_log(last_seen_at);

COMMENT ON TABLE unknown_labels_log IS 'Config Layer: Log de labels encontradas no GitLab que não estão mapeadas em state_mapping ou metadata_mapping.';
COMMENT ON COLUMN unknown_labels_log.occurrence_count IS 'Número de vezes que esta label não mapeada foi encontrada.';
