package mocks

import (
	"context"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/config"
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

func (m *JobRepoMock) Get(ctx context.Context, id uint) (*models.Job, error) {
	args := m.Called(ctx, id)

	job, _ := args.Get(0).(*models.Job)
	return job, args.Error(1)
}

func (m *JobRepoMock) UpdateStatus(ctx context.Context, id uint, status config.JobStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *JobRepoMock) IncrementAttempts(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *JobRepoMock) SaveResult(
	ctx context.Context,
	id uint,
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

func (m *JobRepoMock) AcquireNext(ctx context.Context, queue string, workerID uint, lockDuration time.Duration) (*models.Job, error) {
	args := m.Called(ctx, queue, workerID, lockDuration)

	job, _ := args.Get(0).(*models.Job)
	return job, args.Error(1)
}

func (m *JobRepoMock) Release(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *JobRepoMock) RetryLater(ctx context.Context, id uint, availableAt time.Time) error {
	args := m.Called(ctx, id, availableAt)
	return args.Error(0)
}

func (m *JobRepoMock) ListStuckJobs(ctx context.Context, staleDuration time.Duration) ([]models.Job, error) {
	args := m.Called(ctx, staleDuration)

	jobs, _ := args.Get(0).([]models.Job)
	return jobs, args.Error(1)
}
