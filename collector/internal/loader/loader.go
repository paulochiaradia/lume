package loader

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/paulochiaradia/lume/collector/internal/normalizer"
)

// Loader escreve os dados normalizados no PostgreSQL
type Loader struct {
	db        *sql.DB
	clientKey string
	schema    string
}

// New cria uma nova instância do Loader para um cliente
func New(db *sql.DB, clientKey string) *Loader {
	return &Loader{
		db:        db,
		clientKey: clientKey,
		schema:    "client_" + clientKey,
	}
}

// LoadVendas faz upsert das vendas no banco
func (l *Loader) LoadVendas(vendas []normalizer.Venda) (int, error) {
	if len(vendas) == 0 {
		return 0, nil
	}

	tx, err := l.db.Begin()
	if err != nil {
		slog.Error("erro ao iniciar transacao para vendas", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(fmt.Sprintf(`
		INSERT INTO %s.vendas 
			(venda_key, data_venda, cliente_id, vendedor_id, total, desconto, status, canal, atributos)
		SELECT 
			$1, $2,
			(SELECT id FROM %s.clientes WHERE cliente_key = $3 LIMIT 1),
			$4, $5, $6, $7, $8, $9
		ON CONFLICT (venda_key) DO UPDATE SET
			data_venda  = EXCLUDED.data_venda,
			cliente_id  = EXCLUDED.cliente_id,
			vendedor_id = EXCLUDED.vendedor_id,
			total       = EXCLUDED.total,
			desconto    = EXCLUDED.desconto,
			status      = EXCLUDED.status,
			canal       = EXCLUDED.canal,
			atributos   = EXCLUDED.atributos
	`, l.schema, l.schema))
	if err != nil {
		slog.Error("erro ao preparar statement de vendas", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao preparar statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, v := range vendas {
		atributos, _ := json.Marshal(v.Atributos)

		_, err := stmt.Exec(
			v.VendaKey,
			v.DataVenda,
			nullableString(v.ClienteKey),
			nullableString(v.VendedorID),
			v.Total,
			v.Desconto,
			v.Status,
			v.Canal,
			atributos,
		)
		if err != nil {
			// [SLOG] Loga o erro específico deste registro, mas permite que o lote continue
			slog.Warn("erro ao inserir venda (registro ignorado)",
				slog.String("event", "etl_load_venda_error"),
				slog.String("tenant_id", l.clientKey),
				slog.String("venda_key", v.VendaKey),
				slog.String("error", err.Error()),
			)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		slog.Error("erro ao commitar lote de vendas", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao commitar vendas: %w", err)
	}

	// [SLOG] Resumo de sucesso do batch
	slog.Info("lote de vendas processado",
		slog.String("event", "etl_load_vendas_success"),
		slog.String("tenant_id", l.clientKey),
		slog.Int("inseridos", count),
		slog.Int("total_recebido", len(vendas)),
	)

	return count, nil
}

// LoadProdutos faz upsert dos produtos no banco
func (l *Loader) LoadProdutos(produtos []normalizer.Produto) (int, error) {
	if len(produtos) == 0 {
		return 0, nil
	}

	tx, err := l.db.Begin()
	if err != nil {
		slog.Error("erro ao iniciar transacao para produtos", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(fmt.Sprintf(`
		INSERT INTO %s.produtos
			(produto_key, nome, categoria, subcategoria, unidade, 
			 preco_custo, preco_venda, ativo, atributos)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (produto_key) DO UPDATE SET
			nome         = EXCLUDED.nome,
			categoria    = EXCLUDED.categoria,
			subcategoria = EXCLUDED.subcategoria,
			unidade      = EXCLUDED.unidade,
			preco_custo  = EXCLUDED.preco_custo,
			preco_venda  = EXCLUDED.preco_venda,
			ativo        = EXCLUDED.ativo,
			atributos    = EXCLUDED.atributos,
			updated_at   = NOW()
	`, l.schema))
	if err != nil {
		slog.Error("erro ao preparar statement de produtos", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao preparar statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, p := range produtos {
		atributos, _ := json.Marshal(p.Atributos)

		_, err := stmt.Exec(
			p.ProdutoKey,
			p.Nome,
			nullableString(p.Categoria),
			nullableString(p.Subcategoria),
			p.Unidade,
			p.PrecoCusto,
			p.PrecoVenda,
			p.Ativo,
			atributos,
		)
		if err != nil {
			slog.Warn("erro ao inserir produto (registro ignorado)",
				slog.String("event", "etl_load_produto_error"),
				slog.String("tenant_id", l.clientKey),
				slog.String("produto_key", p.ProdutoKey),
				slog.String("error", err.Error()),
			)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		slog.Error("erro ao commitar lote de produtos", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao commitar produtos: %w", err)
	}

	slog.Info("lote de produtos processado",
		slog.String("event", "etl_load_produtos_success"),
		slog.String("tenant_id", l.clientKey),
		slog.Int("inseridos", count),
		slog.Int("total_recebido", len(produtos)),
	)

	return count, nil
}

// LoadItensVenda faz upsert dos itens de venda no banco
func (l *Loader) LoadItensVenda(itens []normalizer.ItemVenda) (int, error) {
	if len(itens) == 0 {
		return 0, nil
	}

	tx, err := l.db.Begin()
	if err != nil {
		slog.Error("erro ao iniciar transacao para itens_venda", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(fmt.Sprintf(`
		INSERT INTO %s.itens_venda
			(venda_id, produto_key, descricao, quantidade, unidade,
			 preco_unitario, desconto, total, atributos)
		SELECT
			v.id, $2, $3, $4, $5, $6, $7, $8, $9
		FROM %s.vendas v
		WHERE v.venda_key = $1
		ON CONFLICT DO NOTHING
	`, l.schema, l.schema))
	if err != nil {
		slog.Error("erro ao preparar statement de itens_venda", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao preparar statement de itens: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, item := range itens {
		atributos, _ := json.Marshal(item.Atributos)

		_, err := stmt.Exec(
			item.VendaKey,
			item.ProdutoKey,
			item.Descricao,
			item.Quantidade,
			item.Unidade,
			item.PrecoUnitario,
			item.Desconto,
			item.Total,
			atributos,
		)
		if err != nil {
			slog.Warn("erro ao inserir item da venda (registro ignorado)",
				slog.String("event", "etl_load_item_venda_error"),
				slog.String("tenant_id", l.clientKey),
				slog.String("venda_key", item.VendaKey),
				slog.String("produto_key", item.ProdutoKey),
				slog.String("error", err.Error()),
			)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		slog.Error("erro ao commitar lote de itens_venda", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao commitar itens: %w", err)
	}

	slog.Info("lote de itens de venda processado",
		slog.String("event", "etl_load_itens_venda_success"),
		slog.String("tenant_id", l.clientKey),
		slog.Int("inseridos", count),
		slog.Int("total_recebido", len(itens)),
	)

	return count, nil
}

// LoadClientes faz upsert dos clientes no banco
func (l *Loader) LoadClientes(clientes []normalizer.Cliente) (int, error) {
	if len(clientes) == 0 {
		return 0, nil
	}

	tx, err := l.db.Begin()
	if err != nil {
		slog.Error("erro ao iniciar transacao para clientes", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(fmt.Sprintf(`
		INSERT INTO %s.clientes
			(cliente_key, nome, tipo, documento, telefone, cidade, bairro, cep, ativo, atributos)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (cliente_key) DO UPDATE SET
			nome       = EXCLUDED.nome,
			tipo       = EXCLUDED.tipo,
			cidade     = EXCLUDED.cidade,
			bairro     = EXCLUDED.bairro,
			ativo      = EXCLUDED.ativo,
			updated_at = NOW()
	`, l.schema))
	if err != nil {
		slog.Error("erro ao preparar statement de clientes", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao preparar statement de clientes: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, c := range clientes {
		atributos, _ := json.Marshal(c.Atributos)

		_, err := stmt.Exec(
			c.ClienteKey,
			c.Nome,
			nullableString(c.Tipo),
			nullableString(c.Documento),
			nullableString(c.Telefone),
			nullableString(c.Cidade),
			nullableString(c.Bairro),
			nullableString(c.CEP),
			c.Ativo,
			atributos,
		)
		if err != nil {
			slog.Warn("erro ao inserir cliente (registro ignorado)",
				slog.String("event", "etl_load_cliente_error"),
				slog.String("tenant_id", l.clientKey),
				slog.String("cliente_key", c.ClienteKey),
				slog.String("error", err.Error()),
			)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		slog.Error("erro ao commitar lote de clientes", slog.String("tenant_id", l.clientKey), slog.String("error", err.Error()))
		return 0, fmt.Errorf("erro ao commitar clientes: %w", err)
	}

	slog.Info("lote de clientes processado",
		slog.String("event", "etl_load_clientes_success"),
		slog.String("tenant_id", l.clientKey),
		slog.Int("inseridos", count),
		slog.Int("total_recebido", len(clientes)),
	)

	return count, nil
}

// nullableString converte string vazia para nil (NULL no banco)
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
