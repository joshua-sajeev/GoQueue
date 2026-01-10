package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/models"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestJobRepository_Create(t *testing.T) {
	tests := []struct {
		name    string
		job     *models.Job
		wantErr bool
		setup   func(db *gorm.DB)
	}{
		{
			name: "success case",
			job: &models.Job{
				ID:         1,
				Queue:      "email",
				Payload:    datatypes.JSON([]byte(`{"email":"test@example.com","foo":"bar"}`)),
				Status:     config.JobStatusQueued,
				Attempts:   0,
				MaxRetries: 10,
				Result:     datatypes.JSON([]byte(`{"status":"ok"}`)),
				Error:      "s",
			},
			wantErr: false,
		},
		{
			name: "db error on duplicate primary key",
			job: &models.Job{
				ID:    2,
				Queue: "email",
			},
			setup: func(db *gorm.DB) {
				_ = db.Create(&models.Job{
					ID:    2,
					Queue: "reports",
				}).Error
			},
			wantErr: true,
		},
		{
			name: "error when db connection is closed",
			job: &models.Job{
				ID:    3,
				Queue: "email",
			},
			setup: func(db *gorm.DB) {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			db, ctx := setupTestDB(t)
			defer closeTestDB(db)

			// Create repository
			repo := postgres.NewJobRepository(db)

			// Run setup if provided
			if tt.setup != nil {
				tt.setup(db)
			}

			// Execute the test
			err := repo.Create(ctx, tt.job)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "create job")
				return
			}

			require.NoError(t, err)

			// Verify the job was saved
			var saved models.Job
			dbErr := db.First(&saved, tt.job.ID).Error
			require.NoError(t, dbErr)

			// Basic field checks
			assert.Equal(t, tt.job.Queue, saved.Queue)
			assert.Equal(t, tt.job.Status, saved.Status)
			assert.Equal(t, tt.job.Attempts, saved.Attempts)
			assert.Equal(t, tt.job.MaxRetries, saved.MaxRetries)
			assert.Equal(t, tt.job.Error, saved.Error)

			// Validate JSON content
			if len(saved.Payload) > 0 {
				var payload map[string]any
				err = json.Unmarshal(saved.Payload, &payload)
				require.NoError(t, err)
				assert.Equal(t, "test@example.com", payload["email"])
				assert.Equal(t, "bar", payload["foo"])
			}

			if len(saved.Result) > 0 {
				var result map[string]any
				err = json.Unmarshal(saved.Result, &result)
				require.NoError(t, err)
				assert.Equal(t, "ok", result["status"])
			}
		})
	}
}

func TestJobRepository_Get(t *testing.T) {
	tests := []struct {
		name        string
		id          uint
		setup       func(db *gorm.DB)
		wantErr     bool
		errContains string
	}{
		{
			name: "successfully get existing job",
			id:   1,
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{
					ID:         1,
					Queue:      "email",
					Payload:    datatypes.JSON([]byte(`{"email":"test@example.com","foo":"bar"}`)),
					Status:     config.JobStatusCompleted,
					Attempts:   0,
					MaxRetries: 10,
					Result:     datatypes.JSON([]byte(`{"status":"ok"}`)),
					Error:      "s",
				})
			},
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "job not found",
			id:          999,
			wantErr:     true,
			errContains: "job not found",
		},
		{
			name: "db failure during get",
			id:   1,
			setup: func(db *gorm.DB) {
				sqlDB, _ := db.DB()
				_ = sqlDB.Close()
			},
			wantErr:     true,
			errContains: "get job",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			db, ctx := setupTestDB(t)
			defer closeTestDB(db)

			repo := postgres.NewJobRepository(db)

			if tt.setup != nil {
				tt.setup(db)
			}

			got, err := repo.Get(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, uint(1), got.ID)
			assert.Equal(t, "email", got.Queue)
			assert.Equal(t, config.JobStatusCompleted, got.Status)
			assert.Equal(t, 0, got.Attempts)
			assert.Equal(t, 10, got.MaxRetries)
			assert.Equal(t, "s", got.Error)

			var payload map[string]any
			err = json.Unmarshal(got.Payload, &payload)
			require.NoError(t, err)
			assert.Equal(t, "test@example.com", payload["email"])
			assert.Equal(t, "bar", payload["foo"])

			var result map[string]any
			err = json.Unmarshal(got.Result, &result)
			require.NoError(t, err)
			assert.Equal(t, "ok", result["status"])
		})
	}
}

