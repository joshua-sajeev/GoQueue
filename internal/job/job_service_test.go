package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/mocks"
	"github.com/joshu-sajeev/goqueue/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestJobService_CreateJob(t *testing.T) {
	validPayload := []byte(`{
    "to": "test@example.com",
    "subject": "Test Subject",
    "body": "This is a test email body."
}`)

	validWebhookPayload := []byte(`{
    "url": "https://example.com/webhook",
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": {"message": "test"},
    "timeout": 10
}`)

	validPaymentPayload := []byte(`{
    "payment_id": "pay_123",
    "user_id": "user_456",
    "amount": 100.50,
    "currency": "USD",
    "method": "card"
}`)
	invalidPayload := []byte(`{invalid json}`)

	tests := []struct {
		name         string
		dto          *dto.JobCreateDTO
		setupMock    func(*mocks.JobRepoMock)
		setupCtx     func() context.Context
		wantErr      bool
		errContains  string
		skipRepoCall bool
	}{
		{
			name: "successful job creation with default max retries",
			dto: &dto.JobCreateDTO{
				Queue:      "default",
				Type:       "send_email",
				Payload:    validPayload,
				MaxRetries: 0,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(job *models.Job) bool {
					return job.Queue == "default" &&
						job.Type == "send_email" &&
						job.MaxRetries == 3 &&
						job.Attempts == 0
				})).Return(nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name: "successful job creation with custom max retries",
			dto: &dto.JobCreateDTO{
				Queue:      "email",
				Type:       "send_email",
				Payload:    validPayload,
				MaxRetries: 5,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(job *models.Job) bool {
					return job.Queue == "email" &&
						job.Type == "send_email" &&
						job.MaxRetries == 5
				})).Return(nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name: "invalid JSON payload",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: invalidPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:      true,
			errContains:  "payload must be valid JSON",
			skipRepoCall: true,
		},
		{
			name: "invalid queue",
			dto: &dto.JobCreateDTO{
				Queue:   "invalid_queue",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:      true,
			errContains:  "invalid queue",
			skipRepoCall: true,
		},
		{
			name: "invalid job type",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "invalid_type",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:      true,
			errContains:  "invalid job type",
			skipRepoCall: true,
		},
		{
			name: "empty job type",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:      true,
			errContains:  "invalid job type",
			skipRepoCall: true,
		},
		{
			name: "valid queue - webhooks",
			dto: &dto.JobCreateDTO{
				Queue:   "webhooks",
				Type:    "send_webhook",
				Payload: validWebhookPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name: "valid job type - process_payment",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "process_payment",
				Payload: validPaymentPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name: "empty JSON object payload",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: []byte(`{}`),
			},
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "payload validation failed",
		},
		{
			name: "JSON primitive payloads",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: []byte(`123`),
			},
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "invalid payload format",
		},
		{
			name: "JSON with special characters",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: []byte(`{"message":"Hello \"World\"\nNew line\tTab"}`),
			},
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "payload validation failed",
		},
		{
			name: "max retries set to 1",
			dto: &dto.JobCreateDTO{
				Queue:      "default",
				Type:       "send_email",
				Payload:    validPayload,
				MaxRetries: 1,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(job *models.Job) bool {
					return job.MaxRetries == 1
				})).Return(nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name: "max retries set to high number",
			dto: &dto.JobCreateDTO{
				Queue:      "default",
				Type:       "send_email",
				Payload:    validPayload,
				MaxRetries: 100,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(job *models.Job) bool {
					return job.MaxRetries == 100
				})).Return(nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name: "repository error - database failure",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.Anything).
					Return(errors.New("database connection failed"))
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "failed to add job to database",
		},
		// Context-specific tests
		{
			name: "context canceled - repo returns cancellation error",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Job")).
					Return(context.Canceled)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "request was canceled",
		},
		{
			name: "context deadline exceeded - repo returns deadline error",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Job")).
					Return(context.DeadlineExceeded)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "request timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.JobRepoMock)
			tt.setupMock(mockRepo)

			s := NewJobService(mockRepo)
			ctx := tt.setupCtx()
			err := s.CreateJob(ctx, tt.dto)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)

			if tt.skipRepoCall {
				mockRepo.AssertNumberOfCalls(t, "Create", 0)
			}
		})
	}
}

