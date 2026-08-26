package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/paulochiaradia/lume/collector/internal/mailer"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// Server é o servidor HTTP da API
type Server struct {
	db     *sql.DB
	rdb    *redis.Client
	router *chi.Mux
	http   *http.Server
	mailer *mailer.Service
}

// New cria uma nova instância do servidor injetando Postgres e Redis
func New(db *sql.DB, rdb *redis.Client, mailer *mailer.Service) *Server {
	s := &Server{
		db:     db,
		rdb:    rdb,
		mailer: mailer,
	}
	s.router = s.setupRouter()
	s.http = &http.Server{
		Addr:         ":8080",
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

// Start inicia o servidor HTTP
func (s *Server) Start() error {
	log.Println("api: servidor iniciando na porta 8080")
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("erro ao iniciar servidor: %w", err)
	}
	return nil
}

// Stop para o servidor graciosamente
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.http.Shutdown(ctx); err != nil {
		log.Printf("api: erro ao parar servidor: %v", err)
	}
	log.Println("api: servidor parado")
}

// setupRouter configura todas as rotas e middlewares
func (s *Server) setupRouter() *chi.Mux {
	r := chi.NewRouter()

	// Inicializa os limitadores de taxa com a conexão do Redis
	limiter := NewRateLimiter(s.rdb)

	// ── Middlewares globais ──────────────────────────────────
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// ── CORS ─────────────────────────────────────────────────
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost",
			"http://localhost:3000",
			"http://localhost:8501",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ── Headers de segurança (Atualizado) ────────────────────
	r.Use(SecurityHeaders)

	// ── Rate limiting global (100 req/min por IP) ────────────
	r.Use(limiter.Global)

	// ── Rotas públicas ───────────────────────────────────────
	r.Get("/health", s.handleHealth)

	r.Route("/api/v1", func(r chi.Router) {
		// Health — público, sem auth
		r.Get("/health", s.handleHealth)
		r.Handle("/metrics", promhttp.Handler()) // Prometheus scrape endpoint

		// Auth — sem JWT, mas com proteção rígida contra Força Bruta (10 req/min)
		r.With(limiter.Strict).Post("/auth/login", s.handleLogin)
		r.With(limiter.Strict).Post("/auth/refresh", s.handleRefresh)
		r.With(limiter.Strict).Post("/auth/password/forgot", s.handleForgotPassword)
		r.With(limiter.Strict).Post("/auth/password/reset", s.handleResetPassword)
		r.With(limiter.Strict).Post("/auth/invites/accept", s.handleAcceptInvite)

		// Rotas protegidas — exigem JWT válido
		r.Group(func(r chi.Router) {
			r.Use(s.jwtMiddleware)

			// Core
			r.Get("/auth/me", s.handleAuthMe)
			r.Post("/auth/logout", s.handleLogout)
			r.Get("/auth/sessions", s.handleGetSessions)
			r.Post("/auth/sessions/revoke", s.handleRevokeSession)
			r.Post("/auth/sessions/global-logout", s.handleGlobalLogout)
			r.Post("/auth/sessions/logout-others", s.handleLogoutOthers)
			r.Post("/auth/password/change", s.handleChangePassword)

			// Admin — apenas para usuários com role "admin" ou "gerente"
			r.Post("/admin/invites", s.handleCreateInvite)
			r.Get("/admin/invites", s.handleListInvites)
			r.Delete("/admin/invites/{id}", s.handleRevokeInvite)
			r.Get("/admin/users", s.handleListTeam)
			r.Patch("/admin/users/{id}/deactivate", s.handleDeactivateUser)

			// Home
			r.Get("/home/kpis", s.handleHomeKPIs)

			// Vendas
			r.Get("/vendas/resumo", s.handleVendasResumo)
			r.Get("/vendas/por-dia", s.handleVendasPorDia)
			r.Get("/vendas/tendencia-diaria", s.handleTendenciaDiaria)
			r.Get("/vendas/top-dias", s.handleTopDias)
			r.Get("/vendas/por-hora", s.handleVendasPorHora)
			r.Get("/vendas/mix", s.handleMixVendas)
			r.Get("/vendas/kpis", s.handleVendasKPIs)
			r.Get("/vendas/ranking-vendedores", s.handleRankingVendedores)
			r.Get("/vendas/heatmap", s.handleVendasHeatmap)
			r.Get("/vendas/insights", s.handleVendasInsights)

			// Produtos
			r.Get("/produtos/abc", s.handleProdutosABC)
			r.Get("/produtos/kpis", s.handleProdutosKPIs)
			r.Get("/produtos/matriz", s.handleProdutosMatriz)
			r.Get("/produtos/ranking", s.handleProdutosRanking)
			r.Get("/produtos/elasticidade", s.handleProdutosElasticidade)
			r.Get("/produtos/basket", s.handleProdutosBasket)
			r.Get("/produtos/dead-stock", s.handleProdutosDeadStock)

			// Clientes
			r.Get("/clientes/rfm", s.handleClientesRFM)
			r.Get("/clientes/segmentos", s.handleResumoSegmentos)

			// Insights
			r.Get("/insights", s.handleInsights)

			// Estoque
			r.Get("/estoque/alertas", s.handleEstoqueAlertas)
			r.Get("/estoque/completo", s.handleEstoqueCompleto)
			r.Get("/estoque/reposicao", s.handleEstoqueReposicao)
			r.Get("/estoque/kpis", s.handleEstoqueKPIs)
		})
	})

	return r
}