func TestJobRepository_UpdateStatus(t *testing.T) {
	tests := []struct {
		name        string
		id          uint
		status      config.JobStatus
		setup       func(db *gorm.DB)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successfully update job status",
			id:     1,
			status: config.JobStatusRunning,
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{
					ID:     1,
					Queue:  "email",
					Status: config.JobStatusQueued,
				})
			},
			wantErr: false,
		},
		{
			name:   "db failure during update",
			id:     1,
			status: config.JobStatusCompleted,
			setup: func(db *gorm.DB) {
				sqlDB, _ := db.DB()
				_ = sqlDB.Close()
			},
			wantErr:     true,
			errContains: "update status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, ctx := setupTestDB(t)
			defer closeTestDB(db)

			repo := postgres.NewJobRepository(db)

			if tt.setup != nil {
				tt.setup(db)
			}

			err := repo.UpdateStatus(ctx, tt.id, tt.status)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)

			var job models.Job
			err = db.First(&job, tt.id).Error
			require.NoError(t, err)
			assert.Equal(t, tt.status, job.Status)
		})
	}
}

func TestJobRepository_IncrementAttempts(t *testing.T) {
	tests := []struct {
		name        string
		id          uint
		setup       func(db *gorm.DB)
		wantErr     bool
		errContains string
	}{
		{
			name: "successfully increment attempts",
			id:   1,
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{
					ID:       1,
					Attempts: 0,
				})
			},
			wantErr: false,
		},
		{
			name: "db failure during increment",
			id:   1,
			setup: func(db *gorm.DB) {
				sqlDB, _ := db.DB()
				_ = sqlDB.Close()
			},
			wantErr:     true,
			errContains: "increment attempts",
		},
		{
			name: "increment attempts multiple times",
			id:   1,
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{
					ID:       1,
					Attempts: 2,
				})
			},
			wantErr: false,
		},
		{
			name: "increment attempts on non-existent job",
			id:   999,
			setup: func(db *gorm.DB) {
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, ctx := setupTestDB(t)
			defer closeTestDB(db)
			repo := postgres.NewJobRepository(db)

			if tt.setup != nil {
				tt.setup(db)
			}

			err := repo.IncrementAttempts(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)

			// Skip verification for non-existent job test
			if tt.name == "increment attempts on non-existent job" {
				return
			}

			var job models.Job
			require.NoError(t, db.First(&job, tt.id).Error)

			switch tt.name {
			case "increment attempts multiple times":
				assert.Equal(t, 3, job.Attempts)
			case "successfully increment attempts":
				assert.Equal(t, 1, job.Attempts)
			}
		})
	}
}

func TestJobRepository_SaveResult(t *testing.T) {
	tests := []struct {
		name        string
		id          uint
		result      datatypes.JSON
		errMsg      string
		setup       func(db *gorm.DB)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successfully save result",
			id:     1,
			result: datatypes.JSON([]byte(`{"status":"ok"}`)),
			errMsg: "",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{ID: 1})
			},
			wantErr: false,
		},
		{
			name:   "db failure while saving result",
			id:     1,
			result: datatypes.JSON([]byte(`{"status":"failed"}`)),
			errMsg: "error",
			setup: func(db *gorm.DB) {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			},
			wantErr:     true,
			errContains: "save result",
		},
		{
			name:   "save result with empty json",
			id:     1,
			result: datatypes.JSON([]byte(`{}`)),
			errMsg: "",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{ID: 1})
			},
			wantErr: false,
		},
		{
			name:   "save result with empty error message",
			id:     1,
			result: datatypes.JSON([]byte(`{"status":"completed"}`)),
			errMsg: "",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{ID: 1})
			},
			wantErr: false,
		},
		{
			name:   "save result with both empty",
			id:     1,
			result: datatypes.JSON([]byte(`{}`)),
			errMsg: "",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{ID: 1})
			},
			wantErr: false,
		},
		{
			name:   "save result with null json array",
			id:     1,
			result: datatypes.JSON([]byte(`[]`)),
			errMsg: "test error",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{ID: 1})
			},
			wantErr: false,
		},
		{
			name:   "save result with complex nested json",
			id:     1,
			result: datatypes.JSON([]byte(`{"data":{"nested":{"value":"test"}},"count":42}`)),
			errMsg: "",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{ID: 1})
			},
			wantErr: false,
		},
		{
			name:   "save result overwrites previous values",
			id:     1,
			result: datatypes.JSON([]byte(`{"new":"result"}`)),
			errMsg: "new error",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{
					ID:     1,
					Result: datatypes.JSON([]byte(`{"old":"result"}`)),
					Error:  "old error",
				})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, ctx := setupTestDB(t)
			defer closeTestDB(db)

			repo := postgres.NewJobRepository(db)

			if tt.setup != nil {
				tt.setup(db)
			}

			err := repo.SaveResult(ctx, tt.id, tt.result, tt.errMsg)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)

			var job models.Job
			require.NoError(t, db.First(&job, tt.id).Error)
			assert.JSONEq(t, string(tt.result), string(job.Result))
			assert.Equal(t, tt.errMsg, job.Error)
		})
	}
}

