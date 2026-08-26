package db

import (
	"database/sql"
	"fmt"
	"time"
)

type VendasResumo struct {
	Faturamento   float64 `json:"faturamento"`
	TotalVendas   int     `json:"total_vendas"`
	TicketMedio   float64 `json:"ticket_medio"`
	TotalDesconto float64 `json:"total_desconto"`
}

type VendaDia struct {
	Dia         string  `json:"dia"`
	Vendas      int     `json:"vendas"`
	Faturamento float64 `json:"faturamento"`
	MediaMovel  float64 `json:"mediaMovel"`
}

type VendaHora struct {
	Hora        string  `json:"hora"`
	Faturamento float64 `json:"faturamento"`
}

type MixCategoria struct {
	Categoria   string  `json:"categoria"`
	Faturamento float64 `json:"faturamento"`
}

type TrendValue struct {
	Atual    float64 `json:"atual"`
	Anterior float64 `json:"anterior"`
	Variacao float64 `json:"variacao"`
}

type VendasKPIsTrend struct {
	Faturamento     TrendValue `json:"faturamento"`
	TicketMedio     TrendValue `json:"ticket_medio"`
	TotalTransacoes TrendValue `json:"total_transacoes"`
	PercDesconto    TrendValue `json:"perc_desconto"`
}

type RankingVendedor struct {
	ID          string  `json:"id"`
	Nome        string  `json:"nome"`
	Vendas      int     `json:"vendas"`
	Faturamento float64 `json:"faturamento"`
	Ticket      float64 `json:"ticket"`
	UPA         float64 `json:"upa"`
	Desconto    float64 `json:"desconto"`
}

type HeatmapPonto struct {
	Dia         string  `json:"dia"`
	Hora        string  `json:"hora"`
	Faturamento float64 `json:"faturamento"`
}

type Insight struct {
	Tipo       string `json:"tipo"`
	Prioridade int    `json:"prioridade"`
	Titulo     string `json:"titulo"`
	Mensagem   string `json:"mensagem"`
	Acao       string `json:"acao"`
	Href       string `json:"href"`
	Categoria  string `json:"categoria"`
	Icone      string `json:"icone"`
	GeradoEm   string `json:"gerado_em"`
}

type HomeKPIs struct {
	Faturamento   float64 `json:"faturamento"`
	TicketMedio   float64 `json:"ticket_medio"`
	TotalVendas   int     `json:"total_vendas"`
	ItensPorVenda float64 `json:"itens_por_venda"`
}

// GetVendasResumo retorna o resumo das vendas do cliente
func GetVendasResumo(db *sql.DB, clientKey string) (*VendasResumo, error) {
	schema := "client_" + clientKey
	var resumo VendasResumo

	err := db.QueryRow(fmt.Sprintf(`
		SELECT
			COALESCE(SUM(total), 0),
			COUNT(*),
			COALESCE(AVG(total), 0),
			COALESCE(SUM(desconto), 0)
		FROM %s.vendas
		WHERE status = 'concluida'
	`, schema)).Scan(
		&resumo.Faturamento,
		&resumo.TotalVendas,
		&resumo.TicketMedio,
		&resumo.TotalDesconto,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar resumo: %w", err)
	}
	return &resumo, nil
}

