package scheduler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paulochiaradia/lume/collector/internal/db"
)

func TestLoadClientEnvUsesFileAndDbConfig(t *testing.T) {
	tempDir := t.TempDir()
	clientDir := filepath.Join(tempDir, "clients", "loja_teste")
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		t.Fatalf("mkdir client dir: %v", err)
	}

	configFile := filepath.Join(clientDir, ".env")
	content := "ERP_FILE_PATH=/app/testdata/vendas_teste.csv\nSYNC_SCHEDULE=*/15 * * * *\n"
	if err := os.WriteFile(configFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	env, err := loadClientEnv("loja_teste", []byte(`{"ERP_API_KEY":"abc"}`))
	if err != nil {
		t.Fatalf("loadClientEnv: %v", err)
	}

	if got := env["ERP_API_KEY"]; got != "abc" {
		t.Fatalf("expected ERP_API_KEY from db config, got %q", got)
	}
	if got := env["ERP_FILE_PATH"]; got != "/app/testdata/vendas_teste.csv" {
		t.Fatalf("expected ERP_FILE_PATH from file, got %q", got)
	}
	if got := env["SYNC_SCHEDULE"]; got != "*/15 * * * *" {
		t.Fatalf("expected SYNC_SCHEDULE from file, got %q", got)
	}
}

func TestBuildConnectorConfigDefaultsSchedule(t *testing.T) {
	cfg, err := buildConnectorConfig(db.Client{
		ClientKey: "loja_teste",
		ERPType:   "csv",
		ERPConfig: []byte(`{"ERP_FILE_PATH":"/tmp/vendas.csv"}`),
	})
	if err != nil {
		t.Fatalf("buildConnectorConfig: %v", err)
	}

	if cfg.FilePath != "/tmp/vendas.csv" {
		t.Fatalf("expected file path to be loaded, got %q", cfg.FilePath)
	}
	if cfg.Schedule != "*/30 * * * *" {
		t.Fatalf("expected default schedule, got %q", cfg.Schedule)
	}
}