func TestJobRepository_List(t *testing.T) {
	tests := []struct {
		name        string
		queue       string
		setup       func(db *gorm.DB)
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name:  "list jobs by queue",
			queue: "email",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{ID: 1, Queue: "email"})
				db.Create(&models.Job{ID: 2, Queue: "email"})
				db.Create(&models.Job{ID: 3, Queue: "sms"})
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:  "db failure during list",
			queue: "email",
			setup: func(db *gorm.DB) {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			},
			wantErr:     true,
			errContains: "list jobs",
		},
		{
			name:  "list jobs from empty queue",
			queue: "nonexistent",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{ID: 1, Queue: "email"})
				db.Create(&models.Job{ID: 2, Queue: "sms"})
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:  "list jobs with single queue",
			queue: "sms",
			setup: func(db *gorm.DB) {
				db.Create(&models.Job{ID: 1, Queue: "email"})
				db.Create(&models.Job{ID: 2, Queue: "sms"})
				db.Create(&models.Job{ID: 3, Queue: "email"})
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:  "list jobs from empty database",
			queue: "email",
			setup: func(db *gorm.DB) {
				// Empty database
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:  "list all jobs from queue with many entries",
			queue: "reports",
			setup: func(db *gorm.DB) {
				for i := 1; i <= 10; i++ {
					db.Create(&models.Job{ID: uint(i), Queue: "reports"})
				}
				db.Create(&models.Job{ID: 11, Queue: "email"})
			},
			wantCount: 10,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, ctx := setupTestDB(t)
			defer closeTestDB(db)

			repo := postgres.NewJobRepository(db)

			if tt.setup != nil {
				tt.setup(db)
			}

			jobs, err := repo.List(ctx, tt.queue)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Len(t, jobs, tt.wantCount)
		})
	}
}

func TestJobRepository_AcquireNext(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		queue        string
		workerID     uint
		lockDuration time.Duration
		setup        func(db *gorm.DB)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "no jobs available",
			queue:        "default",
			workerID:     1,
			lockDuration: time.Minute,
			setup:        func(db *gorm.DB) {},
			wantErr:      true,
			errContains:  "no jobs available",
		},
		{
			name:         "successfully acquires queued job",
			queue:        "default",
			workerID:     42,
			lockDuration: time.Minute,
			setup: func(db *gorm.DB) {
				job := models.Job{
					Queue:       "default",
					Payload:     datatypes.JSON([]byte(`{"to":"test@example.com"}`)),
					Status:      config.JobStatusQueued,
					AvailableAt: now.Add(-time.Minute),
				}
				require.NoError(t, db.Create(&job).Error)
			},
			wantErr: false,
		},
		{
			name:         "job not yet available",
			queue:        "default",
			workerID:     1,
			lockDuration: time.Minute,
			setup: func(db *gorm.DB) {
				job := models.Job{
					Queue:       "default",
					Status:      config.JobStatusQueued,
					AvailableAt: now.Add(time.Hour),
				}
				require.NoError(t, db.Create(&job).Error)
			},
			wantErr:     true,
			errContains: "no jobs available",
		},
		{
			name:         "locked job is skipped",
			queue:        "default",
			workerID:     2,
			lockDuration: time.Minute,
			setup: func(db *gorm.DB) {
				lockedAt := now
				job := models.Job{
					Queue:       "default",
					Status:      config.JobStatusQueued,
					AvailableAt: now.Add(-time.Minute),
					LockedAt:    &lockedAt,
					LockedBy:    ptrUint(1),
				}
				require.NoError(t, db.Create(&job).Error)
			},
			wantErr:     true,
			errContains: "no jobs available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, ctx := setupTestDB(t)
			defer closeTestDB(db)

			repo := postgres.NewJobRepository(db)

			if tt.setup != nil {
				tt.setup(db)
			}

			job, err := repo.AcquireNext(ctx, tt.queue, tt.workerID, tt.lockDuration)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, job)

			assert.Equal(t, tt.queue, job.Queue)
			assert.NotZero(t, job.ID)
			assert.NotNil(t, job.Payload)

			var lockedJob models.Job
			require.NoError(t, db.First(&lockedJob, job.ID).Error)
			assert.Equal(t, config.JobStatusRunning, lockedJob.Status)
			assert.NotNil(t, lockedJob.LockedAt)
			assert.Equal(t, tt.workerID, *lockedJob.LockedBy)
		})
	}
}

