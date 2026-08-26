package db

import (
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	ID                  string
	ClientID            string
	Email               string
	PasswordHash        string
	Role                string
	Name                string
	Active              bool
	FailedLoginAttempts int
	LockedUntil         *time.Time
}

// GetUserByEmailAndClient busca um usuário ativo pelo e-mail dentro de um cliente específico.
func GetUserByEmailAndClient(db *sql.DB, email, clientID string) (*User, error) {
	var u User
	err := db.QueryRow(`
		SELECT id, client_id, email, password_hash, role, name, active, failed_login_attempts, locked_until
		FROM lume_system.users
		WHERE email = $1 AND client_id = $2 AND active = true
	`, email, clientID).Scan(&u.ID, &u.ClientID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.Active, &u.FailedLoginAttempts, &u.LockedUntil)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("usuário não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	return &u, nil
}

type Session struct {
	UserID       string
	RefreshToken string
	ExpiresAt    time.Time
}

type UserProfile struct {
	ID        string `json:"id"`
	Nome      string `json:"nome"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ClientKey string `json:"client_key"`
	LojaNome  string `json:"loja_nome"`
}

type ActiveSession struct {
	DeviceID  string    `json:"device_id"`
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetUserByEmail busca os dados do usuário usando o email (necessário para o Login)
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	var u User
	err := db.QueryRow(`
		SELECT id, client_id, email, password_hash, role, name, active, failed_login_attempts, locked_until
		FROM lume_system.users
		WHERE email = $1 AND active = true
	`, email).Scan(&u.ID, &u.ClientID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.Active, &u.FailedLoginAttempts, &u.LockedUntil)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("usuário não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	return &u, nil
}

// CreateUser insere um novo usuário ativo no tenant informado.
func CreateUser(db *sql.DB, clientID, email, passwordHash, role, name string) error {
	query := `
		INSERT INTO lume_system.users (client_id, email, password_hash, role, name, active)
		VALUES ($1, $2, $3, $4, $5, true)
	`
	_, err := db.Exec(query, clientID, email, passwordHash, role, name)
	return err
}

// GetUserByID busca os dados do usuário usando o ID (necessário para o Refresh)
func GetUserByID(db *sql.DB, id string) (*User, error) {
	var user User
	query := `SELECT id, client_id, password_hash, role, name FROM lume_system.users WHERE id = $1`
	err := db.QueryRow(query, id).Scan(&user.ID, &user.ClientID, &user.PasswordHash, &user.Role, &user.Name)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserProfile busca os dados seguros do usuário e do tenant (loja) logado
func GetUserProfile(db *sql.DB, userID string) (UserProfile, error) {
	var profile UserProfile

	query := `
		SELECT 
			u.id, 
			u.name, 
			u.email, 
			u.role, 
			c.client_key, 
			c.name AS loja_nome
		FROM lume_system.users u
		JOIN lume_system.clients c ON c.id = u.client_id
		WHERE u.id = $1 AND u.active = true AND c.active = true
	`

	err := db.QueryRow(query, userID).Scan(
		&profile.ID,
		&profile.Nome,
		&profile.Email,
		&profile.Role,
		&profile.ClientKey,
		&profile.LojaNome,
	)

	if err != nil {
		return profile, fmt.Errorf("erro ao buscar perfil do usuário: %w", err)
	}

	return profile, nil
}

// CreateAuditLog registra as ações de segurança (sucesso ou falha de login)
func CreateAuditLog(db *sql.DB, userID, tenantID, action, ip, userAgent string) error {
	// Se a tentativa de login for de um email que não existe, o userID pode vir vazio
	// Tratamos isso para gravar NULL no banco em vez de quebrar a query
	var uid interface{} = userID
	if userID == "" {
		uid = nil
	}

	var tid interface{} = tenantID
	if tenantID == "" {
		tid = nil
	}

	query := `
		INSERT INTO lume_system.audit_logs (user_id, tenant_id, action, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(query, uid, tid, action, ip, userAgent)
	return err
}

// CreateSession insere um novo refresh token atrelado ao usuário no banco de dados
func CreateSession(db *sql.DB, userID, refreshToken, deviceID, userAgent, ip string, expiresAt time.Time) error {
	query := `
		INSERT INTO lume_system.sessions (user_id, refresh_token, device_id, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(query, userID, refreshToken, deviceID, userAgent, ip, expiresAt)
	return err
}

// GetSessionByToken busca uma sessão ativa pelo Refresh Token
func GetSessionByToken(db *sql.DB, refreshToken string) (*Session, error) {
	var session Session
	query := `
		SELECT user_id, refresh_token, expires_at 
		FROM lume_system.sessions 
		WHERE refresh_token = $1
	`
	err := db.QueryRow(query, refreshToken).Scan(&session.UserID, &session.RefreshToken, &session.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession remove a sessão do banco (Logout) com verificação estrita
func DeleteSession(db *sql.DB, refreshToken string) error {
	query := `DELETE FROM lume_system.sessions WHERE refresh_token = $1`

	// Executa o comando
	res, err := db.Exec(query, refreshToken)
	if err != nil {
		return err // Erro de conexão ou sintaxe
	}

	// Verifica se a sessão realmente existia e foi apagada
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("nenhuma sessão encontrada com este token")
	}

	return nil
}

// GetActiveSessions lista todas as sessões válidas de um usuário
func GetActiveSessions(db *sql.DB, userID string) ([]ActiveSession, error) {
	query := `
		SELECT device_id, ip_address, created_at, expires_at 
		FROM lume_system.sessions 
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ActiveSession
	for rows.Next() {
		var s ActiveSession
		if err := rows.Scan(&s.DeviceID, &s.IPAddress, &s.CreatedAt, &s.ExpiresAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

// DeleteSessionByDevice encerra uma sessão específica (Revogação)
func DeleteSessionByDevice(db *sql.DB, userID, deviceID string) error {
	query := `DELETE FROM lume_system.sessions WHERE user_id = $1 AND device_id = $2`
	res, err := db.Exec(query, userID, deviceID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("sessão não encontrada ou já encerrada")
	}
	return nil
}

// DeleteAllSessions encerra todas as sessões de um usuário (Logout Global)
func DeleteAllSessions(db *sql.DB, userID string) error {
	query := `DELETE FROM lume_system.sessions WHERE user_id = $1`
	_, err := db.Exec(query, userID)
	return err
}

// DeleteOtherSessions encerra todas as sessões do usuário, exceto a atual
func DeleteOtherSessions(db *sql.DB, userID string, currentRefreshToken string) error {
	// Deleta todas as linhas do usuário, exceto aquela que possui o token protegido
	query := `DELETE FROM lume_system.sessions WHERE user_id = $1 AND refresh_token != $2`
	_, err := db.Exec(query, userID, currentRefreshToken)
	return err
}

// IncrementFailedLogin soma 1 nas falhas e retorna o novo total
func IncrementFailedLogin(db *sql.DB, userID string) (int, error) {
	var attempts int
	query := `
		UPDATE lume_system.users 
		SET failed_login_attempts = failed_login_attempts + 1 
		WHERE id = $1 
		RETURNING failed_login_attempts
	`
	err := db.QueryRow(query, userID).Scan(&attempts)
	return attempts, err
}

// LockUserAccount bloqueia o acesso inserindo um prazo futuro
func LockUserAccount(db *sql.DB, userID string) error {
	// Trava por 15 minutos
	query := `
		UPDATE lume_system.users 
		SET locked_until = NOW() + INTERVAL '15 minutes' 
		WHERE id = $1
	`
	_, err := db.Exec(query, userID)
	return err
}

// ResetFailedLogin limpa a ficha do usuário quando ele acerta a senha
func ResetFailedLogin(db *sql.DB, userID string) error {
	query := `
		UPDATE lume_system.users 
		SET failed_login_attempts = 0, locked_until = NULL 
		WHERE id = $1 AND (failed_login_attempts > 0 OR locked_until IS NOT NULL)
	`
	_, err := db.Exec(query, userID)
	return err
}

// GetPasswordHashByID busca apenas o hash da senha atual do usuário
func GetPasswordHashByID(db *sql.DB, userID string) (string, error) {
	var hash string
	query := `SELECT password_hash FROM lume_system.users WHERE id = $1`
	err := db.QueryRow(query, userID).Scan(&hash)
	return hash, err
}

// UpdatePassword atualiza a senha do usuário no banco
func UpdatePassword(db *sql.DB, userID string, newHash string) error {
	query := `UPDATE lume_system.users SET password_hash = $1 WHERE id = $2`
	_, err := db.Exec(query, newHash, userID)
	return err
}

// CreatePasswordResetToken salva o token gerado associado ao usuário com validade de 1 hora
func CreatePasswordResetToken(db *sql.DB, userID string, token string) error {
	query := `
		INSERT INTO lume_system.password_resets (user_id, token, expires_at)
		VALUES ($1, $2, NOW() + INTERVAL '1 hour')
	`
	_, err := db.Exec(query, userID, token)
	return err
}

// GetValidPasswordResetToken busca o token. Só retorna o userID se o token existir, não tiver expirado e não tiver sido usado.
func GetValidPasswordResetToken(db *sql.DB, token string) (string, error) {
	var userID string
	query := `
		SELECT user_id 
		FROM lume_system.password_resets 
		WHERE token = $1 AND used = FALSE AND expires_at > NOW()
	`
	err := db.QueryRow(query, token).Scan(&userID)
	return userID, err
}

// MarkTokenAsUsed "queima" o token para que ele não possa ser reutilizado pelo mesmo link
func MarkTokenAsUsed(db *sql.DB, token string) error {
	query := `UPDATE lume_system.password_resets SET used = TRUE WHERE token = $1`
	_, err := db.Exec(query, token)
	return err
}
