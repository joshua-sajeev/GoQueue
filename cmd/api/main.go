package main

import (
	"context"
	"log"

	"github.com/joshu-sajeev/goqueue/internal/models"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	log.Println("Starting...")

	ctx := context.Background()
	cfg, err := postgres.LoadConfigFromEnv(ctx)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := postgres.ConnectDB(cfg)
	if err != nil {
		log.Fatal("Connection failed:", err)
	}

	db.Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Silent),
	}).AutoMigrate(&models.Job{})
	log.Println("SUCCESS! Database connected")
	_ = db
	select {}
}
