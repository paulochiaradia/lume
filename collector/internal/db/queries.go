package db

import (
	"database/sql"
	"fmt"
	"time"
)

// ── Structs ──────────────────────────────────────────────────

type Client struct {
	ID        string
	ClientKey string
	Name      string
	Segment   string
	ERPType   string
	ERPConfig []byte
	Active    bool
}

type ETLLog struct {
	ID             string
	ClientID       string
	ConnectorType  string
	Status         string
	RecordsRead    int
	RecordsWritten int
	ErrorMessage   string
	StartedAt      time.Time
	FinishedAt     *time.Time
}

// User representa um usuário do sistema
type User struct {
	ID           string
	ClientID     string
	Email        string
	PasswordHash string
	Role         string
	Name         string
	Active       bool
}

// ── Clients ──────────────────────────────────────────────────

// GetActiveClients retorna todos os clientes ativos
func GetActiveClients(db *sql.DB) ([]Client, error) {
	rows, err := db.Query(`
		SELECT id, client_key, name, segment, erp_type, erp_config, active
		FROM lume_system.clients
		WHERE active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		if err := rows.Scan(
			&c.ID, &c.ClientKey, &c.Name,
			&c.Segment, &c.ERPType, &c.ERPConfig, &c.Active,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler cliente: %w", err)
		}
		clients = append(clients, c)
	}

	return clients, nil
}

// GetClientByKey retorna um cliente pelo client_key
func GetClientByKey(db *sql.DB, clientKey string) (*Client, error) {
	var c Client
	err := db.QueryRow(`
		SELECT id, client_key, name, segment, erp_type, erp_config, active
		FROM lume_system.clients
		WHERE client_key = $1 AND active = true
	`, clientKey).Scan(
		&c.ID, &c.ClientKey, &c.Name,
		&c.Segment, &c.ERPType, &c.ERPConfig, &c.Active,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cliente %s não encontrado", clientKey)
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}
	return &c, nil
}

// ── ETL Logs ─────────────────────────────────────────────────

// InsertETLLog cria um novo registro de log e retorna o ID
func InsertETLLog(db *sql.DB, clientID, connectorType string) (string, error) {
	var id string
	err := db.QueryRow(`
		INSERT INTO lume_system.etl_logs (client_id, connector_type, status)
		VALUES ($1, $2, 'running')
		RETURNING id
	`, clientID, connectorType).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("erro ao criar etl_log: %w", err)
	}
	return id, nil
}

// UpdateETLLogSuccess marca o log como sucesso
func UpdateETLLogSuccess(db *sql.DB, logID string, recordsRead, recordsWritten int) error {
	_, err := db.Exec(`
		UPDATE lume_system.etl_logs
		SET status = 'success',
		    records_read = $2,
		    records_written = $3,
		    finished_at = NOW()
		WHERE id = $1
	`, logID, recordsRead, recordsWritten)
	if err != nil {
		return fmt.Errorf("erro ao atualizar etl_log: %w", err)
	}
	return nil
}

// UpdateETLLogError marca o log como erro
func UpdateETLLogError(db *sql.DB, logID string, errMsg string) error {
	_, err := db.Exec(`
		UPDATE lume_system.etl_logs
		SET status = 'error',
		    error_message = $2,
		    finished_at = NOW()
		WHERE id = $1
	`, logID, errMsg)
	if err != nil {
		return fmt.Errorf("erro ao atualizar etl_log com erro: %w", err)
	}
	return nil
}

// ── Jobs ─────────────────────────────────────────────────────

// EnqueueJob enfileira um job para o Python processar
func EnqueueJob(db *sql.DB, clientID, jobType string, payload []byte) error {
	_, err := db.Exec(`
		INSERT INTO lume_system.jobs (client_id, job_type, payload)
		VALUES ($1, $2, $3)
	`, clientID, jobType, payload)
	if err != nil {
		return fmt.Errorf("erro ao enfileirar job: %w", err)
	}
	return nil
}

// ── Users ─────────────────────────────────────────────────
// GetUserByEmail busca um usuário pelo email
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	var u User
	err := db.QueryRow(`
		SELECT id, client_id, email, password_hash, role, name, active
		FROM lume_system.users
		WHERE email = $1 AND active = true
	`, email).Scan(&u.ID, &u.ClientID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.Active)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("usuário não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	return &u, nil
}

// GetClientByID busca um cliente pelo ID
func GetClientByID(db *sql.DB, clientID string) (*Client, error) {
	var c Client
	err := db.QueryRow(`
		SELECT id, client_key, name, segment, erp_type, erp_config, active
		FROM lume_system.clients
		WHERE id = $1
	`, clientID).Scan(
		&c.ID, &c.ClientKey, &c.Name,
		&c.Segment, &c.ERPType, &c.ERPConfig, &c.Active,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cliente não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}
	return &c, nil
}

// ── Home KPIs ────────────────────────────────────────────────

type HomeKPIs struct {
	Faturamento   float64 `json:"faturamento"`
	TicketMedio   float64 `json:"ticket_medio"`
	TotalVendas   int     `json:"total_vendas"`
	ItensPorVenda float64 `json:"itens_por_venda"`
}

// GetHomeKPIs busca os indicadores do mês atual
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

// ── Vendas ───────────────────────────────────────────────────

type VendasResumo struct {
	Faturamento   float64 `json:"faturamento"`
	TotalVendas   int     `json:"total_vendas"`
	TicketMedio   float64 `json:"ticket_medio"`
	TotalDesconto float64 `json:"total_desconto"`
}

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

type VendaDia struct {
	Dia         string  `json:"dia"`
	Vendas      int     `json:"vendas"`
	Faturamento float64 `json:"faturamento"`
}

// GetVendasPorDia busca as vendas agrupadas por dia a partir de uma data inicial
func GetVendasPorDia(db *sql.DB, clientKey string, startTime time.Time) ([]VendaDia, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			DATE(data_venda)::text,
			COUNT(*),
			COALESCE(SUM(total), 0)
		FROM %s.vendas
		WHERE status = 'concluida'
		  AND data_venda >= $1
		GROUP BY DATE(data_venda)
		ORDER BY DATE(data_venda) ASC
	`, schema), startTime)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar vendas por dia: %w", err)
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

	return vendas, nil
}

