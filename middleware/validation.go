package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joshu-sajeev/goqueue/common"
)

var validate = validator.New()

func Bind[T any](c *gin.Context, dest *T) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		c.Error(common.Errf(http.StatusBadRequest, "invalid json: %v", err.Error()))
		return false
	}

	if err := validate.Struct(dest); err != nil {
		c.Error(common.APIError{
			Status:  http.StatusBadRequest,
			Message: "validation failed",
			Fields:  FormatValidationErrors(err),
		})
		return false
	}

	return true
}

func FormatValidationErrors(err error) map[string]any {
	errors := map[string]any{}
	for _, e := range err.(validator.ValidationErrors) {
		errors[e.Field()] = "failed " + e.Tag()
	}
	return errors
}
