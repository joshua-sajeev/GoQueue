# GoQueue Development Guide

Complete guide for setting up, developing, and contributing to GoQueue.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Initial Setup](#initial-setup)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Database Operations](#database-operations)
- [Code Organization](#code-organization)
- [Coding Standards](#coding-standards)
- [Contributing](#contributing)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required

- **Go 1.25+**: [Download](https://go.dev/dl/)
- **Docker**: [Install Docker](https://docs.docker.com/get-docker/)
- **Docker Compose**: [Install Docker Compose](https://docs.docker.com/compose/install/)
- **Make**: Usually pre-installed on Unix systems

### Optional

- **PostgreSQL Client** (psql): For direct database access
- **Redis CLI**: For Redis debugging (Phase 2)
- **Air**: For standalone hot reloading (installed in container)
- **Goose**: For standalone migrations (installed in container)

### Verify Installation

```bash
go version        # Should show 1.25+
docker --version
docker-compose --version
make --version
```

## Initial Setup

### 1. Clone Repository

```bash
git clone https://github.com/yourusername/goqueue.git
cd goqueue
```

### 2. Configure Environment

```bash
cp deployments/.env.example deployments/.env
```

Edit `deployments/.env`:

```bash
# Database Configuration
POSTGRES_USER=goqueue_user
POSTGRES_PASSWORD=secure_password_here
POSTGRES_DB=goqueue
POSTGRES_HOST=postgres
POSTGRES_PORT=5432

# Database Connection Settings
DB_MAX_RETRIES=10
DB_RETRY_DELAY=2s
DB_CONNECT_TIMEOUT=5
DB_LOG_LEVEL=warn    # Options: silent, error, warn, info
```

### 3. Start Development Environment

```bash
make compose-up
```

This command:
- Starts PostgreSQL container
- Starts GoQueue API container with hot reload
- Runs database migrations automatically
- Exposes API on `http://localhost:8080`

### 4. Verify Setup

Check health:
```bash
curl http://localhost:8080/health
# Response: {"status":"ok"}
```

Check database:
```bash
curl -X POST http://localhost:8080/health/db
# Response: {"status":"ok"}
```

## Development Workflow

### Hot Reloading

The development environment uses Air for automatic code reloading:

1. Start services: `make compose-up`
2. Edit any `.go` file
3. Save the file
4. Air automatically rebuilds and restarts the server

**Air Configuration** (`.air.toml`):
- Watches: `*.go`, `*.tpl`, `*.tmpl`, `*.html` files
- Excludes: `*_test.go`, `tmp/`, `vendor/`
- Rebuild delay: 1 second

### View Logs

```bash
# Follow all logs
make compose-logs

# View specific service
docker logs goqueue_container -f
docker logs postgres_container -f
```

### Stop Services

```bash
# Stop services
make compose-down

# Stop and remove volumes (deletes database)
make compose-reset-db
```

### Running API Standalone (without Docker)

```bash
# Set environment variables
export POSTGRES_USER=goqueue_user
export POSTGRES_PASSWORD=secure_password
export POSTGRES_DB=goqueue
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432

# Run migrations
goose -dir ./migrations postgres "postgres://user:pass@localhost:5432/goqueue?sslmode=disable" up

# Run API
go run cmd/api/main.go
```

## Testing

### Unit Tests

Run all unit tests:
```bash
go test ./... -v
```

Run specific package:
```bash
go test ./internal/job/... -v
```

Run with coverage:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests use dockertest to spin up a real PostgreSQL instance.

Run integration tests:
```bash
go test ./test/integration/... -v
```

### Test Organization

**Unit Tests**: Located next to source files
- `internal/job/job_handler_test.go`
- `internal/job/job_service_test.go`
- `internal/storage/postgres/connection_test.go`

**Integration Tests**: In `test/integration/`
- `test/integration/db_integration_test.go`
- `test/integration/job_repo_test.go`

### Writing Tests

**Unit Test Example**:
```go
func TestJobService_CreateJob(t *testing.T) {
    mockRepo := new(mocks.JobRepoMock)
    service := job.NewJobService(mockRepo)
    
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
    
    err := service.CreateJob(context.Background(), &dto.JobCreateDTO{
        Queue: "email",
        Type: "send_email",
        Payload: json.RawMessage(`{"to":"test@example.com"}`),
    })
    
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

**Integration Test Example**:
```go
func TestJobRepository_Create(t *testing.T) {
    db, ctx := setupTestDB(t)
    defer closeTestDB(db)
    
    repo := postgres.NewJobRepository(db)
    job := &models.Job{
        Queue: "email",
        Type: "send_email",
    }
    
    err := repo.Create(ctx, job)
    
    require.NoError(t, err)
    assert.NotZero(t, job.ID)
}
```


## Database Operations

### Migrations

**Check migration status**:
```bash
make migrate-status
```

**Run pending migrations**:
```bash
make migrate-up
```

**Rollback last migration**:
```bash
make migrate-down
```

**Create new migration**:
```bash
make migrate-create NAME=add_priority_to_jobs
```

This creates:
```
migrations/20251220120000_add_priority_to_jobs.sql
```

**Migration file format**:
```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs ADD COLUMN priority INT DEFAULT 0;
CREATE INDEX idx_jobs_priority ON jobs(priority);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_jobs_priority;
ALTER TABLE jobs DROP COLUMN priority;
-- +goose StatementEnd
```

### Direct Database Access

Connect to PostgreSQL:
```bash
make db-connect
```

Or manually:
```bash
docker exec -it postgres_container psql -U goqueue_user -d goqueue
```

Common queries:
```sql
-- List all jobs
SELECT id, queue, type, status, attempts FROM jobs;

-- Count jobs by status
SELECT status, COUNT(*) FROM jobs GROUP BY status;

-- View recent jobs
SELECT * FROM jobs ORDER BY created_at DESC LIMIT 10;

-- Clear all jobs (development only)
DELETE FROM jobs;
```

### Database Reset

**Warning**: This deletes all data.

```bash
make compose-reset-db
make compose-up
```

## Code Organization

### Package Structure

```
internal/
├── config/          # Configuration constants
├── dto/             # Data Transfer Objects
├── job/             # Job domain logic
│   ├── interface.go         # Interfaces
│   ├── job_handler.go       # HTTP handlers
│   ├── job_service.go       # Business logic
│   └── payload_validation.go # Validators
├── models/          # Database models
├── storage/         # Data access layer
│   └── postgres/
└── mocks/           # Test mocks
```

### Adding a New Job Type

1. **Define payload DTO** (`internal/dto/`):
```go
// internal/dto/sms.go
package dto

type SendSMSPayload struct {
    Phone   string `json:"phone" validate:"required,e164"`
    Message string `json:"message" validate:"required"`
}
```

2. **Update constants** (`internal/config/constants.go`):
```go
var AllowedJobTypes = []string{
    "send_email",
    "process_payment",
    "send_webhook",
    "send_sms", // New
}
```

3. **Add validation** (`internal/job/job_service.go`):
```go
case "send_sms":
    if err := s.validateSendSMSPayload(dto.Payload); err != nil {
        return err
    }
```

4. **Implement validator** (`internal/job/job_service.go`):
```go
func (s *JobService) validateSendSMSPayload(raw json.RawMessage) error {
    return validatePayload[dto.SendSMSPayload](raw)
}
```

5. **Write tests**:
```go
func TestJobService_CreateJob_SendSMS(t *testing.T) {
    // Test valid SMS payload
    // Test invalid phone numbers
    // Test missing fields
}
```

### Adding a New Queue

1. Update constants:
```go
var AllowedQueues = []string{
    "default",
    "email",
    "webhooks",
    "sms", // New
}
```

2. Document in API.md and ARCHITECTURE.md

## Coding Standards

### Go Style

Follow official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

### Naming Conventions

- **Interfaces**: Suffix with `Interface` (e.g., `JobRepoInterface`)
- **Implementations**: Descriptive names (e.g., `JobRepository`)
- **DTOs**: Suffix with `DTO` (e.g., `JobCreateDTO`)
- **Mocks**: Suffix with `Mock` (e.g., `JobRepoMock`)

### Error Handling

Use custom `APIError`:
```go
return common.Errf(http.StatusBadRequest, "invalid queue: %s", queue)
```

With fields:
```go
return common.NewAPIError(
    http.StatusBadRequest,
    "invalid queue",
    map[string]any{
        "provided": queue,
        "allowed": config.AllowedQueues,
    },
)
```

### Context Usage

Always pass `context.Context` as first parameter:
```go
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
    return r.db.WithContext(ctx).Create(job).Error
}
```

### Testing Standards

- Table-driven tests
- Descriptive test names
- Setup and teardown functions
- Mock all external dependencies
- Test error cases

### Documentation

- Package comments for all packages
- Exported function comments
- Complex logic comments
- Update docs/ when adding features

## Contributing

### Workflow

1. **Fork** the repository
2. **Create branch**: `git checkout -b feature/my-feature`
3. **Write code** following standards
4. **Write tests** (unit + integration)
5. **Run tests**: `go test -tags=integration ./... -v`
6. **Run linter** (if configured)
7. **Commit**: `git commit -m "Add feature X"`
8. **Push**: `git push origin feature/my-feature`
9. **Open Pull Request**

### Commit Messages

Format:
```
<type>: <subject>

<body>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Tests
- `refactor`: Code refactoring
- `chore`: Maintenance

Example:
```
feat: add SMS job type

- Add SendSMSPayload DTO
- Implement SMS validation
- Update allowed job types
- Add tests for SMS jobs
```

### Pull Request Guidelines

- Clear title and description
- Reference related issues
- Include test results
- Update relevant documentation
- Ensure CI passes

## Troubleshooting

### Common Issues

**Port already in use**:
```bash
# Find process using port 8080
lsof -i :8080
# Kill process
kill -9 <PID>
```

**Docker build fails**:
```bash
# Clean Docker cache
docker system prune -a
make compose-up
```

**Database connection fails**:
```bash
# Check PostgreSQL logs
docker logs postgres_container

# Verify container is running
docker ps | grep postgres

# Restart services
make compose-down
make compose-up
```

**Migrations fail**:
```bash
# Check migration status
make migrate-status

# Reset and re-run
make migrate-reset
make migrate-up
```

### Getting Help

1. Check [ERRORS.md](./ERRORS.md) for known issues
2. Search existing GitHub issues
3. Open a new issue with:
   - Go version
   - Docker version
   - Error messages
   - Steps to reproduce

## Useful Commands

```bash
# Development
make compose-up              # Start services
make compose-down            # Stop services
make compose-logs            # View logs

# Database
make migrate-up              # Run migrations
make migrate-down            # Rollback migration
make migrate-status          # Check status
make db-connect              # Connect to database

# Testing
go test ./...                           # Unit tests
go test -tags=integration ./...         # All tests
go test -cover ./...                    # With coverage
go test -v -run TestName ./internal/... # Specific test

# Code Quality
go fmt ./...                 # Format code
go vet ./...                # Vet code
go mod tidy                 # Clean dependencies
```