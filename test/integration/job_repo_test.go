package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/models"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
				Queue:      "hi",
				Type:       "hi",
				Payload:    datatypes.JSON([]byte(`{"email":"test@example.com","foo":"bar"}`)),
				Status:     "done",
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
				Queue: "q",
				Type:  "t",
			},
			setup: func(db *gorm.DB) {
				_ = db.Create(&models.Job{
					ID:    2,
					Queue: "existing",
					Type:  "existing",
				}).Error
			},
			wantErr: true,
		},
		{
			name: "error when db connection is closed",
			job: &models.Job{
				ID:    3,
				Queue: "q",
				Type:  "email",
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
			// Get a fresh DB connection from the test database set up by TestMain
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			config := &postgres.Config{
				User:       "testuser",
				Password:   "testpass",
				Host:       "localhost",
				Port:       testPort, // Set by TestMain
				Database:   "example",
				MaxRetries: 3,
				RetryDelay: 100 * time.Millisecond,
				LogLevel:   logger.Silent,
			}

			db, err := postgres.ConnectDB(ctx, config)
			require.NoError(t, err, "Failed to connect to test database")

			defer func() {
				sqlDB, _ := db.DB()
				if sqlDB != nil {
					sqlDB.Close()
				}
			}()

			// Clean up the jobs table before each test
			db.Exec("DELETE FROM jobs WHERE id IN (1, 2, 3)")

			// Create repository
			repo := postgres.NewJobRepository(db)

			// Run setup if provided
			if tt.setup != nil {
				tt.setup(db)
			}

			// Execute the test
			err = repo.Create(context.Background(), tt.job)

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
			assert.Equal(t, tt.job.Type, saved.Type)
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