// GetTopDias retorna os 5 dias com maior faturamento dentro da janela selecionada
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

	return vendas, nil
}

// ── Estoque ──────────────────────────────────────────────────

type EstoqueAlerta struct {
	ProdutoKey    string  `json:"produto_key"`
	Nome          string  `json:"nome"`
	Quantidade    float64 `json:"quantidade"`
	QuantidadeMin float64 `json:"quantidade_min"`
}

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

	return alertas, nil
}

// ── Produtos ABC ─────────────────────────────────────────────

type ProdutoABC struct {
	ProdutoKey  string  `json:"produto_key"`
	Nome        string  `json:"nome"`
	Categoria   string  `json:"categoria"`
	Faturamento float64 `json:"faturamento"`
	Classe      string  `json:"classe"`
}

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

	return produtos, nil
}

// ── Clientes RFM ─────────────────────────────────────────────

type ClienteRFM struct {
	ClienteID  string  `json:"cliente_id"`
	Recencia   int     `json:"recencia"`
	Frequencia int     `json:"frequencia"`
	ValorTotal float64 `json:"valor_total"`
	ScoreR     int     `json:"score_r"`
	ScoreF     int     `json:"score_f"`
	ScoreM     int     `json:"score_m"`
	ScoreRFM   string  `json:"score_rfm"`
	ScoreTotal int     `json:"score_total"`
	Segmento   string  `json:"segmento"`
}

type ResumoSegmento struct {
	Segmento      string  `json:"segmento"`
	Clientes      int     `json:"clientes"`
	ValorMedio    float64 `json:"valor_medio"`
	ValorTotal    float64 `json:"valor_total"`
	RecenciaMedia float64 `json:"recencia_media"`
}

