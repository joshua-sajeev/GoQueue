# GoQueue Roadmap

**Goal:** Build a job queue system in Go. Right now it just saves jobs to a database. Next, I need to make workers actually process those jobs using Redis as a queue.

---

## What's Done

**API Server (Working)**
- POST `/jobs/create` - creates job in database
- GET `/jobs/:id` - get job status
- PUT `/jobs/:id/status` - update status
- POST `/jobs/:id/increment` - increment retry attempts
- POST `/jobs/:id/save` - save result after running
- GET `/jobs?queue=name` - list jobs

**Database (Working)**
- PostgreSQL with GORM
- Jobs table with migrations
- Repository pattern for database operations
- Tests with dockertest

**Project Structure**
```
cmd/api/          - API server
internal/
  job/            - business logic
  models/         - database models
  storage/        - postgres repo
  dto/            - request/response types
middleware/       - gin middleware
migrations/       - database migrations
```

---

## What's Next - Make It Actually Work

Right now jobs just sit in the database. Need to make workers process them.

### 1. Add Redis Queue (Week 1)

**Why:** Need something to hold job IDs that workers can pull from. Redis is fast and has blocking operations.

**What to do:**
- Add Redis to docker-compose
- Create broker package that talks to Redis
- Use LPUSH to add job IDs to queue
- Use BRPOP to pull job IDs (blocking, so worker waits)
- Update API: after saving job to DB, push ID to Redis

**Files to create:**
```
internal/broker/
  interface.go           - Enqueue(), Dequeue()
  redis/
    redis.go             - actual Redis code
    redis_test.go        - tests with miniredis
```

**Redis operations:**
```
Enqueue: LPUSH "queue:email" 123
Dequeue: BRPOP "queue:email" 0  (blocks until job available)
```

**API change:**
```
POST /jobs/create
  1. Save job to PostgreSQL
  2. Push job.ID to Redis queue
  3. Return response
```

---

### 2. Build Worker Service (Week 2)

**Why:** Need something to actually run the jobs. Worker pulls from Redis and executes handlers.

**What to do:**
- Create new binary `cmd/worker/main.go`
- Worker loop: pull job ID from Redis → fetch from DB → execute → update DB
- Write handlers for send_email, process_payment, send_webhook
- Add retry logic with exponential backoff
- Run multiple workers with goroutines

**Worker flow:**
```
1. BRPOP from Redis (blocking) → get job ID
2. Fetch full job from PostgreSQL using ID
3. Update status = "running"
4. Run handler based on job.Queue
5. If success:
   - status = "completed"
   - save result
6. If fail:
   - increment attempts
   - if attempts < max_retries:
     - status = "queued"
     - push back to Redis with delay
   - else:
     - status = "failed"
     - save error
```

**Files to create:**
```
cmd/worker/
  main.go                - worker binary

internal/worker/
  worker.go              - main worker loop
  handlers.go            - send_email, process_payment, send_webhook
  registry.go            - map queue to handlers
```

**Handler example:**
```go
func SendEmailHandler(ctx context.Context, payload json.RawMessage) error {
    var email dto.SendEmailPayload
    json.Unmarshal(payload, &email)
    
    // simulate sending email
    time.Sleep(100 * time.Millisecond)
    log.Printf("Sent email to %s", email.To)
    
    return nil
}
```

**Docker compose:**
```yaml
worker:
  build: .
  command: go run cmd/worker/main.go
  depends_on:
    - postgres
    - redis
```

---

### 3. Add Logging and Metrics (Week 3)

**Why:** Need to see what's happening. Right now just using fmt.Println everywhere.

**What to do:**
- Replace log.Println with structured logging (slog or zap)
- Add Prometheus metrics endpoint
- Track: jobs created, jobs processed, queue depth, errors
- Add Grafana for visualization

**Metrics to add:**
```
API:
- jobs_created_total{queue}
- http_request_duration_seconds

Worker:
- jobs_processed_total{queue, status}
- job_duration_seconds
- queue_depth{queue}
```

**Docker compose:**
```yaml
prometheus:
  image: prom/prometheus
  ports: ["9090:9090"]

grafana:
  image: grafana/grafana
  ports: ["3000:3000"]
```

---

### 4. Test It (Week 4)

**Why:** Make sure it actually works under load.

**What to do:**
- Write load test that creates 1000 jobs
- Run worker and watch it process them
- Check metrics in Grafana
- Make sure nothing crashes
- Document throughput (jobs/sec)

**Load test:**
```go
// test/load/load_test.go
func TestLoadJobs(t *testing.T) {
    for i := 0; i < 1000; i++ {
        // POST to /jobs/create
    }
    // wait for workers to finish
    // check all jobs completed
}
```

---

### 5. Clean Up (Week 4)

**What to do:**
- Fix any bugs from load testing
- Clean up git history (squash commits)
- Update README with setup instructions
- Add architecture diagram
- Document how to run everything

---

## After This Works

Once workers are running jobs, can add:

**Later (Phase 2):**
- Scheduler service (cron jobs)
- Web dashboard to view jobs
- Job priorities
- Dead letter queue for failed jobs
- Better retry strategies
- Job dependencies

**Much Later:**
- Distributed tracing
- Rate limiting
- Job TTL
- Scheduled jobs (run at specific time)
- Job chains (job A then job B)

---

## Questions I Still Have

- Should I use Redis Streams instead of lists?
- How to handle worker crashes mid-job?
- Should workers pull from multiple queues?
- How many workers to run?
- Should I add job priorities now or later?

---

## File Structure After Phase 1.5

```
goqueue/
├── cmd/
│   ├── api/
│   │   └── main.go          (existing)
│   └── worker/
│       └── main.go          (new)
├── internal/
│   ├── broker/              (new)
│   │   ├── interface.go
│   │   └── redis/
│   │       └── redis.go
│   ├── worker/              (new)
│   │   ├── worker.go
│   │   ├── handlers.go
│   │   └── registry.go
│   ├── job/                 (existing)
│   ├── models/              (existing)
│   └── storage/             (existing)
├── deployments/
│   ├── docker-compose.dev.yml
│   └── prometheus.yml       (new)
├── test/
│   └── load/                (new)
│       └── load_test.go
└── README.md
```

---

## Timeline

**Week 1:** Redis broker
**Week 2:** Worker service  
**Week 3:** Logging/metrics
**Week 4:** Load testing + cleanup
