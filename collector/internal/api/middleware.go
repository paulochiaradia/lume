package api

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// ── Headers de segurança ─────────────────────────────────────

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// ── Rate limiting ────────────────────────────────────────────

// ipLimiter mantém um limiter por IP
type ipLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
}

var (
	globalLimiter = &ipLimiter{limiters: make(map[string]*rate.Limiter)}
	strictLimiter = &ipLimiter{limiters: make(map[string]*rate.Limiter)}
)

func (il *ipLimiter) get(ip string, r rate.Limit, b int) *rate.Limiter {
	il.mu.Lock()
	defer il.mu.Unlock()

	if limiter, exists := il.limiters[ip]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(r, b)
	il.limiters[ip] = limiter
	return limiter
}

// globalRateLimiter — 100 requisições por segundo por IP
func globalRateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := globalLimiter.get(r.RemoteAddr, 100, 200)
		if !limiter.Allow() {
			http.Error(w, `{"error":"rate limit excedido"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// strictRateLimiter — 10 requisições por segundo por IP para rotas autenticadas
func strictRateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := strictLimiter.get(r.RemoteAddr, 10, 20)
		if !limiter.Allow() {
			http.Error(w, `{"error":"rate limit excedido"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
