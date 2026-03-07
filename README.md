# GoQueue

Distributed job queue system in Go with API, Worker, Scheduler and PostgreSQL/Redis backend.

## Overview

GoQueue is a background job processing system built around a polling-based worker model. Jobs are submitted via REST API, persisted in PostgreSQL, and processed by a configurable worker pool. Workers claim jobs using row-level locking (FOR UPDATE SKIP LOCKED) to prevent double-processing across concurrent goroutines.

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- Make

### Installation

```bash
git clone https://github.com/joshua-sajeev/goqueue.git
cd goqueue
cp deployments/.env.example deployments/.env
# Edit deployments/.env with your credentials
make up
```

API available at `http://localhost:8080`

### Basic Usage

```bash
# Create a job
curl -X POST http://localhost:8080/jobs/create \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "email",
    "payload": {
      "to": "user@example.com",
      "subject": "Hello",
      "body": "Welcome!"
    }
  }'

# Get job status
curl http://localhost:8080/jobs/1

# List jobs
curl http://localhost:8080/jobs?queue=email
```

## Documentation

- **[API.md](./docs/API.md)** - API endpoints and examples
- **[ARCHITECTURE.md](./docs/ARCHITECTURE.md)** - System design and components
- **[DEVELOPMENT.md](./docs/DEVELOPMENT.md)** - Setup and contribution guide
- **[ERRORS.md](./docs/ERRORS.md)** - Troubleshooting guide

## Key Features

- REST API with Gin — create, fetch, update, and list jobs
- Worker pool — configurable number of concurrent workers (MAX_WORKERS env)
- Atomic job locking — FOR UPDATE SKIP LOCKED prevents double-processing
- Exponential backoff with jitter — failed jobs retry with 10s base, 1h cap, ±20% jitter
- Janitor goroutine — auto-recovers stuck jobs (locked longer than 2× lock duration)
- Three job queues — email, payment, webhook with per-queue payload validation
- DB migrations — Goose-managed schema versioning
- Comprehensive tests — unit tests with mocks, integration tests against real PostgreSQL via dockertest

## Common Commands

```bash
make up              # Start API + Worker + Postgres
make down            # Stop all services
make logs            # Tail all logs
make api-logs        # Tail API logs only
make worker-logs     # Tail worker logs only
make migrate-up      # Run pending migrations
make migrate-status  # Check migration state
make db-connect      # PostgreSQL shell
make test            # Run all unit tests
make bench           # Run benchmarks
make help            # Full command reference
```

## Testing

```bash
go test ./...                          # All unit tests
go test -v ./...                       # Verbose
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Integration tests (requires Docker)
go test -run=NONE -bench=. -benchmem ./test/integration   # Benchmarks
go test ./test/integration/... -v                          # Integration tests
```

## Contributing

1. Fork the repository
2. Create feature branch
3. Write tests
4. Submit pull request

See [DEVELOPMENT.md](./docs/DEVELOPMENT.md) for details.

## License

MIT License - see [LICENCE](LICENCE) file.
