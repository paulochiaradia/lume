package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims define a estrutura do que vai dentro do JWT
type Claims struct {
	UserID    string `json:"user_id"`
	TenantID  string `json:"tenant_id"`  // O ID interno (UUID)
	ClientKey string `json:"client_key"` // A string de identificação (ex: "loja_teste") para o DuckDB
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// CheckPasswordHash compara a senha em texto plano com o hash do banco
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateJWT cria o token de acesso de vida curta (15 minutos)
func GenerateJWT(userID, tenantID, clientKey, role, secret string) (string, error) {
	expirationTime := time.Now().Add(15 * time.Minute)

	claims := &Claims{
		UserID:    userID,
		TenantID:  tenantID,
		ClientKey: clientKey, // Adicionamos de volta!
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "lume_auth_service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken cria um token longo e criptograficamente seguro
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("falha ao gerar bytes aleatorios: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateToken verifica a assinatura do JWT e extrai os Claims
func ValidateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Valida se o método de assinatura é o esperado (evita ataques de troca de algoritmo)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("token inválido ou expirado")
}
