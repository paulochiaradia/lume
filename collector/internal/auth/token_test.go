package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Testa se o hash do bcrypt está comparando senhas corretamente
func TestCheckPasswordHash(t *testing.T) {
	password := "senha_super_segura_123"
	// Gera o hash simulando o que estaria no banco
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Erro ao gerar hash para o teste: %v", err)
	}

	// Cenário 1: Senha correta (Happy Path)
	if !CheckPasswordHash(password, string(hash)) {
		t.Errorf("Esperava que a senha %s fosse validada com sucesso", password)
	}

	// Cenário 2: Senha incorreta (Ataque)
	if CheckPasswordHash("senha_errada", string(hash)) {
		t.Error("A função falhou gravemente e validou uma senha incorreta")
	}
}

// Testa a criação e validação do JWT com todos os claims de Multi-Tenant
func TestGenerateAndValidateJWT(t *testing.T) {
	userID := "uuid-user-123"
	tenantID := "uuid-tenant-456"
	clientKey := "loja_teste"
	role := "admin"
	secret := "chave_secreta_de_teste"

	// 1. Gera o Token
	tokenString, err := GenerateJWT(userID, tenantID, clientKey, role, secret)
	if err != nil {
		t.Fatalf("Erro inesperado ao gerar JWT: %v", err)
	}
	if tokenString == "" {
		t.Fatal("A função retornou uma string de token vazia")
	}

	// 2. Valida o Token gerado
	claims, err := ValidateToken(tokenString, secret)
	if err != nil {
		t.Fatalf("Erro inesperado ao validar o JWT: %v", err)
	}

	// 3. Verifica se os dados gravados dentro do token continuam íntegros
	if claims.UserID != userID {
		t.Errorf("UserID alterado. Esperado: %s, Recebido: %s", userID, claims.UserID)
	}
	if claims.TenantID != tenantID {
		t.Errorf("TenantID alterado. Esperado: %s, Recebido: %s", tenantID, claims.TenantID)
	}
	if claims.ClientKey != clientKey {
		t.Errorf("ClientKey alterado. Esperado: %s, Recebido: %s", clientKey, claims.ClientKey)
	}

	// 4. Teste de Segurança: Força o uso de uma chave secreta falsa (Ataque)
	_, err = ValidateToken(tokenString, "chave_falsa_hackeada")
	if err == nil {
		t.Error("A validação deveria ter falhado ao usar uma chave secreta incorreta, mas aceitou")
	}
}

// Testa a força criptográfica do Refresh Token
func TestGenerateRefreshToken(t *testing.T) {
	token1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("Erro ao gerar o primeiro refresh token: %v", err)
	}

	token2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("Erro ao gerar o segundo refresh token: %v", err)
	}

	// Verifica se gerou colisões (O que seria catastrófico em base64 com alta entropia)
	if token1 == token2 {
		t.Error("Aviso crítico de segurança: A função gerou tokens repetidos")
	}

	// Verifica se a string gerada é 100% URL Safe (sem '+' ou '/')
	if strings.Contains(token1, "+") || strings.Contains(token1, "/") {
		t.Error("O token gerado possui caracteres inválidos para URLs ou Headers HTTP")
	}
}
