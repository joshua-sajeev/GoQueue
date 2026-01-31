package app

import (
	"context"
	"log"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/pool"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"gorm.io/gorm"
)

type WorkerApp struct {
	Config     *config.Config
	DB         *gorm.DB
	WorkerPool *pool.WorkerPool
}

func NewWorkerApp(db *gorm.DB, cfg *config.Config) *WorkerApp {
	repo := postgres.NewJobRepository(db)
	queues := []string{"email", "payment", "default", "webhook"}

	maxWorkers := 10
	if v := cfg.MaxWorkers; v > 0 {
		maxWorkers = v
	}

	workerPool := pool.NewWorkerPool(maxWorkers, repo, queues, 1*time.Minute)

	return &WorkerApp{
		Config:     cfg,
		DB:         db,
		WorkerPool: workerPool,
	}
}

func (a *WorkerApp) Run(ctx context.Context) {
	a.WorkerPool.Start()
	log.Println("Worker pool active...")

	<-ctx.Done()
	log.Println("Shutdown signal received. Stopping worker pool...")

	a.WorkerPool.Stop()

	if a.DB != nil {
		sqlDB, err := a.DB.DB()
		if err != nil {
			log.Printf("Error getting underlying DB: %v", err)
		} else {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Error closing database: %v", err)
			} else {
				log.Println("Database connection closed cleanly")
			}
		}
	}
}
