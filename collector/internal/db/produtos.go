package db

import (
	"database/sql"
	"fmt"
)

type CatalogKPIs struct {
	TotalSKUs       int     `json:"total_skus"`
	QtdClasseA      int     `json:"qtd_classe_a"`
	PctFaturamentoA float64 `json:"pct_faturamento_a"`
	MargemMedia     float64 `json:"margem_media"`
	DeadStockSKUs   int     `json:"dead_stock_skus"`
	CapitalParado   float64 `json:"capital_parado"`
}

type MatrizABCXYZ struct {
	ClasseABC string `json:"classe_abc"`
	ClasseXYZ string `json:"classe_xyz"`
	Qtd       int    `json:"qtd"`
}

type RankingProduto struct {
	ID          string  `json:"id"`
	Nome        string  `json:"nome"`
	Categoria   string  `json:"categoria"`
	Faturamento float64 `json:"faturamento"`
	Margem      float64 `json:"margem"`
	ClasseABC   string  `json:"classe_abc"`
	ClasseXYZ   string  `json:"classe_xyz"`
	Tendencia   string  `json:"tendencia"` // subindo, estavel, caindo
}

type ElasticidadeProduto struct {
	ProdutoKey    string  `json:"produto_key"`
	Elasticidade  float64 `json:"elasticidade"`
	Interpretacao string  `json:"interpretacao"`
	Receita       float64 `json:"receita"`
}

type RegraAssociacao struct {
	Antecedents string  `json:"antecedents"`
	Consequents string  `json:"consequents"`
	Confidence  float64 `json:"confidence"`
	Lift        float64 `json:"lift"`
}

type ProdutoABC struct {
	ProdutoKey  string  `json:"produto_key"`
	Nome        string  `json:"nome"`
	Categoria   string  `json:"categoria"`
	Faturamento float64 `json:"faturamento"`
	Classe      string  `json:"classe"`
}

