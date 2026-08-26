package scheduler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/paulochiaradia/lume/collector/internal/connector"
	"github.com/paulochiaradia/lume/collector/internal/db"
)

// buildConnectorConfig monta a configuração efetiva do cliente a partir do banco
// e do arquivo .env montado para o cliente, quando existir.
func buildConnectorConfig(client db.Client) (connector.Config, error) {
	env, err := loadClientEnv(client.ClientKey, client.ERPConfig)
	if err != nil {
		slog.Error("falha ao carregar variaveis de ambiente do cliente",
			slog.String("event", "scheduler_load_env_error"),
			slog.String("tenant_id", client.ClientKey),
			slog.String("error", err.Error()),
		)
		return connector.Config{}, err
	}

	cfg := connector.NewConfig(client.ClientKey, client.ERPType, env)
	if cfg.Schedule == "" {
		cfg.Schedule = connector.DefaultSchedule
	}

	return cfg, nil
}

// loadClientEnv reconstrói o mapa de variáveis do cliente a partir de erp_config
// e do arquivo clients/<client_key>/.env, nessa ordem.
func loadClientEnv(clientKey string, erpConfig []byte) (map[string]string, error) {
	env := map[string]string{}

	if len(erpConfig) > 0 && string(erpConfig) != "null" {
		if err := json.Unmarshal(erpConfig, &env); err != nil {
			return nil, fmt.Errorf("erp_config inválido para o cliente %s: %w", clientKey, err)
		}
	}

	envLoadedFromFile := false

	for _, candidate := range []string{
		filepath.Join("clients", clientKey, ".env"),
		filepath.Join("/app/clients", clientKey, ".env"),
	} {
		info, err := os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("erro ao verificar config do cliente %s: %w", clientKey, err)
		}
		if info.IsDir() {
			continue
		}

		fileEnv, err := godotenv.Read(candidate)
		if err != nil {
			// [SLOG] Logamos um WARN pois um arquivo existe, mas está corrompido/ilegível
			slog.Warn("arquivo .env de cliente encontrado porem ilegivel",
				slog.String("tenant_id", clientKey),
				slog.String("file_path", candidate),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("erro ao ler config do cliente %s em %s: %w", clientKey, candidate, err)
		}

		for key, value := range fileEnv {
			env[key] = value
		}

		envLoadedFromFile = true
		break
	}

	// [SLOG] Log de debug útil em ambiente de produção para garantir que a infra montou os volumes corretos
	if envLoadedFromFile {
		slog.Info("configuracao extra (arquivo .env) anexada com sucesso",
			slog.String("event", "scheduler_env_file_loaded"),
			slog.String("tenant_id", clientKey),
		)
	}

	return env, nil
}
