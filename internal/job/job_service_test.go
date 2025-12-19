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
