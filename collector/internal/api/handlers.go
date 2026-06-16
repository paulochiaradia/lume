package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/paulochiaradia/lume/collector/internal/db"
)

// ── Helpers ──────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// getStartTimeFromPeriod converte o query param ("week", "month", "year") em uma data de corte
func getStartTimeFromPeriod(periodo string) time.Time {
	// TRUQUE PARA TESTES: Congelamos o "Hoje" no último dia da base sintética (31/12/2024).
	// Quando for para produção com vendas reais, basta voltar para: now := time.Now()
	now := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	switch periodo {
	case "month":
		return now.AddDate(0, -1, 0)
	case "year":
		return now.AddDate(-1, 0, 0)
	default: // "week" ou qualquer outro fallback
		return now.AddDate(0, 0, -7)
	}
}

// ── Health ───────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"service": "lume-collector",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// ── Home KPIs ────────────────────────────────────────────────

func (s *Server) handleHomeKPIs(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	// 1. Data base do seu sistema (atualmente congelada no último dia da base de testes)
	// Quando for para produção com dados reais, basta trocar por: baseDate := time.Now()
	baseDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	// 2. Trava o período para o Primeiro Dia do mês atual (Month-to-Date)
	startTime := time.Date(baseDate.Year(), baseDate.Month(), 1, 0, 0, 0, 0, baseDate.Location())

	kpis, err := db.GetHomeKPIs(s.db, claims.ClientKey, startTime)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar KPIs")
		return
	}

	writeJSON(w, http.StatusOK, kpis)
}

// ── Vendas ───────────────────────────────────────────────────

func (s *Server) handleVendasResumo(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	resumo, err := db.GetVendasResumo(s.db, claims.ClientKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar resumo de vendas")
		return
	}

	writeJSON(w, http.StatusOK, resumo)
}

// ── Estoque ──────────────────────────────────────────────────

func (s *Server) handleEstoqueAlertas(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	alertas, err := db.GetEstoqueAlertas(s.db, claims.ClientKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar alertas de estoque")
		return
	}

	writeJSON(w, http.StatusOK, alertas)
}

// ── Produtos ─────────────────────────────────────────────────

func (s *Server) handleProdutosABC(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	produtos, err := db.GetProdutosABC(s.db, claims.ClientKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar produtos ABC")
		return
	}

	writeJSON(w, http.StatusOK, produtos)
}

// handleClientesRFM retorna os clientes classificados por RFM (Recência, Frequência, Valor)
func (s *Server) handleClientesRFM(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	clientes, err := db.GetClientesRFM(s.db, claims.ClientKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar clientes")
		return
	}

	writeJSON(w, http.StatusOK, clientes)
}

func (s *Server) handleResumoSegmentos(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	segmentos, err := db.GetResumoSegmentos(s.db, claims.ClientKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar segmentos")
		return
	}

	writeJSON(w, http.StatusOK, segmentos)
}

func (s *Server) handleEstoqueCompleto(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	estoque, err := db.GetEstoqueCompleto(s.db, claims.ClientKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar estoque")
		return
	}

	writeJSON(w, http.StatusOK, estoque)
}

func (s *Server) handleInsights(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	insights, err := db.GetInsights(s.db, claims.ClientKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar insights")
		return
	}

	writeJSON(w, http.StatusOK, insights)
}

// ── Gráficos Analíticos (React Dashboard) ────────────────────

func (s *Server) handleVendasPorHora(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	periodo := r.URL.Query().Get("periodo")
	startTime := getStartTimeFromPeriod(periodo)

	vendas, err := db.GetVendasPorHora(s.db, claims.ClientKey, startTime)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar vendas por hora")
		return
	}

	writeJSON(w, http.StatusOK, vendas)
}

func (s *Server) handleMixVendas(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	periodo := r.URL.Query().Get("periodo")
	startTime := getStartTimeFromPeriod(periodo)

	mix, err := db.GetMixCategorias(s.db, claims.ClientKey, startTime)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar mix de vendas")
		return
	}

	writeJSON(w, http.StatusOK, mix)
}

