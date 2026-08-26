package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
		// Coleta o IP para rastreabilidade nos logs de bloqueio
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// [SLOG] Bloqueio silencioso (muito comum em varreduras de bots)
			slog.Warn("acesso negado: token nao fornecido",
				slog.String("event", "token_missing"),
				slog.String("ip", ip),
				slog.String("route", r.URL.Path),
			)
			writeError(w, http.StatusUnauthorized, "token não fornecido")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			// [SLOG] Erro de integração do Frontend ou manipulação manual
			slog.Warn("acesso negado: formato invalido",
				slog.String("event", "token_invalid_format"),
				slog.String("ip", ip),
				slog.String("route", r.URL.Path),
			)
			writeError(w, http.StatusUnauthorized, "formato de token inválido")
			return
		}

		secret := os.Getenv("JWT_SECRET")
		claims, err := auth.ValidateToken(parts[1], secret)
		if err != nil {
			// [SLOG] O caso mais grave: token expirado, assinatura adulterada ou JWT falso
			slog.Warn("acesso negado: falha na validacao do token",
				slog.String("event", "token_validation_failed"),
				slog.String("ip", ip),
				slog.String("route", r.URL.Path),
				slog.String("reason", err.Error()), // Aqui vai dizer se expirou ou se a assinatura é inválida
			)
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
		// [SLOG] Login falho - usuário inexistente
		slog.Warn("falha de login",
			slog.String("event", "login_failed"),
			slog.String("email", req.Email),
			slog.String("ip", ip),
			slog.String("reason", "usuario_nao_encontrado"),
		)
		// Boa prática: nunca confirmar se o e-mail existe ou não em falhas
		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}

	// 3. Verifica se a conta está atualmente congelada
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		// [SLOG] Tentativa de login em conta bloqueada
		slog.Warn("tentativa de login em conta bloqueada",
			slog.String("event", "account_locked_attempt"),
			slog.String("user_id", user.ID),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusForbidden, "conta bloqueada por múltiplas tentativas. Tente novamente mais tarde.")
		return
	}

	// 4. Valida a senha do usuário
	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		// Se errou a senha, adiciona um "strike"
		attempts, _ := db.IncrementFailedLogin(s.db, user.ID)

		// Log de auditoria no DB para monitorarmos ataques
		_ = db.CreateAuditLog(s.db, user.ID, user.ClientID, fmt.Sprintf("login_failed_attempt_%d", attempts), ip, userAgent)

		// [SLOG] Login falho - senha errada
		slog.Warn("falha de login",
			slog.String("event", "login_failed"),
			slog.String("user_id", user.ID),
			slog.String("ip", ip),
			slog.String("reason", "senha_invalida"),
			slog.Int("attempts", attempts),
		)

		if attempts >= 5 {
			_ = db.LockUserAccount(s.db, user.ID)
			_ = db.CreateAuditLog(s.db, user.ID, user.ClientID, "account_locked_brute_force", ip, userAgent)

			// [SLOG] Conta bloqueada por limite de tentativas
			slog.Warn("conta bloqueada por forca bruta",
				slog.String("event", "account_locked"),
				slog.String("user_id", user.ID),
				slog.String("ip", ip),
				slog.Int("attempts", attempts),
			)

			writeError(w, http.StatusForbidden, "conta bloqueada por 15 minutos por segurança.")
			return
		}

		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}

	// 5. Se a senha está correta, limpamos a ficha dele para os próximos logins!
	_ = db.ResetFailedLogin(s.db, user.ID)

	// Busca o cliente associado para colocar no Token
	client, err := db.GetClientByID(s.db, user.ClientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar cliente")
		return
	}

	// 6. Gera o JWT (15 minutos)
	secret := os.Getenv("JWT_SECRET")
	token, err := auth.GenerateJWT(user.ID, user.ClientID, client.ClientKey, user.Role, secret)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar token de acesso")
		return
	}

	// 7. Gera o Refresh Token (Seguro)
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar token de sessão")
		return
	}

	// 8. Salva a Sessão no Banco (Expira em 7 dias, por exemplo)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	err = db.CreateSession(s.db, user.ID, refreshToken, req.DeviceID, userAgent, ip, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao registrar sessão")
		return
	}

	// 9. Grava o log de Auditoria (Sucesso) no DB
	err = db.CreateAuditLog(s.db, user.ID, user.ClientID, "login_success", ip, userAgent)
	if err != nil {
		slog.Error("erro ao salvar log de auditoria no bd", slog.String("error", err.Error()))
	}

	// [SLOG] Login bem-sucedido
	slog.Info("login realizado",
		slog.String("event", "login_success"),
		slog.String("user_id", user.ID),
		slog.String("tenant_id", user.ClientID),
		slog.String("ip", ip),
		slog.String("method", r.Method),
	)

	// 10. Retorna o sucesso para o frontend
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
	// Coleta de IP para rastreabilidade de requisições maliciosas
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

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
		// [SLOG] A sessão não existe. Isso acontece se o token for falso, ou se
		// um Admin acionou o Kill Switch (DeactivateUser) e apagou a sessão.
		slog.Warn("falha na renovacao: sessao invalida ou revogada",
			slog.String("event", "refresh_invalid"),
			slog.String("ip", ip),
			slog.String("reason", "session_not_found_or_revoked"),
		)
		writeError(w, http.StatusUnauthorized, "sessão inválida ou não encontrada")
		return
	}

	// 2. Verifica se o Refresh Token já expirou (os 7 dias passaram)
	if time.Now().After(session.ExpiresAt) {
		// Limpa do banco para não acumular lixo
		_ = db.DeleteSession(s.db, req.RefreshToken)

		// [SLOG] O token venceu naturalmente pelo tempo.
		slog.Warn("falha na renovacao: sessao expirada",
			slog.String("event", "refresh_expired"),
			slog.String("user_id", session.UserID),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusUnauthorized, "sessão expirada, faça login novamente")
		return
	}

	// 3. Busca o usuário para gerar o novo JWT
	user, err := db.GetUserByID(s.db, session.UserID)
	if err != nil {
		slog.Error("erro interno: usuario da sessao nao encontrado", slog.String("user_id", session.UserID))
		writeError(w, http.StatusInternalServerError, "erro ao buscar dados do usuário")
		return
	}

	client, err := db.GetClientByID(s.db, user.ClientID)
	if err != nil {
		slog.Error("erro interno: tenant da sessao nao encontrado", slog.String("tenant_id", user.ClientID))
		writeError(w, http.StatusInternalServerError, "erro ao buscar dados do cliente")
		return
	}

	// 4. Gera um NOVO JWT fresquinho de 15 minutos
	secret := os.Getenv("JWT_SECRET")
	newToken, err := auth.GenerateJWT(user.ID, user.ClientID, client.ClientKey, user.Role, secret)
	if err != nil {
		slog.Error("erro interno ao assinar novo jwt", slog.String("user_id", user.ID))
		writeError(w, http.StatusInternalServerError, "erro ao gerar novo token")
		return
	}

	// [SLOG] Sucesso na renovação da sessão
	slog.Info("sessao renovada",
		slog.String("event", "refresh_success"),
		slog.String("user_id", user.ID),
		slog.String("tenant_id", user.ClientID),
		slog.String("ip", ip),
	)

	// Devolve o novo JWT (o frontend continua usando o mesmo Refresh Token antigo)
	writeJSON(w, http.StatusOK, map[string]string{
		"token": newToken,
	})
}

