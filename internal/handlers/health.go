package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/common"
	"gorm.io/gorm"
)

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
