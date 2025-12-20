.PHONY: help compose-up compose-down compose-logs \
        migrate-status migrate-up migrate-down migrate-reset \
        migrate-create db-connect

CONTAINER_NAME=goqueue_container
MIGRATIONS_DIR=./migrations
ENV_FILE=deployments/.env

# Load env variables
include $(ENV_FILE)
export $(shell sed 's/=.*//' $(ENV_FILE))

# Construct DATABASE_URL from env variables
DATABASE_URL=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

help:
	@echo "Available commands:"
	@echo "  make compose-up          - Start Docker containers"
	@echo "  make compose-down        - Stop Docker containers"
	@echo "  make compose-logs        - View container logs"
	@echo "  make migrate-status      - Check migration status"
	@echo "  make migrate-up          - Run pending migrations"
	@echo "  make migrate-down        - Rollback last migration"
	@echo "  make migrate-reset       - Rollback all migrations"
	@echo "  make migrate-create NAME=table_name - Create new migration"
	@echo "  make db-connect          - Connect to PostgreSQL shell"
	@echo "  make compose-reset-db    - Delete Postgres volume (ALL DATA LOST)"

## Docker Compose Commands
compose-up:
	@echo "Starting Docker containers..."
	cd deployments && docker-compose -f docker-compose.dev.yml up
	@echo "Containers started. Use 'make compose-logs' to view logs"

compose-up-detach:
	@echo "Starting Docker containers..."
	cd deployments && docker-compose -f docker-compose.dev.yml up -d
	@echo "Containers started. Use 'make compose-logs' to view logs"

compose-down:
	@echo "Stopping Docker containers..."
	cd deployments && docker-compose -f docker-compose.dev.yml down
	@echo "Containers stopped"

compose-logs:
	cd deployments && docker-compose -f docker-compose.dev.yml logs -f

## Migration Commands
migrate-status:
	@echo "Migration status:"
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

migrate-up:
	@echo "Running pending migrations..."
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up
	@echo "Migrations completed"

migrate-down:
	@echo "Rolling back last migration..."
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down
	@echo "Rollback completed"

migrate-reset:
	@echo "Resetting database (rolling back all migrations)..."
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" reset
	@echo "Database reset completed"

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME not specified. Usage: make migrate-create NAME=table_name"; \
		exit 1; \
	fi
	@echo "Creating migration: $(NAME)"
	docker exec $(CONTAINER_NAME) goose -dir $(MIGRATIONS_DIR) create $(NAME) sql
	@echo "Migration created in $(MIGRATIONS_DIR)"

## Database Commands
db-connect:
	docker exec -it postgres_container psql -U postgres -d goqueue

## Delete postgres volume
compose-reset-db:
	@echo "⚠️  WARNING: This will DELETE the PostgreSQL volume and ALL data"
	@echo "Stopping containers..."
	cd deployments && docker-compose -f docker-compose.dev.yml down -v
	@echo "PostgreSQL volume deleted. Run 'make compose-up' to recreate."
