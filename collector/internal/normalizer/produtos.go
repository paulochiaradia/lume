package normalizer

import (
	"github.com/paulochiaradia/lume/collector/internal/connector"
)

// NormalizeProdutos converte registros brutos em Produtos canônicos
func NormalizeProdutos(records []connector.RawRecord) ([]Produto, []string) {
	var produtos []Produto
	var errors []string

	for _, record := range records {
		produtoKey := safeGet(record,
			"id", "codigo", "sku", "produto_id",
			"id_produto", "codigo_produto", "product_id",
		)
		if produtoKey == "" {
			continue
		}

		produto := Produto{
			ProdutoKey:   produtoKey,
			Nome:         safeGet(record, "nome", "descricao", "name", "produto", "description"),
			Categoria:    safeGet(record, "categoria", "grupo", "category", "grupo_produto"),
			Subcategoria: safeGet(record, "subcategoria", "subgrupo", "subcategory"),
			Unidade:      safeGet(record, "unidade", "un", "unit", "unidade_medida"),
			PrecoCusto:   parseFloat(safeGet(record, "custo", "preco_custo", "vl_custo", "cost")),
			PrecoVenda:   parseFloat(safeGet(record, "preco", "preco_venda", "vl_venda", "price")),
			Ativo:        parseBool(safeGet(record, "ativo", "active", "status", "situacao")),
			Atributos:    make(map[string]interface{}),
		}

		if produto.Unidade == "" {
			produto.Unidade = "UN"
		}

		if produto.Nome == "" {
			produto.Nome = produto.ProdutoKey
		}

		produtos = append(produtos, produto)
	}

	return produtos, errors
}