func TestJobService_GetJobByID(t *testing.T) {
	validJob := &dto.JobResponseDTO{
		ID:         1,
		Queue:      "email",
		Type:       "send_email",
		Status:     config.JobStatusQueued,
		Attempts:   0,
		MaxRetries: 3,
		Payload:    json.RawMessage(`{"to":"test@example.com","subject":"Test","body":"Hello"}`),
	}

	tests := []struct {
		name         string
		jobID        uint
		setupMock    func(*mocks.JobRepoMock)
		setupCtx     func() context.Context
		wantErr      bool
		errContains  string
		wantJob      *dto.JobResponseDTO
		skipRepoCall bool
	}{
		{
			name:  "successful job fetch",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(&models.Job{
						ID:         1,
						Queue:      "email",
						Type:       "send_email",
						Status:     config.JobStatusQueued,
						Attempts:   0,
						MaxRetries: 3,
						Payload:    []byte(`{"to":"test@example.com","subject":"Test","body":"Hello"}`),
					}, nil)
			},
			setupCtx: func() context.Context { return context.Background() },
			wantJob:  validJob,
			wantErr:  false,
		},
		{
			name:  "job not found - gorm error",
			jobID: 99,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(99)).
					Return(nil, gorm.ErrRecordNotFound)
			},
			setupCtx:    func() context.Context { return context.Background() },
			wantErr:     true,
			errContains: "job not found",
		},
		{
			name:  "repository error - generic failure",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(nil, errors.New("database unavailable"))
			},
			setupCtx:    func() context.Context { return context.Background() },
			wantErr:     true,
			errContains: "failed to get job",
		},
		{
			name:  "context canceled - repo returns canceled",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(nil, context.Canceled)
			},
			setupCtx:    func() context.Context { return context.Background() },
			wantErr:     true,
			errContains: "request timed out",
		},
		{
			name:  "context with sufficient timeout - successful",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(&models.Job{
						ID:         1,
						Queue:      "email",
						Type:       "send_email",
						Status:     config.JobStatusQueued,
						Attempts:   0,
						MaxRetries: 3,
						Payload:    []byte(`{"to":"test@example.com","subject":"Test","body":"Hello"}`),
					}, nil)
			},
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				t.Cleanup(cancel)
				return ctx
			},
			wantJob: validJob,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.JobRepoMock)
			tt.setupMock(mockRepo)

			s := NewJobService(mockRepo)
			ctx := tt.setupCtx()

			job, err := s.GetJobByID(ctx, tt.jobID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.NotNil(t, job)
				assert.Equal(t, uint(0), job.ID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantJob, job)
			}

			mockRepo.AssertExpectations(t)

			if tt.skipRepoCall {
				mockRepo.AssertNumberOfCalls(t, "Get", 0)
			}
		})
	}
}

