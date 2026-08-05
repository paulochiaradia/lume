package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/paulochiaradia/lume/collector/internal/auth"
	"github.com/paulochiaradia/lume/collector/internal/db"
)

type contextKey string

const claimsKey contextKey = "claims"

// jwtMiddleware valida o token JWT em todas as rotas protegidas
func (s *Server) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "token não fornecido")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "formato de token inválido")
			return
		}

		secret := os.Getenv("JWT_SECRET")
		// Nota: Certifique-se de que a função ValidateToken no pacote auth existe e suporta o JWT/v5
		claims, err := auth.ValidateToken(parts[1], secret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}

		// Injeta os claims no contexto da requisição
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getClaims extrai os claims do contexto da requisição
func getClaims(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return claims
}

// LoginRequest representa o body do login (adicionamos DeviceID opcional)
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"` // Novo: Identificador do aparelho/navegador
}

// LoginResponse representa a resposta do login
type LoginResponse struct {
	Token        string `json:"token"`         // O JWT (Access Token de 15 min)
	RefreshToken string `json:"refresh_token"` // Novo: O token longo para renovação
	Role         string `json:"role"`
	ClientKey    string `json:"client_key"`
	Name         string `json:"name"`
}

// handleLogin autentica o usuário, gera os tokens e grava os logs de auditoria
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// 1. Coleta metadados de segurança
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded // Pega o IP real se estiver atrás de um Nginx/Proxy
	}
	userAgent := r.UserAgent()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email e senha são obrigatórios")
		return
	}

	// 2. Busca o usuário no banco
	user, err := db.GetUserByEmail(s.db, req.Email)
	if err != nil {
		// Boa prática: nunca confirmar se o e-mail existe ou não em falhas
		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}

	// 1. [NOVO] Verifica se a conta está atualmente congelada
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		writeError(w, http.StatusForbidden, "conta bloqueada por múltiplas tentativas. Tente novamente mais tarde.")
		return
	}

	// 2. Valida a senha do usuário
	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		// [NOVO] Se errou a senha, adiciona um "strike"
		attempts, _ := db.IncrementFailedLogin(s.db, user.ID)

		// [NOVO] Log de auditoria para monitorarmos ataques
		_ = db.CreateAuditLog(s.db, user.ID, user.ClientID, fmt.Sprintf("login_failed_attempt_%d", attempts), ip, userAgent)

		if attempts >= 5 {
			_ = db.LockUserAccount(s.db, user.ID)
			_ = db.CreateAuditLog(s.db, user.ID, user.ClientID, "account_locked_brute_force", ip, userAgent)
			writeError(w, http.StatusForbidden, "conta bloqueada por 15 minutos por segurança.")
			return
		}

		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}

	// 3. [NOVO] Se a senha está correta, limpamos a ficha dele para os próximos logins!
	_ = db.ResetFailedLogin(s.db, user.ID)

	// Busca o cliente associado para colocar no Token
	client, err := db.GetClientByID(s.db, user.ClientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar cliente")
		return
	}

	// 4. Gera o JWT (15 minutos)
	secret := os.Getenv("JWT_SECRET")
	token, err := auth.GenerateJWT(user.ID, user.ClientID, client.ClientKey, user.Role, secret)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar token de acesso")
		return
	}

	// 5. Gera o Refresh Token (Seguro)
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar token de sessão")
		return
	}

	// 6. Salva a Sessão no Banco (Expira em 7 dias, por exemplo)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	err = db.CreateSession(s.db, user.ID, refreshToken, req.DeviceID, userAgent, ip, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao registrar sessão")
		return
	}

	// 7. Grava o log de Auditoria (Sucesso)
	err = db.CreateAuditLog(s.db, user.ID, user.ClientID, "login_success", ip, userAgent)
	if err != nil {
		// Loga o erro internamente, mas não impede o login do usuário
	}

	// 8. Retorna o sucesso para o frontend
	writeJSON(w, http.StatusOK, LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		Role:         user.Role,
		ClientKey:    client.ClientKey,
		Name:         user.Name,
	})
}

