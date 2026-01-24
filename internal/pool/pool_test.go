package pool

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	pg "github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestWorkerPool(t *testing.T) {
	mockDb, _, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{Conn: mockDb})
	db, _ := gorm.Open(dialector, &gorm.Config{})
	repo := pg.NewJobRepository(db)

	count := 3
	queues := []string{"default"}
	dur := 1 * time.Minute

	t.Run("NewWorkerPool initializes correctly", func(t *testing.T) {
		p := NewWorkerPool(count, repo, queues, dur)
		assert.NotNil(t, p)
		assert.Equal(t, count, len(p.workers))
		assert.NotNil(t, p.ctx)
		assert.NotNil(t, p.cancel)
	})

	t.Run("Lifecycle: Start and Stop", func(t *testing.T) {
		p := NewWorkerPool(count, repo, queues, dur)

		p.Start()

		done := make(chan bool)
		go func() {
			p.Stop()
			done <- true
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("WorkerPool.Stop() timed out")
		}
	})
}
