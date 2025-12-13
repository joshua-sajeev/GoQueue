package main

import (
	"context"
	"log"

	"github.com/joshu-sajeev/goqueue/internal/models"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
)

func main() {
	log.Println("Starting...")

	ctx := context.Background()
	cfg, err := postgres.LoadConfigFromEnv(ctx)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := postgres.ConnectDB(ctx, cfg)
	if err != nil {
		log.Fatal("Connection failed:", err)
	}

	if err := postgres.MigrateModels(db, &models.Job{}); err != nil {
		log.Fatalf("Failed to migrate Job: %v", err)
	}
	log.Println("SUCCESS! Database connected")
	_ = db
	select {}
}
