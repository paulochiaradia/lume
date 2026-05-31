-- ============================================================
-- Lume Inteligência Comercial
-- init.sql — executado uma única vez na criação do banco
-- ============================================================

-- Extensões necessárias
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- Schema interno do sistema
-- ============================================================
CREATE SCHEMA IF NOT EXISTS lume_system;

-- ============================================================
-- Tabela de controle de migrations
-- Garante que cada migration rode apenas uma vez
-- ============================================================
CREATE TABLE IF NOT EXISTS lume_system.schema_migrations (
    version     VARCHAR(255) PRIMARY KEY,
    applied_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- Tabela de clientes
-- Cada cliente ativo no sistema tem um registro aqui
-- ============================================================
CREATE TABLE IF NOT EXISTS lume_system.clients (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_key      VARCHAR(100) UNIQUE NOT NULL, -- ex: loja_joao
    name            VARCHAR(255) NOT NULL,
    segment         VARCHAR(100) NOT NULL,        -- ex: construcao, farmacia
    erp_type        VARCHAR(100) NOT NULL,        -- ex: csv, bling, omie
    erp_config      JSONB NOT NULL DEFAULT '{}',  -- configurações do conector
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- Tabela de usuários
-- Usuários com acesso ao dashboard
-- ============================================================
CREATE TABLE IF NOT EXISTS lume_system.users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id       UUID NOT NULL REFERENCES lume_system.clients(id),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(50) NOT NULL DEFAULT 'viewer', -- admin, compras, gerente, viewer
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- Tabela de logs de ETL
-- Registra cada execução de pipeline — lida pelo Grafana
-- ============================================================
CREATE TABLE IF NOT EXISTS lume_system.etl_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id       UUID NOT NULL REFERENCES lume_system.clients(id),
    connector_type  VARCHAR(100) NOT NULL,
    status          VARCHAR(50) NOT NULL,  -- running, success, error
    records_read    INTEGER DEFAULT 0,
    records_written INTEGER DEFAULT 0,
    error_message   TEXT,
    started_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    finished_at     TIMESTAMP WITH TIME ZONE
);

-- ============================================================
-- Tabela de fila de jobs
-- Jobs de processamento Python são enfileirados aqui
-- ============================================================
CREATE TABLE IF NOT EXISTS lume_system.jobs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id       UUID NOT NULL REFERENCES lume_system.clients(id),
    job_type        VARCHAR(100) NOT NULL, -- abc_xyz, rfm, forecast, etc
    status          VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, done, error
    payload         JSONB NOT NULL DEFAULT '{}',
    error_message   TEXT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- Índices de sistema
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_etl_logs_client_id
    ON lume_system.etl_logs(client_id);

CREATE INDEX IF NOT EXISTS idx_etl_logs_started_at
    ON lume_system.etl_logs(started_at DESC);

CREATE INDEX IF NOT EXISTS idx_jobs_status
    ON lume_system.jobs(status)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_jobs_client_id
    ON lume_system.jobs(client_id);

-- ============================================================
-- Função para atualizar updated_at automaticamente
-- ============================================================
CREATE OR REPLACE FUNCTION lume_system.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers de updated_at
CREATE TRIGGER trg_clients_updated_at
    BEFORE UPDATE ON lume_system.clients
    FOR EACH ROW EXECUTE FUNCTION lume_system.update_updated_at();

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON lume_system.users
    FOR EACH ROW EXECUTE FUNCTION lume_system.update_updated_at();

CREATE TRIGGER trg_jobs_updated_at
    BEFORE UPDATE ON lume_system.jobs
    FOR EACH ROW EXECUTE FUNCTION lume_system.update_updated_at();