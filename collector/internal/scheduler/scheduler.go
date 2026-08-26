package scheduler

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/paulochiaradia/lume/collector/internal/connector"
	"github.com/paulochiaradia/lume/collector/internal/db"
	"github.com/paulochiaradia/lume/collector/internal/loader"
	"github.com/paulochiaradia/lume/collector/internal/normalizer"
	"github.com/robfig/cron/v3"
)

// Scheduler orquestra os jobs de coleta de todos os clientes
type Scheduler struct {
	db   *sql.DB
	cron *cron.Cron
	mu   sync.Mutex
}

// New cria uma nova instância do Scheduler
func New(database *sql.DB) *Scheduler {
	return &Scheduler{
		db:   database,
		cron: cron.New(),
	}
}

// Start carrega os clientes ativos e registra os jobs
func (s *Scheduler) Start() error {
	slog.Info("iniciando orquestrador de etl", slog.String("event", "scheduler_start"))

	clients, err := db.GetActiveClients(s.db)
	if err != nil {
		slog.Error("erro ao buscar clientes ativos para agendamento", slog.String("error", err.Error()))
		return err
	}

	if len(clients) == 0 {
		slog.Warn("nenhum cliente ativo encontrado para agendamento", slog.String("event", "scheduler_no_clients"))
	}

	for _, client := range clients {
		c := client // captura para a goroutine

		cfg, err := buildConnectorConfig(c)
		if err != nil {
			slog.Warn("cliente ignorado no agendamento por erro de configuracao",
				slog.String("event", "scheduler_client_config_error"),
				slog.String("tenant_id", c.ClientKey),
				slog.String("error", err.Error()),
			)
			continue
		}

		schedule := cfg.Schedule
		slog.Info("registrando job de etl",
			slog.String("event", "scheduler_job_registered"),
			slog.String("tenant_id", c.ClientKey),
			slog.String("schedule", schedule),
		)

		go s.runSync(c.ClientKey, c.ID)

		s.cron.AddFunc(schedule, func() {
			s.runSync(c.ClientKey, c.ID)
		})
	}

	s.cron.Start()
	slog.Info("orquestrador rodando",
		slog.String("event", "scheduler_started"),
		slog.Int("jobs_registrados", len(clients)),
	)
	return nil
}

// Stop para o scheduler graciosamente
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	select {
	case <-ctx.Done():
		slog.Info("orquestrador de etl parado com sucesso", slog.String("event", "scheduler_stopped"))
	case <-time.After(30 * time.Second):
		slog.Error("timeout ao tentar parar o orquestrador", slog.String("event", "scheduler_stop_timeout"))
	}
}

// runSync executa a sincronização de um cliente
func (s *Scheduler) runSync(clientKey, clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("iniciando sync de etl",
		slog.String("event", "etl_sync_started"),
		slog.String("tenant_id", clientKey),
	)

	client, err := db.GetClientByID(s.db, clientID)
	if err != nil {
		slog.Error("erro ao recarregar configuracao do cliente no sync",
			slog.String("event", "etl_load_client_error"),
			slog.String("tenant_id", clientKey),
			slog.String("error", err.Error()),
		)
		return
	}

	cfg, err := buildConnectorConfig(*client)
	if err != nil {
		slog.Error("erro ao montar configuracao no sync",
			slog.String("event", "etl_build_config_error"),
			slog.String("tenant_id", clientKey),
			slog.String("error", err.Error()),
		)
		return
	}

	// Registra início no etl_log (Tabela no banco)
	logID, err := db.InsertETLLog(s.db, clientID, cfg.ERPType)
	if err != nil {
		slog.Error("erro ao criar registro de etl_log",
			slog.String("event", "etl_db_log_error"),
			slog.String("tenant_id", clientKey),
			slog.String("error", err.Error()),
		)
		return
	}

	// Cria o conector
	conn, err := connector.Factory(cfg)
	if err != nil {
		db.UpdateETLLogError(s.db, logID, err.Error())
		slog.Error("erro ao instanciar conector",
			slog.String("event", "etl_connector_factory_error"),
			slog.String("tenant_id", clientKey),
			slog.String("etl_log_id", logID),
			slog.String("error", err.Error()),
		)
		return
	}

	// Valida a configuração
	if err := conn.Validate(); err != nil {
		db.UpdateETLLogError(s.db, logID, err.Error())
		slog.Warn("configuracao de conector invalida",
			slog.String("event", "etl_validation_error"),
			slog.String("tenant_id", clientKey),
			slog.String("etl_log_id", logID),
			slog.String("error", err.Error()),
		)
		return
	}

	// Extrai os dados com timeout
	ctx, cancel := context.WithTimeout(context.Background(), connector.SyncTimeout)
	defer cancel()

	_ = ctx // será usado quando implementarmos Extract com context

	records, err := conn.Extract()
	if err != nil {
		db.UpdateETLLogError(s.db, logID, err.Error())
		slog.Error("falha critica na extracao de dados",
			slog.String("event", "etl_extraction_error"),
			slog.String("tenant_id", clientKey),
			slog.String("etl_log_id", logID),
			slog.String("error", err.Error()),
		)
		return
	}

	// Normaliza e carrega
	l := loader.New(s.db, clientKey)
	totalWritten := 0

	vendas, _ := normalizer.NormalizeVendas(records)
	written, err := l.LoadVendas(vendas)
	if err != nil {
		slog.Error("erro ao carregar vendas no banco",
			slog.String("event", "etl_load_error"),
			slog.String("tenant_id", clientKey),
			slog.String("etl_log_id", logID),
			slog.String("error", err.Error()),
		)
	} else {
		totalWritten += written
	}

	// Registra sucesso
	db.UpdateETLLogSuccess(s.db, logID, len(records), totalWritten)

	slog.Info("sync de etl concluido com sucesso",
		slog.String("event", "etl_sync_success"),
		slog.String("tenant_id", clientKey),
		slog.String("etl_log_id", logID),
		slog.Int("total_recebido", len(records)),
		slog.Int("total_escrito", totalWritten),
	)
}
