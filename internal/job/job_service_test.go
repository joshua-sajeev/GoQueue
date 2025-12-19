package job

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/mocks"
	"github.com/joshu-sajeev/goqueue/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestJobService_CreateJob(t *testing.T) {
	validPayload := []byte(`{"email": "test@example.com", "subject": "Test"}`)
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
						job.Status == "pending" &&
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
						job.MaxRetries == 5 &&
						job.Status == "pending"
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
			name: "nil payload",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: nil,
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
			name: "empty byte slice payload",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: []byte{},
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
			name: "empty queue",
			dto: &dto.JobCreateDTO{
				Queue:   "",
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
			name: "valid queue - reports",
			dto: &dto.JobCreateDTO{
				Queue:   "reports",
				Type:    "generate_report",
				Payload: validPayload,
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
			name: "valid queue - webhooks",
			dto: &dto.JobCreateDTO{
				Queue:   "webhooks",
				Type:    "send_webhook",
				Payload: validPayload,
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
				Payload: validPayload,
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
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name: "complex nested JSON payload",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: []byte(`{"user":{"id":123,"details":{"email":"test@test.com","preferences":["opt1","opt2"]}}}`),
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
			name: "JSON primitive payloads",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: []byte(`123`),
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
			name: "JSON with special characters",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: []byte(`{"message":"Hello \"World\"\nNew line\tTab"}`),
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
		{
			name: "repository error - constraint violation",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.Anything).
					Return(errors.New("unique constraint violation"))
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "failed to add job to database",
		},
		{
			name: "repository error - context timeout",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.Anything).
					Return(errors.New("context deadline exceeded"))
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "failed to add job to database",
		},
		{
			name: "repository error - wrapped error",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.Anything).
					Return(fmt.Errorf("create job: %w", errors.New("connection refused")))
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "create job",
		},
		{
			name: "repository error - GORM error",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.Anything).
					Return(fmt.Errorf("create job: %w", gorm.ErrRecordNotFound))
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "create job",
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
		{
			name: "context propagation - context with value",
			dto: &dto.JobCreateDTO{
				Queue:      "default",
				Type:       "send_email",
				Payload:    validPayload,
				MaxRetries: 3,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Job")).
					Return(nil).
					Run(func(args mock.Arguments) {
						receivedCtx := args.Get(0).(context.Context)
						assert.Equal(t, "test-123", receivedCtx.Value("request_id"))
					})
			},
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), "request_id", "test-123")
			},
			wantErr: false,
		},
		{
			name: "context timeout before repo call",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
			},
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(10 * time.Millisecond)
				return ctx
			},
			wantErr:      true,
			errContains:  "request",
			skipRepoCall: true,
		},
		{
			name: "context with sufficient timeout - successful",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Job")).
					Return(nil)
			},
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				t.Cleanup(cancel)
				return ctx
			},
			wantErr: false,
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
	validJob := &models.Job{
		ID:         1,
		Queue:      "email",
		Type:       "send_email",
		Status:     "pending",
		Attempts:   0,
		MaxRetries: 3,
	}

	tests := []struct {
		name         string
		jobID        uint
		setupMock    func(*mocks.JobRepoMock)
		setupCtx     func() context.Context
		wantErr      bool
		errContains  string
		wantJob      *models.Job
		skipRepoCall bool
	}{
		{
			name:  "successful job fetch",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(validJob, nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantJob: validJob,
			wantErr: false,
		},
		{
			name:  "job not found - gorm error",
			jobID: 99,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(99)).
					Return(nil, gorm.ErrRecordNotFound)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "job not found",
		},
		{
			name:  "job not found - wrapped error",
			jobID: 99,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(99)).
					Return(nil, fmt.Errorf("job not found: %w", gorm.ErrRecordNotFound))
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
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
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "failed to get job",
		},
		{
			name:  "repository error - wrapped internal error",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(nil, fmt.Errorf("get job: %w", errors.New("connection refused")))
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "failed to get job",
		},

		// Context-related tests
		{
			name:  "context canceled - repo returns canceled",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(nil, context.Canceled)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "request timed out",
		},
		{
			name:  "context deadline exceeded - repo returns deadline error",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(nil, context.DeadlineExceeded)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "request timed out",
		},
		{
			name:      "context timeout before repo call",
			jobID:     1,
			setupMock: func(m *mocks.JobRepoMock) {},
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(10 * time.Millisecond)
				return ctx
			},
			wantErr:      true,
			errContains:  "request timed out",
			skipRepoCall: true,
		},
		{
			name:  "context propagation - context with value",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(validJob, nil).
					Run(func(args mock.Arguments) {
						receivedCtx := args.Get(0).(context.Context)
						assert.Equal(t, "req-123", receivedCtx.Value("request_id"))
					})
			},
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), "request_id", "req-123")
			},
			wantJob: validJob,
			wantErr: false,
		},
		{
			name:  "context with sufficient timeout - successful",
			jobID: 1,
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("Get", mock.Anything, uint(1)).
					Return(validJob, nil)
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
				assert.Nil(t, job)
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
		status      string
		setupMock   func(*mocks.JobRepoMock)
		setupCtx    func() context.Context
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful status update",
			jobID:  1,
			status: "processing",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("UpdateStatus", mock.Anything, uint(1), "processing").Return(nil)
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name:   "repository returns internal error",
			jobID:  2,
			status: "completed",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("UpdateStatus", mock.Anything, uint(2), "completed").
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
			status:    "failed",
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
			status:    "pending",
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
			status: "processing",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("UpdateStatus", mock.Anything, uint(5), "processing").
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
			status: "completed",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("UpdateStatus", mock.Anything, uint(6), "completed").
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
		{ID: 1, Queue: "default"},
		{ID: 2, Queue: "default"},
	}

	tests := []struct {
		name        string
		queue       string
		setupMock   func(*mocks.JobRepoMock)
		setupCtx    func() context.Context
		wantErr     bool
		errContains string
		wantJobs    []models.Job
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
			wantJobs: []models.Job{},
		},
		{
			name:  "success",
			queue: "default",
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("List", mock.Anything, "default").Return(jobs, nil)
			},
			setupCtx: func() context.Context { return context.Background() },
			wantErr:  false,
			wantJobs: jobs,
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
