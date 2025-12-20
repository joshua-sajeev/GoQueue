package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/common"
	"github.com/joshu-sajeev/goqueue/internal/job"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"github.com/joshu-sajeev/goqueue/middleware"
	"gorm.io/gorm"
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

	log.Println("SUCCESS! Database connected")

	jobRepo := postgres.NewJobRepository(db)
	jobService := job.NewJobService(jobRepo)
	jobHandler := job.NewJobHandler(jobService)
	r := gin.Default()

	r.Use(middleware.TimeoutMiddleware(5*time.Second), middleware.ErrorHandler())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/health/db", HealthCheckHandler(db))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	jobs := r.Group("/jobs")
	{
		jobs.POST("/create", jobHandler.Create)
		jobs.GET("/:id", jobHandler.Get)
		jobs.PUT("/:id/status", jobHandler.Update)
		jobs.POST("/:id/increment", jobHandler.Increment)
		jobs.POST("/:id/save", jobHandler.Save)
		jobs.GET("/", jobHandler.List)
	}
	log.Println("Starting server on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func HealthCheckHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			apiErr := common.APIError{
				Status:  http.StatusServiceUnavailable,
				Message: "failed to get database instance",
			}
			c.JSON(apiErr.Status, apiErr)
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			apiErr := common.APIError{
				Status:  http.StatusServiceUnavailable,
				Message: "database is unavailable",
			}
			c.JSON(apiErr.Status, apiErr)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}
}
