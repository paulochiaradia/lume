package mailer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
)

func TestSendRealEmail(t *testing.T) {
	// Procura e carrega o arquivo .env dinamicamente
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

	// Puxa as variáveis de ambiente carregadas
	apiKey := os.Getenv("RESEND_API_KEY")
	meuEmail := os.Getenv("RESEND_EMAIL") // Usando o e-mail que você já cadastrou para os outros testes

	// Trava de segurança para não rodar sem querer ou falhar no servidor do Git (CI/CD)
	if apiKey == "" || meuEmail == "" {
		t.Skip("⏭️ Pulei o teste: RESEND_API_KEY ou EMAIL não encontrados no .env.")
	}

	// 1. Inicializa o nosso serviço
	m, err := New(apiKey)
	if err != nil {
		t.Fatalf("❌ Erro ao iniciar mailer: %v", err)
	}

	// 2. O Template do E-mail
	html := `
	<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #eaeaec; border-radius: 8px;">
		<h1 style="color: #2563eb;">Lume Backend 🚀</h1>
		<p>Olá!</p>
		<p>Se você está lendo isso, significa que a integração do <strong>Golang</strong> com a API do <strong>Resend</strong> foi um sucesso usando o carregamento automático do <code>.env</code>.</p>
		<p>A arquitetura está pronta para os fluxos de Recuperação de Senha e Convite B2B.</p>
		<hr style="border: none; border-top: 1px solid #eaeaea; margin: 20px 0;" />
		<p style="font-size: 12px; color: #888;">Enviado automaticamente pelo robô de testes E2E do Lume.</p>
	</div>
	`

	// 3. Dispara a mensagem
	t.Log("🚀 Disparando o e-mail para os servidores do Resend...")
	err = m.Send(meuEmail, "Sucesso: O Lume agora sabe enviar e-mails via .env! 🎉", html)

	if err != nil {
		t.Fatalf("❌ Erro ao enviar e-mail: %v", err)
	}

	t.Log("✅ E-mail despachado com sucesso! Vá checar a sua caixa de entrada (e o Spam por via das dúvidas).")
}
