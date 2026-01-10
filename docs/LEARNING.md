# Learnings from GoQueue project

# 2025-12-06
## Mocking functions
```go
var envProcess = envconfig.Process
```
`envconfig.Process` is actually a function. But to simulate errors we can assign it to a var and then play with it like 
```go
envProcess = func(ctx context.Context, i any, mus ...envconfig.Mutator) error {
	return fmt.Errorf("mock envconfig error")
}
```

## context.Context
“Pass context.Context as the first parameter to any function that might block or take a long time.”
- Without ctx, your DB operations keep running even after request is gone → memory leaks + connection exhaustion.
- If your repo methods don’t accept ctx, your service cannot control cancellation or deadlines.
---
---

# 2025-12-15
## Postgres Connection
Postgres won't ask for a password when connecting locally inside the container, and Postgres because configured to trust local connections.

---
---

# 2025-12-18
## Marking Integration Tests

To exclude integration tests during normal runs:

```go
//go:build integration
// +build integration
```

* **Place at the top of the test file**, before `package`.
* All tests in that file are treated as **integration tests**.
* Run unit tests without them: `go test -tags=unit ./...`
* Run integration tests explicitly: `go test -tags=integration ./...`
---
---

# 2025-12-22
## Worker related Job Repo Functions and Fields
### new columns in job
```sql
status VARCHAR(50) NOT NULL DEFAULT 'queued',

available_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
locked_at TIMESTAMP WITH TIME ZONE,
locked_by VARCHAR(255),
```
The status field must be always in `queued` state when creating a job. It is fixed. It cannot be anything else.

#### `available_at` this field marks the earliest time a job is allowed to be executed.
```go
if job.AvailableAt.IsZero() {
	job.AvailableAt = time.Now()
}
```
In repo layer, i add like this so that someone can schedule a job for later, otherwise it is set to now(creation of job at repo level).
It will be used by something like
```sql
WHERE available_at <= now()
```

#### `locked_at` is used to understand when did the worker claim this job
This is used like below
```sql
locked_at IS NULL
OR locked_at < now() - lock_duration
```
So a worker will only process if no worker has taken it for processing

#### `locked_by` will give us an idea on the worker
Used to identify which worker owns this job

### New Repo Functions
```go
AcquireNext(ctx context.Context, queue string, workerID uint, lockDuration time.Duration) (*models.Job, error)
Release(ctx context.Context, id uint) error
RetryLater(ctx context.Context, id uint, availableAt time.Time) error
ListStuckJobs(ctx context.Context, staleDuration time.Duration) ([]models.Job, error)
```

I’ll use:
* **Queue:** `email`
* **Worker IDs:** `1`, `2`
* **Lock duration:** `30s`

---

#### 1. `AcquireNext` — claiming a job safely

##### What this function does

> Picks **one safe job** from the database and **locks it** so only one worker can process it.

---

##### **Example scenario**
 *Initial jobs table*

| id  | queue | status | available_at | locked_at | locked_by |
| --- | ----- | ------ | ------------ | --------- | --------- |
| 1   | email | queued | 10:00:00     | NULL      | NULL      |
| 2   | email | queued | 10:01:00     | NULL      | NULL      |
###### Worker 1 calls:

```go
job, _ := AcquireNext(ctx, "email", 1, 30*time.Second)
```

###### What happens internally

1. Finds jobs:
   * `queue = email`
   * `status = queued`
   * `available_at <= now`
2. Orders by:
   * earliest `available_at`
   * then smallest `id`
1. Locks row using `FOR UPDATE SKIP LOCKED`
2. Updates job 1:

| id  | status     | locked_at | locked_by |
| --- | ---------- | --------- | --------- |
| 1   | running | 10:00:30  | worker 1  |
###### Return value

```go
Job{ID: 1, Queue: "email"}
```

Worker `1` now owns job 1.

---
##### Concurrent worker example

```go
AcquireNext(ctx, "email", 2, 30*time.Second)
```

* Job 1 is locked → skipped
* Worker `2` gets job 2 instead

This prevents **double processing**.

---

#### 2. `Release` — emergency unlock

##### What this function does

> Unlocks a job and puts it back in the queue.

---

##### Example scenario
Worker 1 crashes while running job 1.
###### Job state (stuck)

| id  | status     | locked_at | locked_by |
| --- | ---------- | --------- | --------- |
| 1   | running | 10:00:30  | 1         |

Admin or reaper calls:

```go
Release(ctx, 1)
```

###### After release

| id  | status | locked_at | locked_by |
| --- | ------ | --------- | --------- |
| 1   | queued | NULL      | NULL      |

Job 1 is now available for any worker again.

