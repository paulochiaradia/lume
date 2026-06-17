package db

import (
	"database/sql"
	"fmt"
	"time"
)

// ── Structs Base do Sistema ──────────────────────────────────

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

// ── ETL Logs ─────────────────────────────────────────────────

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

// ── Users ────────────────────────────────────────────────────

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

// ── Home KPIs ────────────────────────────────────────────────

type HomeKPIs struct {
	Faturamento   float64 `json:"faturamento"`
	TicketMedio   float64 `json:"ticket_medio"`
	TotalVendas   int     `json:"total_vendas"`
	ItensPorVenda float64 `json:"itens_por_venda"`
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

// ── Vendas (Módulo Novo) ─────────────────────────────────────

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
	MediaMovel  float64 `json:"mediaMovel"`
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
	return vendas, nil
}

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

type MixCategoria struct {
	Categoria   string  `json:"categoria"`
	Faturamento float64 `json:"faturamento"`
}

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
	return categorias, nil
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

// ── Ranking de Vendedores ────────────────────────────────────

type RankingVendedor struct {
	ID          string  `json:"id"`
	Nome        string  `json:"nome"`
	Vendas      int     `json:"vendas"`
	Faturamento float64 `json:"faturamento"`
	Ticket      float64 `json:"ticket"`
	UPA         float64 `json:"upa"`
	Desconto    float64 `json:"desconto"`
}

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

	return ranking, nil
}

// ── Heatmap Comercial ────────────────────────────────────────

type HeatmapPonto struct {
	Dia         string  `json:"dia"`
	Hora        string  `json:"hora"`
	Faturamento float64 `json:"faturamento"`
}

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
	return insights, nil
}

// ── ESTRUTURAS PARA O PAINEL DE PRODUTOS ──────────────────────

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

type DeadStockProduto struct {
	Nome          string  `json:"nome"`
	Quantidade    float64 `json:"quantidade"`
	PrecoCusto    float64 `json:"preco_custo"`
	CapitalParado float64 `json:"capital_parado"`
}

// ── FUNÇÕES DE BUSCA ─────────────────────────────────────────

// 1. KPIs DO CATÁLOGO (SKUs, Margem, Resumo Classe A e Dead Stock agrupados)
func GetCatalogKPIs(db *sql.DB, clientKey string) (*CatalogKPIs, error) {
	schema := "client_" + clientKey
	var kpis CatalogKPIs

	// Total SKUs ativos
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s.produtos", schema)).Scan(&kpis.TotalSKUs)
	if err != nil {
		return nil, err
	}

	// Classe A da tabela abc_xyz_cache
	_ = db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*), COALESCE(SUM(faturamento), 0) 
		FROM %s.abc_xyz_cache WHERE classe = 'A'
	`, schema)).Scan(&kpis.QtdClasseA, &kpis.PctFaturamentoA)

	// Margem média da cache de elasticidade
	_ = db.QueryRow(fmt.Sprintf("SELECT COALESCE(AVG(pct_margem), 0) FROM %s.margem_resultado", schema)).Scan(&kpis.MargemMedia)

	// Dead Stock (usando a sua query de especificação)
	_ = db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(p.id), COALESCE(SUM(e.quantidade * p.preco_custo), 0)
		FROM %s.produtos p
		JOIN %s.estoque e ON e.produto_key = p.produto_key
		LEFT JOIN %s.itens_venda iv ON iv.produto_key = p.produto_key AND iv.data_venda >= NOW() - INTERVAL '90 days'
		WHERE iv.id IS NULL AND e.quantidade > 0
	`, schema, schema, schema)).Scan(&kpis.DeadStockSKUs, &kpis.CapitalParado)

	return &kpis, nil
}

// 2. MATRIZ ABC x XYZ
func GetMatrizABCXYZ(db *sql.DB, clientKey string) ([]MatrizABCXYZ, error) {
	schema := "client_" + clientKey
	rows, err := db.Query(fmt.Sprintf(`
		SELECT classe, xyz, COUNT(*) 
		FROM %s.abc_xyz_cache 
		GROUP BY classe, xyz
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
	return matriz, nil
}

// 3. RANKING DE PRODUTOS COM TENDÊNCIA (Window Function comparando últimos 28 dias vs 28 anteriores)
func GetRankingProdutos(db *sql.DB, clientKey string) ([]RankingProduto, error) {
	schema := "client_" + clientKey
	rows, err := db.Query(fmt.Sprintf(`
		WITH faturamento_periodos AS (
			SELECT 
				produto_key,
				SUM(CASE WHEN data_venda >= NOW() - INTERVAL '28 days' THEN total ELSE 0 END) as fat_atual,
				SUM(CASE WHEN data_venda >= NOW() - INTERVAL '56 days' AND data_venda < NOW() - INTERVAL '28 days' THEN total ELSE 0 END) as fat_anterior
			FROM %s.itens_venda
			GROUP BY produto_key
		)
		SELECT 
			p.produto_key,
			p.nome,
			COALESCE(p.categoria, 'Geral'),
			COALESCE(fp.fat_atual, 0) as faturamento,
			COALESCE(m.pct_margem, 0) as margem,
			COALESCE(cache.classe, 'C') as classe_abc,
			COALESCE(cache.xyz, 'Z') as classe_xyz,
			CASE 
				WHEN COALESCE(fp.fat_atual, 0) > COALESCE(fp.fat_anterior, 0) * 1.05 THEN 'subindo'
				WHEN COALESCE(fp.fat_atual, 0) < COALESCE(fp.fat_anterior, 0) * 0.95 THEN 'caindo'
				ELSE 'estavel'
			END as tendencia
		FROM %s.produtos p
		LEFT JOIN faturamento_periodos fp ON fp.produto_key = p.produto_key
		LEFT JOIN %s.abc_xyz_cache cache ON cache.produto_key = p.produto_key
		LEFT JOIN %s.margem_resultado m ON m.produto_key = p.produto_key
		ORDER BY faturamento DESC
	`, schema, schema, schema, schema))

	if err != nil {
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
			continue
		}
		lista = append(lista, rp)
	}
	return lista, nil
}

// 4. ELASTICIDADE DE PREÇO
func GetElasticidadeProdutos(db *sql.DB, clientKey string) ([]ElasticidadeProduto, error) {
	schema := "client_" + clientKey
	rows, err := db.Query(fmt.Sprintf(`
		SELECT produto_key, elasticidade, interpretacao, receita 
		FROM %s.elasticidade_cache 
		WHERE receita > 0
		ORDER BY receita DESC
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
	return lista, nil
}

// 5. PRODUTOS COMPRADOS JUNTOS (Market Basket)
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
	return lista, nil
}

// 6. DEAD STOCK / PRODUTOS SEM GIRO (Sua Query de Especificação Completa)
func GetDeadStock(db *sql.DB, clientKey string) ([]DeadStockProduto, error) {
	schema := "client_" + clientKey
	rows, err := db.Query(fmt.Sprintf(`
		SELECT p.nome, e.quantidade, p.preco_custo, (e.quantidade * p.preco_custo) as capital_parado
		FROM %s.produtos p
		JOIN %s.estoque e ON e.produto_key = p.produto_key
		LEFT JOIN %s.itens_venda iv ON iv.produto_key = p.produto_key AND iv.data_venda >= NOW() - INTERVAL '90 days'
		WHERE iv.id IS NULL AND e.quantidade > 0
		ORDER BY capital_parado DESC
	`, schema, schema, schema))

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
	return lista, nil
}
