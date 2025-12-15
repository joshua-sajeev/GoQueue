package job

import (
	"context"
	"errors"
	"fmt"
	"testing"

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
		name        string
		dto         *dto.JobCreateDTO
		setupMock   func(*mocks.JobRepoMock)
		wantErr     bool
		errContains string
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
			wantErr: false,
		},

		{
			name: "invalid JSON payload",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: invalidPayload,
			},
			setupMock:   func(m *mocks.JobRepoMock) {},
			wantErr:     true,
			errContains: "payload must be valid JSON",
		},
		{
			name: "nil payload",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: nil,
			},
			setupMock:   func(m *mocks.JobRepoMock) {},
			wantErr:     true,
			errContains: "payload must be valid JSON",
		},
		{
			name: "empty byte slice payload",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "send_email",
				Payload: []byte{},
			},
			setupMock:   func(m *mocks.JobRepoMock) {},
			wantErr:     true,
			errContains: "payload must be valid JSON",
		},
		{
			name: "invalid queue",
			dto: &dto.JobCreateDTO{
				Queue:   "invalid_queue",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock:   func(m *mocks.JobRepoMock) {},
			wantErr:     true,
			errContains: "invalid queue",
		},
		{
			name: "empty queue",
			dto: &dto.JobCreateDTO{
				Queue:   "",
				Type:    "send_email",
				Payload: validPayload,
			},
			setupMock:   func(m *mocks.JobRepoMock) {},
			wantErr:     true,
			errContains: "invalid queue",
		},
		{
			name: "invalid job type",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "invalid_type",
				Payload: validPayload,
			},
			setupMock:   func(m *mocks.JobRepoMock) {},
			wantErr:     true,
			errContains: "invalid job type",
		},
		{
			name: "empty job type",
			dto: &dto.JobCreateDTO{
				Queue:   "default",
				Type:    "",
				Payload: validPayload,
			},
			setupMock:   func(m *mocks.JobRepoMock) {},
			wantErr:     true,
			errContains: "invalid job type",
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
			wantErr:     true,
			errContains: "create job",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.JobRepoMock)
			tt.setupMock(mockRepo)

			s := NewJobService(mockRepo)
			err := s.CreateJob(context.Background(), tt.dto)

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
