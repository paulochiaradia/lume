package normalizer

import (
	"time"
)

// ── Structs do schema canônico ───────────────────────────────
// Estas structs representam o formato padronizado
// que todo conector deve produzir independente do ERP

type Venda struct {
	VendaKey   string
	DataVenda  time.Time
	ClienteKey string
	VendedorID string
	Total      float64
	Desconto   float64
	Status     string
	Canal      string
	Atributos  map[string]interface{}
}

type ItemVenda struct {
	VendaKey      string
	ProdutoKey    string
	Descricao     string
	Quantidade    float64
	Unidade       string
	PrecoUnitario float64
	Desconto      float64
	Total         float64
	Atributos     map[string]interface{}
}

type Produto struct {
	ProdutoKey   string
	Nome         string
	Categoria    string
	Subcategoria string
	Unidade      string
	PrecoCusto   float64
	PrecoVenda   float64
	Ativo        bool
	Atributos    map[string]interface{}
}

type Cliente struct {
	ClienteKey string
	Nome       string
	Tipo       string
	Documento  string
	Telefone   string
	Cidade     string
	Bairro     string
	CEP        string
	Ativo      bool
	Atributos  map[string]interface{}
}

type Estoque struct {
	ProdutoKey    string
	Quantidade    float64
	QuantidadeMin float64
	QuantidadeMax float64
	Localizacao   string
}

// NormalizeResult agrupa todos os dados normalizados de uma extração
type NormalizeResult struct {
	Vendas   []Venda
	Itens    []ItemVenda
	Produtos []Produto
	Clientes []Cliente
	Estoques []Estoque
	Errors   []string
}