func GetClientesRFM(db *sql.DB, clientKey string) ([]ClienteRFM, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		WITH ultima_data AS (
			SELECT MAX(data_venda) AS max_data FROM %s.vendas
		)
		SELECT
			c.cliente_key,
			COUNT(v.id) AS frequencia,
			COALESCE(SUM(v.total), 0) AS valor_total,
			EXTRACT(DAY FROM (SELECT max_data FROM ultima_data) - MAX(v.data_venda))::int AS recencia
		FROM %s.clientes c
		LEFT JOIN %s.vendas v ON v.cliente_id = c.id
		WHERE c.ativo = true
		GROUP BY c.cliente_key
		HAVING COUNT(v.id) > 0
		ORDER BY valor_total DESC
	`, schema, schema, schema))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	var clientes []ClienteRFM
	for rows.Next() {
		var c ClienteRFM
		if err := rows.Scan(
			&c.ClienteID, &c.Frequencia,
			&c.ValorTotal, &c.Recencia,
		); err != nil {
			continue
		}
		clientes = append(clientes, c)
	}

	return clientes, nil
}
func GetResumoSegmentos(db *sql.DB, clientKey string) ([]ResumoSegmento, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		WITH ultima_data AS (
			SELECT MAX(data_venda) AS max_data FROM %s.vendas
		),
		rfm AS (
			SELECT
				c.cliente_key,
				COUNT(v.id) AS frequencia,
				COALESCE(SUM(v.total), 0) AS valor_total,
				EXTRACT(DAY FROM (SELECT max_data FROM ultima_data) - MAX(v.data_venda))::int AS recencia
			FROM %s.clientes c
			LEFT JOIN %s.vendas v ON v.cliente_id = c.id
			WHERE c.ativo = true
			GROUP BY c.cliente_key
			HAVING COUNT(v.id) > 0
		),
		percentis AS (
			SELECT
				PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY recencia) AS p25_r,
				PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY recencia) AS p75_r,
				PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY frequencia) AS p25_f,
				PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY frequencia) AS p75_f,
				PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY valor_total) AS p25_m,
				PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY valor_total) AS p75_m
			FROM rfm
		),
		scored AS (
			SELECT
				r.*,
				CASE WHEN r.recencia  <= p.p25_r THEN 3
				     WHEN r.recencia  <= p.p75_r THEN 2
				     ELSE 1 END AS score_r,
				CASE WHEN r.frequencia >= p.p75_f THEN 3
				     WHEN r.frequencia >= p.p25_f THEN 2
				     ELSE 1 END AS score_f,
				CASE WHEN r.valor_total >= p.p75_m THEN 3
				     WHEN r.valor_total >= p.p25_m THEN 2
				     ELSE 1 END AS score_m
			FROM rfm r, percentis p
		)
		SELECT
			CASE
				WHEN score_r = 3 AND score_f = 3 AND score_m = 3 THEN 'campeon'
				WHEN score_f >= 2 AND score_m >= 2               THEN 'fiel'
				WHEN score_r = 1 AND score_f >= 2                THEN 'em_risco'
				WHEN score_r = 1 AND score_f = 1                 THEN 'perdido'
				WHEN score_r = 3 AND score_f = 1                 THEN 'promissor'
				WHEN score_r >= 2 AND score_f = 1                THEN 'novo'
				WHEN score_r <= 2 AND score_f >= 2               THEN 'hibernando'
				ELSE 'outros'
			END AS segmento,
			COUNT(*) AS clientes,
			ROUND(AVG(valor_total)::numeric, 2) AS valor_medio,
			ROUND(SUM(valor_total)::numeric, 2) AS valor_total,
			ROUND(AVG(recencia)::numeric, 2) AS recencia_media
		FROM scored
		GROUP BY 1
		ORDER BY valor_total DESC
	`, schema, schema, schema))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar segmentos: %w", err)
	}
	defer rows.Close()

	var segmentos []ResumoSegmento
	for rows.Next() {
		var s ResumoSegmento
		if err := rows.Scan(
			&s.Segmento, &s.Clientes,
			&s.ValorMedio, &s.ValorTotal, &s.RecenciaMedia,
		); err != nil {
			continue
		}
		segmentos = append(segmentos, s)
	}

	return segmentos, nil
}

// ── Estoque completo ─────────────────────────────────────────

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

	return itens, nil
}

// ── Insights ─────────────────────────────────────────────────

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

	return insights, nil
}

// ── Vendas por Hora (Pico de Atendimento) ────────────────────

type VendaHora struct {
	Hora        string  `json:"hora"`
	Faturamento float64 `json:"faturamento"`
}

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

	return vendas, nil
}

// ── Mix de Vendas por Categoria (Doughnut Chart) ─────────────

type MixCategoria struct {
	Categoria   string  `json:"categoria"`
	Faturamento float64 `json:"faturamento"`
}

func GetMixCategorias(db *sql.DB, clientKey string, startTime time.Time) ([]MixCategoria, error) {
	schema := "client_" + clientKey

	// Fazemos o JOIN com 'vendas' para filtrar pela data e status
	// e com 'produtos' para pegar o nome da categoria
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

	return categorias, nil
}

// TrendValue armazena o valor atual, o passado e o crescimento percentual
type TrendValue struct {
	Atual    float64 `json:"atual"`
	Anterior float64 `json:"anterior"`
	Variacao float64 `json:"variacao"`
}

// VendasKPIsTrend representa o Bloco 1 da página de Vendas
type VendasKPIsTrend struct {
	Faturamento     TrendValue `json:"faturamento"`
	TicketMedio     TrendValue `json:"ticket_medio"`
	TotalTransacoes TrendValue `json:"total_transacoes"`
	PercDesconto    TrendValue `json:"perc_desconto"`
}

// GetVendasKPIsTrend calcula os KPIs comparando o período atual com o anterior
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

	// Helpers de cálculo seguro (evita divisão por zero)
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

	// 1. Faturamento
	kpis.Faturamento = TrendValue{
		Atual: fatAtual, Anterior: fatAnt, Variacao: calcTrend(fatAtual, fatAnt),
	}

	// 2. Transações
	kpis.TotalTransacoes = TrendValue{
		Atual: float64(transAtual), Anterior: float64(transAnt), Variacao: calcTrend(float64(transAtual), float64(transAnt)),
	}

	// 3. Ticket Médio
	tmAtual := safeDiv(fatAtual, transAtual)
	tmAnt := safeDiv(fatAnt, transAnt)
	kpis.TicketMedio = TrendValue{
		Atual: tmAtual, Anterior: tmAnt, Variacao: calcTrend(tmAtual, tmAnt),
	}

	// 4. % Desconto (Variação em Pontos Percentuais)
	percDescAtual := calcPercDesconto(fatAtual, descAtual)
	percDescAnt := calcPercDesconto(fatAnt, descAnt)
	kpis.PercDesconto = TrendValue{
		Atual: percDescAtual, Anterior: percDescAnt, Variacao: percDescAtual - percDescAnt, // Diferença direta
	}

	return kpis, nil
}
