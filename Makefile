.PHONY: up down logs migrate new-client ps build restart

# Sobe todos os serviços
up:
	docker compose up -d

# Derruba todos os serviços
down:
	docker compose down

# Logs em tempo real (todos os serviços)
logs:
	docker compose logs -f

# Logs de um serviço específico — uso: make logs-s s=collector
logs-s:
	docker compose logs -f $(s)

# Build das imagens
build:
	docker compose build

# Reinicia um serviço específico — uso: make restart s=collector
restart:
	docker compose restart $(s)

# Status dos containers
ps:
	docker compose ps

# Roda as migrations do banco
migrate:
	docker compose run --rm collector /app/migrate

# Cria um novo cliente — uso: make new-client id=loja_joao
new-client:
	@if [ -z "$(id)" ]; then echo "Erro: informe o id do cliente. Uso: make new-client id=nome_loja"; exit 1; fi
	bash infra/scripts/new_client.sh $(id)

# Remove todos os dados de um cliente — uso: make offboard id=loja_joao
offboard:
	@if [ -z "$(id)" ]; then echo "Erro: informe o id do cliente. Uso: make offboard id=nome_loja"; exit 1; fi
	bash infra/scripts/offboard_client.sh $(id)