# Architecture

System design and internal components of GoQueue.

## Overview

GoQueue is a polling-based distributed job queue. Jobs are submitted via REST API and stored in PostgreSQL. A worker pool continuously polls for available jobs, claiming them with row-level locks to prevent concurrent workers from processing the same job.

```
Client → API Server → PostgreSQL ← Worker Pool
                                        │
                                    Janitor
```

## Repository Structure

```
goqueue/
├── cmd/
│   ├── api/              # API server binary
│   └── worker/           # Worker service binary
├── internal/
│   ├── app/              # Application wiring (ApiApp, WorkerApp)
│   ├── config/           # Config loading from env + constants
│   ├── dto/              # Data Transfer Objects (request/response + payloads)
│   ├── job/              # Job domain: handler, service, payload validation, interfaces
│   ├── mocks/            # Testify mocks for repo and service
│   ├── models/           # GORM database model
│   ├── pool/             # Worker pool + janitor
│   ├── router/           # Gin router setup
│   ├── storage/
│   │   └── postgres/     # DB connection + JobRepository implementation
│   └── worker/           # Worker loop + job handlers (email, payment, webhook)
├── middleware/           # Error handler, timeout, request validation
├── common/               # APIError type
├── migrations/           # Goose SQL migrations
├── test/
│   └── integration/      # DB integration tests + benchmarks
└── deployments/
    └── docker-compose.dev.yml
```

## Layer Architecture

```
HTTP Request
     │
     ▼
┌─────────────────┐
│  JobHandler     │  Binds request, calls service, formats response
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  JobService     │  Validates queue, validates payload schema, maps errors
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  JobRepository  │  GORM database operations, atomic updates
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   PostgreSQL    │
└─────────────────┘
```

## Worker Architecture

Each binary runs independently. The worker binary starts a `WorkerPool` which spawns N goroutines plus one janitor.

```
WorkerPool
├── Worker 1  (goroutine)
├── Worker 2  (goroutine)
├── Worker N  (goroutine)
└── Janitor   (30s ticker goroutine)
```

### Worker loop

```go
for {
    job = AcquireNext(queue, workerID, lockDuration)

    if job != nil:
        execute(job)
        → success: MarkCompleted
        → failure: IncrementAttemptsAndGet
            → attempts < maxRetries: RetryLater(backoff)
            → attempts >= maxRetries: SaveResult(error) + UpdateStatus(failed)
        currentDelay = baseInterval
    else:
        currentDelay = min(currentDelay * 2, maxDelay)  // exponential backoff on idle

    select:
        case <-time.After(currentDelay)
        case <-quit / ctx.Done: return
}
```

### Job locking strategy

Workers use `FOR UPDATE SKIP LOCKED` to atomically claim jobs from PostgreSQL without coordination overhead:

```sql
SELECT * FROM jobs
WHERE queue = $1
  AND status = 'queued'
  AND available_at <= now()
  AND (locked_at IS NULL OR locked_at < now() - lock_duration)
ORDER BY available_at ASC, id ASC
LIMIT 1
FOR UPDATE SKIP LOCKED
```

`SKIP LOCKED` means concurrent workers never block on each other — they simply skip rows locked by another transaction and move to the next available job. This gives maximum parallelism with zero double-processing.

After acquiring, the row is updated atomically in the same transaction:

```sql
UPDATE jobs SET status='running', locked_at=$1, locked_by=$2 WHERE id=$3
```

### Stuck job recovery (Janitor)

The janitor runs every 30 seconds and finds jobs whose lock has expired:

```sql
SELECT * FROM jobs
WHERE status = 'running'
  AND locked_at < now() - (lock_duration * 2)
```

For each stuck job it calls `Release`, which clears the lock and resets status to `queued` so another worker can pick it up.

### Retry with exponential backoff

Failed jobs are retried with:

- **Base delay:** 10 seconds
- **Formula:** `base * 2^(attempts-1)`
- **Cap:** 1 hour
- **Jitter:** ±20% randomness to prevent thundering herd

Example: attempt 1 → ~10s, attempt 2 → ~20s, attempt 3 → ~40s, …, capped at ~1h.

The `IncrementAttemptsAndGet` call is atomic (uses a DB transaction to increment and read in one operation), preventing race conditions when multiple workers might be retrying the same job.

## Database Schema

```sql
CREATE TABLE jobs (
    id           BIGSERIAL PRIMARY KEY,
    queue        VARCHAR(255) NOT NULL,
    payload      JSONB,
    status       VARCHAR(50)  NOT NULL DEFAULT 'queued',
    attempts     INT          NOT NULL DEFAULT 0,
    max_retries  INT          NOT NULL DEFAULT 5,
    available_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    locked_at    TIMESTAMP WITH TIME ZONE,
    locked_by    BIGINT,
    result       JSONB,
    error        TEXT,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX idx_jobs_status         ON jobs(status);
CREATE INDEX idx_jobs_queue_status   ON jobs(queue, status);
CREATE INDEX idx_jobs_available_at   ON jobs(available_at);
CREATE INDEX idx_jobs_locked_at      ON jobs(locked_at);
```

**Key fields:**

| Field | Purpose |
|-------|---------|
| `available_at` | Earliest time a worker may claim this job — enables delayed/scheduled jobs |
| `locked_at` | Timestamp when a worker claimed the job; used to detect stuck jobs |
| `locked_by` | Worker ID that holds the lock |
| `attempts` | Incremented atomically on each failure |
| `max_retries` | Job-specific retry limit (0–20, default 3) |

## Middleware

| Middleware | Purpose |
|-----------|---------|
| `TimeoutMiddleware` | Adds 5s request context timeout |
| `ErrorHandler` | Catches errors from handlers, formats consistent JSON error responses |
| `Bind[T]` (generic) | Binds and validates request body using struct tags |

## Error Handling

All errors flow as `APIError` structs through `c.Error()` and are caught by `ErrorHandler` middleware:

```
Handler → c.Error(apiErr) → ErrorHandler → JSON response
```

Service layer maps repository and context errors to appropriate HTTP codes (400, 404, 408, 500).

## Configuration

All config loaded from environment via `sethvargo/go-envconfig`:

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | required | DB username |
| `POSTGRES_PASSWORD` | required | DB password |
| `POSTGRES_HOST` | required | DB host |
| `POSTGRES_PORT` | required | DB port |
| `POSTGRES_DB` | required | DB name |
| `DB_MAX_RETRIES` | 10 | Connection retry attempts |
| `DB_RETRY_DELAY` | 2s | Delay between retries |
| `DB_LOG_LEVEL` | warn | GORM log level (silent/error/warn/info) |
| `SERVER_PORT` | 8080 | API server port |
| `MAX_WORKERS` | 10 | Worker pool size |

## Performance

### Connection pooling

```go
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(50)
sqlDB.SetConnMaxLifetime(time.Hour)
```

### Context propagation

Every repository method accepts `context.Context`. The API middleware sets a 5s timeout on all requests. Workers propagate the pool's cancellation context through all DB operations for clean shutdown.

## Testing strategy

| Layer      | Approach                                                                     |
| ---------- | ---------------------------------------------------------------------------- |
| Handler    | Mock service via `JobServiceMock`                                            |
| Service    | Mock repository via `JobRepoMock`                                            |
| Repository | Integration tests against real PostgreSQL (dockertest)                       |
| Worker     | Mock repository; tests cover process, handleFailure, backoff, pullJob, Start |
| App        | Mock HTTP server + sqlmock                                                   |