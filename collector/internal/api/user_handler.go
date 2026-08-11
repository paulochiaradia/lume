package api

import (
	"log"
	"net/http"

	"github.com/paulochiaradia/lume/collector/internal/db"
)

// -----------------------------------------------------------------------------
// 1. O Admin lista a sua equipe
// -----------------------------------------------------------------------------
func (s *Server) handleListTeam(w http.ResponseWriter, r *http.Request) {
	// 1. Usa nossa função auxiliar para pegar com segurança os dados do JWT
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminClientID := claims.TenantID
	adminRole := claims.Role

	// 2. Regra de Negócio: Apenas cargos de chefia podem ver a lista completa
	if adminRole != "admin" && adminRole != "gerente" {
		writeError(w, http.StatusForbidden, "apenas administradores podem visualizar a equipe")
		return
	}

	// 3. Busca a equipe inteira do Tenant
	team, err := db.GetUsersByClient(s.db, adminClientID)
	if err != nil {
		log.Printf("Erro ao listar equipe: %v", err)
		writeError(w, http.StatusInternalServerError, "erro ao buscar usuários")
		return
	}

	// 4. Retorna a lista limpa (UserSummary, sem senhas)
	writeJSON(w, http.StatusOK, team)
}

// -----------------------------------------------------------------------------
// 2. O Admin desativa (demite) um funcionário
// -----------------------------------------------------------------------------
func (s *Server) handleDeactivateUser(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminClientID := claims.TenantID
	adminRole := claims.Role
	adminID := claims.UserID

	// 1. Regra de Negócio: Apenas chefia desliga
	if adminRole != "admin" && adminRole != "gerente" {
		writeError(w, http.StatusForbidden, "apenas administradores podem desativar usuários")
		return
	}

	// 2. Extrai o ID do alvo da URL (ex: PATCH /admin/users/1234-abcd)
	targetUserID := r.PathValue("id")
	if targetUserID == "" {
		writeError(w, http.StatusBadRequest, "ID do usuário é obrigatório")
		return
	}

	// 3. Regra de Segurança Crítica: Um Admin não pode desativar a si mesmo!
	// (Isso evita que todos fiquem trancados para fora da empresa acidentalmente)
	if targetUserID == adminID {
		writeError(w, http.StatusForbidden, "você não pode desativar sua própria conta")
		return
	}

	// 4. Aciona o Kill Switch no banco
	err := db.DeactivateUser(s.db, targetUserID, adminClientID)
	if err != nil {
		// Se o ID não existir ou o cara for de outra empresa, o banco recusa.
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "usuário desativado com sucesso. todos os acessos foram revogados.",
	})
}
