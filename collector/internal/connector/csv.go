package connector

import (
	"encoding/csv"
	"fmt"
	"io"
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
	file, err := os.Open(c.cfg.FilePath)
	if err != nil {
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
		return nil, fmt.Errorf("erro ao ler cabeçalho do CSV: %w", err)
	}

	// Normaliza os cabeçalhos — remove espaços e converte para minúsculas
	for i, h := range headers {
		headers[i] = strings.ToLower(strings.TrimSpace(h))
	}

	// Lê os registros
	var records []RawRecord
	lineNum := 1

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Loga o erro mas continua processando as demais linhas
			fmt.Printf("aviso: erro na linha %d do CSV: %v\n", lineNum, err)
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