func ptrUint(v uint) *uint {
	return &v
}

func TestJobRepository_Release(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		setup   func(db *gorm.DB) uint
		wantErr bool
	}{
		{
			name: "release existing job",
			setup: func(db *gorm.DB) uint {
				job := models.Job{
					Queue:    "default",
					Status:   config.JobStatusRunning,
					LockedAt: &now,
					LockedBy: ptrUint(42),
				}
				require.NoError(t, db.Create(&job).Error)
				return job.ID
			},
			wantErr: false,
		},
		{
			name: "release non-existent job (idempotent)",
			setup: func(db *gorm.DB) uint {
				return 9999
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, ctx := setupTestDB(t)
			defer closeTestDB(db)

			repo := postgres.NewJobRepository(db)
			id := tt.setup(db)

			err := repo.Release(ctx, id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			var job models.Job
			if db.First(&job, id).Error == nil {
				assert.Equal(t, config.JobStatusQueued, job.Status)
				assert.Nil(t, job.LockedBy)
				assert.Nil(t, job.LockedAt)
			}
		})
	}
}

func TestJobRepository_RetryLater(t *testing.T) {
	now := time.Now()
	availableAt := now.Add(time.Minute)

	tests := []struct {
		name  string
		setup func(db *gorm.DB) uint
	}{
		{
			name: "retry existing job",
			setup: func(db *gorm.DB) uint {
				job := models.Job{
					Queue:    "default",
					Status:   config.JobStatusRunning,
					LockedAt: &now,
					LockedBy: ptrUint(1),
				}
				require.NoError(t, db.Create(&job).Error)
				return job.ID
			},
		},
		{
			name: "retry non-existent job (idempotent)",
			setup: func(db *gorm.DB) uint {
				return 9999
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, ctx := setupTestDB(t)
			defer closeTestDB(db)

			repo := postgres.NewJobRepository(db)
			id := tt.setup(db)

			err := repo.RetryLater(ctx, id, availableAt)
			require.NoError(t, err)

			var job models.Job
			if db.First(&job, id).Error == nil {
				assert.Equal(t, config.JobStatusQueued, job.Status)
				assert.Nil(t, job.LockedBy)
				assert.Nil(t, job.LockedAt)
				assert.WithinDuration(t, availableAt, job.AvailableAt, time.Millisecond)
			}
		})
	}
}

func TestJobRepository_ListStuckJobs(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		setup       func(db *gorm.DB)
		stale       time.Duration
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name: "returns stuck jobs",
			setup: func(db *gorm.DB) {
				old := now.Add(-2 * time.Hour)
				jobs := []models.Job{
					{Queue: "default", Status: config.JobStatusRunning, LockedAt: &old},
					{Queue: "default", Status: config.JobStatusRunning, LockedAt: &old},
					{Queue: "default", Status: config.JobStatusQueued},
				}
				for _, j := range jobs {
					require.NoError(t, db.Create(&j).Error)
				}
			},
			stale:     time.Hour,
			wantCount: 2,
		},
		{
			name: "no stuck jobs",
			setup: func(db *gorm.DB) {
				j := models.Job{Queue: "default", Status: config.JobStatusRunning, LockedAt: &now}
				require.NoError(t, db.Create(&j).Error)
			},
			stale:     time.Hour,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, ctx := setupTestDB(t)
			defer closeTestDB(db)

			repo := postgres.NewJobRepository(db)

			if tt.setup != nil {
				tt.setup(db)
			}

			jobs, err := repo.ListStuckJobs(ctx, tt.stale)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Len(t, jobs, tt.wantCount)
		})
	}
}
