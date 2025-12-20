# GoQueue

Distributed job queue system in Go with API, Worker, Scheduler and PostgreSQL/Redis backend.

## Overview

GoQueue provides a REST API for asynchronous job processing with PostgreSQL persistence, multiple queue support, and configurable retry mechanisms.

**Current Status**: Phase 1 (85% complete) - Core API and storage layer implemented. Redis broker and worker service in progress.

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- Make

### Installation

```bash
git clone https://github.com/joshu-sajeev/goqueue.git
cd goqueue
cp deployments/.env.example deployments/.env
make compose-up
```

API available at `http://localhost:8080`

### Basic Usage

```bash
# Create a job
curl -X POST http://localhost:8080/jobs/create \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "email",
    "type": "send_email",
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
- **[ROADMAP.md](./docs/ROADMAP.md)** - Project plan and timeline
- **[ERRORS.md](./docs/ERRORS.md)** - Troubleshooting guide

## Key Features

- REST API with Gin framework
- PostgreSQL storage with GORM
- Multiple queues (default, email, webhooks)
- Three job types (send_email, process_payment, send_webhook)
- Request validation and timeout handling
- Database migrations with Goose
- Hot reload development environment
- Comprehensive test coverage

## Common Commands

```bash
make compose-up          # Start services
make compose-down        # Stop services
make compose-logs        # View logs
make migrate-up          # Run migrations
make db-connect          # Database shell
make help               # Show all commands
```

## Testing

```bash
go test ./... -v                              # Unit tests
go test -coverprofile=coverage.out ./...      # With coverage
```

## Contributing

1. Fork the repository
2. Create feature branch
3. Write tests
4. Submit pull request

See [DEVELOPMENT.md](./docs/DEVELOPMENT.md) for details.

## License

MIT License - see [LICENCE](LICENCE) file.