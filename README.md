# Lume Inteligência Comercial

Plataforma de analytics e inteligência comercial para pequenos e médios varejistas.
Transforma dados operacionais em decisões concretas — unindo algoritmos de ciência
de dados ao conhecimento profundo do setor.

## Sobre o projeto

A Lume conecta nos dados gerados pelo ERP da loja e entrega inteligência real sobre
vendas, estoque, clientes e produtos. Não são apenas gráficos — são ações concretas
que o gestor consegue executar no dia seguinte.

**Segmento inicial:** materiais de construção e ferragens.
**Arquitetura:** Go (ingestão e API) + Python (analytics e IA) + Streamlit (interface).

## Stack

| Camada | Tecnologia |
|---|---|
| Servidor | Oracle Cloud Free — VM ARM 4 OCPU / 24GB RAM |
| Orquestração | Docker Compose + Nginx |
| Banco operacional | PostgreSQL 16 |
| Banco analítico | DuckDB |
| ETL e API | Go 1.22 |
| Analytics e IA | Python 3.12 |
| Interface | Streamlit (MVP) → React (fase 2) |
| Observabilidade | Grafana |
| Alertas | E-mail via Brevo SMTP |

## Estrutura do projeto
ume/
├── collector/          Go — ETL, conectores, API, scheduler
├── engine/             Python — algoritmos, ML, LLM, worker
├── app/
│   ├── streamlit/      Interface MVP
│   └── frontend/       React — fase 2
├── infra/              Nginx, Grafana, PostgreSQL, scripts
├── docs/               Arquitetura, playbooks, ERPs, segmentos
└── clients/            Configurações por cliente (não versionado)

## Setup local

### Pré-requisitos

- Docker e Docker Compose
- Go 1.22+
- Python 3.12+
- Make

### Instalação

```bash
# Clone o repositório
git clone https://github.com/SEU_USUARIO/lume.git
cd lume

# Configure as variáveis de ambiente
cp .env.example .env
# Edite o .env com suas configurações

# Suba os serviços
make up

# Rode as migrations
make migrate
```

### Comandos disponíveis

```bash
make up                      # Sobe todos os serviços
make down                    # Derruba todos os serviços
make logs                    # Logs de todos os serviços
make logs-s s=collector      # Logs de um serviço específico
make build                   # Build das imagens Docker
make restart s=collector     # Reinicia um serviço
make ps                      # Status dos containers
make migrate                 # Roda as migrations do banco
make new-client id=loja_xyz  # Cria um novo cliente
make offboard id=loja_xyz    # Remove todos os dados de um cliente
```

## Desenvolvimento

### Fluxo de branches
feature/xxx → develop → main
Nunca commitar diretamente na `main`. Cada feature ou fix em branch própria.

### Padrão de commits
feat: nova funcionalidade
fix: correção de bug
chore: configuração, infra, dependências
docs: documentação
refactor: refatoração sem mudança de comportamento
test: testes

### Adicionando um novo segmento

1. Crie a pasta `engine/segments/<segmento>/`
2. Implemente os 4 arquivos: `engine.py`, `config.py`, `kpis.py`, `terminology.py`
3. Documente em `docs/segments/<segmento>.md`
4. Nenhum outro arquivo precisa ser alterado

### Adicionando um novo conector de ERP

1. Crie `collector/internal/connector/<erp>.go` implementando a interface `Connector`
2. Registre no `factory.go`
3. Documente em `docs/erps/<erp>.md`
4. Nenhum outro arquivo precisa ser alterado

## Licença

Proprietário — todos os direitos reservados.