package mocks

import (
	"context"

	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"
)

type JobServiceMock struct {
	mock.Mock
}

func (m *JobServiceMock) CreateJob(ctx context.Context, dto *dto.JobCreateDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}

func (m *JobServiceMock) GetJobByID(ctx context.Context, id uint) (*dto.JobResponseDTO, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.JobResponseDTO), args.Error(1)
}

func (m *JobServiceMock) UpdateStatus(ctx context.Context, id uint, status config.JobStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *JobServiceMock) IncrementAttempts(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *JobServiceMock) SaveResult(ctx context.Context, id uint, result datatypes.JSON, err string) error {
	args := m.Called(ctx, id, result, err)
	return args.Error(0)
}

func (m *JobServiceMock) ListJobs(ctx context.Context, queue string) ([]dto.JobResponseDTO, error) {
	args := m.Called(ctx, queue)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.JobResponseDTO), args.Error(1)
}
