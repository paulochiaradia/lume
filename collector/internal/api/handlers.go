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

	kpis, err := db.GetHomeKPIs(s.db, claims.ClientKey)
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

func (s *Server) handleVendasPorDia(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	vendas, err := db.GetVendasPorDia(s.db, claims.ClientKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar vendas por dia")
		return
	}

	writeJSON(w, http.StatusOK, vendas)
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
