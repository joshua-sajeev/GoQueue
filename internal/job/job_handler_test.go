package job

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/common"
	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/mocks"
	"github.com/joshu-sajeev/goqueue/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestJobHandler_Create(t *testing.T) {

	tests := []struct {
		name           string
		body           string
		setupMock      func(*mocks.JobServiceMock)
		setupContext   func(*gin.Context)
		expectedStatus int
	}{
		{
			name: "successful job creation",
			body: `{"queue":"default","payload":{"email":"test@example.com","subject":"Test"},"maxRetries":3}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},

		{
			name:           "invalid request body JSON",
			body:           "{invalid json}",
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid JSON payload",
			body: `{"queue":"default","payload":"{invalid}"}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusBadRequest, "payload must be valid JSON"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid queue",
			body: `{"queue":"invalid_queue","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.NewAPIError(http.StatusBadRequest, "invalid queue", map[string]any{
						"provided": "invalid_queue",
						"allowed":  []string{"default", "email", "reports", "webhooks"},
					}))
			},
			expectedStatus: http.StatusBadRequest,
		},

		{
			name: "database connection error",
			body: `{"queue":"default","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusInternalServerError, "failed to add job to database: database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(mocks.JobServiceMock)
			tt.setupMock(mockService)

			req := httptest.NewRequest(
				http.MethodPost,
				"/jobs",
				bytes.NewReader([]byte(tt.body)),
			)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			if tt.setupContext != nil {
				tt.setupContext(c)
			}

			r := gin.New()
			r.Use(middleware.TimeoutMiddleware(5*time.Second), middleware.ErrorHandler())
			handler := NewJobHandler(mockService)
			r.POST("/jobs", handler.Create)

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch for test: %s", tt.name)
			mockService.AssertExpectations(t)
		})
	}
}

func TestJobHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validJobResponse := &dto.JobResponseDTO{
		ID:         1,
		Queue:      "email",
		Payload:    json.RawMessage(`{"email":"test@example.com","subject":"Test"}`),
		Status:     config.JobStatusQueued,
		Attempts:   0,
		MaxRetries: 3,
	}

	tests := []struct {
		name           string
		jobID          string
		setupMock      func(*mocks.JobServiceMock)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "successful fetch",
			jobID: "1",
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("GetJobByID", mock.Anything, uint(1)).Return(validJobResponse, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":1,"queue":"email","payload":{"email":"test@example.com","subject":"Test"},"status":"queued","attempts":0,"max_retries":3,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:           "invalid ID param",
			jobID:          "abc",
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid ID"}`,
		},
		{
			name:  "job not found",
			jobID: "99",
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("GetJobByID", mock.Anything, uint(99)).Return(nil, common.Errf(http.StatusNotFound, "job not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Job not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.JobServiceMock)
			tt.setupMock(mockService)

			r := gin.New()
			handler := NewJobHandler(mockService)
			r.GET("/jobs/:id", handler.Get)

			req := httptest.NewRequest(http.MethodGet, "/jobs/"+tt.jobID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}

func TestJobHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		jobID          string
		body           string
		setupMock      func(*mocks.JobServiceMock)
		expectedStatus int
	}{
		{
			name:  "successful update",
			jobID: "1",
			body:  `{"status":"running"}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("UpdateStatus", mock.Anything, uint(1), config.JobStatusRunning).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid ID",
			jobID:          "abc",
			body:           `{"status":"running"}`,
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "service error",
			jobID: "1",
			body:  `{"status":"running"}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("UpdateStatus", mock.Anything, uint(1), config.JobStatusRunning).
					Return(common.Errf(http.StatusInternalServerError, "%s", string(config.JobStatusFailed)))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.JobServiceMock)
			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodPatch, "/jobs/"+tt.jobID, bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r := gin.New()
			r.Use(middleware.ErrorHandler())
			handler := NewJobHandler(mockService)
			r.PATCH("/jobs/:id", handler.Update)

			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestJobHandler_Increment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		jobID          string
		setupMock      func(*mocks.JobServiceMock)
		expectedStatus int
	}{
		{
			name:  "successful increment",
			jobID: "1",
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("IncrementAttempts", mock.Anything, uint(1)).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid ID",
			jobID:          "abc",
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "service error",
			jobID: "1",
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("IncrementAttempts", mock.Anything, uint(1)).
					Return(common.Errf(http.StatusInternalServerError, "%s", string(config.JobStatusFailed)))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.JobServiceMock)
			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodPatch, "/jobs/"+tt.jobID+"/increment", nil)
			w := httptest.NewRecorder()

			r := gin.New()
			r.Use(middleware.ErrorHandler())
			handler := NewJobHandler(mockService)
			r.PATCH("/jobs/:id/increment", handler.Increment)

			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestJobHandler_Save(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		jobID          string
		body           string
		setupMock      func(*mocks.JobServiceMock)
		expectedStatus int
	}{
		{
			name:  "successful save",
			jobID: "1",
			body:  `{"result":{"ok":true},"error":""}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("SaveResult", mock.Anything, uint(1), mock.Anything, "").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid ID",
			jobID:          "abc",
			body:           `{"result":{"ok":true},"error":""}`,
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "service error",
			jobID: "1",
			body:  `{"result":{"ok":true},"error":""}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("SaveResult", mock.Anything, uint(1), mock.Anything, "").
					Return(common.Errf(http.StatusInternalServerError, "%s", string(config.JobStatusFailed)))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.JobServiceMock)
			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodPatch, "/jobs/"+tt.jobID+"/save", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r := gin.New()
			r.Use(middleware.ErrorHandler())
			handler := NewJobHandler(mockService)
			r.PATCH("/jobs/:id/save", handler.Save)

			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestJobHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedDTOs := []dto.JobResponseDTO{
		{
			ID: 1, Queue: "default",  Status: config.JobStatusQueued,
			Payload:    json.RawMessage(`{}`),
			Attempts:   0,
			MaxRetries: 0,
			CreatedAt:  time.Time{},
			UpdatedAt:  time.Time{},
		},
		{
			ID: 2, Queue: "default",  Status: config.JobStatusQueued,
			Payload:    json.RawMessage(`{}`),
			Attempts:   0,
			MaxRetries: 0,
			CreatedAt:  time.Time{},
			UpdatedAt:  time.Time{},
		},
	}

	tests := []struct {
		name           string
		queueParam     string
		setupMock      func(*mocks.JobServiceMock)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing queue param",
			queueParam:     "",
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: 400,
			expectedBody:   `{"error":"queue parameter is required"}`,
		},
		{
			name:       "service error",
			queueParam: "default",
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("ListJobs", mock.Anything, "default").
					Return(nil, common.Errf(500, "failed to list jobs"))
			},
			expectedStatus: 500,
			expectedBody:   `{"error":"failed to list jobs"}`,
		},
		{
			name:       "success",
			queueParam: "default",
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("ListJobs", mock.Anything, "default").Return(expectedDTOs, nil)
			},
			expectedStatus: 200,
			expectedBody: `[
				{"id":1,"queue":"default","status":"queued","payload":{},"attempts":0,"max_retries":0,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"},
				{"id":2,"queue":"default","status":"queued","payload":{},"attempts":0,"max_retries":0,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}
			]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.JobServiceMock)
			tt.setupMock(mockService)

			r := gin.New()
			handler := NewJobHandler(mockService)
			r.GET("/jobs", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/jobs?queue="+tt.queueParam, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}
