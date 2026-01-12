package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/common"
	"github.com/joshu-sajeev/goqueue/middleware"
	"github.com/stretchr/testify/assert"
)

func TestErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		handlerErr     error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "no error",
			handlerErr:     nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `"ok"`,
		},
		{
			name: "api error",
			handlerErr: common.APIError{
				Status:  http.StatusBadRequest,
				Message: "invalid input",
				Fields: map[string]any{
					"name": "required",
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"invalid input"`,
		},
		{
			name:           "generic error",
			handlerErr:     errors.New("boom"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"boom"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(middleware.ErrorHandler())

			r.GET("/test", func(c *gin.Context) {
				if tt.handlerErr != nil {
					c.Error(tt.handlerErr)
					return
				}
				c.JSON(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}
