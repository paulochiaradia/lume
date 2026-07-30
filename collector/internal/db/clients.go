package db

import (
	"database/sql"
	"fmt"
)

type Client struct {
	ID        string
	ClientKey string
	Name      string
	Segment   string
	ERPType   string
	ERPConfig []byte
	Active    bool
}

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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar clientes: %w", err)
	}
	return clients, nil
}

// GetClientByKey busca os dados do cliente usando a chave
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

// GetClientByID busca os dados do cliente usando o ID
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

// GetClientBySegment busca os clientes por segmento
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar clientes rfm: %w", err)
	}
	return clientes, nil
}

// GetResumoSegmentos retorna o resumo dos segmentos de clientes
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar segmentos: %w", err)
	}
	return segmentos, nil
}
