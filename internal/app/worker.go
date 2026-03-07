package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	queues := []string{"email", "payment", "default", "webhooks"}

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

// health checks DB connectivity and returns pool state.
// Called by the /health handler on every request.
func (a *WorkerApp) health() (status int, body map[string]any) {
	// Check DB
	if a.DB == nil {
		return http.StatusServiceUnavailable, map[string]any{
			"status": "unhealthy",
			"error":  "database not initialised",
		}
	}

	sqlDB, err := a.DB.DB()
	if err != nil {
		return http.StatusServiceUnavailable, map[string]any{
			"status": "unhealthy",
			"error":  fmt.Sprintf("db unavailable: %v", err),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return http.StatusServiceUnavailable, map[string]any{
			"status": "unhealthy",
			"error":  fmt.Sprintf("db ping failed: %v", err),
		}
	}

	return http.StatusOK, map[string]any{
		"status":  "healthy",
		"workers": a.WorkerPool.WorkerCount(),
	}
}

// startHealthServer runs a minimal HTTP server on port 8081.
// It is started in a goroutine and respects ctx cancellation via
// server.Shutdown so it doesn't leak after the worker stops.
func (a *WorkerApp) startHealthServer(ctx context.Context) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		code, body := a.health()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(body)
	})

	srv := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	// Start server
	go func() {
		log.Println("Worker health server listening on :8081")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Worker health server error: %v", err)
		}
	}()

	// Shut down cleanly when ctx is cancelled (same signal that stops the pool)
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Worker health server shutdown error: %v", err)
	}
}

func (a *WorkerApp) Run(ctx context.Context) {
	a.WorkerPool.Start()
	log.Println("Worker pool active...")

	// Health server runs alongside the pool, shuts down with it
	go a.startHealthServer(ctx)

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
