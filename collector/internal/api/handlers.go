package api

import (
	"encoding/json"
	"net/http"
	"time"
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

// ── Placeholders — serão implementados nas próximas tarefas ──

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "em implementação"})
}

func (s *Server) handleHomeKPIs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "em implementação"})
}

func (s *Server) handleVendasResumo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "em implementação"})
}

func (s *Server) handleVendasPorDia(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "em implementação"})
}

func (s *Server) handleEstoqueAlertas(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "em implementação"})
}

func (s *Server) handleProdutosABC(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "em implementação"})
}
