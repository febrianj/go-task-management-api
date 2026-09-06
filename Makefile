include .env
export

PSQL := docker compose exec -T db psql -U postgres -d taskapidb -v ON_ERROR_STOP=1

.PHONY: run up down migrate migrate-status db-reset

run:
	go run ./cmd/api

up:
	docker-compose up -d db

down:
	docker-compose down

migrate:
	cat migrations/*.sql | $(PSQL)
	@echo "migration applied"

migrate-status:
	$(PSQL) -c '\dt'

db-reset:
	docker-compose down -v
	docker-compose up -d db
	@sleep 6
	@(MAKE) migrate

test:
	go test ./...