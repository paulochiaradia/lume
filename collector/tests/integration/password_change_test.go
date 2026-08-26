package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
)

func TestPasswordChangeFlow(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if ok {
		baseDir := filepath.Dir(currentFile)
		for _, candidate := range []string{
			filepath.Join(baseDir, "..", "..", "..", ".env"),
			filepath.Join(baseDir, "..", "..", ".env"),
			filepath.Join(baseDir, "..", ".env"),
			filepath.Join(baseDir, ".env"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				_ = godotenv.Load(candidate)
				break
			}
		}
	}

	email := os.Getenv("EMAIL")
	originalPassword := os.Getenv("PASSWORD")
	t.Logf("EMAIL=%q PASSWORD=%q", email, originalPassword)
	newPassword := "NovaSenha_123_TESTE" // Senha temporária do teste

	// Função helper para disparar requisições
	doRequest := func(method, url, token string, body interface{}) (*http.Response, error) {
		var buf io.Reader
		if body != nil {
			jsonBody, _ := json.Marshal(body)
			buf = bytes.NewBuffer(jsonBody)
		}
		req, _ := http.NewRequest(method, baseURL+url, buf)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return http.DefaultClient.Do(req)
	}

	// Função helper para Login
	doLogin := func(pass, device string) (int, string, string) {
		payload := map[string]string{"email": email, "password": pass, "device_id": device}
		resp, err := doRequest(http.MethodPost, "/login", "", payload)
		if err != nil {
			t.Fatalf("Erro de rede no login: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var res map[string]string
			json.NewDecoder(resp.Body).Decode(&res)
			return resp.StatusCode, res["token"], res["refresh_token"]
		}
		return resp.StatusCode, "", ""
	}

	t.Log("🚀 Iniciando Teste de Troca de Senha...")

	// 1. Criamos a Sessão A (que fará a mudança de senha)
	statusA, jwtTokenA, refreshTokenA := doLogin(originalPassword, "pc_principal")
	if statusA != http.StatusOK {
		t.Fatalf("❌ Falha no login inicial. A sua senha original está correta? Status: %d", statusA)
	}

	// 2. Criamos a Sessão B (Sessão paralela em outro aparelho)
	_, _, refreshTokenB := doLogin(originalPassword, "celular_paralelo")
	t.Log("✅ Logins iniciais simulados com sucesso.")

	// 3. Realiza a troca de senha usando a Sessão A
	changePayload := map[string]string{
		"current_password": originalPassword,
		"new_password":     newPassword,
		"refresh_token":    refreshTokenA, // Enviamos este para ele NÃO ser derrubado
	}

	respChange, err := doRequest(http.MethodPost, "/password/change", jwtTokenA, changePayload)
	if err != nil || respChange.StatusCode != http.StatusOK {
		t.Fatalf("❌ Falha ao tentar trocar a senha. Status: %d", respChange.StatusCode)
	}
	respChange.Body.Close()
	t.Log("✅ Senha alterada com sucesso no banco de dados!")

	// 4. SEGURANÇA: Prova que a Sessão B (celular paralelo) foi expulsa do sistema
	refreshPayloadB := map[string]string{"refresh_token": refreshTokenB}
	respRefB, _ := doRequest(http.MethodPost, "/refresh", "", refreshPayloadB)
	if respRefB.StatusCode != http.StatusUnauthorized {
		t.Fatalf("🚨 Falha Crítica: A sessão do celular NÃO foi derrubada após a troca de senha! Status: %d", respRefB.StatusCode)
	}
	respRefB.Body.Close()
	t.Log("✅ Segurança: Aparelho paralelo foi ejetado corretamente.")

	// 5. Prova que tentar logar com a SENHA ANTIGA agora falha
	badStatus, _, _ := doLogin(originalPassword, "pc_principal")
	if badStatus != http.StatusUnauthorized {
		t.Fatalf("🚨 Falha Crítica: A senha antiga continuou funcionando! Status: %d", badStatus)
	}
	t.Log("✅ Senha antiga revogada com sucesso (401 Unauthorized).")

	// 6. Prova que logar com a NOVA SENHA funciona
	goodStatus, newJwtToken, newRefreshToken := doLogin(newPassword, "pc_principal")
	if goodStatus != http.StatusOK {
		t.Fatalf("🚨 Falha Crítica: A senha nova falhou no login! Status: %d", goodStatus)
	}
	t.Log("✅ Login efetuado com a nova senha perfeitamente!")

	// =====================================================================
	// 7. CLEANUP: Reverte a senha para o estado original!
	// Isso garante que os outros testes do seu projeto continuarão passando amanhã.
	// =====================================================================
	t.Log("🧹 Desfazendo alterações para manter o banco de dados limpo...")
	revertPayload := map[string]string{
		"current_password": newPassword,
		"new_password":     originalPassword,
		"refresh_token":    newRefreshToken,
	}
	respRevert, _ := doRequest(http.MethodPost, "/password/change", newJwtToken, revertPayload)
	if respRevert.StatusCode != http.StatusOK {
		t.Fatalf("🚨 ATENÇÃO: Falha ao reverter a senha! O banco de dados ficou com a senha temporária 'NovaSenha_123_TESTE'. Status: %d", respRevert.StatusCode)
	}

	t.Log("✅ Teste finalizado e banco de dados devolvido intacto!")
}
