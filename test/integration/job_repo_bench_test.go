package integration

import (
	"encoding/json"
	"testing"

	"github.com/joshu-sajeev/goqueue/internal/models"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"gorm.io/datatypes"
)

// BenchmarkJobRepository_Create benchmarks the Create method
func BenchmarkJobRepository_Create(b *testing.B) {
	db, ctx := setupTestDB(b)
	defer closeTestDB(db)

	repo := postgres.NewJobRepository(db)

	job := &models.Job{
		Queue:      "bench",
		Type:       "test_create",
		Payload:    datatypes.JSON([]byte(`{"foo":"bar"}`)),
		Status:     "pending",
		Attempts:   0,
		MaxRetries: 5,
	}

	for i := 0; b.Loop(); i++ {
		job.ID = uint(i + 1) // unique ID for each iteration
		_ = repo.Create(ctx, job)
	}
}

// BenchmarkJobRepository_Get benchmarks the Get method
func BenchmarkJobRepository_Get(b *testing.B) {
	db, ctx := setupTestDB(b)
	defer closeTestDB(db)

	repo := postgres.NewJobRepository(db)

	// Create a test job
	job := &models.Job{Queue: "bench", Type: "test_get", Status: "pending"}
	_ = repo.Create(ctx, job)

	for b.Loop() {
		_, _ = repo.Get(ctx, job.ID)
	}
}

// BenchmarkJobRepository_UpdateStatus benchmarks the UpdateStatus method
func BenchmarkJobRepository_UpdateStatus(b *testing.B) {
	db, ctx := setupTestDB(b)
	defer closeTestDB(db)

	repo := postgres.NewJobRepository(db)

	job := &models.Job{Queue: "bench", Type: "test_update_status", Status: "pending"}
	_ = repo.Create(ctx, job)

	for b.Loop() {
		_ = repo.UpdateStatus(ctx, job.ID, "processing")
	}
}

// BenchmarkJobRepository_IncrementAttempts benchmarks IncrementAttempts
func BenchmarkJobRepository_IncrementAttempts(b *testing.B) {
	db, ctx := setupTestDB(b)
	defer closeTestDB(db)

	repo := postgres.NewJobRepository(db)

	job := &models.Job{Queue: "bench", Type: "test_increment", Attempts: 0}
	_ = repo.Create(ctx, job)

	for b.Loop() {
		_ = repo.IncrementAttempts(ctx, job.ID)
	}
}

// BenchmarkJobRepository_SaveResult benchmarks SaveResult
func BenchmarkJobRepository_SaveResult(b *testing.B) {
	db, ctx := setupTestDB(b)
	defer closeTestDB(db)

	repo := postgres.NewJobRepository(db)

	job := &models.Job{Queue: "bench", Type: "test_save_result"}
	_ = repo.Create(ctx, job)

	resultJSON := datatypes.JSON([]byte(`{"status":"ok"}`))
	errMsg := "error message"

	for b.Loop() {
		_ = repo.SaveResult(ctx, job.ID, resultJSON, errMsg)
	}
}

// BenchmarkJobRepository_List benchmarks List
func BenchmarkJobRepository_List(b *testing.B) {
	db, ctx := setupTestDB(b)
	defer closeTestDB(db)

	repo := postgres.NewJobRepository(db)

	// Create multiple jobs for the queue
	for range 100 {
		_ = repo.Create(ctx, &models.Job{Queue: "bench_list", Type: "test_list"})
	}

	for b.Loop() {
		_, _ = repo.List(ctx, "bench_list")
	}
}

// Optional: BenchmarkJobRepository_GetWithJSONUnmarshal
// Includes JSON unmarshal overhead like your tests
func BenchmarkJobRepository_GetWithJSONUnmarshal(b *testing.B) {
	db, ctx := setupTestDB(b)
	defer closeTestDB(db)

	repo := postgres.NewJobRepository(db)

	payload := datatypes.JSON([]byte(`{"email":"test@example.com","foo":"bar"}`))
	result := datatypes.JSON([]byte(`{"status":"ok"}`))
	job := &models.Job{Queue: "bench_json", Type: "test_json", Payload: payload, Result: result}
	_ = repo.Create(ctx, job)

	for b.Loop() {
		got, _ := repo.Get(ctx, job.ID)
		var p map[string]any
		_ = json.Unmarshal(got.Payload, &p)
		var r map[string]any
		_ = json.Unmarshal(got.Result, &r)
	}
}
