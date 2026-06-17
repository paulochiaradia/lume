-- Migration 005 — Tabela de cache de insights por cliente
-- Criada dentro do schema canônico de cada cliente via função
CREATE OR REPLACE FUNCTION lume_system.create_insights_cache(p_client_key VARCHAR)
RETURNS void AS $$
DECLARE
    v_schema VARCHAR := 'client_' || p_client_key;
BEGIN
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.insights_cache (
            id          SERIAL PRIMARY KEY,
            tipo        VARCHAR(100) NOT NULL,
            prioridade  INTEGER NOT NULL DEFAULT 99,
            titulo      VARCHAR(255) NOT NULL,
            mensagem    TEXT NOT NULL,
            acao        VARCHAR(100),
            href        VARCHAR(255),
            categoria   VARCHAR(100),
            icone       VARCHAR(50) DEFAULT ''success'',
            gerado_em   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        )
    ', v_schema);
    RAISE NOTICE 'insights_cache criado em %', v_schema;
END;
$$ LANGUAGE plpgsql;