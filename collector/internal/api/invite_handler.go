package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/paulochiaradia/lume/collector/internal/auth"
	"github.com/paulochiaradia/lume/collector/internal/db"
)

// Request body para o Admin convidar
type InviteUserRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Request body para o funcionário aceitar o convite
type AcceptInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	Name     string `json:"name"` // Opcional, o cara pode colocar o próprio nome
}

// -----------------------------------------------------------------------------
// 1. O Admin cria o convite
// -----------------------------------------------------------------------------
func (s *Server) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminID := claims.UserID
	adminClientID := claims.TenantID
	adminRole := claims.Role

	// Regra de Negócio: Só admin ou gerente pode convidar
	if adminRole != "admin" && adminRole != "gerente" {
		writeError(w, http.StatusForbidden, "apenas administradores podem convidar usuários")
		return
	}

	var req InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload inválido")
		return
	}

	// Validações básicas
	if req.Email == "" || req.Role == "" {
		writeError(w, http.StatusBadRequest, "email e role são obrigatórios")
		return
	}

	// Verifica se o e-mail já existe no sistema para ESTE cliente
	_, err := db.GetUserByEmailAndClient(s.db, req.Email, adminClientID)
	if err == nil {
		// Se achou o usuário, já existe
		writeError(w, http.StatusConflict, "este e-mail já possui conta ativa na sua empresa")
		return
	}

	// Gera o token super seguro (32 bytes = 64 caracteres hexadecimais)
	token, err := auth.GenerateSecureToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar token de convite")
		return
	}

	// Define validade (ex: 48 horas)
	expiresAt := time.Now().Add(48 * time.Hour)

	// Salva no banco (amarrando o convite ao ClientID do Admin)
	err = db.CreateInvitation(s.db, adminClientID, req.Email, req.Role, token, adminID, expiresAt)
	if err != nil {
		log.Printf("Erro ao criar convite: %v", err)
		writeError(w, http.StatusInternalServerError, "erro ao registrar convite")
		return
	}

	// Constrói e dispara o E-mail
	inviteLink := fmt.Sprintf("http://localhost:3000/accept-invite?token=%s", token)
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
			<h2 style="color: #2563eb;">Você foi convidado para o Lume!</h2>
			<p>Um administrador convidou você para acessar o sistema.</p>
			<p>Seu cargo será: <strong>%s</strong></p>
			<p><a href="%s" style="background-color: #2563eb; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; display: inline-block;">Ativar minha conta</a></p>
			<p style="font-size: 12px; color: #666;">Este link expira em 48 horas.</p>
		</div>
	`, req.Role, inviteLink)

	go func() {
		err := s.mailer.Send(req.Email, "Convite de Acesso - Lume", html)
		if err != nil {
			log.Printf("🚨 Falha ao enviar e-mail de convite para %s: %v", req.Email, err)
		} else {
			log.Printf("✅ Convite enviado com sucesso para %s", req.Email)
		}
	}()

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "convite gerado e e-mail enviado com sucesso",
	})
}

// -----------------------------------------------------------------------------
// 2. O Funcionário aceita o convite e define a senha
// -----------------------------------------------------------------------------
func (s *Server) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload inválido")
		return
	}

	if req.Token == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "token inválido ou senha muito curta (mín. 8 caracteres)")
		return
	}

	// 1. Tenta buscar o convite no banco (já valida se expirou ou foi usado)
	invitation, err := db.GetValidInvitationByToken(s.db, req.Token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// 2. Hasheia a nova senha do abençoado
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao processar senha")
		return
	}

	// (Opcional) Nome Padrão
	finalName := req.Name
	if finalName == "" {
		finalName = "Novo Usuário"
	}

	// 3. Cria o Usuário de Fato! (Copiando o ClientID e Role do convite)
	err = db.CreateUser(s.db, invitation.ClientID, invitation.Email, hashedPassword, invitation.Role, finalName)
	if err != nil {
		log.Printf("Erro ao criar usuário via convite: %v", err)
		writeError(w, http.StatusInternalServerError, "falha ao ativar conta")
		return
	}

	// 4. Queima o convite para não ser usado de novo
	_ = db.MarkInvitationAsUsed(s.db, invitation.ID)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "conta ativada com sucesso. você já pode fazer login.",
	})
}

// -----------------------------------------------------------------------------
// 3. O Admin lista os convites pendentes
// -----------------------------------------------------------------------------
func (s *Server) handleListInvites(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminClientID := claims.TenantID
	adminRole := claims.Role

	if adminRole != "admin" && adminRole != "gerente" {
		writeError(w, http.StatusForbidden, "apenas administradores podem visualizar convites")
		return
	}

	invites, err := db.GetPendingInvitations(s.db, adminClientID)
	if err != nil {
		log.Printf("Erro ao listar convites: %v", err)
		writeError(w, http.StatusInternalServerError, "erro ao buscar convites pendentes")
		return
	}

	// Limpa o token da resposta por segurança (não deve ser exposto na listagem)
	for i := range invites {
		invites[i].Token = "[OMITIDO_POR_SEGURANCA]"
	}

	writeJSON(w, http.StatusOK, invites)
}

// -----------------------------------------------------------------------------
// 4. O Admin revoga/cancela um convite
// -----------------------------------------------------------------------------
func (s *Server) handleRevokeInvite(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminClientID := claims.TenantID
	adminRole := claims.Role

	if adminRole != "admin" && adminRole != "gerente" {
		writeError(w, http.StatusForbidden, "apenas administradores podem revogar convites")
		return
	}

	// Extrai o ID do convite da URL (Go 1.22+)
	inviteID := r.PathValue("id")
	if inviteID == "" {
		writeError(w, http.StatusBadRequest, "ID do convite é obrigatório")
		return
	}

	err := db.RevokeInvitation(s.db, inviteID, adminClientID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "convite revogado com sucesso",
	})
}
