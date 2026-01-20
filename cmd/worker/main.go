package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joshu-sajeev/goqueue/internal/app"
	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("Starting Worker...")

	cfg, err := config.LoadConfigFromEnv(ctx)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := postgres.ConnectDB(ctx, cfg)
	if err != nil {
		log.Fatal("Connection failed:", err)
	}

	log.Println("SUCCESS! Database connected")

	worker := app.NewWorkerApp(db, cfg)
	worker.Run(ctx)
}