// handleLogout deleta a sessão do banco
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// 1. Coleta o IP logo no início para garantir a rastreabilidade em todos os fluxos
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token é obrigatório")
		return
	}

	// Tenta extrair os claims (funciona se a rota de logout estiver envolvida no jwtMiddleware)
	claims := getClaims(r)
	var userID, tenantID string
	if claims != nil {
		userID = claims.UserID
		tenantID = claims.TenantID
	}

	// 2. Deleta a sessão do banco de dados (o token longo perde a validade na hora)
	err := db.DeleteSession(s.db, req.RefreshToken)
	if err != nil {
		// [SLOG] Falha no logout: o cara tentou apagar uma sessão que já não existia.
		// Pode ser bug no frontend (chamou 2x) ou tentativa maliciosa.
		slog.Warn("falha no logout: token invalido ou ja removido",
			slog.String("event", "logout_failed"),
			slog.String("user_id", userID),
			slog.String("ip", ip),
			slog.String("reason", "invalid_session"),
		)

		writeError(w, http.StatusUnauthorized, "token inválido, inexistente ou já deslogado")
		return
	}

	// 3. Sucesso! Grava a auditoria e o log
	if claims != nil {
		// Auditoria no Banco de Dados (Histórico permanente)
		_ = db.CreateAuditLog(s.db, userID, tenantID, "logout_success", ip, r.UserAgent())

		// [SLOG] Telemetria em tempo real
		slog.Info("logout realizado",
			slog.String("event", "logout_success"),
			slog.String("user_id", userID),
			slog.String("tenant_id", tenantID),
			slog.String("ip", ip),
		)
	}

	// 4. Retorna sucesso
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "logout realizado com sucesso",
	})
}

