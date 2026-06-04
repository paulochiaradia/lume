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
	TotalDesconto float64 `json:"total_desconto"`
}

func GetHomeKPIs(db *sql.DB, clientKey string) (*HomeKPIs, error) {
	schema := "client_" + clientKey
	var kpis HomeKPIs

	err := db.QueryRow(fmt.Sprintf(`
		SELECT
			COALESCE(SUM(total), 0),
			COALESCE(AVG(total), 0),
			COUNT(*),
			COALESCE(SUM(desconto), 0)
		FROM %s.vendas
		WHERE status = 'concluida'
	`, schema)).Scan(
		&kpis.Faturamento,
		&kpis.TicketMedio,
		&kpis.TotalVendas,
		&kpis.TotalDesconto,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar KPIs: %w", err)
	}

	return &kpis, nil
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

func GetVendasPorDia(db *sql.DB, clientKey string) ([]VendaDia, error) {
	schema := "client_" + clientKey

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			DATE(data_venda)::text,
			COUNT(*),
			COALESCE(SUM(total), 0)
		FROM %s.vendas
		WHERE status = 'concluida'
		GROUP BY DATE(data_venda)
		ORDER BY DATE(data_venda)
	`, schema))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar vendas por dia: %w", err)
	}
	defer rows.Close()

	var vendas []VendaDia
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
