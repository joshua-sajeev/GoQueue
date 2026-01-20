package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/job"
	"github.com/joshu-sajeev/goqueue/internal/router"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"gorm.io/gorm"
)

type ApiApp struct {
	Config     *config.Config
	DB         *gorm.DB
	Router     *gin.Engine
	Server     Server
	JobHandler *job.JobHandler
}

func NewApiApp(db *gorm.DB, cfg *config.Config) *ApiApp {
	app := &ApiApp{}

	app.Config = cfg
	app.DB = db

	jobRepo := postgres.NewJobRepository(app.DB)
	jobService := job.NewJobService(jobRepo)
	app.JobHandler = job.NewJobHandler(jobService)

	app.Router = router.NewRouter(app.JobHandler, app.DB, app.Ready)

	addr := fmt.Sprintf(":%s", app.Config.ServerPort)
	app.Server = &HTTPServer{
		Srv: &http.Server{
			Addr:    addr,
			Handler: app.Router.Handler(),
		},
	}

	return app
}

// Ready performs a deep health check of the app dependencies
func (a *ApiApp) Ready() error {
	if a.DB == nil {
		return fmt.Errorf("DB is not present")
	}
	sqlDB, err := a.DB.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return sqlDB.PingContext(ctx)
}

func (a *ApiApp) Run(ctx context.Context) {
	go func() {
		log.Printf("Starting server on %s...", a.Config.ServerPort)
		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received. Cleaning up...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.Server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if a.DB != nil {
		if sqlDB, err := a.DB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Error closing database: %v", err)
			} else {
				log.Println("Database connection closed cleanly")
			}
		}
	}

	log.Println("Server exiting")
}
