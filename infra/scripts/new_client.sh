#!/bin/bash
# ============================================================
# Lume Inteligência Comercial
# new_client.sh — onboarding de novo cliente
# Uso: bash infra/scripts/new_client.sh <client_id> <nome> <segmento> <erp_type>
# Exemplo: bash infra/scripts/new_client.sh loja_joao "Loja do João" construcao csv
# ============================================================

set -e

# ── Argumentos ───────────────────────────────────────────────
CLIENT_KEY=$1
CLIENT_NAME=$2
SEGMENT=$3
ERP_TYPE=$4

# ── Validações ───────────────────────────────────────────────
if [ -z "$CLIENT_KEY" ] || [ -z "$CLIENT_NAME" ] || [ -z "$SEGMENT" ] || [ -z "$ERP_TYPE" ]; then
    echo "Uso: bash infra/scripts/new_client.sh <client_key> <nome> <segmento> <erp_type>"
    echo "Exemplo: bash infra/scripts/new_client.sh loja_joao 'Loja do João' construcao csv"
    exit 1
fi

# ── Carrega variáveis de ambiente ────────────────────────────
if [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
fi

echo "╔══════════════════════════════════════╗"
echo "║     Lume — Novo Cliente              ║"
echo "╚══════════════════════════════════════╝"
echo ""
echo "Cliente:  $CLIENT_NAME"
echo "Key:      $CLIENT_KEY"
echo "Segmento: $SEGMENT"
echo "ERP:      $ERP_TYPE"
echo ""

# ── Cria o schema no banco ───────────────────────────────────
echo "[1/4] Criando schema no PostgreSQL..."

docker compose exec -T postgres psql \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    -c "SELECT lume_system.create_client_schema('$CLIENT_KEY');"

echo "      Schema client_$CLIENT_KEY criado."

# ── Cria índices no schema do cliente ───────────────────────
echo "[2/4] Criando índices..."

docker compose exec -T postgres psql \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" << EOF
CREATE INDEX IF NOT EXISTS idx_${CLIENT_KEY}_vendas_data
    ON client_${CLIENT_KEY}.vendas(data_venda DESC);

CREATE INDEX IF NOT EXISTS idx_${CLIENT_KEY}_vendas_cliente
    ON client_${CLIENT_KEY}.vendas(cliente_id);

CREATE INDEX IF NOT EXISTS idx_${CLIENT_KEY}_itens_venda_id
    ON client_${CLIENT_KEY}.itens_venda(venda_id);

CREATE INDEX IF NOT EXISTS idx_${CLIENT_KEY}_itens_produto
    ON client_${CLIENT_KEY}.itens_venda(produto_key);

CREATE INDEX IF NOT EXISTS idx_${CLIENT_KEY}_estoque_produto
    ON client_${CLIENT_KEY}.estoque(produto_key);
EOF

echo "      Índices criados."

# ── Registra o cliente na tabela de sistema ──────────────────
echo "[3/4] Registrando cliente no sistema..."

docker compose exec -T postgres psql \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    -c "INSERT INTO lume_system.clients (client_key, name, segment, erp_type)
        VALUES ('$CLIENT_KEY', '$CLIENT_NAME', '$SEGMENT', '$ERP_TYPE')
        ON CONFLICT (client_key) DO NOTHING;"

echo "      Cliente registrado."

# ── Gera o arquivo .env do cliente ──────────────────────────
echo "[4/4] Gerando arquivo de configuração..."

mkdir -p "clients/$CLIENT_KEY"

cat > "clients/$CLIENT_KEY/.env" << EOF
# ============================================================
# Lume — Configuração do cliente: $CLIENT_NAME
# Gerado em: $(date)
# ============================================================

CLIENT_KEY=$CLIENT_KEY
CLIENT_NAME=$CLIENT_NAME
SEGMENT=$SEGMENT
ERP_TYPE=$ERP_TYPE

# Configuração do ERP (preencher conforme o tipo)
ERP_FILE_PATH=
ERP_API_KEY=
ERP_API_URL=
ERP_DB_HOST=
ERP_DB_PORT=
ERP_DB_NAME=
ERP_DB_USER=
ERP_DB_PASSWORD=

# Schedule de sincronização (formato cron)
SYNC_SCHEDULE=*/30 * * * *
EOF

echo "      Arquivo clients/$CLIENT_KEY/.env gerado."
echo ""
echo "✅ Cliente $CLIENT_NAME criado com sucesso!"
echo ""
echo "Próximos passos:"
echo "  1. Preencha as credenciais do ERP em clients/$CLIENT_KEY/.env"
echo "  2. Teste a conexão com o ERP"
echo "  3. Rode a primeira sincronização"