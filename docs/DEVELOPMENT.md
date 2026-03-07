# GoQueue Development Guide

# Development Guide

## Prerequisites

- Go 1.25+
- Docker and Docker Compose
- Make

```bash
go version        # Should show 1.25+
docker --version
make --version
```

## Initial Setup

```bash
git clone https://github.com/joshua-sajeev/goqueue.git
cd goqueue
cp deployments/.env.example deployments/.env
```

Edit `deployments/.env`:

```env
POSTGRES_USER=goqueue_user
POSTGRES_PASSWORD=your_password_here
POSTGRES_DB=goqueue
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
DB_MAX_RETRIES=10
DB_RETRY_DELAY=2s
DB_LOG_LEVEL=warn
MAX_WORKERS=10
```

Start everything:

```bash
make up
```

This starts:
- PostgreSQL container
- API server with hot reload (Air)
- Worker service with hot reload (Air)
- Goose migrations run automatically on API startup

Verify:

```bash
curl http://localhost:8080/health
# {"status":"healthy"}
```

## Development Workflow

### Hot Reload

Both the API and worker use Air for automatic rebuild on file save. They share one Docker image but use different Air configs (`.air-api.toml` and `.air-worker.toml`), controlled by the `SERVICE_TYPE` env var in docker-compose.

### Logs

```bash
make logs           # All services
make api-logs       # API only
make worker-logs    # Worker only
```

### Rebuilding

```bash
make rebuild        # Rebuild image, restart services
make rebuild-clean  # No-cache rebuild (slower, use when dependencies change)
```

### Database

```bash
make migrate-status   # Check pending migrations
make migrate-up       # Apply pending migrations
make migrate-down     # Rollback one migration
make migrate-reset    # Rollback all migrations
make db-connect       # psql shell inside container
make reset-db         # WARNING!  Delete postgres volume (all data lost)
```

## Running Without Docker

```bash
# Requires a local PostgreSQL instance

export POSTGRES_USER=goqueue_user
export POSTGRES_PASSWORD=your_password
export POSTGRES_DB=goqueue
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432

# Run migrations
goose -dir ./migrations postgres \
  "postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB?sslmode=disable" up

# Start API
go run cmd/api/main.go

# Start worker (separate terminal)
go run cmd/worker/main.go
```

## Testing

```bash
# Unit tests
make test
go test ./... -v

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific package
go test ./internal/job/... -v
go test ./internal/worker/... -v

# Integration tests (requires Docker — spins up its own Postgres via dockertest)
go test ./test/integration/... -v

# Benchmarks
make bench
go test -run=NONE -bench=. -benchmem ./test/integration
go test -run=NONE -bench=BenchmarkJobRepository_Get -benchmem ./test/integration
```

### Test organization

| Location | Type | Uses |
|----------|------|------|
| `internal/job/*_test.go` | Unit | `JobServiceMock`, `JobRepoMock` |
| `internal/worker/*_test.go` | Unit | `JobRepoMock` |
| `internal/app/*_test.go` | Unit | `sqlmock`, mock server |
| `internal/config/*_test.go` | Unit | injected `envProcess` |
| `test/integration/` | Integration | Real PostgreSQL via dockertest |

## Code Organization

### Adding a new queue

1. Add to `AllowedQueues` in `internal/config/constants.go`
2. Add a payload DTO in `internal/dto/`
3. Add payload validation in `internal/job/job_service.go` (switch case)
4. Add a handler in `internal/worker/handler.go`
5. Add the case to `worker.execute()` in `internal/worker/worker.go`
6. Update `docs/API.md` with the new payload schema

### Adding a new migration

```bash
# Create migration file manually with timestamp prefix:
# migrations/20260201120000_your_migration_name.sql

# Then apply:
make migrate-up
```

Migration format (Goose):

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs ADD COLUMN priority INT DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jobs DROP COLUMN priority;
-- +goose StatementEnd
```

## Coding Standards

### Naming

| Thing | Convention |
|-------|-----------|
| Interfaces | `JobRepoInterface`, `JobServiceInterface` |
| Implementations | `JobRepository`, `JobService` |
| DTOs | `JobCreateDTO`, `JobResponseDTO` |
| Mocks | `JobRepoMock`, `JobServiceMock` |

### Errors

Use `common.Errf` or `common.NewAPIError` — these implement `error` and carry an HTTP status code that the error handler middleware reads:

```go
// Simple
return common.Errf(http.StatusBadRequest, "invalid queue: %s", queue)

// With structured fields
return common.NewAPIError(http.StatusBadRequest, "invalid queue", map[string]any{
    "provided": queue,
    "allowed":  config.AllowedQueues,
})
```

### Context

Always propagate `context.Context` as the first argument. This allows the DB layer to respect request timeouts and cancellations:

```go
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
    return r.db.WithContext(ctx).Create(job).Error
}
```

### Tests

- Table-driven tests with descriptive case names
- Mock all external dependencies (DB, HTTP server)
- Cover both success and error paths
- Use `mock.MatchedBy` for flexible argument matching

## Troubleshooting

**Port 8080 already in use:**
```bash
lsof -i :8080 | grep LISTEN
kill -9 <PID>
```

**Database won't connect:**
```bash
docker logs postgres_container
make down && make up
```

**`tmp/` files owned by root (created by Docker):**
```bash
make clean-force   # Uses sudo
```

**Migrations fail:**
```bash
make migrate-status
make migrate-reset && make migrate-up
```