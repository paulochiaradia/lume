package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

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

// LoginRequest representa o body do login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse representa a resposta do login
type LoginResponse struct {
	Token     string `json:"token"`
	Role      string `json:"role"`
	ClientKey string `json:"client_key"`
	Name      string `json:"name"`
}

// handleLogin autentica o usuário e retorna o JWT
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email e senha são obrigatórios")
		return
	}

	// Busca o usuário no banco
	user, err := db.GetUserByEmail(s.db, req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}

	// Verifica a senha
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}

	// Busca o cliente associado
	client, err := db.GetClientByID(s.db, user.ClientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar cliente")
		return
	}

	// Gera o JWT
	secret := os.Getenv("JWT_SECRET")
	token, err := auth.GenerateToken(user.ID, user.ClientID, client.ClientKey, user.Role, secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar token")
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     token,
		Role:      user.Role,
		ClientKey: client.ClientKey,
		Name:      user.Name,
	})
}
