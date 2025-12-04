# GoQueue Roadmap

**Goal:** Build a distributed job queue system in Go, with API, Worker, Scheduler, and Dashboard. Users can submit jobs, process them asynchronously, track status, schedule tasks

---

## Phase 1 — Core Job Queue Backend (2 Weeks)

**Goal:** Have a working API + Worker system.

### TASK 1 — Job Model + Storage Layer

**Implement:**

1. **Job model in Go**

```go
type Job struct {
    ID         string         `gorm:"primaryKey"`
    Queue      string
    Type       string
    Payload    datatypes.JSON
    Status     string         // queued, processing, completed, failed
    Attempts   int
    MaxRetries int
    Result     datatypes.JSON
    Error      string
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

2. **Storage interface**

```go
type JobStorage interface {
    CreateJob(job *Job) error
    GetJobByID(id string) (*Job, error)
    UpdateStatus(id string, status string) error
    IncrementAttempts(id string) error
    SaveResult(id string, result datatypes.JSON, err string) error
    ListJobs(queue string) ([]Job, error)
}
```

3. **PostgreSQL Implementation**

* Use GORM
* Docker Compose will have Postgres ready

---

### TASK 2 — Broker Layer (Redis Queue)

**Responsibilities:**

* Queue job IDs in Redis
* Workers pick jobs via BRPOP
* Acknowledge completion with HSET

**Interface:**

```go
type Broker interface {
    Enqueue(jobID string, queue string) error
    Dequeue(queue string) (string, error)
    Ack(jobID string) error
}
```

**Implementation Notes:**

* Use `LPUSH queue_name jobID` to push
* Use `BRPOP queue_name 0` to pop
* Keep job metadata in PostgreSQL

---

### TASK 3 — API Server (`cmd/api`)

**Endpoints:**

1. **POST `/jobs`**

* Validate payload
* Create job in Postgres
* Push job ID to Redis

2. **GET `/jobs/:id`**

* Fetch job by ID from Postgres
* Return status, result, attempts, queue

3. **POST `/jobs/:id/retry`**

* Reset status → `queued`
* Increment `Attempts` if needed
* Push back to Redis

**Other Notes:**

* Keep a separate `handlers` package
* Use Gin for HTTP routing

---

### TASK 4 — Worker Service (`cmd/worker`)

**Responsibilities:**

* Poll Redis for new jobs
* Fetch metadata from Postgres
* Execute handler based on `job.Type`
* Update job status
* Retry with exponential backoff if job fails

**Handler registration example:**

```go
worker.Register("email.send", EmailHandler)
worker.Register("image.process", ImageHandler)
```

**Concurrency:**

* Start multiple goroutines per worker process
* Each goroutine listens to Redis queue

---

### TASK 5 — Testing & Local Deployment

**Docker Compose setup:**

```yaml
services:
  api:
    build: ./cmd/api
    ports: ["8080:8080"]
    environment:
      - REDIS_URL=redis:6379
      - DATABASE_URL=postgres://user:pass@postgres:5432/goqueue

  worker:
    build: ./cmd/worker
    environment:
      - REDIS_URL=redis:6379
      - DATABASE_URL=postgres://user:pass@postgres:5432/goqueue

  redis:
    image: redis:7-alpine

  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: goqueue
```

**Goal:** After this, you can:

* Submit a job via API
* Worker picks it up and updates DB
* Query job status via API


---
## Phase 2 — Scheduler Service & Dashboard 
**Goal:** Add scheduled jobs and a dashboard to monitor and manage the queue system.

### TASK 6 — Scheduler Service (`cmd/scheduler`)

**Responsibilities:**

* Read a schedule configuration file (YAML or JSON)
* Trigger jobs at specified times via API
* Support cron-style scheduling

**Example schedule file (`schedules.yaml`):**

```yaml
- type: "report.generate"
  queue: "reports"
  cron: "0 0 * * *"  # Every day at 00:00
- type: "email.daily_summary"
  queue: "emails"
  cron: "0 8 * * *"  # Every day at 08:00
```

**Implementation Notes:**

* Scheduler runs as a separate service
* Uses the Go client SDK to submit jobs to the API
* Optionally, implement leader election if running multiple scheduler instances

---

### TASK 7 — Dashboard (`goqueue-dashboard`)

**Goal:** Build a simple web UI to monitor jobs and worker activity

**Features:**

* List all jobs with filters (queue, status)
* Show job details (payload, result, attempts, error)
* Retry failed jobs via UI
* Optional: live worker activity / job progress

**Tech stack:**

* React or Vue
* REST API integration with GoQueue API service

**Example folder structure:**

```
goqueue-dashboard/
├── src/
│   ├── components/
│   ├── pages/
│   └── api/
├── public/
├── Dockerfile
└── package.json
```

---

### TASK 8 — Integration & Deployment

* Update `docker-compose.yml` to include Scheduler and Dashboard

```yaml
  scheduler:
    build: ./cmd/scheduler
    environment:
      - REDIS_URL=redis:6379
      - DATABASE_URL=postgres://user:pass@postgres:5432/goqueue

  dashboard:
    build: ./goqueue-dashboard
    ports: ["3000:3000"]
    environment:
      - API_URL=http://api:8080
```

* Verify all services communicate correctly:

  * Scheduler submits jobs via API
  * Worker processes jobs
  * Dashboard displays real-time status