// handleGetSessions retorna a lista de dispositivos logados do usuário
func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	// Coleta o IP para rastreabilidade de erros
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	claims := getClaims(r)
	if claims == nil {
		// [SLOG] Se chegou aqui sem claims, significa que alguém esqueceu de colocar
		// o jwtMiddleware na declaração desta rota no server.go! É uma falha crítica.
		slog.Error("falha critica: rota protegida acessada sem claims",
			slog.String("event", "missing_claims_in_protected_route"),
			slog.String("ip", ip),
			slog.String("route", r.URL.Path),
		)
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	sessions, err := db.GetActiveSessions(s.db, claims.UserID)
	if err != nil {
		// [SLOG] Erro interno do banco de dados ao buscar a lista
		slog.Error("erro ao buscar sessoes ativas no bd",
			slog.String("event", "get_sessions_failed"),
			slog.String("user_id", claims.UserID),
			slog.String("error", err.Error()),
			slog.String("ip", ip),
		)
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
	// 1. Coleta o IP para rastreabilidade
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

	var req RevokeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id é obrigatório")
		return
	}

	// 2. Apaga a sessão no banco
	err := db.DeleteSessionByDevice(s.db, claims.UserID, req.DeviceID)
	if err != nil {
		// [SLOG] Tentativa de deletar dispositivo inexistente. Pode ser duplo-clique no frontend
		// ou alguém tentando chutar (adivinhar) nomes de dispositivos via API.
		slog.Warn("falha ao revogar sessao: dispositivo nao encontrado",
			slog.String("event", "revoke_session_failed"),
			slog.String("user_id", claims.UserID),
			slog.String("device_id", req.DeviceID),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// 3. Sucesso! Grava a auditoria e o log

	// Auditoria no Banco (Histórico do Usuário)
	_ = db.CreateAuditLog(s.db, claims.UserID, claims.TenantID, "session_revoked_"+req.DeviceID, ip, r.UserAgent())

	// [SLOG] Telemetria em tempo real
	slog.Info("sessao de dispositivo revogada pelo usuario",
		slog.String("event", "session_revoked"),
		slog.String("user_id", claims.UserID),
		slog.String("tenant_id", claims.TenantID),
		slog.String("device_id", req.DeviceID),
		slog.String("ip", ip),
	)

	// 4. Retorna sucesso
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "dispositivo " + req.DeviceID + " desconectado com sucesso",
	})
}

// handleGlobalLogout aciona o botão de pânico e derruba todas as sessões
func (s *Server) handleGlobalLogout(w http.ResponseWriter, r *http.Request) {
	// 1. Captura o IP para o rastro forense (o atacante pode ter clicado nisso para trancar o usuário legítimo fora)
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

	// 2. Aciona o botão de pânico no banco (Deleta todas as sessões do usuário)
	err := db.DeleteAllSessions(s.db, claims.UserID)
	if err != nil {
		slog.Error("falha critica no banco ao executar global logout",
			slog.String("event", "global_logout_db_error"),
			slog.String("user_id", claims.UserID),
			slog.String("error", err.Error()),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusInternalServerError, "erro ao encerrar sessões")
		return
	}

	// 3. Sucesso! Grava a auditoria e o log

	_ = db.CreateAuditLog(s.db, claims.UserID, claims.TenantID, "global_logout_triggered", ip, r.UserAgent())

	// [SLOG] Telemetria em tempo real. Usamos WARN porque é um evento incomum e de alto impacto.
	slog.Warn("botao de panico acionado: todas as sessoes derrubadas",
		slog.String("event", "global_logout"),
		slog.String("user_id", claims.UserID),
		slog.String("tenant_id", claims.TenantID),
		slog.String("ip", ip),
	)

	// 4. Retorna sucesso
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "todos os dispositivos foram desconectados com sucesso",
	})
}

