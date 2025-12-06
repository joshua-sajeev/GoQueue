package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/joshu-sajeev/goqueue/internal/models"
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
			db := SetupTestDB(t)
			repo := NewJobRepository(db)

			// Call setup BEFORE creating the job
			if tt.setup != nil {
				tt.setup(db)
			}

			err := repo.Create(context.Background(), tt.job)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "create job")
				return
			}

			require.NoError(t, err)

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