// GetVendasPorDia retorna o faturamento diário com a média móvel de 7 dias
// 1. QUERY EXCLUSIVA DA HOME (Simples, foca em trazer 2 meses de dados para comparar)
func GetVendasPorDia(db *sql.DB, clientKey string, startTime time.Time) ([]VendaDia, error) {
	schema := "client_" + clientKey
	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			TO_CHAR(data_venda, 'YYYY-MM-DD') as dia,
			COUNT(*) as vendas,
			COALESCE(SUM(total), 0) as faturamento,
			0 as media_movel
		FROM %s.vendas
		WHERE status = 'concluida' AND data_venda >= $1
		GROUP BY TO_CHAR(data_venda, 'YYYY-MM-DD')
		ORDER BY dia
	`, schema), startTime)

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar vendas por dia: %w", err)
	}
	defer rows.Close()

	vendas := []VendaDia{}
	for rows.Next() {
		var v VendaDia
		if err := rows.Scan(&v.Dia, &v.Vendas, &v.Faturamento, &v.MediaMovel); err != nil {
			continue
		}
		vendas = append(vendas, v)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultado de vendas por dia: %w", err)
	}
	return vendas, nil
}

// 2. QUERY EXCLUSIVA DA TELA DE VENDAS (Média Móvel com Window Function)
func GetVendasTendencia(db *sql.DB, clientKey string, startTime time.Time) ([]VendaDia, error) {
	schema := "client_" + clientKey
	rows, err := db.Query(fmt.Sprintf(`
		WITH diarios AS (
			SELECT
				DATE(data_venda) as data_real,
				COUNT(*) as vendas,
				COALESCE(SUM(total), 0) as faturamento
			FROM %s.vendas
			WHERE status = 'concluida' AND data_venda >= $1
			GROUP BY DATE(data_venda)
		)
		SELECT
			TO_CHAR(data_real, 'YYYY-MM-DD') as dia,
			vendas,
			faturamento,
			ROUND(COALESCE(AVG(faturamento) OVER (
				ORDER BY data_real
				ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
			), 0), 2) as media_movel
		FROM diarios
		ORDER BY data_real
	`, schema), startTime)

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar tendencia: %w", err)
	}
	defer rows.Close()

	vendas := []VendaDia{}
	for rows.Next() {
		var v VendaDia
		if err := rows.Scan(&v.Dia, &v.Vendas, &v.Faturamento, &v.MediaMovel); err != nil {
			continue
		}
		vendas = append(vendas, v)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultado de tendencia: %w", err)
	}
	return vendas, nil
}

// GetTopDias retorna os 5 dias com maior faturamento
func GetTopDias(db *sql.DB, clientKey string, startTime time.Time) ([]VendaDia, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			DATE(data_venda)::text,
			COUNT(*),
			COALESCE(SUM(total), 0) as faturamento
		FROM %s.vendas
		WHERE status = 'concluida'
		  AND data_venda >= $1
		GROUP BY DATE(data_venda)
		ORDER BY faturamento DESC
		LIMIT 5
	`, schema), startTime)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar top dias: %w", err)
	}
	defer rows.Close()

	vendas := []VendaDia{}
	for rows.Next() {
		var v VendaDia
		if err := rows.Scan(&v.Dia, &v.Vendas, &v.Faturamento); err != nil {
			continue
		}
		vendas = append(vendas, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultado de top dias: %w", err)
	}
	return vendas, nil
}

// GetVendasPorHora retorna o faturamento por hora do dia
func GetVendasPorHora(db *sql.DB, clientKey string, startTime time.Time) ([]VendaHora, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			TO_CHAR(data_venda, 'HH24:00') as hora,
			COALESCE(SUM(total), 0) as faturamento
		FROM %s.vendas
		WHERE status = 'concluida' 
		  AND data_venda >= $1
		GROUP BY TO_CHAR(data_venda, 'HH24:00')
		ORDER BY hora ASC
	`, schema), startTime)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar vendas por hora: %w", err)
	}
	defer rows.Close()

	var vendas []VendaHora
	for rows.Next() {
		var v VendaHora
		if err := rows.Scan(&v.Hora, &v.Faturamento); err != nil {
			continue
		}
		vendas = append(vendas, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultado de vendas por hora: %w", err)
	}
	return vendas, nil
}

// GetMixCategorias retorna o mix de categorias com faturamento
func GetMixCategorias(db *sql.DB, clientKey string, startTime time.Time) ([]MixCategoria, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			COALESCE(p.categoria, 'Sem Categoria') as categoria,
			COALESCE(SUM(iv.total), 0) as faturamento
		FROM %s.itens_venda iv
		JOIN %s.vendas v ON iv.venda_id = v.id
		LEFT JOIN %s.produtos p ON p.produto_key = iv.produto_key
		WHERE v.status = 'concluida' 
		  AND v.data_venda >= $1
		GROUP BY COALESCE(p.categoria, 'Sem Categoria')
		ORDER BY faturamento DESC
		LIMIT 5
	`, schema, schema, schema), startTime)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar mix de categorias: %w", err)
	}
	defer rows.Close()

	var categorias []MixCategoria
	for rows.Next() {
		var c MixCategoria
		if err := rows.Scan(&c.Categoria, &c.Faturamento); err != nil {
			continue
		}
		categorias = append(categorias, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultado do mix de categorias: %w", err)
	}
	return categorias, nil
}