---

#### 3. `RetryLater` — retry with delay

#### What this function does

> Reschedules a failed job for retry **after a delay**.

---

##### Example scenario

Worker `2` processes job 2 but email server is down.

```go
RetryLater(ctx, 2, time.Now().Add(2*time.Minute))
```

###### Before retry

| id  | status     | available_at |
| --- | ---------- | ------------ |
| 2   | running | 10:01:00     |

###### After retry

| id  | status | available_at |
| --- | ------ | ------------ |
| 2   | queued | 10:03:00     |

For the next 2 minutes:

* Workers will **not pick this job**

This prevents **retry storms**.

---

#### 4. `ListStuckJobs` — detecting crashed workers

##### What this function does

> Finds jobs that were locked too long and are likely abandoned.

---

##### Example scenario

Worker `3` crashed and never released job 3.

###### Job state

| id  | status     | locked_at |
| --- | ---------- | --------- |
| 3   | running | 09:45:00  |

Reaper runs every minute:

```go
jobs, _ := ListStuckJobs(ctx, 10*time.Minute)
```

###### Cutoff time

```
now = 10:00
cutoff = 09:50
```

Job 3 qualifies:

* `locked_at < cutoff`

Returned result:

```go
[]Job{ {ID: 3} }
```

Reaper then calls:

```go
Release(ctx, 3)
```

Job is rescued automatically.

---

#### How all functions work together (big picture)

##### Normal success flow

```
AcquireNext → run → completed
```

##### Failure with retry

```
AcquireNext → run → RetryLater → AcquireNext
```

##### Worker crash recovery

```
AcquireNext → crash
ListStuckJobs → Release → AcquireNext
```

---

#### Advantages of this

* No double processing
* Handles worker crashes
* Supports retries & backoff
* Safe under concurrency
* Uses DB as source of truth

---
---

## Clauses
### 1. `FOR UPDATE SKIP LOCKED` (row-level locking)

```go
Clauses(clause.Locking{
    Strength: "UPDATE",
    Options:  "SKIP LOCKED",
})
```

In PostgreSQL, this becomes:

```sql
FOR UPDATE SKIP LOCKED
```
The above line will 
* Locks the selected **row**
* Prevents other transactions from:
  * updating it
  * locking it again

This lock lasts **until the transaction commits or rolls back**.

### What `SKIP LOCKED` does

* If another transaction already locked a row:
  * **skip it**
  * don’t wait
  * move to the next available row

### Without `FOR UPDATE`

Two workers could do this:

| Time | Worker 1     | Worker 2     |
| ---- | ------------ | ------------ |
| t1   | SELECT job 1 | SELECT job 1 |
| t2   | UPDATE job 1 | UPDATE job 1 |

**Same job processed twice**
### With `FOR UPDATE` but **without** `SKIP LOCKED`

| Time | Worker 1   | Worker 2      |
| ---- | ---------- | ------------- |
| t1   | Lock job 1 | waits         |
| t2   | running | still waiting |

Worker 2 blocks → **throughput collapses**

### With `FOR UPDATE SKIP LOCKED`

| Time | Worker 1   | Worker 2    |
| ---- | ---------- | ----------- |
| t1   | Lock job 1 | skips job 1 |
| t2   | processes  | locks job 2 |

 **Maximum parallelism, zero duplicates**

---

## Why this condition exists

```go
Where("(locked_at IS NULL OR locked_at < ?)", now.Add(-lockDuration))
```

This means

> “Give me jobs that are either:
>
> * not locked at all
>   **OR**
> * locked so long ago that the lock has expired”

---

### Why the `- lockDuration` part matters


```go
now.Add(-lockDuration)
```

This means:
> current time **minus** lock duration

#### Example

* `now = 10:00`
* `lockDuration = 30s`
* cutoff = `09:59:30`

So this condition becomes:

```sql
locked_at < '09:59:30'
```

#### Scenario: worker crash

1. Worker 1 locks job at `09:59:00`
2. Worker 1 crashes
3. Job remains `running` forever

**Without this condition:**

* Job is **never reclaimed**
* Queue slowly dies

 **With this condition**

* Lock expires after 30s
* Another worker can safely reclaim the job

This is called a **visibility timeout**.
# 2025-12-29
```go
LockedBy    *uint
```
We use a pointer because `0` can be a worker may be in future, by using a pointer I can make sure nil value or NULL for LockedBy means no worker is locking the job.

# 2026-01-03
**Use constants when:**
- Values have semantic meaning in your code
	- Example: job statuses, error codes, API endpoints.
	- They are not just arbitrary text; they represent a concept
-  Values are reused across multiple places
- Type safety matters
