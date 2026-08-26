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

// Define o endereço base como se fôssemos o frontend batendo no Nginx
const baseURL = "http://localhost/api/v1/auth"

func TestAuthenticationFlowE2E(t *testing.T) {
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
	password := os.Getenv("PASSWORD")
	t.Logf("EMAIL=%q PASSWORD=%q", email, password)

	// ---------------------------------------------------------
	// PASSO 1: O Login (Happy Path)
	// ---------------------------------------------------------
	loginPayload := map[string]interface{}{
		"email":     email,
		"password":  password,
		"device_id": "go_e2e_robot",
	}

	body, _ := json.Marshal(loginPayload)
	resp, err := http.Post(baseURL+"/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("❌ Nginx ou Docker inativo: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyErr, _ := io.ReadAll(resp.Body)
		t.Fatalf("❌ Login falhou. Esperado 200, recebido %d. Erro: %s", resp.StatusCode, string(bodyErr))
	}

	// Extrai os tokens salvando-os em variáveis
	var loginResult map[string]string
	json.NewDecoder(resp.Body).Decode(&loginResult)

	jwtToken := loginResult["token"]
	refreshToken := loginResult["refresh_token"]

	if jwtToken == "" || refreshToken == "" {
		t.Fatal("❌ A API não retornou os tokens de acesso.")
	}
	t.Log("✅ Passo 1: Login realizado com sucesso!")

	// ---------------------------------------------------------
	// PASSO 2: O Refresh (Renovando o JWT)
	// ---------------------------------------------------------
	refreshPayload := map[string]string{"refresh_token": refreshToken}
	refBody, _ := json.Marshal(refreshPayload)

	respRefresh, err := http.Post(baseURL+"/refresh", "application/json", bytes.NewBuffer(refBody))
	if err != nil {
		t.Fatalf("❌ Falha na rede ao acionar /refresh: %v", err)
	}
	defer respRefresh.Body.Close()

	if respRefresh.StatusCode != http.StatusOK {
		bodyErr, _ := io.ReadAll(respRefresh.Body)
		t.Fatalf("❌ Refresh falhou. Esperado 200, recebido %d. Erro: %s", respRefresh.StatusCode, string(bodyErr))
	}

	var refreshResult map[string]string
	json.NewDecoder(respRefresh.Body).Decode(&refreshResult)
	newJWT := refreshResult["token"]
	t.Log("✅ Passo 2: Novo JWT gerado pelo Refresh Token!")

	// ---------------------------------------------------------
	// PASSO 3: O Logout (Destruindo a sessão)
	// ---------------------------------------------------------
	// O Logout precisa de um request customizado para enviarmos o Header Bearer Token
	reqLogout, _ := http.NewRequest(http.MethodPost, baseURL+"/logout", bytes.NewBuffer(refBody))
	reqLogout.Header.Set("Content-Type", "application/json")
	reqLogout.Header.Set("Authorization", "Bearer "+newJWT)

	client := &http.Client{}
	respLogout, err := client.Do(reqLogout)
	if err != nil {
		t.Fatalf("❌ Falha na rede ao acionar /logout: %v", err)
	}
	defer respLogout.Body.Close()

	if respLogout.StatusCode != http.StatusOK {
		bodyErr, _ := io.ReadAll(respLogout.Body)
		t.Fatalf("❌ Logout falhou. Esperado 200, recebido %d. Erro: %s", respLogout.StatusCode, string(bodyErr))
	}
	t.Log("✅ Passo 3: Logout executado e sessão explodida no banco!")

	// ---------------------------------------------------------
	// PASSO 4: A Prova do Crime (Refresh Bloqueado)
	// ---------------------------------------------------------
	respFinal, err := http.Post(baseURL+"/refresh", "application/json", bytes.NewBuffer(refBody))
	if err != nil {
		t.Fatalf("❌ Falha na rede no último request: %v", err)
	}
	defer respFinal.Body.Close()

	// Aqui a mágica acontece: nós EXIGIMOS que o status seja 401 Unauthorized
	if respFinal.StatusCode != http.StatusUnauthorized {
		bodyErr, _ := io.ReadAll(respFinal.Body)
		t.Fatalf("🚨 FALHA CRÍTICA DE SEGURANÇA! O servidor aceitou um refresh de uma sessão deslogada. Recebido: HTTP %d. %s", respFinal.StatusCode, string(bodyErr))
	}
	t.Log("✅ Passo 4: Segurança absoluta comprovada. O servidor barrou a tentativa zumbi.")
}
