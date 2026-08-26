package db

import (
	"database/sql"
	"errors"
	"time"
)

// UserSummary representa os dados limpos do usuário para serem enviados ao Frontend
type UserSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetUsersByClient busca todos os usuários de uma empresa específica
func GetUsersByClient(db *sql.DB, clientID string) ([]UserSummary, error) {
	query := `
		SELECT id, name, email, role, active, created_at, updated_at
		FROM lume_system.users
		WHERE client_id = $1
		ORDER BY active DESC, created_at DESC
	`
	// O "ORDER BY active DESC" faz com que os inativos fiquem no final da lista

	rows, err := db.Query(query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserSummary
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Email, &u.Role, &u.Active, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if users == nil {
		users = []UserSummary{}
	}

	return users, nil
}

// DeactivateUser inativa um usuário e apaga todas as suas sessões ativas (Kill Switch)
func DeactivateUser(db *sql.DB, targetUserID, adminClientID string) error {
	// Iniciamos uma transação
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// O defer garante que, se der qualquer erro (panic) no meio do caminho,
	// o banco desfaz (Rollback) tudo o que foi feito dentro da transação.
	defer tx.Rollback()

	// 1. Inativa o usuário (Garantindo que ele pertence à MESMA empresa do admin)
	updateQuery := `UPDATE lume_system.users SET active = false WHERE id = $1 AND client_id = $2`
	result, err := tx.Exec(updateQuery, targetUserID, adminClientID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// Se 0 linhas foram afetadas, significa que o ID não existe ou o usuário é de OUTRA empresa
	if rowsAffected == 0 {
		return errors.New("usuário não encontrado ou não pertence a esta empresa")
	}

	// 2. Kill Switch: Apaga todas as sessões abertas deste usuário
	deleteSessionsQuery := `DELETE FROM lume_system.sessions WHERE user_id = $1`
	_, err = tx.Exec(deleteSessionsQuery, targetUserID)
	if err != nil {
		return err
	}

	// Se chegou até aqui, tudo ocorreu perfeitamente. Commita as duas ações juntas!
	return tx.Commit()
}