// GetVendasKPIsTrend retorna os KPIs de vendas com tendência de crescimento ou queda
func GetVendasKPIsTrend(db *sql.DB, clientKey string, currentStart, previousStart time.Time) (VendasKPIsTrend, error) {
	schema := "client_" + clientKey
	var kpis VendasKPIsTrend

	query := fmt.Sprintf(`
		SELECT
			-- Período Atual
			COALESCE(SUM(total) FILTER (WHERE data_venda >= $1), 0) as fat_atual,
			COUNT(id) FILTER (WHERE data_venda >= $1) as trans_atual,
			COALESCE(SUM(desconto) FILTER (WHERE data_venda >= $1), 0) as desc_atual,

			-- Período Anterior
			COALESCE(SUM(total) FILTER (WHERE data_venda >= $2 AND data_venda < $1), 0) as fat_ant,
			COUNT(id) FILTER (WHERE data_venda >= $2 AND data_venda < $1) as trans_ant,
			COALESCE(SUM(desconto) FILTER (WHERE data_venda >= $2 AND data_venda < $1), 0) as desc_ant
		FROM %s.vendas
		WHERE status = 'concluida' AND data_venda >= $2
	`, schema)

	var fatAtual, descAtual, fatAnt, descAnt float64
	var transAtual, transAnt int

	err := db.QueryRow(query, currentStart, previousStart).Scan(
		&fatAtual, &transAtual, &descAtual,
		&fatAnt, &transAnt, &descAnt,
	)
	if err != nil {
		return kpis, fmt.Errorf("erro na query de tendencia: %w", err)
	}

	safeDiv := func(a float64, b int) float64 {
		if b == 0 {
			return 0
		}
		return a / float64(b)
	}

	calcTrend := func(atual, anterior float64) float64 {
		if anterior == 0 {
			if atual > 0 {
				return 100.0
			}
			return 0.0
		}
		return ((atual - anterior) / anterior) * 100.0
	}

	calcPercDesconto := func(fat, desc float64) float64 {
		if (fat + desc) == 0 {
			return 0
		}
		return (desc / (fat + desc)) * 100.0
	}

	kpis.Faturamento = TrendValue{Atual: fatAtual, Anterior: fatAnt, Variacao: calcTrend(fatAtual, fatAnt)}
	kpis.TotalTransacoes = TrendValue{Atual: float64(transAtual), Anterior: float64(transAnt), Variacao: calcTrend(float64(transAtual), float64(transAnt))}
	tmAtual := safeDiv(fatAtual, transAtual)
	tmAnt := safeDiv(fatAnt, transAnt)
	kpis.TicketMedio = TrendValue{Atual: tmAtual, Anterior: tmAnt, Variacao: calcTrend(tmAtual, tmAnt)}
	percDescAtual := calcPercDesconto(fatAtual, descAtual)
	percDescAnt := calcPercDesconto(fatAnt, descAnt)
	kpis.PercDesconto = TrendValue{Atual: percDescAtual, Anterior: percDescAnt, Variacao: percDescAtual - percDescAnt}

	return kpis, nil
}

// GetRankingVendedores retorna o ranking de vendedores com base no faturamento
func GetRankingVendedores(db *sql.DB, clientKey string, startTime time.Time) ([]RankingVendedor, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		WITH itens_agrupados AS (
			SELECT venda_id, SUM(quantidade) as qtd_itens
			FROM %s.itens_venda
			GROUP BY venda_id
		)
		SELECT
			COALESCE(v.vendedor_id, 'Não Informado') AS id,
			COUNT(v.id) AS qtd_vendas,
			COALESCE(SUM(v.total), 0) AS faturamento,
			COALESCE(SUM(v.total) / NULLIF(COUNT(v.id), 0), 0) AS ticket_medio,
			COALESCE(SUM(ia.qtd_itens) / NULLIF(COUNT(v.id), 0), 0) AS upa,
			COALESCE((SUM(v.desconto) / NULLIF(SUM(v.total) + SUM(v.desconto), 0)) * 100, 0) AS perc_desconto
		FROM %s.vendas v
		LEFT JOIN itens_agrupados ia ON ia.venda_id = v.id
		WHERE v.status = 'concluida' AND v.data_venda >= $1
		GROUP BY v.vendedor_id
		ORDER BY faturamento DESC
	`, schema, schema), startTime)

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar ranking de vendedores: %w", err)
	}
	defer rows.Close()

	var ranking []RankingVendedor
	for rows.Next() {
		var r RankingVendedor
		if err := rows.Scan(
			&r.ID, &r.Vendas, &r.Faturamento,
			&r.Ticket, &r.UPA, &r.Desconto,
		); err != nil {
			continue
		}
		// Como não temos tabela de Vendedores com nome, usamos o ID/Código como nome
		r.Nome = r.ID
		ranking = append(ranking, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar ranking de vendedores: %w", err)
	}

	return ranking, nil
}

// GetVendasHeatmap retorna os dados para o heatmap de vendas (dia da semana x hora do dia)
func GetVendasHeatmap(db *sql.DB, clientKey string, startTime time.Time) ([]HeatmapPonto, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			CASE EXTRACT(DOW FROM data_venda)
				WHEN 1 THEN 'Seg' WHEN 2 THEN 'Ter' WHEN 3 THEN 'Qua'
				WHEN 4 THEN 'Qui' WHEN 5 THEN 'Sex' WHEN 6 THEN 'Sáb'
				ELSE 'Dom'
			END as dia,
			TO_CHAR(data_venda, 'HH24"h"') as hora,
			COALESCE(SUM(total), 0) as faturamento
		FROM %s.vendas
		WHERE status = 'concluida' AND data_venda >= $1
		GROUP BY EXTRACT(DOW FROM data_venda), TO_CHAR(data_venda, 'HH24"h"')
	`, schema), startTime)

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar dados do heatmap: %w", err)
	}
	defer rows.Close()

	var pontos []HeatmapPonto
	for rows.Next() {
		var p HeatmapPonto
		if err := rows.Scan(&p.Dia, &p.Hora, &p.Faturamento); err != nil {
			continue
		}
		pontos = append(pontos, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar dados do heatmap: %w", err)
	}

	return pontos, nil
}

