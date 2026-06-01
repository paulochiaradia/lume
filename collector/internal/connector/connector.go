package connector

import "time"

// RawRecord representa um registro bruto extraído do ERP
// antes de qualquer normalização — é um mapa genérico
type RawRecord map[string]string

// SyncResult resume o resultado de uma sincronização
type SyncResult struct {
	RecordsRead    int
	RecordsWritten int
	Errors         []string
}

// Connector é a interface que todo conector de ERP deve implementar
// O scheduler só conhece essa interface — nunca o conector específico
type Connector interface {
	// Extract extrai os registros brutos do ERP
	// Retorna os registros e um possível erro
	Extract() ([]RawRecord, error)

	// Validate verifica se a configuração do conector está correta
	// Deve ser chamado na inicialização antes do primeiro Extract
	Validate() error

	// GetSchedule retorna a frequência de sincronização no formato cron
	// Exemplo: "*/30 * * * *" para a cada 30 minutos
	GetSchedule() string

	// GetType retorna o identificador do tipo de conector
	// Exemplo: "csv", "bling", "omie", "sql"
	GetType() string
}

// Config contém as configurações necessárias para inicializar um conector
type Config struct {
	ClientKey  string
	ERPType    string
	FilePath   string // para conectores de arquivo
	APIURL     string // para conectores de API
	APIKey     string // para conectores de API
	DBHost     string // para conectores SQL
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	Schedule   string
	Extra      map[string]string // configurações adicionais por conector
}

// NewConfig cria uma Config a partir das variáveis de ambiente do cliente
func NewConfig(clientKey, erpType string, env map[string]string) Config {
	return Config{
		ClientKey:  clientKey,
		ERPType:    erpType,
		FilePath:   env["ERP_FILE_PATH"],
		APIURL:     env["ERP_API_URL"],
		APIKey:     env["ERP_API_KEY"],
		DBHost:     env["ERP_DB_HOST"],
		DBPort:     env["ERP_DB_PORT"],
		DBName:     env["ERP_DB_NAME"],
		DBUser:     env["ERP_DB_USER"],
		DBPassword: env["ERP_DB_PASSWORD"],
		Schedule:   env["SYNC_SCHEDULE"],
		Extra:      env,
	}
}

// DefaultSchedule é o schedule padrão caso não seja configurado
const DefaultSchedule = "*/30 * * * *"

// SyncTimeout é o tempo máximo que um sync pode durar
const SyncTimeout = 10 * time.Minute
