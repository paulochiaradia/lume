package normalizer

import (
	"fmt"
	"log/slog"

	"github.com/paulochiaradia/lume/collector/internal/connector"
)

func NormalizeClientes(records []connector.RawRecord) ([]Cliente, []string) {
	var clientes []Cliente
	var errors []string
	skipped := 0

	for i, record := range records {
		clienteKey := safeGet(record,
			"id", "codigo", "cliente_id", "id_cliente",
		)

		// Detecção de anomalia na qualidade dos dados (Falta de Chave Primária)
		if clienteKey == "" {
			skipped++
			errMsg := fmt.Sprintf("linha %d ignorada: chave do cliente não encontrada", i+1)
			errors = append(errors, errMsg)

			// [SLOG] Telemetria de descarte de dados
			slog.Warn("registro de cliente ignorado por falta de id",
				slog.String("event", "normalize_cliente_skipped"),
				slog.Int("row_index", i),
			)
			continue
		}

		cliente := Cliente{
			ClienteKey: clienteKey,
			Nome:       safeGet(record, "nome", "name", "razao_social"),
			Tipo:       safeGet(record, "tipo", "type", "perfil"),
			Documento:  safeGet(record, "documento", "cpf", "cnpj"),
			Telefone:   safeGet(record, "telefone", "fone", "phone"),
			Cidade:     safeGet(record, "cidade", "city"),
			Bairro:     safeGet(record, "bairro", "district"),
			CEP:        safeGet(record, "cep", "zipcode"),
			Ativo:      parseBool(safeGet(record, "ativo", "active", "status")),
			Atributos:  make(map[string]interface{}),
		}

		if cliente.Nome == "" {
			cliente.Nome = cliente.ClienteKey
		}

		clientes = append(clientes, cliente)
	}

	// [SLOG] Resumo final do processo de normalização do lote
	slog.Info("normalizacao de clientes concluida",
		slog.String("event", "normalize_clientes_success"),
		slog.Int("total_recebido", len(records)),
		slog.Int("normalizados", len(clientes)),
		slog.Int("ignorados", skipped),
	)

	return clientes, errors
}
