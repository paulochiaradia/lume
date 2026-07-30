package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ── Helper para extrair IP Real atrás do Proxy ────────────────
func getRealIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return strings.Split(ip, ",")[0]
	}
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

// ── Headers de Segurança (Auditoria Enterprise) ───────────────
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Obrigatórios pela Auditoria
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Proteções padrão
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}

// ── Rate Limiter com Redis ────────────────────────────────────

// RateLimiter agrupa os middlewares que dependem do Redis
type RateLimiter struct {
	redisClient *redis.Client
}

// NewRateLimiter inicializa os middlewares de limite de taxa injetando o Redis
func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{
		redisClient: rdb,
	}
}

// checkRate executa a lógica atômica no Redis (Fixed Window)
func (rl *RateLimiter) checkRate(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	// Incrementa o contador para esta chave
	count, err := rl.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// Se for o primeiro acesso nesta janela, define o tempo de expiração
	if count == 1 {
		rl.redisClient.Expire(ctx, key, window)
	}

	return count <= limit, nil
}

// Global — 100 requisições por MINUTO por IP
func (rl *RateLimiter) Global(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getRealIP(r)
		key := "rate_limit:global:" + ip

		// Limite de 100 requisições por minuto conforme a auditoria
		allowed, err := rl.checkRate(r.Context(), key, 100, time.Minute)

		if err != nil || !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"message":"Muitas requisições, tente novamente mais tarde"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Strict — 10 requisições por MINUTO por IP (Para Rotas Críticas como Login)
func (rl *RateLimiter) Strict(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getRealIP(r)
		key := "rate_limit:strict:" + ip

		// Limite estrito de 10 requisições por minuto contra força bruta
		allowed, err := rl.checkRate(r.Context(), key, 10, time.Minute)

		if err != nil || !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"message":"Múltiplas tentativas falhas. Bloqueio temporário ativo."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
