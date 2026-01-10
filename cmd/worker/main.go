package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/pool"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
)

func main() {
	log.Println("Starting Worker...")

	ctx := context.Background()
	cfg, err := postgres.LoadConfigFromEnv(ctx)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := postgres.ConnectDB(ctx, cfg)
	if err != nil {
		log.Fatal("Connection failed:", err)
	}

	log.Println("SUCCESS! Database connected")

	repo := postgres.NewJobRepository(db)
	queues := []string{"email", "payment", "default", "webhooks"}
	temp := os.Getenv("MAX_WORKERS")
	maxWorkers := 10

	if v, err := strconv.Atoi(temp); err == nil && v > 0 {
		maxWorkers = v
	}

	workerPool := pool.NewWorkerPool(maxWorkers, repo, queues, 1*time.Minute)

	workerPool.Start()
	log.Println("Worker pool active. Press Ctrl+C to stop.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	workerPool.Stop()
	log.Println("Shutdown complete.")
}
