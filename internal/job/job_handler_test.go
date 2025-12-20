package job

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/common"
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
			body: `{"queue":"default","type":"send_email","payload":{"email":"test@example.com","subject":"Test"},"maxRetries":3}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "job creation with custom max retries",
			body: `{"queue":"email","type":"process_payment","payload":{"amount":100},"maxRetries":5}`,
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
			name:           "missing required field - queue",
			body:           `{"type":"send_email","payload":{"test":true}}`,
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing required field - type",
			body:           `{"queue":"default","payload":{"test":true}}`,
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing required field - payload",
			body:           `{"queue":"default","type":"send_email"}`,
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty request body",
			body:           "",
			setupMock:      func(m *mocks.JobServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},

		{
			name: "invalid JSON payload",
			body: `{"queue":"default","type":"send_email","payload":"{invalid}"}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusBadRequest, "payload must be valid JSON"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid queue",
			body: `{"queue":"invalid_queue","type":"send_email","payload":{"test":true}}`,
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
			name: "invalid job type",
			body: `{"queue":"default","type":"invalid_type","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.NewAPIError(http.StatusBadRequest, "invalid job type", map[string]any{
						"provided": "invalid_type",
						"allowed":  []string{"send_email", "process_payment", "generate_report", "send_webhook"},
					}))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty queue",
			body: `{"queue":"","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty job type",
			body: `{"queue":"default","type":"","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
			},
			expectedStatus: http.StatusBadRequest,
		},

		{
			name: "database connection error",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusInternalServerError, "failed to add job to database: database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "database constraint violation",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusInternalServerError, "failed to add job to database: unique constraint violation"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "context deadline exceeded",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusInternalServerError, "failed to add job to database: context deadline exceeded"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "invalid queue with detailed error info",
			body: `{"queue":"bad_queue","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.NewAPIError(http.StatusBadRequest, "queue validation failed", map[string]any{
						"provided": "bad_queue",
						"allowed":  []string{"default", "email", "reports"},
						"reason":   "queue does not exist",
					}))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "concurrent job creation limit exceeded",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.NewAPIError(http.StatusTooManyRequests, "rate limit exceeded", map[string]any{
						"retryAfter": 60,
						"limit":      100,
					}))
			},
			expectedStatus: http.StatusTooManyRequests,
		},

		// Context-related test cases
		{
			name: "context canceled before service call",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusRequestTimeout, "request was canceled"))
			},
			setupContext: func(c *gin.Context) {
				ctx, cancel := context.WithCancel(c.Request.Context())
				cancel()
				c.Request = c.Request.WithContext(ctx)
			},
			expectedStatus: http.StatusRequestTimeout,
		},
		{
			name: "context deadline exceeded in service layer",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusRequestTimeout, "request timeout"))
			},
			setupContext: func(c *gin.Context) {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(2 * time.Millisecond)
				c.Request = c.Request.WithContext(ctx)
			},
			expectedStatus: http.StatusRequestTimeout,
		},
		{
			name: "context timeout with valid job data",
			body: `{"queue":"default","type":"send_email","payload":{"email":"test@example.com"},"maxRetries":3}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						time.Sleep(10 * time.Millisecond)
					}).
					Return(common.Errf(http.StatusRequestTimeout, "request timeout"))
			},
			setupContext: func(c *gin.Context) {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Millisecond)
				defer cancel()
				c.Request = c.Request.WithContext(ctx)
			},
			expectedStatus: http.StatusRequestTimeout,
		},
		{
			name: "context canceled after validation but before database insert",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusRequestTimeout, "request was canceled"))
			},
			expectedStatus: http.StatusRequestTimeout,
		},
		{
			name: "parent context canceled propagates to service",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusRequestTimeout, "request was canceled"))
			},
			setupContext: func(c *gin.Context) {
				parentCtx, parentCancel := context.WithCancel(context.Background())
				childCtx, childCancel := context.WithCancel(parentCtx)
				defer childCancel()
				parentCancel()
				c.Request = c.Request.WithContext(childCtx)
			},
			expectedStatus: http.StatusRequestTimeout,
		},
		{
			name: "context with very short deadline",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusRequestTimeout, "request timeout"))
			},
			setupContext: func(c *gin.Context) {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Microsecond)
				defer cancel()
				time.Sleep(1 * time.Millisecond)
				c.Request = c.Request.WithContext(ctx)
			},
			expectedStatus: http.StatusRequestTimeout,
		},
		{
			name: "context error with generic failure",
			body: `{"queue":"default","type":"send_email","payload":{"test":true}}`,
			setupMock: func(m *mocks.JobServiceMock) {
				m.On("CreateJob", mock.Anything, mock.Anything).
					Return(common.Errf(http.StatusInternalServerError, "request failed"))
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
