-- ============================================================
-- Migration 001 — Tabelas de sistema
-- ============================================================

CREATE TABLE IF NOT EXISTS lume_system.schema_migrations (
    version     VARCHAR(255) PRIMARY KEY,
    applied_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lume_system.clients (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_key  VARCHAR(100) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    segment     VARCHAR(100) NOT NULL,
    erp_type    VARCHAR(100) NOT NULL,
    erp_config  JSONB NOT NULL DEFAULT '{}',
    active      BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lume_system.users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id     UUID NOT NULL REFERENCES lume_system.clients(id),
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(50) NOT NULL DEFAULT 'viewer',
    active        BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lume_system.etl_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id       UUID NOT NULL REFERENCES lume_system.clients(id),
    connector_type  VARCHAR(100) NOT NULL,
    status          VARCHAR(50) NOT NULL,
    records_read    INTEGER DEFAULT 0,
    records_written INTEGER DEFAULT 0,
    error_message   TEXT,
    started_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    finished_at     TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS lume_system.jobs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id     UUID NOT NULL REFERENCES lume_system.clients(id),
    job_type      VARCHAR(100) NOT NULL,
    status        VARCHAR(50) NOT NULL DEFAULT 'pending',
    payload       JSONB NOT NULL DEFAULT '{}',
    error_message TEXT,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);