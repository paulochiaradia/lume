package normalizer

import (
	"fmt"

	"github.com/paulochiaradia/lume/collector/internal/connector"
)

// NormalizeVendas converte registros brutos em Vendas canônicas
// Aceita múltiplos nomes de coluna para cada campo
func NormalizeVendas(records []connector.RawRecord) ([]Venda, []string) {
	var vendas []Venda
	var errors []string

	for i, record := range records {
		// ID da venda — tenta múltiplos nomes de coluna
		vendaKey := safeGet(record,
			"id", "codigo", "numero", "num_venda",
			"id_venda", "codigo_venda", "nf", "numero_nf",
		)
		if vendaKey == "" {
			errors = append(errors, fmt.Sprintf("linha %d: ID da venda não encontrado", i+1))
			continue
		}

		// Data da venda
		dataStr := safeGet(record,
			"data", "data_venda", "dt_venda", "data_emissao",
			"dt_emissao", "created_at", "date",
		)
		dataVenda, err := parseDate(dataStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("linha %d: data inválida '%s'", i+1, dataStr))
			continue
		}

		venda := Venda{
			VendaKey:   vendaKey,
			DataVenda:  dataVenda,
			ClienteKey: safeGet(record, "cliente", "id_cliente", "codigo_cliente", "customer_id"),
			VendedorID: safeGet(record, "vendedor", "id_vendedor", "codigo_vendedor", "seller_id"),
			Total:      parseFloat(safeGet(record, "total", "valor_total", "vl_total", "total_venda")),
			Desconto:   parseFloat(safeGet(record, "desconto", "vl_desconto", "discount")),
			Status:     normalizeStatus(safeGet(record, "status", "situacao", "status_venda")),
			Canal:      normalizeCanal(safeGet(record, "canal", "origem", "channel")),
			Atributos:  make(map[string]interface{}),
		}

		vendas = append(vendas, venda)
	}

	return vendas, errors
}

func normalizeStatus(s string) string {
	if s == "" {
		return "concluida"
	}
	return s
}

func normalizeCanal(s string) string {
	if s == "" {
		return "balcao"
	}
	return s
}
