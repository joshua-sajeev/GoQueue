.PHONY: help dev-up dev-down dev-logs api worker api-logs worker-logs clean test

help:
	@echo "Available commands:"
	@echo "  make dev-up        - Start all services (API + Worker + Postgres)"
	@echo "  make dev-down      - Stop all services"
	@echo "  make dev-logs      - Follow logs for all services"
	@echo "  make api-logs      - Follow logs for API service only"
	@echo "  make worker-logs   - Follow logs for worker service only"
	@echo "  make clean         - Remove tmp directories and build artifacts"
	@echo "  make test          - Run tests"
	@echo ""
	@echo "Local development (without Docker):"
	@echo "  make api           - Run API with hot-reload locally"
	@echo "  make worker        - Run worker with hot-reload locally"

# Docker commands
dev-up:
	cd deployments && docker-compose -f docker-compose.dev.yml up

dev-down:
	cd deployments && docker-compose -f docker-compose.dev.yml down

dev-logs:
	cd deployments && docker-compose -f docker-compose.dev.yml logs -f

api-logs:
	cd deployments && docker-compose -f docker-compose.dev.yml logs -f goapp

worker-logs:
	cd deployments && docker-compose -f docker-compose.dev.yml logs -f worker

# Local development (without Docker)
api:
	air -c .air-api.toml

worker:
	air -c .air-worker.toml

# Cleanup
clean:
	rm -rf tmp/api tmp/worker
	rm -f tmp/build-errors.log

# Testing
test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
