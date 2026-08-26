package api

import (
	"log/slog"
	"net/http"

	"github.com/paulochiaradia/lume/collector/internal/db"
)

// -----------------------------------------------------------------------------
// 1. O Admin lista a sua equipe
// -----------------------------------------------------------------------------
func (s *Server) handleListTeam(w http.ResponseWriter, r *http.Request) {
	// Captura de IP para rastreabilidade de acessos indevidos
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	claims := getClaims(r)
	if claims == nil {
		slog.Error("falha critica: rota protegida acessada sem claims",
			slog.String("event", "missing_claims_in_protected_route"),
			slog.String("ip", ip),
			slog.String("route", r.URL.Path),
		)
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminClientID := claims.TenantID
	adminRole := claims.Role

	// 2. Regra de Negócio: Apenas cargos de chefia podem ver a lista completa
	if adminRole != "admin" && adminRole != "gerente" {
		// [SLOG] Tentativa de quebra de privilégio (escalonamento)
		slog.Warn("tentativa de acesso nao autorizado: listar equipe",
			slog.String("event", "unauthorized_access_attempt"),
			slog.String("user_id", claims.UserID),
			slog.String("role", adminRole),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusForbidden, "apenas administradores podem visualizar a equipe")
		return
	}

	// 3. Busca a equipe inteira do Tenant
	team, err := db.GetUsersByClient(s.db, adminClientID)
	if err != nil {
		slog.Error("erro ao listar equipe no bd",
			slog.String("event", "list_team_db_error"),
			slog.String("tenant_id", adminClientID),
			slog.String("error", err.Error()),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusInternalServerError, "erro ao buscar usuários")
		return
	}

	// 4. Retorna a lista limpa (UserSummary, sem senhas).
	// Não logamos o sucesso (INFO) aqui para não poluir os logs em uma rota de leitura constante.
	writeJSON(w, http.StatusOK, team)
}

// -----------------------------------------------------------------------------
// 2. O Admin desativa (demite) um funcionário
// -----------------------------------------------------------------------------
func (s *Server) handleDeactivateUser(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	claims := getClaims(r)
	if claims == nil {
		slog.Error("falha critica: rota protegida acessada sem claims",
			slog.String("event", "missing_claims_in_protected_route"),
			slog.String("ip", ip),
			slog.String("route", r.URL.Path),
		)
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	adminClientID := claims.TenantID
	adminRole := claims.Role
	adminID := claims.UserID

	// 1. Regra de Negócio: Apenas chefia desliga
	if adminRole != "admin" && adminRole != "gerente" {
		slog.Warn("tentativa de acesso nao autorizado: desativar usuario",
			slog.String("event", "unauthorized_access_attempt"),
			slog.String("user_id", adminID),
			slog.String("role", adminRole),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusForbidden, "apenas administradores podem desativar usuários")
		return
	}

	// 2. Extrai o ID do alvo da URL (ex: PATCH /admin/users/1234-abcd)
	targetUserID := r.PathValue("id")
	if targetUserID == "" {
		writeError(w, http.StatusBadRequest, "ID do usuário é obrigatório")
		return
	}

	// 3. Regra de Segurança Crítica: Um Admin não pode desativar a si mesmo!
	if targetUserID == adminID {
		slog.Warn("tentativa de auto-desativacao bloqueada",
			slog.String("event", "self_deactivation_attempt"),
			slog.String("user_id", adminID),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusForbidden, "você não pode desativar sua própria conta")
		return
	}

	// 4. Aciona o Kill Switch no banco
	err := db.DeactivateUser(s.db, targetUserID, adminClientID)
	if err != nil {
		// Se o ID não existir ou o cara for de outra empresa, o banco recusa.
		slog.Warn("falha ao desativar usuario: nao encontrado ou erro no bd",
			slog.String("event", "deactivate_user_failed"),
			slog.String("admin_id", adminID),
			slog.String("target_id", targetUserID),
			slog.String("error", err.Error()),
			slog.String("ip", ip),
		)
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// 5. Sucesso! Grava a auditoria e a telemetria (Evento Crítico)

	// Registro permanente na tabela de auditoria do banco
	_ = db.CreateAuditLog(s.db, adminID, adminClientID, "user_deactivated_"+targetUserID, ip, r.UserAgent())

	// Telemetria imediata para o Grafana/Logs
	slog.Info("usuario desativado com sucesso (kill switch acionado)",
		slog.String("event", "user_deactivated"),
		slog.String("admin_id", adminID),
		slog.String("target_id", targetUserID),
		slog.String("tenant_id", adminClientID),
		slog.String("ip", ip),
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "usuário desativado com sucesso. todos os acessos foram revogados.",
	})
}