// Helpers temporais específicos para as janelas do painel (congelados em 31/12/2024 para a base de testes)
func getStartTimeForComparativo(periodo string) time.Time {
	now := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	switch periodo {
	case "month":
		return now.AddDate(0, -2, 0) // Puxa 2 meses para cobrir o mês atual e o anterior completo
	case "year":
		return now.AddDate(-2, 0, 0) // Puxa 2 anos para cobrir o ano atual e o anterior completo
	default:
		return now.AddDate(0, 0, -14) // Puxa 14 dias para cobrir a semana atual e a anterior completa
	}
}

func getStartTimeForTopDias(periodo string) time.Time {
	switch periodo {
	case "month":
		return time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC) // Início do mês atual da base de teste
	case "year":
		return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Início do ano atual da base de teste
	default:
		return time.Date(2024, 12, 30, 0, 0, 0, 0, time.UTC) // Segunda-feira da última semana ativa
	}
}

// Handlers HTTP para os novos fluxos Server-Side
func (s *Server) handleVendasPorDia(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	periodo := r.URL.Query().Get("periodo")
	if periodo == "" {
		periodo = "month"
	}

	// Data base congelada da base de testes
	baseDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	var startTime time.Time
	var diasParaCortar int

	switch periodo {
	case "week":
		// Busca 13 dias atrás (7 da semana + 6 para aquecer a média móvel)
		startTime = baseDate.AddDate(0, 0, -13)
		diasParaCortar = 6
	case "month":
		// Busca o dia 1º do mês, e recua mais 6 dias no mês anterior para aquecer o cálculo
		primeiroDiaMes := time.Date(baseDate.Year(), baseDate.Month(), 1, 0, 0, 0, 0, baseDate.Location())
		startTime = primeiroDiaMes.AddDate(0, 0, -6)
		diasParaCortar = 6
	case "year":
		// Num gráfico anual, não costumamos cortar dias de aquecimento (o impacto visual é mínimo)
		startTime = time.Date(baseDate.Year(), 1, 1, 0, 0, 0, 0, baseDate.Location())
		diasParaCortar = 0
	}

	series, err := db.GetVendasPorDia(s.db, claims.ClientKey, startTime)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao calcular vendas por dia")
		return
	}

	// Remove os dias "fantasmas" que usamos apenas para o Postgres calcular a média com precisão
	if len(series) > diasParaCortar {
		series = series[diasParaCortar:]
	}

	writeJSON(w, http.StatusOK, series)
}
func (s *Server) handleTopDias(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	periodo := r.URL.Query().Get("periodo")
	startTime := getStartTimeForTopDias(periodo)

	vendas, err := db.GetTopDias(s.db, claims.ClientKey, startTime)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar top dias analíticos")
		return
	}

	writeJSON(w, http.StatusOK, vendas)
}

func (s *Server) handleVendasKPIs(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	periodo := r.URL.Query().Get("periodo")
	if periodo == "" {
		periodo = "month"
	}

	// Data base congelada do seu banco (Substituir por time.Now() em prod)
	baseDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	var currentStart, previousStart time.Time

	switch periodo {
	case "week": // Últimos 7 dias vs 7 dias anteriores
		currentStart = time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day()-6, 0, 0, 0, 0, baseDate.Location())
		previousStart = currentStart.AddDate(0, 0, -7)
	case "month": // Mês Atual vs Mês Anterior
		currentStart = time.Date(baseDate.Year(), baseDate.Month(), 1, 0, 0, 0, 0, baseDate.Location())
		previousStart = currentStart.AddDate(0, -1, 0)
	case "year": // Ano Atual vs Ano Anterior
		currentStart = time.Date(baseDate.Year(), 1, 1, 0, 0, 0, 0, baseDate.Location())
		previousStart = currentStart.AddDate(-1, 0, 0)
	}

	kpis, err := db.GetVendasKPIsTrend(s.db, claims.ClientKey, currentStart, previousStart)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao calcular kpis de vendas")
		return
	}

	writeJSON(w, http.StatusOK, kpis)
}
