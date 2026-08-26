package connector

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// CSVConnector lê arquivos CSV exportados de qualquer ERP
type CSVConnector struct {
	cfg       Config
	separator rune
}

// NewCSVConnector cria uma nova instância do conector CSV
func NewCSVConnector(cfg Config) *CSVConnector {
	return &CSVConnector{
		cfg:       cfg,
		separator: detectSeparator(cfg.FilePath),
	}
}

// Validate verifica se o arquivo existe e é legível
func (c *CSVConnector) Validate() error {
	if c.cfg.FilePath == "" {
		return fmt.Errorf("ERP_FILE_PATH não configurado para o cliente %s", c.cfg.ClientKey)
	}

	info, err := os.Stat(c.cfg.FilePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("arquivo não encontrado: %s", c.cfg.FilePath)
	}
	if err != nil {
		return fmt.Errorf("erro ao verificar arquivo: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("caminho é um diretório, não um arquivo: %s", c.cfg.FilePath)
	}

	return nil
}

// Extract lê o CSV e retorna os registros brutos
func (c *CSVConnector) Extract() ([]RawRecord, error) {
	// [SLOG] Marca o início do processo de ingestão
	slog.Info("iniciando extracao de csv",
		slog.String("event", "etl_csv_extract_started"),
		slog.String("tenant_id", c.cfg.ClientKey),
		slog.String("file", c.cfg.FilePath),
		slog.String("separator", string(c.separator)),
	)

	file, err := os.Open(c.cfg.FilePath)
	if err != nil {
		slog.Error("erro fatal ao abrir arquivo csv",
			slog.String("event", "etl_csv_open_error"),
			slog.String("tenant_id", c.cfg.ClientKey),
			slog.String("file", c.cfg.FilePath),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("erro ao abrir arquivo %s: %w", c.cfg.FilePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = c.separator
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Lê o cabeçalho
	headers, err := reader.Read()
	if err != nil {
		slog.Error("erro ao ler cabecalho do csv",
			slog.String("event", "etl_csv_header_error"),
			slog.String("tenant_id", c.cfg.ClientKey),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("erro ao ler cabeçalho do CSV: %w", err)
	}

	// Normaliza os cabeçalhos — remove espaços e converte para minúsculas
	for i, h := range headers {
		headers[i] = strings.ToLower(strings.TrimSpace(h))
	}

	// Lê os registros
	var records []RawRecord
	lineNum := 1 // Começa em 1 (ou 2, se considerar o cabeçalho como linha 1, mas mantive sua lógica)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// [SLOG] Substitui o fmt.Printf. O WARN é perfeito aqui pois o pipeline não morre,
			// ele apenas pula a linha defeituosa, mas deixa o rastro no Grafana.
			slog.Warn("erro ao processar linha do csv (linha ignorada)",
				slog.String("event", "etl_csv_row_parse_error"),
				slog.String("tenant_id", c.cfg.ClientKey),
				slog.String("file", c.cfg.FilePath),
				slog.Int("line", lineNum),
				slog.String("error", err.Error()),
			)
			lineNum++
			continue
		}

		record := make(RawRecord)
		for i, header := range headers {
			if i < len(row) {
				record[header] = strings.TrimSpace(row[i])
			} else {
				record[header] = ""
			}
		}

		records = append(records, record)
		lineNum++
	}

	// [SLOG] Telemetria de sucesso. Útil para medir volumetria diária por cliente.
	slog.Info("extracao de csv concluida com sucesso",
		slog.String("event", "etl_csv_extract_success"),
		slog.String("tenant_id", c.cfg.ClientKey),
		slog.Int("records_extracted", len(records)),
		slog.Int("total_lines_processed", lineNum),
	)

	return records, nil
}

// GetSchedule retorna o schedule configurado ou o padrão
func (c *CSVConnector) GetSchedule() string {
	if c.cfg.Schedule != "" {
		return c.cfg.Schedule
	}
	return DefaultSchedule
}

// GetType retorna o identificador do conector
func (c *CSVConnector) GetType() string {
	return "csv"
}

// detectSeparator tenta detectar o separador do CSV pelo nome do arquivo
// e pelo conteúdo da primeira linha
func detectSeparator(filePath string) rune {
	// Tenta detectar pela extensão
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".tsv" {
		return '\t'
	}

	// Tenta detectar pelo conteúdo da primeira linha
	file, err := os.Open(filePath)
	if err != nil {
		return ','
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return ','
	}

	firstLine := string(buf[:n])
	semicolons := strings.Count(firstLine, ";")
	commas := strings.Count(firstLine, ",")
	tabs := strings.Count(firstLine, "\t")

	if semicolons > commas && semicolons > tabs {
		return ';'
	}
	if tabs > commas && tabs > semicolons {
		return '\t'
	}

	return ','
}
