package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/common"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		if apiErr, ok := err.(common.APIError); ok {
			response := gin.H{"error": apiErr.Message}
			if apiErr.Fields != nil {
				response["fields"] = apiErr.Fields
			}
			c.JSON(apiErr.Status, response)
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
