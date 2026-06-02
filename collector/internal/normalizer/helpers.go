package normalizer

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseFloat converte string para float64
// aceita vírgula ou ponto como separador decimal
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Remove símbolo de moeda se existir
	s = strings.ReplaceAll(s, "R$", "")
	s = strings.TrimSpace(s)

	// Detecta o formato — brasileiro (1.250,00) ou internacional (1250.00)
	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")

	if hasComma && hasDot {
		// Formato brasileiro: 1.250,00 → remove ponto, troca vírgula por ponto
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if hasComma && !hasDot {
		// Formato com vírgula decimal: 1250,00 → troca vírgula por ponto
		s = strings.ReplaceAll(s, ",", ".")
	}
	// Se só tem ponto: 1250.00 → usa como está

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

// parseDate tenta converter string para time.Time
// aceita múltiplos formatos comuns no Brasil
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("data vazia")
	}

	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
		"02-01-2006",
		"01/02/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("formato de data não reconhecido: %s", s)
}

// parseBool converte string para bool
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "sim" || s == "s" || s == "ativo" || s == "a"
}

// safeGet retorna o valor de um mapa ou string vazia se não existir
func safeGet(record map[string]string, keys ...string) string {
	for _, key := range keys {
		if val, ok := record[key]; ok && val != "" {
			return val
		}
	}
	return ""
}
