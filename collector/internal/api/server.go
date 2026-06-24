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
)

// Server é o servidor HTTP da API
type Server struct {
	db     *sql.DB
	router *chi.Mux
	http   *http.Server
}

// New cria uma nova instância do servidor
func New(db *sql.DB) *Server {
	s := &Server{db: db}
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

	// ── Headers de segurança ─────────────────────────────────
	r.Use(securityHeaders)

	// ── Rate limiting global ─────────────────────────────────
	r.Use(globalRateLimiter)

	// ── Rotas públicas ───────────────────────────────────────
	r.Get("/health", s.handleHealth)

	r.Route("/api/v1", func(r chi.Router) {
		// Health — público, sem auth
		r.Get("/health", s.handleHealth)

		// Auth — sem JWT
		r.Post("/auth/login", s.handleLogin)

		// Rotas protegidas — exigem JWT válido
		r.Group(func(r chi.Router) {
			r.Use(s.jwtMiddleware)
			r.Use(strictRateLimiter)

			// Home
			r.Get("/home/kpis", s.handleHomeKPIs)

			// Vendas
			r.Get("/vendas/resumo", s.handleVendasResumo)
			r.Get("/vendas/por-dia", s.handleVendasPorDia)             // ← A Home usa esse
			r.Get("/vendas/tendencia-diaria", s.handleTendenciaDiaria) // ← A página de Vendas usa esse!
			r.Get("/vendas/top-dias", s.handleTopDias)                 // ← Agora ele existe e não vai dar erro
			r.Get("/vendas/por-hora", s.handleVendasPorHora)
			r.Get("/vendas/mix", s.handleMixVendas)
			r.Get("/vendas/kpis", s.handleVendasKPIs)
			r.Get("/vendas/ranking-vendedores", s.handleRankingVendedores)
			r.Get("/vendas/heatmap", s.handleVendasHeatmap)
			r.Get("/vendas/insights", s.handleVendasInsights) // ← Nova rota isolada!

			// Estoque
			r.Get("/estoque/alertas", s.handleEstoqueAlertas)
			r.Get("/estoque/completo", s.handleEstoqueCompleto)

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

		})
	})

	return r
}