// GetCatalogKPIs retorna os KPIs do catálogo de produtos
func GetCatalogKPIs(db *sql.DB, clientKey string) (*CatalogKPIs, error) {
	schema := "client_" + clientKey
	var kpis CatalogKPIs

	_ = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s.produtos", schema)).Scan(&kpis.TotalSKUs)

	_ = db.QueryRow(fmt.Sprintf(`
			SELECT 
				COUNT(*), 
				COALESCE((SUM(faturamento) / NULLIF((SELECT SUM(faturamento) FROM %s.abc_xyz_cache), 0)) * 100, 0)
			FROM %s.abc_xyz_cache 
			WHERE classe = 'A'
		`, schema, schema)).Scan(&kpis.QtdClasseA, &kpis.PctFaturamentoA)

	// Alterado para ler da nova tabela margem_cache
	_ = db.QueryRow(fmt.Sprintf("SELECT COALESCE(AVG(pct_margem), 0) FROM %s.margem_cache", schema)).Scan(&kpis.MargemMedia)

	_ = db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(p.id), COALESCE(SUM(e.quantidade * p.preco_custo), 0)
		FROM %s.produtos p
		JOIN %s.estoque e ON e.produto_key = p.produto_key
		LEFT JOIN (
			SELECT DISTINCT iv.produto_key 
			FROM %s.itens_venda iv 
			JOIN %s.vendas v ON v.id = iv.venda_id 
			WHERE v.data_venda >= NOW() - INTERVAL '90 days'
		) recentes ON recentes.produto_key = p.produto_key
		WHERE recentes.produto_key IS NULL AND e.quantidade > 0
	`, schema, schema, schema, schema)).Scan(&kpis.DeadStockSKUs, &kpis.CapitalParado)

	return &kpis, nil
}

// GetMatrizABCXYZ retorna a matriz de produtos por classe ABC e XYZ
func GetMatrizABCXYZ(db *sql.DB, clientKey string) ([]MatrizABCXYZ, error) {
	schema := "client_" + clientKey
	// Corrigido: classe_xyz ao invés de xyz
	rows, err := db.Query(fmt.Sprintf(`
		SELECT classe, classe_xyz, COUNT(*) 
		FROM %s.abc_xyz_cache 
		GROUP BY classe, classe_xyz
	`, schema))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matriz []MatrizABCXYZ
	for rows.Next() {
		var m MatrizABCXYZ
		if err := rows.Scan(&m.ClasseABC, &m.ClasseXYZ, &m.Qtd); err != nil {
			continue
		}
		matriz = append(matriz, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return matriz, nil
}

// GetProdutosABC retorna a lista de produtos com faturamento e classe ABC
func GetRankingProdutos(db *sql.DB, clientKey string) ([]RankingProduto, error) {
	schema := "client_" + clientKey
	rows, err := db.Query(fmt.Sprintf(`
		WITH faturamento_periodos AS (
			SELECT 
				iv.produto_key,
				SUM(CASE WHEN v.data_venda >= NOW() - INTERVAL '28 days' THEN iv.total ELSE 0 END) as fat_atual,
				SUM(CASE WHEN v.data_venda >= NOW() - INTERVAL '56 days' AND v.data_venda < NOW() - INTERVAL '28 days' THEN iv.total ELSE 0 END) as fat_anterior
			FROM %s.itens_venda iv
			JOIN %s.vendas v ON v.id = iv.venda_id
			GROUP BY iv.produto_key
		)
		SELECT 
			p.produto_key,
			p.nome,
			COALESCE(p.categoria, 'Geral') as categoria,
			COALESCE(fp.fat_atual, 0) as faturamento,
			COALESCE(m.pct_margem, 0) as margem,
			COALESCE(cache.classe, 'C') as classe_abc,
			COALESCE(cache.classe_xyz, 'Z') as classe_xyz,
			CASE 
				WHEN COALESCE(fp.fat_atual, 0) > COALESCE(fp.fat_anterior, 0) * 1.05 THEN 'subindo'
				WHEN COALESCE(fp.fat_atual, 0) < COALESCE(fp.fat_anterior, 0) * 0.95 THEN 'caindo'
				ELSE 'estavel'
			END as tendencia
		FROM %s.produtos p
		LEFT JOIN faturamento_periodos fp ON fp.produto_key = p.produto_key
		LEFT JOIN %s.abc_xyz_cache cache ON cache.produto_key = p.produto_key
		LEFT JOIN %s.margem_cache m ON m.produto_key = p.produto_key
		ORDER BY faturamento DESC
	`, schema, schema, schema, schema, schema))

	if err != nil {
		// Se a query falhar, o Go vai gritar o motivo no terminal
		fmt.Printf("\n[ERRO DB] Falha na query de ranking: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var lista []RankingProduto
	for rows.Next() {
		var rp RankingProduto
		if err := rows.Scan(
			&rp.ID, &rp.Nome, &rp.Categoria, &rp.Faturamento,
			&rp.Margem, &rp.ClasseABC, &rp.ClasseXYZ, &rp.Tendencia,
		); err != nil {
			// Se a conversão dos dados falhar, o Go vai gritar o motivo aqui
			fmt.Printf("\n[ERRO SCAN] Falha ao ler linha do ranking: %v\n", err)
			continue
		}
		lista = append(lista, rp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// GetElasticidadeProdutos retorna a elasticidade dos produtos
func GetElasticidadeProdutos(db *sql.DB, clientKey string) ([]ElasticidadeProduto, error) {
	schema := "client_" + clientKey
	// Corrigido: tipo as interpretacao e receita_total as receita
	rows, err := db.Query(fmt.Sprintf(`
		SELECT produto_key, elasticidade, tipo as interpretacao, receita_total as receita 
		FROM %s.elasticidade_cache 
		WHERE receita_total > 0
		ORDER BY receita_total DESC
	`, schema))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []ElasticidadeProduto
	for rows.Next() {
		var ep ElasticidadeProduto
		if err := rows.Scan(&ep.ProdutoKey, &ep.Elasticidade, &ep.Interpretacao, &ep.Receita); err != nil {
			continue
		}
		lista = append(lista, ep)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// GetMarketBasketRules retorna as regras de associação do Market Basket Analysis
func GetMarketBasketRules(db *sql.DB, clientKey string) ([]RegraAssociacao, error) {
	schema := "client_" + clientKey
	rows, err := db.Query(fmt.Sprintf(`
		SELECT antecedents, consequents, confidence * 100, lift 
		FROM %s.basket_cache 
		ORDER BY lift DESC
	`, schema))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []RegraAssociacao
	for rows.Next() {
		var ra RegraAssociacao
		if err := rows.Scan(&ra.Antecedents, &ra.Consequents, &ra.Confidence, &ra.Lift); err != nil {
			continue
		}
		lista = append(lista, ra)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// GetProdutosABC retorna a lista de produtos com faturamento e classe ABC
func GetProdutosABC(db *sql.DB, clientKey string) ([]ProdutoABC, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		WITH vendas_por_produto AS (
			SELECT
				iv.produto_key,
				COALESCE(p.nome, iv.produto_key) as nome,
				COALESCE(p.categoria, '') as categoria,
				SUM(iv.total) as faturamento
			FROM %s.itens_venda iv
			LEFT JOIN %s.produtos p ON p.produto_key = iv.produto_key
			GROUP BY iv.produto_key, p.nome, p.categoria
		),
		total AS (
			SELECT SUM(faturamento) as total FROM vendas_por_produto
		),
		acumulado AS (
			SELECT
				produto_key, nome, categoria, faturamento,
				SUM(faturamento) OVER (ORDER BY faturamento DESC) as acumulado,
				(SELECT total FROM total) as total
			FROM vendas_por_produto
		)
		SELECT
			produto_key, nome, categoria, faturamento,
			CASE
				WHEN acumulado / NULLIF(total, 0) <= 0.8 THEN 'A'
				WHEN acumulado / NULLIF(total, 0) <= 0.95 THEN 'B'
				ELSE 'C'
			END as classe
		FROM acumulado
		ORDER BY faturamento DESC
	`, schema, schema))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar produtos ABC: %w", err)
	}
	defer rows.Close()

	var produtos []ProdutoABC
	for rows.Next() {
		var p ProdutoABC
		if err := rows.Scan(&p.ProdutoKey, &p.Nome, &p.Categoria, &p.Faturamento, &p.Classe); err != nil {
			continue
		}
		produtos = append(produtos, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar produtos ABC: %w", err)
	}
	return produtos, nil
}
