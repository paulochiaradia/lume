package api

import (
	"net/http"
	"os"
	"time"
)

// SetSecureCookie cria um cookie de autenticação no padrão de segurança exigido
func SetSecureCookie(w http.ResponseWriter, name, value string, expires time.Time) {
	// Se estiver rodando em produção (com HTTPS), ativa a flag Secure
	isProd := os.Getenv("ENV") == "production"

	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expires,
		Path:     "/",
		HttpOnly: true,                 // Javascript do Frontend não consegue ler (Evita XSS)
		Secure:   isProd,               // Só trafega em HTTPS
		SameSite: http.SameSiteLaxMode, // Proteção contra CSRF
	}

	http.SetCookie(w, cookie)
}
