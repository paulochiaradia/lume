-- ============================================================
-- Migration 003 — Índices das tabelas de sistema
-- Índices do schema canônico são criados pelo new_client.sh
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

CREATE INDEX IF NOT EXISTS idx_users_client_id
    ON lume_system.users(client_id);

-- Triggers de updated_at
CREATE OR REPLACE FUNCTION lume_system.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
    CREATE TRIGGER trg_clients_updated_at
        BEFORE UPDATE ON lume_system.clients
        FOR EACH ROW EXECUTE FUNCTION lume_system.update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_users_updated_at
        BEFORE UPDATE ON lume_system.users
        FOR EACH ROW EXECUTE FUNCTION lume_system.update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TRIGGER trg_jobs_updated_at
        BEFORE UPDATE ON lume_system.jobs
        FOR EACH ROW EXECUTE FUNCTION lume_system.update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;