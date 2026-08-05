package integration_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func TestPasswordRecoveryFlow(t *testing.T) {
	// 🛑 DADOS DO SEU USUÁRIO DE TESTE
	email := os.Getenv("TEST_EMAIL")
	originalPassword := os.Getenv("TEST_PASSWORD")
	newPassword := os.Getenv("TEST_NEW_PASSWORD")

	// Helper para requisições
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

	t.Log("🚀 Iniciando Teste E2E: Recuperação de Senha...")

	// =========================================================================
	// 1. SOLICITA A RECUPERAÇÃO (Porta da frente)
	// =========================================================================
	forgotPayload := map[string]string{"email": email}
	respForgot, _ := doRequest(http.MethodPost, "/password/forgot", "", forgotPayload)
	if respForgot.StatusCode != http.StatusOK {
		t.Fatalf("❌ Falha ao solicitar recuperação. Status: %d", respForgot.StatusCode)
	}
	respForgot.Body.Close()
	t.Log("✅ Pedido de recuperação aceito pela API.")

	// =========================================================================
	// 2. A INVASÃO (Porta dos fundos: Conecta no DB e rouba o token)
	// =========================================================================
	t.Log("🕵️‍♂️ Robô invadindo o banco de dados para interceptar o Token...")

	connStr := "user=lume_user password=lume_dev_123 dbname=lume sslmode=disable host=localhost port=5432"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("❌ Erro crítico: Robô não conseguiu conectar no banco: %v", err)
	}
	defer db.Close()

	var stolenToken string
	// Busca o token mais recente e não usado deste e-mail
	query := `
		SELECT pr.token 
		FROM lume_system.password_resets pr
		JOIN lume_system.users u ON u.id = pr.user_id
		WHERE u.email = $1 AND pr.used = FALSE
		ORDER BY pr.created_at DESC 
		LIMIT 1;
	`
	err = db.QueryRow(query, email).Scan(&stolenToken)
	if err != nil {
		t.Fatalf("❌ Robô falhou ao roubar o token no banco (não encontrou): %v", err)
	}
	t.Logf("✅ Token interceptado com sucesso: %s...", stolenToken[:10])

	// =========================================================================
	// 3. O ATAQUE (Usa o token para mudar a senha)
	// =========================================================================
	resetPayload := map[string]string{
		"token":        stolenToken,
		"new_password": newPassword,
	}
	respReset, _ := doRequest(http.MethodPost, "/password/reset", "", resetPayload)
	if respReset.StatusCode != http.StatusOK {
		t.Fatalf("❌ Falha ao redefinir a senha com o token. Status: %d", respReset.StatusCode)
	}
	respReset.Body.Close()
	t.Log("✅ Senha redefinida com o token roubado!")

	// =========================================================================
	// 4. A VALIDAÇÃO (Tenta logar com a nova senha)
	// =========================================================================
	loginPayload := map[string]string{"email": email, "password": newPassword, "device_id": "teste_e2e_recovery"}
	respLogin, _ := doRequest(http.MethodPost, "/login", "", loginPayload)
	if respLogin.StatusCode != http.StatusOK {
		t.Fatalf("🚨 Falha Crítica: Login rejeitou a nova senha! Status: %d", respLogin.StatusCode)
	}

	// Extrai os tokens para o cleanup
	var res map[string]string
	json.NewDecoder(respLogin.Body).Decode(&res)
	jwtToken := res["token"]
	refreshToken := res["refresh_token"]
	respLogin.Body.Close()

	t.Log("✅ Login efetuado com a NOVA senha. Sistema 100% validado!")

	// =========================================================================
	// 5. CLEANUP: DEIXA O BANCO COMO ENCONTROU
	// =========================================================================
	t.Log("🧹 Limpando rastros e devolvendo a senha original...")
	revertPayload := map[string]string{
		"current_password": newPassword,
		"new_password":     originalPassword,
		"refresh_token":    refreshToken,
	}
	// Reutilizamos a rota de "Change Password" (do passo anterior) porque já estamos logados!
	respRevert, _ := doRequest(http.MethodPost, "/password/change", jwtToken, revertPayload)
	if respRevert.StatusCode != http.StatusOK {
		t.Fatalf("🚨 ATENÇÃO: Falha ao reverter a senha! Status: %d", respRevert.StatusCode)
	}
	respRevert.Body.Close()
	t.Log("✅ Limpeza concluída. Banco intacto!")
}
