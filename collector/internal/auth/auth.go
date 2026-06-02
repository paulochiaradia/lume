package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("token inválido")
	ErrExpiredToken = errors.New("token expirado")
)

// Claims representa o payload do JWT
type Claims struct {
	UserID    string `json:"user_id"`
	ClientID  string `json:"client_id"`
	ClientKey string `json:"client_key"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken gera um JWT para o usuário
func GenerateToken(userID, clientID, clientKey, role, secret string) (string, error) {
	claims := Claims{
		UserID:    userID,
		ClientID:  clientID,
		ClientKey: clientKey,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "lume",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken valida um JWT e retorna os claims
func ValidateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
