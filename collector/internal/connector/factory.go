package connector

import "fmt"

// Factory retorna o conector correto baseado no erp_type do cliente
// Para adicionar um novo ERP: criar o arquivo e registrar aqui
func Factory(cfg Config) (Connector, error) {
	switch cfg.ERPType {
	// case "csv":
	// 	return NewCSVConnector(cfg), nil  ← será descomentado na próxima tarefa
	default:
		return nil, fmt.Errorf("conector '%s' não implementado", cfg.ERPType)
	}
}
