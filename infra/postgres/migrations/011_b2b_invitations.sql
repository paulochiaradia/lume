-- Tabela de Convites B2B (Fica no schema global junto com os usuários)
CREATE TABLE IF NOT EXISTS lume_system.invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES lume_system.clients(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    token VARCHAR(255) NOT NULL UNIQUE,
    invited_by UUID NOT NULL REFERENCES lume_system.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índices essenciais para velocidade e segurança
CREATE INDEX IF NOT EXISTS idx_invitations_token ON lume_system.invitations(token);
CREATE INDEX IF NOT EXISTS idx_invitations_client_id ON lume_system.invitations(client_id);
CREATE INDEX IF NOT EXISTS idx_invitations_email ON lume_system.invitations(email);

-- Garante que um e-mail só possa ter um convite pendente por vez para a MESMA empresa
-- A verificação de expiração do tempo (NOW) será feita no código backend Go
CREATE UNIQUE INDEX idx_unique_pending_invite 
ON lume_system.invitations (email, client_id) 
WHERE used = FALSE;