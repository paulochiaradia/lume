-- ============================================================
-- Migration 007 — Tabela de Cache de Reposição de Estoque
-- Percorre todos os schemas de clientes existentes para criar a tabela
-- ============================================================

DO $$
DECLARE
    schema_record RECORD;
BEGIN
    FOR schema_record IN
        SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'client_%'
    LOOP
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.estoque_reposicao_cache (
                id VARCHAR(50) PRIMARY KEY,
                nome VARCHAR(255) NOT NULL,
                categoria VARCHAR(100) NOT NULL,
                classe_abc CHAR(1) NOT NULL,
                estoque_atual DOUBLE PRECISION NOT NULL,
                demanda_prevista DOUBLE PRECISION NOT NULL,
                dias_ate_ruptura INT NOT NULL,
                quantidade_sugerida DOUBLE PRECISION NOT NULL,
                urgencia INT NOT NULL,
                atualizado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
            );

            CREATE INDEX IF NOT EXISTS idx_estoque_reposicao_urgencia_%s 
            ON %I.estoque_reposicao_cache (urgencia, dias_ate_ruptura);
        ', schema_record.schema_name, schema_record.schema_name, schema_record.schema_name);
    END LOOP;
END;
$$ LANGUAGE plpgsql;