#!/bin/bash
# ============================================================
# Lume Inteligência Comercial
# offboard_client.sh — remoção completa de um cliente (LGPD)
# Uso: bash infra/scripts/offboard_client.sh <client_key>
# ============================================================

set -e

CLIENT_KEY=$1

if [ -z "$CLIENT_KEY" ]; then
    echo "Uso: bash infra/scripts/offboard_client.sh <client_key>"
    exit 1
fi

if [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
fi

echo "╔══════════════════════════════════════╗"
echo "║     Lume — Offboard Cliente          ║"
echo "╚══════════════════════════════════════╝"
echo ""
echo "⚠️  ATENÇÃO: Esta operação é irreversível."
echo "Cliente: $CLIENT_KEY"
echo ""
read -p "Digite o client_key para confirmar: " CONFIRM

if [ "$CONFIRM" != "$CLIENT_KEY" ]; then
    echo "Confirmação incorreta. Operação cancelada."
    exit 1
fi

echo ""
echo "[1/3] Removendo schema do PostgreSQL..."
docker compose exec -T postgres psql \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    -c "DROP SCHEMA IF EXISTS client_${CLIENT_KEY} CASCADE;"

docker compose exec -T postgres psql \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    -c "DELETE FROM lume_system.clients WHERE client_key = '$CLIENT_KEY';"

echo "      Schema removido."

echo "[2/3] Removendo arquivo DuckDB..."
rm -f "data/duckdb/${CLIENT_KEY}.duckdb"
rm -f "data/duckdb/${CLIENT_KEY}.duckdb.wal"
echo "      DuckDB removido."

echo "[3/3] Removendo configurações do cliente..."
rm -rf "clients/$CLIENT_KEY"
echo "      Configurações removidas."

echo ""
echo "✅ Cliente $CLIENT_KEY removido com sucesso."
echo "   Todos os dados foram deletados conforme LGPD."