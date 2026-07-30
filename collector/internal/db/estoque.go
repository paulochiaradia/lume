package db

import (
	"database/sql"
	"fmt"
)

type EstoqueAlerta struct {
	ProdutoKey    string  `json:"produto_key"`
	Nome          string  `json:"nome"`
	Quantidade    float64 `json:"quantidade"`
	QuantidadeMin float64 `json:"quantidade_min"`
}

type EstoqueItem struct {
	ProdutoKey    string  `json:"produto_key"`
	Nome          string  `json:"nome"`
	Categoria     string  `json:"categoria"`
	Quantidade    float64 `json:"quantidade"`
	QuantidadeMin float64 `json:"quantidade_min"`
	PrecoVenda    float64 `json:"preco_venda"`
	PrecoCusto    float64 `json:"preco_custo"`
	Alerta        bool    `json:"alerta"`
}

type DeadStockProduto struct {
	Nome          string  `json:"nome"`
	Quantidade    float64 `json:"quantidade"`
	PrecoCusto    float64 `json:"preco_custo"`
	CapitalParado float64 `json:"capital_parado"`
}

type EstoqueReposicao struct {
	ID                 string  `json:"id"`
	Nome               string  `json:"nome"`
	Categoria          string  `json:"categoria"`
	ClasseABC          string  `json:"classe_abc"`
	EstoqueAtual       float64 `json:"estoque_atual"`
	DemandaPrevista    float64 `json:"demanda_prevista"`
	DiasAteRuptura     int     `json:"dias_ate_ruptura"`
	QuantidadeSugerida float64 `json:"quantidade_sugerida"`
	Urgencia           int     `json:"urgencia"`
}

type EstoqueKPIs struct {
	TotalSKUs         int     `json:"total_skus"`
	ValorTotalEstoque float64 `json:"valor_total_estoque"`
	ItensEmAlerta     int     `json:"itens_em_alerta"`
	TaxaRuptura       float64 `json:"taxa_ruptura"`
}

// GetEstoqueAlertas busca os produtos que estão abaixo do estoque mínimo
func GetEstoqueAlertas(db *sql.DB, clientKey string) ([]EstoqueAlerta, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			e.produto_key,
			COALESCE(p.nome, e.produto_key),
			e.quantidade,
			e.quantidade_min
		FROM %s.estoque e
		LEFT JOIN %s.produtos p ON p.produto_key = e.produto_key
		WHERE e.quantidade <= e.quantidade_min
		ORDER BY e.quantidade ASC
	`, schema, schema))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar alertas: %w", err)
	}

	defer rows.Close()

	var alertas []EstoqueAlerta
	for rows.Next() {
		var a EstoqueAlerta
		if err := rows.Scan(&a.ProdutoKey, &a.Nome, &a.Quantidade, &a.QuantidadeMin); err != nil {
			continue
		}
		alertas = append(alertas, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler resultados: %w", err)
	}

	return alertas, nil
}

// GetEstoqueCompleto busca todos os produtos do estoque, incluindo informações de alerta
func GetEstoqueCompleto(db *sql.DB, clientKey string) ([]EstoqueItem, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			e.produto_key,
			COALESCE(p.nome, e.produto_key)      AS nome,
			COALESCE(p.categoria, '')             AS categoria,
			e.quantidade,
			e.quantidade_min,
			COALESCE(p.preco_venda, 0)            AS preco_venda,
			COALESCE(p.preco_custo, 0)            AS preco_custo,
			(e.quantidade <= e.quantidade_min)    AS alerta
		FROM %s.estoque e
		LEFT JOIN %s.produtos p ON p.produto_key = e.produto_key
		ORDER BY alerta DESC, e.quantidade ASC
	`, schema, schema))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar estoque: %w", err)
	}
	defer rows.Close()

	var itens []EstoqueItem
	for rows.Next() {
		var i EstoqueItem
		if err := rows.Scan(
			&i.ProdutoKey, &i.Nome, &i.Categoria,
			&i.Quantidade, &i.QuantidadeMin,
			&i.PrecoVenda, &i.PrecoCusto, &i.Alerta,
		); err != nil {
			continue
		}
		itens = append(itens, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler resultados: %w", err)
	}

	return itens, nil
}