func TestJobService_UpdateStatus(t *testing.T) {
	tests := []struct {
		name        string
		jobID       uint
		status      config.JobStatus
		setupMock   func(*mocks.JobRepoMock)
		setupCtx    func() context.Context
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful status update",
			jobID:  1,
			status: config.JobStatusRunning,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("UpdateStatus", mock.Anything, uint(1), config.JobStatusRunning).Return(nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name:   "repository returns internal error",
			jobID:  2,
			status: config.JobStatusCompleted,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("UpdateStatus", mock.Anything, uint(2), config.JobStatusCompleted).
					Return(fmt.Errorf("db failure"))
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "failed to update job status",
		},
		{
			name:      "context canceled before repo call",
			jobID:     3,
			status:    config.JobStatusFailed,
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantErr:     true,
			errContains: "request timed out",
		},
		{
			name:      "context deadline exceeded before repo call",
			jobID:     4,
			status:    config.JobStatusQueued,
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(10 * time.Millisecond)
				return ctx
			},
			wantErr:     true,
			errContains: "request timed out",
		},
		{
			name:   "repository returns context canceled",
			jobID:  5,
			status: config.JobStatusRunning,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("UpdateStatus", mock.Anything, uint(5), config.JobStatusRunning).
					Return(context.Canceled)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "request timed out",
		},
		{
			name:   "repository returns context deadline exceeded",
			jobID:  6,
			status: config.JobStatusCompleted,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("UpdateStatus", mock.Anything, uint(6), config.JobStatusCompleted).
					Return(context.DeadlineExceeded)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "request timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.JobRepoMock)
			tt.setupMock(mockRepo)

			s := NewJobService(mockRepo)
			ctx := tt.setupCtx()
			err := s.UpdateStatus(ctx, tt.jobID, tt.status)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestJobService_IncrementAttempts(t *testing.T) {
	tests := []struct {
		name        string
		jobID       uint
		setupMock   func(*mocks.JobRepoMock)
		setupCtx    func() context.Context
		wantErr     bool
		errContains string
	}{
		{
			name:        "context canceled",
			jobID:       1,
			setupMock:   func(m *mocks.JobRepoMock) {},
			setupCtx:    func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx },
			wantErr:     true,
			errContains: "request timed out",
		},
		{
			name:  "invalid job ID 0",
			jobID: 0,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("IncrementAttempts", mock.Anything, uint(0)).
					Return(errors.New("invalid job ID"))
			},
			setupCtx:    func() context.Context { return context.Background() },
			wantErr:     true,
			errContains: "failed to increment job attempts",
		},
		{
			name:  "repository error",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("IncrementAttempts", mock.Anything, uint(1)).
					Return(errors.New("db failure"))
			},
			setupCtx:    func() context.Context { return context.Background() },
			wantErr:     true,
			errContains: "failed to increment job attempts",
		},
		{
			name:  "success",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("IncrementAttempts", mock.Anything, uint(1)).Return(nil)
			},
			setupCtx: func() context.Context { return context.Background() },
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.JobRepoMock)
			tt.setupMock(mockRepo)
			s := NewJobService(mockRepo)
			err := s.IncrementAttempts(tt.setupCtx(), tt.jobID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestJobService_SaveResult(t *testing.T) {
	emptyResult := datatypes.JSON([]byte(`{}`))
	validResult := datatypes.JSON([]byte(`{"success":true}`))

	tests := []struct {
		name        string
		jobID       uint
		result      datatypes.JSON
		errMsg      string
		setupMock   func(*mocks.JobRepoMock)
		setupCtx    func() context.Context
		wantErr     bool
		errContains string
	}{
		{
			name:      "context deadline exceeded",
			jobID:     1,
			result:    validResult,
			errMsg:    "",
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(10 * time.Millisecond)
				return ctx
			},
			wantErr:     true,
			errContains: "request timed out",
		},
		{
			name:   "repository error",
			jobID:  1,
			result: validResult,
			errMsg: "error occurred",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("SaveResult", mock.Anything, uint(1), validResult, "error occurred").
					Return(errors.New("db failure"))
			},
			setupCtx:    func() context.Context { return context.Background() },
			wantErr:     true,
			errContains: "failed to save job result",
		},
		{
			name:   "nil/empty result",
			jobID:  2,
			result: emptyResult,
			errMsg: "",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("SaveResult", mock.Anything, uint(2), emptyResult, "").Return(nil)
			},
			setupCtx: func() context.Context { return context.Background() },
			wantErr:  false,
		},
		{
			name:   "empty error string",
			jobID:  3,
			result: validResult,
			errMsg: "",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("SaveResult", mock.Anything, uint(3), validResult, "").Return(nil)
			},
			setupCtx: func() context.Context { return context.Background() },
			wantErr:  false,
		},
		{
			name:   "success",
			jobID:  1,
			result: validResult,
			errMsg: "",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("SaveResult", mock.Anything, uint(1), validResult, "").Return(nil)
			},
			setupCtx: func() context.Context { return context.Background() },
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.JobRepoMock)
			tt.setupMock(mockRepo)
			s := NewJobService(mockRepo)
			err := s.SaveResult(tt.setupCtx(), tt.jobID, tt.result, tt.errMsg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestJobService_ListJobs(t *testing.T) {
	jobs := []models.Job{
		{ID: 1, Queue: "default", Type: "send_email", Status: config.JobStatusQueued},
		{ID: 2, Queue: "default", Type: "process_payment", Status: config.JobStatusQueued},
	}

	expectedDTOs := []dto.JobResponseDTO{
		{ID: 1, Queue: "default", Type: "send_email", Status: config.JobStatusQueued},
		{ID: 2, Queue: "default", Type: "process_payment", Status: config.JobStatusQueued},
	}

	tests := []struct {
		name        string
		queue       string
		setupMock   func(*mocks.JobRepoMock)
		setupCtx    func() context.Context
		wantErr     bool
		errContains string
		wantJobs    []dto.JobResponseDTO
	}{
		{
			name:      "context canceled",
			queue:     "default",
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantErr:     true,
			errContains: "request timed out",
		},
		{
			name:  "repository error",
			queue: "default",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("List", mock.Anything, "default").
					Return(nil, errors.New("db failure"))
			},
			setupCtx:    func() context.Context { return context.Background() },
			wantErr:     true,
			errContains: "failed to list jobs",
		},
		{
			name:  "empty queue",
			queue: "",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("List", mock.Anything, "").Return([]models.Job{}, nil)
			},
			setupCtx: func() context.Context { return context.Background() },
			wantErr:  false,
			wantJobs: []dto.JobResponseDTO{},
		},
		{
			name:  "success",
			queue: "default",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("List", mock.Anything, "default").Return(jobs, nil)
			},
			setupCtx: func() context.Context { return context.Background() },
			wantErr:  false,
			wantJobs: expectedDTOs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.JobRepoMock)
			tt.setupMock(mockRepo)
			s := NewJobService(mockRepo)
			got, err := s.ListJobs(tt.setupCtx(), tt.queue)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantJobs, got)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
