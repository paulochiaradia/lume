-- Adiciona colunas para controle de força bruta
ALTER TABLE lume_system.users 
ADD COLUMN failed_login_attempts INT DEFAULT 0,
ADD COLUMN locked_until TIMESTAMP WITH TIME ZONE;