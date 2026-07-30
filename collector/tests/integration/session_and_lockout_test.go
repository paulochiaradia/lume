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

func TestSessionAndLockoutFlow(t *testing.T) {
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
	validPassword := os.Getenv("PASSWORD")
	t.Logf("EMAIL=%q PASSWORD=%q", email, validPassword)
	wrongPassword := "senha_errada_123"

	// Função helper para fazer requisições com token
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

	// Função helper para fazer Login
	doLogin := func(device string, pass string) (int, string, string) {
		payload := map[string]string{"email": email, "password": pass, "device_id": device}
		resp, err := doRequest(http.MethodPost, "/login", "", payload)
		if err != nil {
			t.Fatalf("Erro de rede no login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var result map[string]string
			json.NewDecoder(resp.Body).Decode(&result)
			return resp.StatusCode, result["token"], result["refresh_token"]
		}
		return resp.StatusCode, "", ""
	}

	// =====================================================================
	// PARTE 1: GERENCIAMENTO DE SESSÕES
	// =====================================================================
	t.Log("🚀 Iniciando testes de Múltiplas Sessões...")

	// 1. Logar com 3 aparelhos diferentes
	status1, _, _ := doLogin("pc_escritorio", validPassword)
	status2, _, _ := doLogin("tablet_sala", validPassword)
	status3, jwt3, ref3 := doLogin("celular_bolso", validPassword)

	if status1 != 200 || status2 != 200 || status3 != 200 {
		t.Fatalf("❌ Falha ao criar múltiplas sessões. Status recebidos: %d, %d, %d", status1, status2, status3)
	}
	t.Log("✅ 3 dispositivos conectados com sucesso.")

	// 2. Verificar a lista de sessões
	resp, _ := doRequest(http.MethodGet, "/sessions", jwt3, nil)
	var sessions []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&sessions)
	resp.Body.Close()

	if len(sessions) < 3 {
		t.Fatalf("❌ A API retornou menos de 3 sessões ativas: %d encontradas", len(sessions))
	}
	t.Log("✅ Rota GET /sessions listou todos os aparelhos.")

	// 3. Revogar o PC especificamente (Usando o JWT 1 para provar que a sessão dele cai)
	resp, _ = doRequest(http.MethodPost, "/sessions/revoke", jwt3, map[string]string{"device_id": "pc_escritorio"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("❌ Falha ao revogar sessão específica. Status: %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ Rota Revoke derrubou o pc_escritorio.")

	// 4. Deslogar todos os OUTROS dispositivos (vai matar o tablet_sala e poupar o celular_bolso)
	resp, _ = doRequest(http.MethodPost, "/sessions/logout-others", jwt3, map[string]string{"refresh_token": ref3})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("❌ Falha no logout-others. Status: %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ Rota Logout-Others eliminou os concorrentes.")

	// 5. Botão de Pânico: Global Logout
	resp, _ = doRequest(http.MethodPost, "/sessions/global-logout", jwt3, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("❌ Falha no global logout. Status: %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ Rota Global Logout varreu as sessões restantes do banco.")

	// =====================================================================
	// PARTE 2: PROTEÇÃO CONTRA FORÇA BRUTA (CONGELAMENTO)
	// =====================================================================
	t.Log("🚀 Iniciando Simulação de Ataque Hacker (Força Bruta)...")

	// Dispara 4 requisições seguidas com a senha ERRADA
	for i := 1; i <= 4; i++ {
		status, _, _ := doLogin("hacker_botnet", wrongPassword)
		if status != http.StatusUnauthorized {
			t.Fatalf("❌ Tentativa %d falhou na segurança. Esperava 401, recebeu %d", i, status)
		}
		t.Logf("   - Strike %d/5: Invasor bloqueado (401)", i)
	}

	// A QUINTA tentativa errada já deve travar a conta e retornar 403
	statusStrike5, _, _ := doLogin("hacker_botnet", wrongPassword)
	if statusStrike5 != http.StatusForbidden {
		t.Fatalf("🚨 FALHA CRÍTICA: O 5º erro deveria retornar 403! Recebeu: %d", statusStrike5)
	}
	t.Log("   - Strike 5/5: Acesso negado e conta congelada no ato (403)!")

	// A Sexta tentativa, MESMO COM A SENHA CERTA, continua retornando 403!
	statusBlocked, _, _ := doLogin("hacker_tentativa_final", validPassword)

	if statusBlocked != http.StatusForbidden {
		t.Fatalf("🚨 FALHA CRÍTICA: O sistema aceitou a senha correta de uma conta bloqueada! Recebeu status: %d", statusBlocked)
	}

	t.Log("✅ SEGURANÇA MÁXIMA: A conta continua blindada contra o dono real (403 Forbidden)!")
}
