package normalizer

import (
	"github.com/paulochiaradia/lume/collector/internal/connector"
)

func NormalizeClientes(records []connector.RawRecord) ([]Cliente, []string) {
	var clientes []Cliente
	var errors []string

	for _, record := range records {
		clienteKey := safeGet(record,
			"id", "codigo", "cliente_id", "id_cliente",
		)
		if clienteKey == "" {
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

	return clientes, errors
}
