.PHONY: help up up-detach down logs \
        api-logs worker-logs rebuild rebuild-clean restart \
        migrate-status migrate-up migrate-down migrate-reset \
        db-connect reset-db status \
        test test-verbose bench clean clean-force clean-images

# Container names
API_CONTAINER=goqueue_api_container
WORKER_CONTAINER=goqueue_worker_container
POSTGRES_CONTAINER=postgres_container

# Image name
IMAGE_NAME=goqueue_app:dev

# Paths
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
	@echo "  make up                  - Start all services (API + Worker + Postgres)"
	@echo "  make up-detach           - Start all services in background"
	@echo "  make down                - Stop all services"
	@echo "  make restart             - Restart services without rebuilding"
	@echo "  make rebuild             - Rebuild image and restart services"
	@echo "  make rebuild-clean       - Clean rebuild (no cache, removes old images)"
	@echo "  make logs                - View logs from all services"
	@echo "  make api-logs            - View logs from API service only"
	@echo "  make worker-logs         - View logs from worker service only"
	@echo "  make status              - Show container and image status"
	@echo "  make reset-db            - Delete Postgres volume (ALL DATA LOST)"
	@echo ""
	@echo "Database / Migrations:"
	@echo "  make migrate-status      - Check migration status"
	@echo "  make migrate-up          - Run pending migrations"
	@echo "  make migrate-down        - Rollback last migration"
	@echo "  make migrate-reset       - Rollback all migrations"
	@echo "  make db-connect          - Connect to PostgreSQL shell"
	@echo ""
	@echo "Testing / Benchmarks:"
	@echo "  make test                - Run all tests"
	@echo "  make test-verbose        - Run tests with verbose output"
	@echo "  make bench               - Run all benchmarks"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean               - Remove build artifacts"
	@echo "  make clean-force         - Force remove build artifacts (uses sudo)"
	@echo "  make clean-images        - Remove dangling Docker images"

## ----------------------
## Docker
## ----------------------

up:
	cd deployments && docker-compose -f docker-compose.dev.yml up

up-detach:
	cd deployments && docker-compose -f docker-compose.dev.yml up -d

down:
	cd deployments && docker-compose -f docker-compose.dev.yml down

restart:
	cd deployments && docker-compose -f docker-compose.dev.yml restart

rebuild:
	@echo "Rebuilding image..."
	cd deployments && docker-compose -f docker-compose.dev.yml down
	cd deployments && docker-compose -f docker-compose.dev.yml build goapp
	@echo "Cleaning dangling images..."
	@docker image prune -f
	cd deployments && docker-compose -f docker-compose.dev.yml up -d
	@echo "Rebuild complete!"

rebuild-clean:
	@echo "⚠️  Clean rebuild (no cache)..."
	cd deployments && docker-compose -f docker-compose.dev.yml down
	@echo "Removing old image..."
	@docker rmi $(IMAGE_NAME) 2>/dev/null || true
	@docker image prune -f
	@echo "Building fresh image..."
	cd deployments && docker-compose -f docker-compose.dev.yml build --no-cache goapp
	cd deployments && docker-compose -f docker-compose.dev.yml up -d
	@echo "Clean rebuild complete!"

logs:
	cd deployments && docker-compose -f docker-compose.dev.yml logs -f

api-logs:
	cd deployments && docker-compose -f docker-compose.dev.yml logs -f goapp

worker-logs:
	cd deployments && docker-compose -f docker-compose.dev.yml logs -f worker

status:
	@echo "=== Container Status ==="
	@docker ps -a --filter "name=$(API_CONTAINER)" --filter "name=$(WORKER_CONTAINER)" --filter "name=$(POSTGRES_CONTAINER)" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "=== Application Image ==="
	@docker images --filter "reference=$(IMAGE_NAME)" --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
	@echo ""
	@echo "=== Dangling Images ==="
	@docker images -f "dangling=true" -q | head -n 5 | xargs -r docker images --format "table {{.ID}}\t{{.Size}}\t{{.CreatedAt}}" 2>/dev/null || echo "No dangling images found"

reset-db:
	@echo "⚠️  WARNING: This will DELETE the PostgreSQL volume and ALL data"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd deployments && docker-compose -f docker-compose.dev.yml down -v; \
		echo "PostgreSQL volume deleted."; \
	else \
		echo "Operation cancelled."; \
	fi

## ----------------------
## Migrations
## ----------------------

migrate-status:
	docker exec $(API_CONTAINER) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

migrate-up:
	docker exec $(API_CONTAINER) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	docker exec $(API_CONTAINER) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-reset:
	docker exec $(API_CONTAINER) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" reset

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

## ----------------------
## Cleanup
## ----------------------

clean:
	@echo "Cleaning build artifacts..."
	@if [ -d "tmp/api" ] || [ -d "tmp/worker" ]; then \
		if [ -w "tmp/api/main" ] 2>/dev/null || [ -w "tmp/worker/main" ] 2>/dev/null; then \
			rm -rf tmp/api tmp/worker; \
		else \
			echo "Files created by Docker detected. Cleaning with API container..."; \
			docker exec $(API_CONTAINER) sh -c "rm -rf /app/tmp/api /app/tmp/worker" 2>/dev/null || \
			echo "⚠️  Container not running. Use 'make clean-force' or start containers first."; \
		fi \
	fi
	@rm -f tmp/build-errors.log
	@echo "Clean complete!"

clean-force:
	@echo "Force cleaning with sudo..."
	@sudo rm -rf tmp/api tmp/worker
	@rm -f tmp/build-errors.log
	@echo "Clean complete!"

clean-images:
	@echo "Removing dangling Docker images..."
	@docker image prune -f
	@echo "Dangling images removed!"
