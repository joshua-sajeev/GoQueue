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

	cfg, err := config.LoadConfigFromEnv(ctx)
	if err != nil {
		log.Fatal("Couldn't load the config:", err)
	}
	db, err := postgres.ConnectDB(ctx, cfg)
	if err != nil {
		log.Fatal("Failed connecting to DB", err)
	}
	application := app.NewApiApp(db, cfg)
	application.Run(ctx)
}
