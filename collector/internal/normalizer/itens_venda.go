package normalizer

import (
	"fmt"

	"github.com/paulochiaradia/lume/collector/internal/connector"
)

func NormalizeItensVenda(records []connector.RawRecord) ([]ItemVenda, []string) {
	var itens []ItemVenda
	var errors []string

	for i, record := range records {
		vendaKey := safeGet(record, "venda_id", "id_venda", "venda_key", "numero_venda")
		if vendaKey == "" {
			errors = append(errors, fmt.Sprintf("linha %d: ID da venda não encontrado", i+1))
			continue
		}

		produtoKey := safeGet(record, "produto_key", "produto_id", "codigo_produto", "sku")
		if produtoKey == "" {
			errors = append(errors, fmt.Sprintf("linha %d: ID do produto não encontrado", i+1))
			continue
		}

		quantidade := parseFloat(safeGet(record, "quantidade", "qtd", "qty"))
		precoUnitario := parseFloat(safeGet(record, "preco_unitario", "preco", "valor_unitario", "unit_price"))
		desconto := parseFloat(safeGet(record, "desconto", "vl_desconto", "discount"))
		total := parseFloat(safeGet(record, "total", "valor_total", "vl_total"))

		// Calcula o total se não vier no CSV
		if total == 0 && quantidade > 0 && precoUnitario > 0 {
			total = (quantidade * precoUnitario) - desconto
		}

		item := ItemVenda{
			VendaKey:      vendaKey,
			ProdutoKey:    produtoKey,
			Descricao:     safeGet(record, "descricao", "nome", "description", "produto"),
			Quantidade:    quantidade,
			Unidade:       safeGet(record, "unidade", "un", "unit"),
			PrecoUnitario: precoUnitario,
			Desconto:      desconto,
			Total:         total,
			Atributos:     make(map[string]interface{}),
		}

		if item.Unidade == "" {
			item.Unidade = "UN"
		}

		itens = append(itens, item)
	}

	return itens, errors
}
