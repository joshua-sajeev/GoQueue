.PHONY: help up up-detach down logs \
        migrate-status migrate-up migrate-down migrate-reset \
        migrate-create db-connect reset-db \
        test test-verbose bench bench-integration

CONTAINER_NAME=goqueue_container
POSTGRES_CONTAINER=postgres_container
MIGRATIONS_DIR=./migrations
ENV_FILE=deployments/.env

# Load env variables
include $(ENV_FILE)
export $(shell sed 's/=.*//' $(ENV_FILE))

# Construct DATABASE_URL from env variables
DATABASE_URL=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

help:
	@echo "Available commands:"
	@echo ""
	@echo "Docker:"
	@echo "  make up                  - Start Docker containers"
	@echo "  make up-detach           - Start Docker containers in background"
	@echo "  make down                - Stop Docker containers"
	@echo "  make logs                - View container logs"
	@echo "  make reset-db            - Delete Postgres volume (ALL DATA LOST)"
	@echo ""
	@echo "Database / Migrations:"
	@echo "  make migrate-status      - Check migration status"
	@echo "  make migrate-up          - Run pending migrations"
	@echo "  make migrate-down        - Rollback last migration"
	@echo "  make migrate-reset       - Rollback all migrations"
	@echo "  make migrate-create NAME=x - Create new migration"
	@echo "  make db-connect          - Connect to PostgreSQL shell"
	@echo ""
	@echo "Testing / Benchmarks:"
	@echo "  make test                - Run all tests"
	@echo "  make test-verbose        - Run tests with verbose output"
	@echo "  make bench               - Run all benchmarks"

## ----------------------
## Docker
## ----------------------

up:
	cd deployments && docker-compose -f docker-compose.dev.yml up

up-detach:
	cd deployments && docker-compose -f docker-compose.dev.yml up -d

down:
	cd deployments && docker-compose -f docker-compose.dev.yml down

logs:
	cd deployments && docker-compose -f docker-compose.dev.yml logs -f

reset-db:
	@echo "⚠️  WARNING: This will DELETE the PostgreSQL volume and ALL data"
	cd deployments && docker-compose -f docker-compose.dev.yml down -v
	@echo "PostgreSQL volume deleted."

## ----------------------
## Migrations
## ----------------------

migrate-status:
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

migrate-up:
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-reset:
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" reset

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME not specified. Usage: make migrate-create NAME=table_name"; \
		exit 1; \
	fi
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) create $(NAME) sql

## ----------------------
## Database
## ----------------------

db-connect:
	docker exec -it $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

## ----------------------
## Testing & Benchmarks
## ----------------------

test:
	go test ./...

test-verbose:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./...
