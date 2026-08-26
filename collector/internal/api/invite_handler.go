package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	claims := getClaims(r)
	if claims == nil {
		slog.Error("falha critica: rota protegida acessada sem claims",
			slog.String("event", "missing_claims_in_protected_route"),
			slog.String("ip", ip),
			slog.String("route", r.URL.Path),
		)
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminID := claims.UserID
	adminClientID := claims.TenantID
	adminRole := claims.Role

	// Regra de Negócio: Só admin ou gerente pode convidar
	if adminRole != "admin" && adminRole != "gerente" {
		slog.Warn("tentativa de acesso nao autorizado: criar convite",
			slog.String("event", "unauthorized_access_attempt"),
			slog.String("user_id", adminID),
			slog.String("role", adminRole),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusForbidden, "apenas administradores podem convidar usuários")
		return
	}

	var req InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload inválido")
		return
	}

	if req.Email == "" || req.Role == "" {
		writeError(w, http.StatusBadRequest, "email e role são obrigatórios")
		return
	}

	// Verifica se o e-mail já existe no sistema para ESTE cliente
	_, err := db.GetUserByEmailAndClient(s.db, req.Email, adminClientID)
	if err == nil {
		// [SLOG] Tentativa de convidar alguém que já está na equipe
		slog.Info("tentativa de convite duplicado",
			slog.String("event", "invite_duplicate_email"),
			slog.String("admin_id", adminID),
			slog.String("email", req.Email),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusConflict, "este e-mail já possui conta ativa na sua empresa")
		return
	}

	// Gera o token super seguro (32 bytes = 64 caracteres hexadecimais)
	token, err := auth.GenerateSecureToken(32)
	if err != nil {
		slog.Error("erro ao gerar token de convite", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "erro ao gerar token de convite")
		return
	}

	// Define validade (ex: 48 horas)
	expiresAt := time.Now().Add(48 * time.Hour)

	// Salva no banco (amarrando o convite ao ClientID do Admin)
	err = db.CreateInvitation(s.db, adminClientID, req.Email, req.Role, token, adminID, expiresAt)
	if err != nil {
		slog.Error("erro no bd ao criar convite",
			slog.String("event", "create_invite_db_error"),
			slog.String("error", err.Error()),
			slog.String("tenant_id", adminClientID),
		)
		writeError(w, http.StatusInternalServerError, "erro ao registrar convite")
		return
	}

	// Auditoria: Registra quem enviou o convite
	_ = db.CreateAuditLog(s.db, adminID, adminClientID, "invite_created_for_"+req.Email, ip, r.UserAgent())

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
			slog.Error("falha ao enviar e-mail de convite",
				slog.String("event", "invite_email_failed"),
				slog.String("email", req.Email),
				slog.String("error", err.Error()),
			)
		} else {
			slog.Info("e-mail de convite enviado com sucesso",
				slog.String("event", "invite_email_sent"),
				slog.String("email", req.Email),
			)
		}
	}()

	slog.Info("convite gerado com sucesso",
		slog.String("event", "invite_created"),
		slog.String("admin_id", adminID),
		slog.String("target_email", req.Email),
		slog.String("tenant_id", adminClientID),
		slog.String("ip", ip),
	)

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "convite gerado e e-mail enviado com sucesso",
	})
}

// -----------------------------------------------------------------------------
// 2. O Funcionário aceita o convite e define a senha
// -----------------------------------------------------------------------------
func (s *Server) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

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
		// [SLOG] Possível varredura de bots ou usuário clicando em convite velho
		slog.Warn("falha ao aceitar convite: token invalido ou expirado",
			slog.String("event", "invite_accept_invalid_token"),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// 2. Hasheia a nova senha do abençoado
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		slog.Error("erro ao processar hash de senha no convite", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "erro ao processar senha")
		return
	}

	finalName := req.Name
	if finalName == "" {
		finalName = "Novo Usuário"
	}

	// 3. Cria o Usuário de Fato! (Copiando o ClientID e Role do convite)
	err = db.CreateUser(s.db, invitation.ClientID, invitation.Email, hashedPassword, invitation.Role, finalName)
	if err != nil {
		slog.Error("erro no bd ao criar usuario via convite",
			slog.String("event", "create_user_from_invite_error"),
			slog.String("error", err.Error()),
			slog.String("email", invitation.Email),
		)
		writeError(w, http.StatusInternalServerError, "falha ao ativar conta")
		return
	}

	// 4. Queima o convite para não ser usado de novo
	_ = db.MarkInvitationAsUsed(s.db, invitation.ID)

	// Auditoria: Como não temos o UserID recém-criado em mãos (dependendo da sua função CreateUser),
	// registramos no Tenant que uma nova conta foi ativada.
	_ = db.CreateAuditLog(s.db, "SYSTEM", invitation.ClientID, "user_activated_via_invite_"+invitation.Email, ip, r.UserAgent())

	slog.Info("conta ativada via convite",
		slog.String("event", "invite_accepted"),
		slog.String("email", invitation.Email),
		slog.String("tenant_id", invitation.ClientID),
		slog.String("ip", ip),
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "conta ativada com sucesso. você já pode fazer login.",
	})
}

// -----------------------------------------------------------------------------
// 3. O Admin lista os convites pendentes
// -----------------------------------------------------------------------------
func (s *Server) handleListInvites(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminClientID := claims.TenantID
	adminRole := claims.Role

	if adminRole != "admin" && adminRole != "gerente" {
		slog.Warn("tentativa de acesso nao autorizado: listar convites",
			slog.String("event", "unauthorized_access_attempt"),
			slog.String("user_id", claims.UserID),
			slog.String("role", adminRole),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusForbidden, "apenas administradores podem visualizar convites")
		return
	}

	invites, err := db.GetPendingInvitations(s.db, adminClientID)
	if err != nil {
		slog.Error("erro ao listar convites no bd",
			slog.String("event", "list_invites_db_error"),
			slog.String("error", err.Error()),
			slog.String("tenant_id", adminClientID),
			slog.String("ip", ip),
		)
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
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminClientID := claims.TenantID
	adminRole := claims.Role
	adminID := claims.UserID

	if adminRole != "admin" && adminRole != "gerente" {
		slog.Warn("tentativa de acesso nao autorizado: revogar convite",
			slog.String("event", "unauthorized_access_attempt"),
			slog.String("user_id", adminID),
			slog.String("role", adminRole),
			slog.String("ip", ip),
		)
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
		slog.Warn("falha ao revogar convite: nao encontrado ou ja revogado",
			slog.String("event", "revoke_invite_failed"),
			slog.String("admin_id", adminID),
			slog.String("invite_id", inviteID),
			slog.String("error", err.Error()),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Auditoria e Telemetria
	_ = db.CreateAuditLog(s.db, adminID, adminClientID, "invite_revoked_"+inviteID, ip, r.UserAgent())

	slog.Info("convite revogado com sucesso",
		slog.String("event", "invite_revoked"),
		slog.String("admin_id", adminID),
		slog.String("invite_id", inviteID),
		slog.String("tenant_id", adminClientID),
		slog.String("ip", ip),
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "convite revogado com sucesso",
	})
}
