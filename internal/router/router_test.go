package router

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/internal/job"
	"github.com/stretchr/testify/assert"
)

func TestHealthRoute_Healthy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ready := func() error { return nil }
	r := NewRouter(&job.JobHandler{}, nil, ready)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"healthy"`)
}

func TestHealthRoute_Unhealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ready := func() error { return errors.New("db down") }
	r := NewRouter(&job.JobHandler{}, nil, ready)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"unhealthy"`)
}

func TestRootRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := NewRouter(&job.JobHandler{}, nil, func() error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ok"`)
}
