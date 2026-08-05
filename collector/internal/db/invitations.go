package db

import (
	"database/sql"
	"errors"
	"time"
)

// Invitation representa o modelo da tabela de convites no banco
type Invitation struct {
	ID        string
	ClientID  string
	Email     string
	Role      string
	Token     string
	InvitedBy string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

// CreateInvitation insere um novo convite no banco de dados
func CreateInvitation(db *sql.DB, clientID, email, role, token, invitedBy string, expiresAt time.Time) error {
	query := `
		INSERT INTO lume_system.invitations (client_id, email, role, token, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(query, clientID, email, role, token, invitedBy, expiresAt)
	return err
}

// GetValidInvitationByToken busca um convite pelo token, garantindo que não foi usado e não expirou
func GetValidInvitationByToken(db *sql.DB, token string) (*Invitation, error) {
	query := `
		SELECT id, client_id, email, role, token, invited_by, expires_at, used, created_at
		FROM lume_system.invitations
		WHERE token = $1 AND used = FALSE AND expires_at > NOW()
	`

	var inv Invitation
	err := db.QueryRow(query, token).Scan(
		&inv.ID,
		&inv.ClientID,
		&inv.Email,
		&inv.Role,
		&inv.Token,
		&inv.InvitedBy,
		&inv.ExpiresAt,
		&inv.Used,
		&inv.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("convite inválido, já utilizado ou expirado")
		}
		return nil, err
	}

	return &inv, nil
}

// MarkInvitationAsUsed atualiza o status do convite para 'usado' (queima o token)
func MarkInvitationAsUsed(db *sql.DB, id string) error {
	query := `UPDATE lume_system.invitations SET used = TRUE WHERE id = $1`
	_, err := db.Exec(query, id)
	return err
}

// GetPendingInvitations lista todos os convites válidos e não usados de uma empresa
func GetPendingInvitations(db *sql.DB, clientID string) ([]Invitation, error) {
	query := `
		SELECT id, client_id, email, role, token, invited_by, expires_at, used, created_at
		FROM lume_system.invitations
		WHERE client_id = $1 AND used = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []Invitation
	for rows.Next() {
		var inv Invitation
		if err := rows.Scan(
			&inv.ID, &inv.ClientID, &inv.Email, &inv.Role,
			&inv.Token, &inv.InvitedBy, &inv.ExpiresAt, &inv.Used, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Se for nil, retornamos um slice vazio para o JSON ficar [] em vez de null
	if invites == nil {
		invites = []Invitation{}
	}

	return invites, nil
}

// RevokeInvitation deleta um convite pendente.
// Passamos o clientID junto por segurança, para garantir que um admin não apague o convite de outra empresa.
func RevokeInvitation(db *sql.DB, inviteID, clientID string) error {
	query := `DELETE FROM lume_system.invitations WHERE id = $1 AND client_id = $2`
	result, err := db.Exec(query, inviteID, clientID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("convite não encontrado ou você não tem permissão para apagá-lo")
	}

	return nil
}
