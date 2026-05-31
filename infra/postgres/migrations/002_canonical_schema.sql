-- ============================================================
-- Migration 002 — Schema canônico por cliente
-- Esta função cria o schema isolado de cada cliente
-- Chamada pelo script new_client.sh
-- ============================================================

CREATE OR REPLACE FUNCTION lume_system.create_client_schema(p_client_key VARCHAR)
RETURNS void AS $$
DECLARE
    v_schema VARCHAR := 'client_' || p_client_key;
BEGIN
    -- Cria o schema isolado do cliente
    EXECUTE format('CREATE SCHEMA IF NOT EXISTS %I', v_schema);

    -- ── Tabela de vendas ──────────────────────────────────
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.vendas (
            id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            venda_key       VARCHAR(100) NOT NULL,  -- ID original no ERP
            data_venda      TIMESTAMP WITH TIME ZONE NOT NULL,
            cliente_id      UUID,
            vendedor_id     VARCHAR(100),
            total           NUMERIC(12,2) NOT NULL DEFAULT 0,
            desconto        NUMERIC(12,2) NOT NULL DEFAULT 0,
            status          VARCHAR(50) NOT NULL DEFAULT ''concluida'',
            canal           VARCHAR(50) DEFAULT ''balcao'',
            atributos       JSONB NOT NULL DEFAULT ''{}'',
            created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            UNIQUE(venda_key)
        )', v_schema);

    -- ── Tabela de itens de venda ──────────────────────────
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.itens_venda (
            id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            venda_id        UUID NOT NULL,
            produto_id      UUID,
            produto_key     VARCHAR(100) NOT NULL,  -- ID original no ERP
            descricao       VARCHAR(255) NOT NULL,
            quantidade      NUMERIC(12,3) NOT NULL,
            unidade         VARCHAR(20) DEFAULT ''UN'',
            preco_unitario  NUMERIC(12,2) NOT NULL,
            desconto        NUMERIC(12,2) NOT NULL DEFAULT 0,
            total           NUMERIC(12,2) NOT NULL,
            atributos       JSONB NOT NULL DEFAULT ''{}''
        )', v_schema);

    -- ── Tabela de produtos ────────────────────────────────
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.produtos (
            id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            produto_key     VARCHAR(100) NOT NULL,  -- ID original no ERP
            nome            VARCHAR(255) NOT NULL,
            categoria       VARCHAR(100),
            subcategoria    VARCHAR(100),
            unidade         VARCHAR(20) DEFAULT ''UN'',
            preco_custo     NUMERIC(12,2),
            preco_venda     NUMERIC(12,2),
            ativo           BOOLEAN NOT NULL DEFAULT true,
            atributos       JSONB NOT NULL DEFAULT ''{}'',
            created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            UNIQUE(produto_key)
        )', v_schema);

    -- ── Tabela de clientes da loja ────────────────────────
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.clientes (
            id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            cliente_key     VARCHAR(100) NOT NULL,  -- ID original no ERP
            nome            VARCHAR(255) NOT NULL,
            tipo            VARCHAR(50) DEFAULT ''pessoa_fisica'',
            documento       VARCHAR(20),
            telefone        VARCHAR(20),
            cidade          VARCHAR(100),
            bairro          VARCHAR(100),
            cep             VARCHAR(10),
            ativo           BOOLEAN NOT NULL DEFAULT true,
            atributos       JSONB NOT NULL DEFAULT ''{}'',
            created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            UNIQUE(cliente_key)
        )', v_schema);

    -- ── Tabela de estoque ─────────────────────────────────
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.estoque (
            id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            produto_id      UUID,
            produto_key     VARCHAR(100) NOT NULL,
            quantidade      NUMERIC(12,3) NOT NULL DEFAULT 0,
            quantidade_min  NUMERIC(12,3) NOT NULL DEFAULT 0,
            quantidade_max  NUMERIC(12,3),
            localizacao     VARCHAR(100),
            atualizado_em   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            UNIQUE(produto_key)
        )', v_schema);

    -- ── Tabela de movimentos de estoque ───────────────────
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.movimentos_estoque (
            id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            produto_key     VARCHAR(100) NOT NULL,
            tipo            VARCHAR(50) NOT NULL, -- entrada, saida, ajuste
            quantidade      NUMERIC(12,3) NOT NULL,
            motivo          VARCHAR(100),
            referencia      VARCHAR(100),         -- ID da venda ou compra
            atributos       JSONB NOT NULL DEFAULT ''{}'',
            created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        )', v_schema);

    -- ── Tabela de fornecedores ────────────────────────────
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.fornecedores (
            id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            fornecedor_key  VARCHAR(100) NOT NULL,
            nome            VARCHAR(255) NOT NULL,
            documento       VARCHAR(20),
            telefone        VARCHAR(20),
            ativo           BOOLEAN NOT NULL DEFAULT true,
            atributos       JSONB NOT NULL DEFAULT ''{}'',
            created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            UNIQUE(fornecedor_key)
        )', v_schema);

    RAISE NOTICE 'schema % criado com sucesso', v_schema;
END;
$$ LANGUAGE plpgsql;