// handleLogoutOthers desconecta todos os dispositivos, exceto a sessão que fez a requisição
func (s *Server) handleLogoutOthers(w http.ResponseWriter, r *http.Request) {
	// 1. Captura do IP no início para garantir o rastro em qualquer cenário de erro
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

	// 2. Executa a limpeza no banco de dados
	err := db.DeleteOtherSessions(s.db, claims.UserID, req.RefreshToken)
	if err != nil {
		// [SLOG] Falha no banco ao tentar limpar as outras sessões
		slog.Error("erro no bd ao encerrar outras sessoes",
			slog.String("event", "logout_others_db_error"),
			slog.String("user_id", claims.UserID),
			slog.String("error", err.Error()),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusInternalServerError, "erro ao encerrar outras sessões")
		return
	}

	// 3. Sucesso! Grava a auditoria e a telemetria

	// Registra na trilha de auditoria quem apertou o botão
	_ = db.CreateAuditLog(s.db, claims.UserID, claims.TenantID, "logout_other_sessions", ip, r.UserAgent())

	// [SLOG] Telemetria em tempo real
	slog.Info("outros dispositivos desconectados pelo usuario",
		slog.String("event", "logout_others_success"),
		slog.String("user_id", claims.UserID),
		slog.String("tenant_id", claims.TenantID),
		slog.String("ip", ip),
	)

	// 4. Retorna sucesso
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
	// 1. Coleta o IP para rastreabilidade de segurança
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

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	// 2. Validações básicas da borda
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

	// 3. Busca o hash antigo no banco
	currentHash, err := db.GetPasswordHashByID(s.db, claims.UserID)
	if err != nil {
		slog.Error("erro no bd ao buscar hash da senha",
			slog.String("event", "get_password_hash_failed"),
			slog.String("user_id", claims.UserID),
			slog.String("error", err.Error()),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusInternalServerError, "erro ao buscar dados do usuário")
		return
	}

	// 4. Valida se o usuário sabe mesmo a senha atual (Alarme de Segurança)
	if !auth.CheckPasswordHash(req.CurrentPassword, currentHash) {
		// [SLOG] Alerta: alguém com uma sessão válida está errando a senha antiga.
		// Pode ser o usuário confuso, ou um atacante que invadiu o PC destravado.
		slog.Warn("tentativa de alteracao de senha falhou: senha atual incorreta",
			slog.String("event", "change_password_failed"),
			slog.String("user_id", claims.UserID),
			slog.String("reason", "wrong_current_password"),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusUnauthorized, "a senha atual está incorreta")
		return
	}

	// 5. Criptografa a nova senha e salva
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		slog.Error("erro ao gerar hash da nova senha", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "erro ao processar nova senha")
		return
	}

	if err := db.UpdatePassword(s.db, claims.UserID, newHash); err != nil {
		slog.Error("erro no bd ao atualizar senha", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "erro ao atualizar senha")
		return
	}

	// 6. Segurança máxima: Derruba todas as outras sessões (exceto a atual via Refresh Token)
	_ = db.DeleteOtherSessions(s.db, claims.UserID, req.RefreshToken)

	// 7. Trilha de Auditoria e Telemetria
	_ = db.CreateAuditLog(s.db, claims.UserID, claims.TenantID, "password_changed", ip, r.UserAgent())

	// [SLOG] Telemetria em tempo real
	slog.Info("senha alterada com sucesso",
		slog.String("event", "password_changed"),
		slog.String("user_id", claims.UserID),
		slog.String("tenant_id", claims.TenantID),
		slog.String("ip", ip),
	)

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
	// 1. Coleta o IP para rastrearmos tentativas de abuso
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email é obrigatório")
		return
	}

	// [SLOG] Telemetria: Registra o pedido. Útil para monitorar volume de requisições de uma mesma origem.
	slog.Info("pedido de recuperacao de senha",
		slog.String("event", "password_reset_requested"),
		slog.String("email", req.Email),
		slog.String("ip", ip),
	)

	// Tenta buscar o usuário
	user, err := db.GetUserByEmail(s.db, req.Email)
	if err != nil {
		// [SLOG] Se o e-mail não existe, registramos como WARN.
		// Muitos WARNs seguidos do mesmo IP indicam um bot tentando adivinhar e-mails.
		slog.Warn("recuperacao falhou: e-mail nao encontrado",
			slog.String("event", "password_reset_email_not_found"),
			slog.String("email", req.Email),
			slog.String("ip", ip),
		)
		// Mantém a proteção contra enumeração
		writeJSON(w, http.StatusOK, map[string]string{"message": "se o e-mail existir, um link de recuperação foi enviado."})
		return
	}

	// Gera o token de 32 bytes
	token, err := auth.GenerateSecureToken(32)
	if err != nil {
		slog.Error("erro ao gerar token de recuperacao", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "erro interno do servidor")
		return
	}

	// Salva no banco com validade de 1 hora
	if err := db.CreatePasswordResetToken(s.db, user.ID, token); err != nil {
		slog.Error("erro ao salvar token de recuperacao no bd", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "erro interno do servidor")
		return
	}

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
			// [SLOG] Captura erros no serviço de e-mail (ex: SendGrid/Resend fora do ar)
			slog.Error("falha ao enviar e-mail de recuperacao",
				slog.String("event", "password_reset_email_failed"),
				slog.String("user_id", user.ID),
				slog.String("email", user.Email),
				slog.String("error", err.Error()),
			)
		} else {
			// [SLOG] Sucesso
			slog.Info("e-mail de recuperacao enviado",
				slog.String("event", "password_reset_email_sent"),
				slog.String("user_id", user.ID),
			)
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
	// 1. Coleta o IP para rastreabilidade de segurança forense
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.Token == "" || len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "token inválido ou senha muito curta (mínimo 8 caracteres)")
		return
	}

	// 2. Valida se o token existe e não expirou
	userID, err := db.GetValidPasswordResetToken(s.db, req.Token)
	if err != nil {
		// [SLOG] Tentativa de usar um token inválido, forjado ou que já venceu.
		// Muito comum quando bots testam tokens aleatórios na API.
		slog.Warn("falha na redefinicao: token invalido ou expirado",
			slog.String("event", "password_reset_invalid_token"),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusUnauthorized, "token inválido ou expirado")
		return
	}

	// 3. Criptografa a nova senha
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		slog.Error("erro ao gerar hash da nova senha", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "erro ao processar senha")
		return
	}

	// 4. Atualiza no banco
	if err := db.UpdatePassword(s.db, userID, newHash); err != nil {
		slog.Error("erro no bd ao atualizar senha redefinida", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "erro ao atualizar senha")
		return
	}

	// 5. Queima o token
	_ = db.MarkTokenAsUsed(s.db, req.Token)

	// 6. SEGURANÇA MÁXIMA: Derruba todas as sessões ativas deste usuário (Botão de pânico)
	// Isso garante que se o hacker estava logado, ele perde o acesso na hora.
	_ = db.DeleteAllSessions(s.db, userID)

	// 7. Auditoria e Telemetria
	// Busca rapidamente os dados do usuário para preencher o TenantID na auditoria
	user, err := db.GetUserByID(s.db, userID)
	tenantID := ""
	if err == nil && user != nil {
		tenantID = user.ClientID
	}

	// Grava no histórico permanente do banco
	_ = db.CreateAuditLog(s.db, userID, tenantID, "password_reset_completed", ip, r.UserAgent())

	// [SLOG] Telemetria em tempo real
	slog.Info("senha redefinida via token",
		slog.String("event", "password_reset_success"),
		slog.String("user_id", userID),
		slog.String("tenant_id", tenantID),
		slog.String("ip", ip),
	)

	writeJSON(w, http.StatusOK, map[string]string{"message": "senha redefinida com sucesso. você já pode fazer login."})
}