// GetEstoqueReposicao busca a fila de compras priorizada no cache do Postgres
func GetEstoqueReposicao(db *sql.DB, clientKey string) ([]EstoqueReposicao, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT id, nome, categoria, classe_abc, estoque_atual, demanda_prevista, 
		       dias_ate_ruptura, quantidade_sugerida, urgencia
		FROM %s.estoque_reposicao_cache
		ORDER BY urgencia ASC, dias_ate_ruptura ASC
	`, schema))

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fila []EstoqueReposicao
	for rows.Next() {
		var item EstoqueReposicao
		if err := rows.Scan(
			&item.ID, &item.Nome, &item.Categoria, &item.ClasseABC,
			&item.EstoqueAtual, &item.DemandaPrevista,
			&item.DiasAteRuptura, &item.QuantidadeSugerida, &item.Urgencia,
		); err != nil {
			continue // Seguindo exatamente o seu padrão de ignorar a linha com erro e seguir
		}
		fila = append(fila, item)
	}

	// checa erros de leitura das linhas
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler resultados: %w", err)
	}

	// Garante que retorne um array vazio [] ao invés de null para o JSON do frontend
	if fila == nil {
		fila = []EstoqueReposicao{}
	}

	return fila, nil
}

// GetEstoqueKPIs busca os indicadores principais do topo da tela
func GetEstoqueKPIs(db *sql.DB, clientKey string) (EstoqueKPIs, error) {
	schema := "client_" + clientKey
	var kpis EstoqueKPIs

	// Agregação rápida para os blocos do topo
	query := fmt.Sprintf(`
		SELECT 
			COUNT(e.produto_key) AS total_skus,
			COALESCE(SUM(CASE WHEN e.quantidade > 0 THEN e.quantidade * p.preco_custo ELSE 0 END), 0) AS valor_total_estoque,
			COUNT(CASE WHEN e.quantidade <= e.quantidade_min THEN 1 END) AS itens_em_alerta,
			COALESCE(ROUND(
				(COUNT(CASE WHEN e.quantidade <= 0 THEN 1 END)::numeric / NULLIF(COUNT(e.produto_key), 0)) * 100
			, 2), 0) AS taxa_ruptura
		FROM %s.estoque e
		LEFT JOIN %s.produtos p ON p.produto_key = e.produto_key
	`, schema, schema)

	err := db.QueryRow(query).Scan(
		&kpis.TotalSKUs,
		&kpis.ValorTotalEstoque,
		&kpis.ItensEmAlerta,
		&kpis.TaxaRuptura,
	)

	if err != nil {
		return kpis, fmt.Errorf("erro ao calcular kpis de estoque: %w", err)
	}

	return kpis, nil
}

// GetDeadStock busca os produtos que não tiveram vendas nos últimos 90 dias e ainda possuem estoque
func GetDeadStock(db *sql.DB, clientKey string) ([]DeadStockProduto, error) {
	schema := "client_" + clientKey
	rows, err := db.Query(fmt.Sprintf(`
		SELECT p.nome, e.quantidade, p.preco_custo, (e.quantidade * p.preco_custo) as capital_parado
		FROM %s.produtos p
		JOIN %s.estoque e ON e.produto_key = p.produto_key
		LEFT JOIN (
			SELECT DISTINCT iv.produto_key 
			FROM %s.itens_venda iv 
			JOIN %s.vendas v ON v.id = iv.venda_id 
			WHERE v.data_venda >= NOW() - INTERVAL '90 days'
		) recentes ON recentes.produto_key = p.produto_key
		WHERE recentes.produto_key IS NULL AND e.quantidade > 0
		ORDER BY capital_parado DESC
	`, schema, schema, schema, schema))

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []DeadStockProduto
	for rows.Next() {
		var ds DeadStockProduto
		if err := rows.Scan(&ds.Nome, &ds.Quantidade, &ds.PrecoCusto, &ds.CapitalParado); err != nil {
			continue
		}
		lista = append(lista, ds)
	}
	// checa erros de leitura das linhas
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler resultados: %w", err)
	}

	// garante array vazio em vez de null
	if lista == nil {
		lista = []DeadStockProduto{}
	}

	return lista, nil
}
