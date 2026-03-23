package app

import (
	"context"
	"net/http"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/pool"
	pg "github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWorkerApp(t *testing.T) {
	mockDb, _, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{Conn: mockDb})
	db, _ := gorm.Open(dialector, &gorm.Config{})

	cfg := &config.Config{MaxWorkers: 5}

	t.Run("NewWorkerApp wiring", func(t *testing.T) {
		app := NewWorkerApp(db, cfg)
		assert.NotNil(t, app)
		assert.Equal(t, db, app.DB)
		assert.NotNil(t, app.WorkerPool)
	})

	t.Run("Run responds to context cancellation", func(t *testing.T) {
		app := NewWorkerApp(db, cfg)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			app.Run(ctx)
			close(done)
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("WorkerApp.Run did not exit after context cancellation")
		}
	})
}

// newTestWorkerApp builds a WorkerApp backed by a real WorkerPool.
// Pass nil to get an app with no DB and no pool (covers the nil-DB branch).
func newTestWorkerApp(t *testing.T, db *gorm.DB) *WorkerApp {
	t.Helper()

	var workerPool *pool.WorkerPool
	if db != nil {
		repo := pg.NewJobRepository(db)
		workerPool = pool.NewWorkerPool(3, repo, []string{"default"}, time.Minute)
	}

	return &WorkerApp{
		DB:         db,
		WorkerPool: workerPool,
		Config:     &config.Config{},
	}
}

func TestWorkerApp_Health(t *testing.T) {
	tests := []struct {
		name           string
		setupApp       func(t *testing.T) *WorkerApp
		expectedStatus int
		expectedBody   map[string]any
	}{
		{
			name: "nil DB returns unhealthy",
			setupApp: func(t *testing.T) *WorkerApp {
				return &WorkerApp{DB: nil}
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody: map[string]any{
				"status": "unhealthy",
				"error":  "database not initialised",
			},
		},
		{
			name: "healthy DB returns ok with worker count",
			setupApp: func(t *testing.T) *WorkerApp {
				db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
				require.NoError(t, err)
				return newTestWorkerApp(t, db)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]any{
				"status":  "healthy",
				"workers": 3,
			},
		},
		{
			name: "closed DB connection causes ping failure",
			setupApp: func(t *testing.T) *WorkerApp {
				db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
				require.NoError(t, err)

				// gorm.DB.DB() still succeeds after this —
				// only PingContext will fail
				sqlDB, err := db.DB()
				require.NoError(t, err)
				sqlDB.Close()

				return newTestWorkerApp(t, db)
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody: map[string]any{
				"status": "unhealthy",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupApp(t)

			status, body := app.health()

			assert.Equal(t, tt.expectedStatus, status)
			for k, v := range tt.expectedBody {
				assert.Equal(t, v, body[k], "body mismatch for key %q", k)
			}
		})
	}
}
