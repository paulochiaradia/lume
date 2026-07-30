-- ============================================================
-- Migration 008 (UP) — Limpeza de Débito Técnico e Tabelas de Autenticação
-- ============================================================

-- 1. Limpa a tabela cache fantasma criada indevidamente no schema public por versões anteriores
DROP TABLE IF EXISTS public.estoque_reposicao_cache;

-- 2. Criação da tabela de Sessões para o Refresh Token (Fim dos JWTs eternos)
CREATE TABLE IF NOT EXISTS lume_system.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    refresh_token VARCHAR(512) NOT NULL UNIQUE,
    device_id VARCHAR(255),
    user_agent TEXT,
    ip_address VARCHAR(45),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    -- Relacionamento forte com a tabela de usuários existente
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES lume_system.users (id) ON DELETE CASCADE
);

-- Índice obrigatório pela auditoria para buscar rapidamente as sessões de um usuário
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON lume_system.sessions(user_id);

-- 3. Criação da tabela de Auditoria (Security Logs)
CREATE TABLE IF NOT EXISTS lume_system.audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID, -- Pode ser nulo se a falha for de um "usuário fantasma" tentando logar
    tenant_id UUID,
    action VARCHAR(100) NOT NULL, -- Ex: 'login_success', 'logout_global', 'password_reset'
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Índices para facilitar a busca rápida de logs por tenant (loja) ou usuário específico
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON lume_system.audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON lume_system.audit_logs(user_id);