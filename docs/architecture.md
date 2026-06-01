# Lume — Arquitetura do Sistema

## Visão geral

A Lume é composta por dois serviços principais que se comunicam através do banco de dados:

- **collector** (Go) — ingestão, normalização, API e scheduler
- **engine** (Python) — algoritmos de ML, LLM e worker de jobs
ERP → Collector (Go) → PostgreSQL → Engine (Python) → DuckDB → Streamlit

---

## Decisões de arquitetura

### Por que Go para o collector?
Concorrência nativa com goroutines — cada cliente roda seu pipeline em paralelo sem
bloquear os demais. Binário único sem dependências. Performance superior em I/O.

### Por que Python para o engine?
Ecossistema científico insubstituível — pandas, scikit-learn, Prophet, DuckDB.
O Python não expõe HTTP — é um worker puro que consome jobs e grava resultados.

### Por que DuckDB para analytics?
Motor OLAP que roda em processo, sem servidor. Queries analíticas em cima de grandes
volumes com latência mínima. Um arquivo `.duckdb` por cliente — isolamento total.

### Por que um schema por cliente no PostgreSQL?
Isolamento real de dados sem complexidade de múltiplas instâncias. Fácil de deletar
um cliente por completo (LGPD). Sem risco de um cliente ver dados de outro.

---

## Banco de dados

### PostgreSQL — banco operacional

#### Schema `lume_system` — tabelas internas

| Tabela | Responsabilidade |
|---|---|
| `clients` | Clientes ativos, configurações de ERP e segmento |
| `users` | Usuários com acesso ao dashboard e seus roles |
| `etl_logs` | Log de cada execução de pipeline — lido pelo Grafana |
| `jobs` | Fila de jobs para o engine Python processar |
| `schema_migrations` | Controle de versão do banco |

#### Schema `client_<client_key>` — dados de cada cliente

| Tabela | Responsabilidade |
|---|---|
| `vendas` | Cabeçalho de cada venda |
| `itens_venda` | Produtos de cada venda |
| `produtos` | Catálogo de produtos |
| `clientes` | Cadastro de clientes da loja |
| `estoque` | Posição atual de estoque por produto |
| `movimentos_estoque` | Histórico de entradas e saídas |
| `fornecedores` | Cadastro de fornecedores |

#### Campo `atributos JSONB`
Todas as tabelas do schema canônico possuem uma coluna `atributos JSONB`.
Ela armazena dados específicos de segmento sem poluir o schema genérico.

Exemplos:
- Construção: `{"tipo_obra": "residencial", "metragem": 120}`
- Farmácia: `{"convenio": "Unimed", "principio_ativo": "Paracetamol"}`
- Pet shop: `{"especie": "cachorro", "porte": "grande"}`

---

## Sistema de migrations

Migrations ficam em `infra/postgres/migrations/` numeradas sequencialmente.
O serviço `collector/cmd/migrate` as executa em ordem e registra em `schema_migrations`.

**Regras:**
- Nunca editar uma migration já aplicada — criar uma nova
- Sempre usar `IF NOT EXISTS` para garantir idempotência
- Cada migration roda dentro de uma transação — falhou, fez rollback

---

## Multi-segmento

Adicionar um novo segmento nunca altera o schema canônico nem o código existente.
O que muda por segmento fica exclusivamente em `engine/segments/<segmento>/`:
engine/segments/construcao/
```text
engine.py       → herda SegmentEngine, sobrescreve o que é específico
config.py       → KPIs, janela de churn, parâmetros de sazonalidade
kpis.py         → cálculos exclusivos do segmento
terminology.py  → labels da interface em português do setor
```
---

## Multi-ERP

Adicionar um novo conector de ERP nunca altera o normalizer nem o loader.
O que muda por ERP fica exclusivamente em `collector/internal/connector/<erp>.go`.
O `factory.go` lê o campo `erp_type` do cliente e retorna o conector correto.
Connector (interface)
```text
Extract()     → retorna registros brutos
Validate()    → valida configuração
GetSchedule() → frequência de sincronização
```
---

## Infraestrutura

| Serviço | Imagem | Porta interna | Responsabilidade |
|---|---|---|---|
| postgres | postgres:16-alpine | 5432 | Banco operacional |
| collector | lume-collector | 8080 | ETL, API, scheduler |
| engine | lume-engine | — | Worker de analytics |
| streamlit | lume-streamlit | 8501 | Interface do usuário |
| grafana | grafana/grafana | 3000 | Observabilidade |
| nginx | nginx:alpine | 80/443 | Proxy reverso, SSL |

---

## Fluxo de dados completo
1. Scheduler (Go) dispara job de coleta para o cliente
2. Connector extrai dados brutos do ERP
3. Normalizer transforma para o schema canônico
4. Loader faz upsert no PostgreSQL (schema do cliente)
5. Go enfileira job de analytics na tabela `jobs`
6. Engine (Python) consome o job
7. Algoritmos leem do PostgreSQL e calculam
8. Resultados são gravados no DuckDB do cliente
9. Streamlit lê do DuckDB e renderiza o dashboard
---

## Adicionando um novo cliente

```bash
bash infra/scripts/new_client.sh <client_key> "<nome>" <segmento> <erp_type>
```

O script:
1. Cria o schema isolado no PostgreSQL
2. Cria os índices do schema
3. Registra na tabela `lume_system.clients`
4. Gera `clients/<client_key>/.env` com template de configuração

## Removendo um cliente (LGPD)

```bash
bash infra/scripts/offboard_client.sh <client_key>
```

O script remove: schema do PostgreSQL, arquivo DuckDB e configurações do cliente.