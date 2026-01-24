package app

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestWorkerApp(t *testing.T) {
	// 1. Setup Mock DB
	mockDb, _, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{Conn: mockDb})
	db, _ := gorm.Open(dialector, &gorm.Config{})

	cfg := &config.Config{
		MaxWorkers: 5,
	}

	t.Run("NewWorkerApp wiring", func(t *testing.T) {
		app := NewWorkerApp(db, cfg)
		assert.NotNil(t, app)
		assert.Equal(t, db, app.DB)
		assert.NotNil(t, app.WorkerPool)
	})

	t.Run("Run responds to context cancellation", func(t *testing.T) {
		app := NewWorkerApp(db, cfg)

		// Create a context we can cancel
		ctx, cancel := context.WithCancel(context.Background())

		// Run in background
		errChan := make(chan struct{})
		go func() {
			app.Run(ctx)
			close(errChan)
		}()

		// Simulate some work time
		time.Sleep(100 * time.Millisecond)

		// Signal shutdown
		cancel()

		// Verify Run exits
		select {
		case <-errChan:
			// Success: Run returned
		case <-time.After(2 * time.Second):
			t.Fatal("WorkerApp.Run did not exit after context cancellation")
		}
	})
}
