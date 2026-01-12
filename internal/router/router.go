package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/internal/job"
	"github.com/joshu-sajeev/goqueue/middleware"
	"gorm.io/gorm"
)

func NewRouter(jobHandler *job.JobHandler, db *gorm.DB, readyCheck func() error) *gin.Engine {

	r := gin.Default()

	r.Use(middleware.TimeoutMiddleware(5*time.Second), middleware.ErrorHandler())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/health", func(c *gin.Context) {
		if err := readyCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
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
	return r
}
