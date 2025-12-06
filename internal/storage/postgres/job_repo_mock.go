package postgres

import (
	"context"

	"github.com/joshu-sajeev/goqueue/internal/models"
	"github.com/stretchr/testify/mock"
)

type JobRepoMock struct {
	mock.Mock
}

func (m *JobRepoMock) CreateJob(ctx context.Context, job *models.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}
