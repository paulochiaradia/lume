package db

import (
	"database/sql"
	"fmt"
	"time"
)

// ── Structs ──────────────────────────────────────────────────

type Client struct {
	ID        string
	ClientKey string
	Name      string
	Segment   string
	ERPType   string
	ERPConfig []byte
	Active    bool
}

type ETLLog struct {
	ID             string
	ClientID       string
	ConnectorType  string
	Status         string
	RecordsRead    int
	RecordsWritten int
	ErrorMessage   string
	StartedAt      time.Time
	FinishedAt     *time.Time
}

// User representa um usuário do sistema
type User struct {
	ID           string
	ClientID     string
	Email        string
	PasswordHash string
	Role         string
	Name         string
	Active       bool
}

// ── Clients ──────────────────────────────────────────────────

// GetActiveClients retorna todos os clientes ativos
func GetActiveClients(db *sql.DB) ([]Client, error) {
	rows, err := db.Query(`
		SELECT id, client_key, name, segment, erp_type, erp_config, active
		FROM lume_system.clients
		WHERE active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		if err := rows.Scan(
			&c.ID, &c.ClientKey, &c.Name,
			&c.Segment, &c.ERPType, &c.ERPConfig, &c.Active,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler cliente: %w", err)
		}
		clients = append(clients, c)
	}

	return clients, nil
}

// GetClientByKey retorna um cliente pelo client_key
func GetClientByKey(db *sql.DB, clientKey string) (*Client, error) {
	var c Client
	err := db.QueryRow(`
		SELECT id, client_key, name, segment, erp_type, erp_config, active
		FROM lume_system.clients
		WHERE client_key = $1 AND active = true
	`, clientKey).Scan(
		&c.ID, &c.ClientKey, &c.Name,
		&c.Segment, &c.ERPType, &c.ERPConfig, &c.Active,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cliente %s não encontrado", clientKey)
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}
	return &c, nil
}

// ── ETL Logs ─────────────────────────────────────────────────

// InsertETLLog cria um novo registro de log e retorna o ID
func InsertETLLog(db *sql.DB, clientID, connectorType string) (string, error) {
	var id string
	err := db.QueryRow(`
		INSERT INTO lume_system.etl_logs (client_id, connector_type, status)
		VALUES ($1, $2, 'running')
		RETURNING id
	`, clientID, connectorType).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("erro ao criar etl_log: %w", err)
	}
	return id, nil
}

// UpdateETLLogSuccess marca o log como sucesso
func UpdateETLLogSuccess(db *sql.DB, logID string, recordsRead, recordsWritten int) error {
	_, err := db.Exec(`
		UPDATE lume_system.etl_logs
		SET status = 'success',
		    records_read = $2,
		    records_written = $3,
		    finished_at = NOW()
		WHERE id = $1
	`, logID, recordsRead, recordsWritten)
	if err != nil {
		return fmt.Errorf("erro ao atualizar etl_log: %w", err)
	}
	return nil
}

// UpdateETLLogError marca o log como erro
func UpdateETLLogError(db *sql.DB, logID string, errMsg string) error {
	_, err := db.Exec(`
		UPDATE lume_system.etl_logs
		SET status = 'error',
		    error_message = $2,
		    finished_at = NOW()
		WHERE id = $1
	`, logID, errMsg)
	if err != nil {
		return fmt.Errorf("erro ao atualizar etl_log com erro: %w", err)
	}
	return nil
}

// ── Jobs ─────────────────────────────────────────────────────

// EnqueueJob enfileira um job para o Python processar
func EnqueueJob(db *sql.DB, clientID, jobType string, payload []byte) error {
	_, err := db.Exec(`
		INSERT INTO lume_system.jobs (client_id, job_type, payload)
		VALUES ($1, $2, $3)
	`, clientID, jobType, payload)
	if err != nil {
		return fmt.Errorf("erro ao enfileirar job: %w", err)
	}
	return nil
}

// ── Users ─────────────────────────────────────────────────
// GetUserByEmail busca um usuário pelo email
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	var u User
	err := db.QueryRow(`
		SELECT id, client_id, email, password_hash, role, name, active
		FROM lume_system.users
		WHERE email = $1 AND active = true
	`, email).Scan(&u.ID, &u.ClientID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.Active)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("usuário não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	return &u, nil
}

// GetClientByID busca um cliente pelo ID
func GetClientByID(db *sql.DB, clientID string) (*Client, error) {
	var c Client
	err := db.QueryRow(`
		SELECT id, client_key, name, segment, erp_type, erp_config, active
		FROM lume_system.clients
		WHERE id = $1
	`, clientID).Scan(
		&c.ID, &c.ClientKey, &c.Name,
		&c.Segment, &c.ERPType, &c.ERPConfig, &c.Active,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cliente não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}
	return &c, nil
}