// TokenRequest serve tanto para receber o token de refresh quanto para o logout
type TokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// handleRefresh recebe o Refresh Token longo e devolve um novo JWT de 15 minutos
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token é obrigatório")
		return
	}

	// 1. Busca a sessão no banco
	session, err := db.GetSessionByToken(s.db, req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sessão inválida ou não encontrada")
		return
	}

	// 2. Verifica se o Refresh Token já expirou (os 7 dias passaram)
	if time.Now().After(session.ExpiresAt) {
		// Limpa do banco para não acumular lixo
		_ = db.DeleteSession(s.db, req.RefreshToken)
		writeError(w, http.StatusUnauthorized, "sessão expirada, faça login novamente")
		return
	}

	// 3. Busca o usuário para gerar o novo JWT
	user, err := db.GetUserByID(s.db, session.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar dados do usuário")
		return
	}

	client, err := db.GetClientByID(s.db, user.ClientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar dados do cliente")
		return
	}

	// 4. Gera um NOVO JWT fresquinho de 15 minutos
	secret := os.Getenv("JWT_SECRET")
	newToken, err := auth.GenerateJWT(user.ID, user.ClientID, client.ClientKey, user.Role, secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar novo token")
		return
	}

	// Devolve o novo JWT (o frontend continua usando o mesmo Refresh Token antigo)
	writeJSON(w, http.StatusOK, map[string]string{
		"token": newToken,
	})
}

// handleLogout deleta a sessão do banco
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token é obrigatório")
		return
	}

	// Deleta a sessão do banco de dados (o token longo perde a validade na hora)
	err := db.DeleteSession(s.db, req.RefreshToken)
	if err != nil {
		// Se o token não existir ou já tiver sido apagado, retornamos 401
		writeError(w, http.StatusUnauthorized, "token inválido, inexistente ou já deslogado")
		return
	}

	// Extrai os claims do usuário que está fazendo logout para gravar na auditoria
	claims := getClaims(r)
	if claims != nil {
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}
		// Registra o log de segurança: o usuário saiu voluntariamente
		_ = db.CreateAuditLog(s.db, claims.UserID, claims.TenantID, "logout_success", ip, r.UserAgent())
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "logout realizado com sucesso",
	})
}

// handleGetSessions retorna a lista de dispositivos logados do usuário
func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	sessions, err := db.GetActiveSessions(s.db, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar sessões")
		return
	}

	// Garante que retorne um array vazio [] no JSON em vez de null caso não tenha sessões (boa prática de frontend)
	if sessions == nil {
		sessions = []db.ActiveSession{}
	}

	writeJSON(w, http.StatusOK, sessions)
}

// RevokeSessionRequest estrutura o payload para revogar um dispositivo
type RevokeSessionRequest struct {
	DeviceID string `json:"device_id"`
}

// handleRevokeSession desconecta um dispositivo específico escolhido pelo usuário
func (s *Server) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	var req RevokeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id é obrigatório")
		return
	}

	err := db.DeleteSessionByDevice(s.db, claims.UserID, req.DeviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "dispositivo " + req.DeviceID + " desconectado com sucesso",
	})
}

// handleGlobalLogout aciona o botão de pânico e derruba todas as sessões
func (s *Server) handleGlobalLogout(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	err := db.DeleteAllSessions(s.db, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao encerrar sessões")
		return
	}

	// Log opcional para auditoria de segurança severa
	_ = db.CreateAuditLog(s.db, claims.UserID, claims.TenantID, "global_logout_triggered", r.RemoteAddr, r.UserAgent())

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "todos os dispositivos foram desconectados com sucesso",
	})
}

// handleLogoutOthers desconecta todos os dispositivos, exceto a sessão que fez a requisição
func (s *Server) handleLogoutOthers(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	// Aproveitamos a mesma struct do Logout comum
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token é obrigatório")
		return
	}

	err := db.DeleteOtherSessions(s.db, claims.UserID, req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao encerrar outras sessões")
		return
	}

	// Registra na trilha de auditoria quem apertou o botão
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}
	_ = db.CreateAuditLog(s.db, claims.UserID, claims.TenantID, "logout_other_sessions", ip, r.UserAgent())

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "todos os outros dispositivos foram desconectados com sucesso",
	})
}

// ChangePasswordRequest define o payload esperado
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	RefreshToken    string `json:"refresh_token"` // Precisamos para não derrubar a sessão que está fazendo a troca
}

