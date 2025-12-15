package mocks

import (
	"context"

	"github.com/joshu-sajeev/goqueue/internal/models"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"
)

type JobRepoMock struct {
	mock.Mock
}

func (m *JobRepoMock) Create(ctx context.Context, job *models.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *JobRepoMock) Get(ctx context.Context, id string) (*models.Job, error) {
	args := m.Called(ctx, id)

	job, _ := args.Get(0).(*models.Job)
	return job, args.Error(1)
}

func (m *JobRepoMock) UpdateStatus(ctx context.Context, id string, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *JobRepoMock) IncrementAttempts(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *JobRepoMock) SaveResult(
	ctx context.Context,
	id string,
	result datatypes.JSON,
	errMsg string,
) error {
	args := m.Called(ctx, id, result, errMsg)
	return args.Error(0)
}

func (m *JobRepoMock) List(ctx context.Context, queue string) ([]models.Job, error) {
	args := m.Called(ctx, queue)

	jobs, _ := args.Get(0).([]models.Job)
	return jobs, args.Error(1)
}
