package scheduler

import (
	"context"
	"database/sql"
	"log"
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
	log.Println("scheduler: iniciando...")

	clients, err := db.GetActiveClients(s.db)
	if err != nil {
		return err
	}

	if len(clients) == 0 {
		log.Println("scheduler: nenhum cliente ativo encontrado")
	}

	for _, client := range clients {
		c := client // captura para a goroutine

		cfg := connector.Config{
			ClientKey: c.ClientKey,
			ERPType:   c.ERPType,
			Schedule:  connector.DefaultSchedule,
		}

		schedule := cfg.Schedule
		log.Printf("scheduler: registrando job para cliente %s (schedule: %s)", c.ClientKey, schedule)

		s.cron.AddFunc(schedule, func() {
			s.runSync(c.ClientKey, c.ID, cfg)
		})
	}

	s.cron.Start()
	log.Printf("scheduler: %d jobs registrados e rodando", len(clients))
	return nil
}

// Stop para o scheduler graciosamente
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	select {
	case <-ctx.Done():
		log.Println("scheduler: parado com sucesso")
	case <-time.After(30 * time.Second):
		log.Println("scheduler: timeout ao parar")
	}
}

// runSync executa a sincronização de um cliente
func (s *Scheduler) runSync(clientKey, clientID string, cfg connector.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("scheduler: iniciando sync para cliente %s", clientKey)

	// Registra início no etl_log
	logID, err := db.InsertETLLog(s.db, clientID, cfg.ERPType)
	if err != nil {
		log.Printf("scheduler: erro ao criar etl_log para %s: %v", clientKey, err)
		return
	}

	// Cria o conector
	conn, err := connector.Factory(cfg)
	if err != nil {
		db.UpdateETLLogError(s.db, logID, err.Error())
		log.Printf("scheduler: erro ao criar conector para %s: %v", clientKey, err)
		return
	}

	// Valida a configuração
	if err := conn.Validate(); err != nil {
		db.UpdateETLLogError(s.db, logID, err.Error())
		log.Printf("scheduler: configuração inválida para %s: %v", clientKey, err)
		return
	}

	// Extrai os dados com timeout
	ctx, cancel := context.WithTimeout(context.Background(), connector.SyncTimeout)
	defer cancel()

	_ = ctx // será usado quando implementarmos Extract com context

	records, err := conn.Extract()
	if err != nil {
		db.UpdateETLLogError(s.db, logID, err.Error())
		log.Printf("scheduler: erro ao extrair dados de %s: %v", clientKey, err)
		return
	}

	// Normaliza e carrega
	l := loader.New(s.db, clientKey)
	totalWritten := 0

	vendas, _ := normalizer.NormalizeVendas(records)
	written, err := l.LoadVendas(vendas)
	if err != nil {
		log.Printf("scheduler: erro ao carregar vendas de %s: %v", clientKey, err)
	} else {
		totalWritten += written
	}

	// Registra sucesso
	db.UpdateETLLogSuccess(s.db, logID, len(records), totalWritten)
	log.Printf("scheduler: sync concluído para %s — %d registros processados", clientKey, totalWritten)
}