// handleChangePassword processa a troca de senha do usuário logado
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	// 1. Validações básicas da borda
	if req.CurrentPassword == "" || req.NewPassword == "" || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "current_password, new_password e refresh_token são obrigatórios")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "a nova senha deve ter pelo menos 8 caracteres")
		return
	}
	if req.CurrentPassword == req.NewPassword {
		writeError(w, http.StatusBadRequest, "a nova senha não pode ser igual à atual")
		return
	}

	// 2. Busca o hash antigo no banco
	currentHash, err := db.GetPasswordHashByID(s.db, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar dados do usuário")
		return
	}

	// 3. Valida se o usuário sabe mesmo a senha atual
	if !auth.CheckPasswordHash(req.CurrentPassword, currentHash) {
		writeError(w, http.StatusUnauthorized, "a senha atual está incorreta")
		return
	}

	// 4. Criptografa a nova senha e salva
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao processar nova senha")
		return
	}

	if err := db.UpdatePassword(s.db, claims.UserID, newHash); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao atualizar senha")
		return
	}

	// 5. Segurança máxima: Derruba todas as outras sessões
	_ = db.DeleteOtherSessions(s.db, claims.UserID, req.RefreshToken)

	// 6. Trilha de Auditoria
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}
	_ = db.CreateAuditLog(s.db, claims.UserID, claims.TenantID, "password_changed", ip, r.UserAgent())

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "senha alterada com sucesso! todos os outros dispositivos foram desconectados por segurança.",
	})
}

// ForgotPasswordRequest payload para solicitar o link
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// handleForgotPassword processa o pedido de recuperação (Rota Pública)
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email é obrigatório")
		return
	}

	log.Printf("auth: pedido de recuperação para %s", req.Email)

	// Tenta buscar o usuário
	user, err := db.GetUserByEmail(s.db, req.Email)
	if err != nil {
		log.Printf("auth: e-mail não encontrado para recuperação")
		writeJSON(w, http.StatusOK, map[string]string{"message": "se o e-mail existir, um link de recuperação foi enviado."})
		return
	}

	// Gera o token de 32 bytes
	token, err := auth.GenerateSecureToken(32)
	if err != nil {
		log.Printf("auth: erro ao gerar token de recuperação: %v", err)
		writeError(w, http.StatusInternalServerError, "erro interno do servidor")
		return
	}

	// Salva no banco com validade de 1 hora
	if err := db.CreatePasswordResetToken(s.db, user.ID, token); err != nil {
		log.Printf("auth: erro ao salvar token de recuperação: %v", err)
		writeError(w, http.StatusInternalServerError, "erro interno do servidor")
		return
	}

	log.Printf("auth: token de recuperação gerado para %s", user.Email)

	resetLink := fmt.Sprintf("http://localhost:3000/reset-password?token=%s", token)
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
			<h2 style="color: #2563eb;">Recuperação de Senha - Lume</h2>
			<p>Olá %s,</p>
			<p>Recebemos uma solicitação para redefinir a senha da sua conta.</p>
			<p><a href="%s" style="background-color: #2563eb; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; display: inline-block;">Redefinir Minha Senha</a></p>
			<p style="font-size: 12px; color: #666;">Se você não solicitou isso, pode ignorar este e-mail. O link expira em 1 hora.</p>
		</div>
	`, user.Name, resetLink)

	// Dispara o e-mail em background
	go func() {
		err := s.mailer.Send(user.Email, "Lume - Recuperação de Senha", html)
		if err != nil {
			log.Printf("auth: falha ao enviar e-mail de recuperação para %s: %v", user.Email, err)
		}
	}()

	writeJSON(w, http.StatusOK, map[string]string{"message": "se o e-mail existir, um link de recuperação foi enviado."})
}

// ResetPasswordRequest payload para salvar a nova senha
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// handleResetPassword processa a nova senha (Rota Pública)
func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.Token == "" || len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "token inválido ou senha muito curta (mínimo 8 caracteres)")
		return
	}

	// 1. Valida se o token existe e não expirou
	userID, err := db.GetValidPasswordResetToken(s.db, req.Token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "token inválido ou expirado")
		return
	}

	// 2. Criptografa a nova senha
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao processar senha")
		return
	}

	// 3. Atualiza no banco
	if err := db.UpdatePassword(s.db, userID, newHash); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao atualizar senha")
		return
	}

	// 4. Queima o token
	_ = db.MarkTokenAsUsed(s.db, req.Token)

	// 5. SEGURANÇA MÁXIMA: Derruba todas as sessões ativas deste usuário (Botão de pânico)
	// Isso garante que se o hacker estava logado, ele perde o acesso na hora.
	_ = db.DeleteAllSessions(s.db, userID)

	writeJSON(w, http.StatusOK, map[string]string{"message": "senha redefinida com sucesso. você já pode fazer login."})
}