// GetInsightsVendas busca os 3 insights mais urgentes específicos da categoria vendas
func GetInsightsVendas(db *sql.DB, clientKey string) ([]Insight, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			COALESCE(tipo, ''),
			COALESCE(prioridade::int, 99),
			COALESCE(titulo, ''),
			COALESCE(mensagem, ''),
			COALESCE(acao, ''),
			COALESCE(href, ''),
			COALESCE(categoria, ''),
			COALESCE(icone, 'success'),
			COALESCE(gerado_em::text, NOW()::text)
		FROM %s.insights_cache
		WHERE categoria = 'vendas'
		ORDER BY prioridade ASC
		LIMIT 3
	`, schema))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar insights de vendas: %w", err)
	}
	defer rows.Close()

	var insights []Insight
	for rows.Next() {
		var i Insight
		if err := rows.Scan(
			&i.Tipo, &i.Prioridade, &i.Titulo,
			&i.Mensagem, &i.Acao, &i.Href,
			&i.Categoria, &i.Icone, &i.GeradoEm,
		); err != nil {
			continue
		}
		insights = append(insights, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar insights de vendas: %w", err)
	}

	return insights, nil
}

func GetHomeKPIs(db *sql.DB, clientKey string, startTime time.Time) (HomeKPIs, error) {
	schema := "client_" + clientKey

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(total), 0) as faturamento,
			COALESCE(AVG(total), 0) as ticket_medio,
			COUNT(id) as total_vendas,
			COALESCE((
				SELECT SUM(iv.quantidade)
				FROM %s.itens_venda iv
				JOIN %s.vendas v2 ON v2.id = iv.venda_id
				WHERE v2.status = 'concluida' AND v2.data_venda >= $1
			) / NULLIF(COUNT(id), 0), 0) as itens_por_venda
		FROM %s.vendas
		WHERE status = 'concluida' 
		  AND data_venda >= $1
	`, schema, schema, schema)

	var kpis HomeKPIs
	err := db.QueryRow(query, startTime).Scan(
		&kpis.Faturamento,
		&kpis.TicketMedio,
		&kpis.TotalVendas,
		&kpis.ItensPorVenda,
	)
	if err != nil {
		return kpis, fmt.Errorf("erro ao buscar kpis: %w", err)
	}
	return kpis, nil
}

func GetInsights(db *sql.DB, clientKey string) ([]Insight, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			COALESCE(tipo, ''),
			COALESCE(prioridade::int, 99),
			COALESCE(titulo, ''),
			COALESCE(mensagem, ''),
			COALESCE(acao, ''),
			COALESCE(href, ''),
			COALESCE(categoria, ''),
			COALESCE(icone, 'success'),
			COALESCE(gerado_em::text, NOW()::text)
		FROM %s.insights_cache
		ORDER BY prioridade ASC
		LIMIT 10
	`, schema))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar insights: %w", err)
	}
	defer rows.Close()

	var insights []Insight
	for rows.Next() {
		var i Insight
		if err := rows.Scan(
			&i.Tipo, &i.Prioridade, &i.Titulo,
			&i.Mensagem, &i.Acao, &i.Href,
			&i.Categoria, &i.Icone, &i.GeradoEm,
		); err != nil {
			continue
		}
		insights = append(insights, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar insights: %w", err)
	}
	return insights, nil
}
