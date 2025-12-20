# Benchmarks

This document contains performance benchmarks for the **GoQueue** project.
Benchmarks focus on the **repository layer**, measuring database interaction
costs and memory behavior. These results establish a baseline before introducing
Redis-backed optimizations.

Benchmarks are implemented using Go’s built-in `testing` framework and are run
against a real PostgreSQL instance.

---

## Scope

- Layer: Repository (Postgres-backed `JobRepository`)
- Purpose:
  - Establish baseline latency and allocations
  - Identify high-allocation paths
  - Enable fair comparison with Redis-backed reads later

---

## Environment

| Component | Value |
|---------|------|
| OS | Linux |
| Architecture | amd64 |
| CPU | AMD Ryzen 5 5500U |
| Go Version | 1.25 |
| Database | PostgreSQL 17 (Docker) |
| ORM | GORM |

---

## Running Benchmarks Locally

PostgreSQL must be running (Docker or local install).

```bash
go test -run=NONE -bench=. -benchmem ./test/integration
````

To run a specific benchmark:

```bash
go test -run=NONE -bench=BenchmarkJobRepository_Get -benchmem ./test/integration
```

---

## Baseline Results (PostgreSQL)

> ⚠️ Results vary across machines and environments.
> Use these numbers for **relative comparison**, not absolute guarantees.

| Operation            | Time (ns/op) | Allocations (allocs/op) | Memory (B/op) |
| -------------------- | ------------ | ----------------------- | ------------- |
| Create               | ~1,784,401   | 120                     | ~10,021       |
| Get                  | ~512,400     | 129                     | ~7,447        |
| UpdateStatus         | ~1,615,837   | 82                      | ~7,754        |
| IncrementAttempts    | ~1,709,650   | 84                      | ~7,818        |
| SaveResult           | ~1,693,771   | 91                      | ~8,312        |
| List                 | ~1,194,111   | 1,817                   | ~115,293      |
| Get + JSON Unmarshal | ~536,061     | 162                     | ~8,872        |

---

## Observations

* **Read-heavy operations (`Get`) are relatively fast** but still incur
  noticeable allocation overhead.
* **`List` is the most expensive operation**, primarily due to:

  * Large result sets
  * JSON decoding
  * Slice growth allocations
* Write operations (`Create`, `UpdateStatus`) are dominated by:

  * Network round-trips
  * GORM object mapping

---

## Optimization Roadmap

Planned improvements based on benchmark data:

* Introduce **Redis caching** for:

  * `Get`
  * `List`
* Reduce allocations by:

  * Avoiding unnecessary JSON unmarshalling
  * Reusing buffers where possible
* Compare:

  * Postgres-only vs Redis-assisted reads
  * Allocation deltas per operation

All future optimizations will be measured against this baseline.
