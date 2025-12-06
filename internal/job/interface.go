package job

import (
	"context"

	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/models"
	"gorm.io/datatypes"
)

type JobRepoInterface interface {
	Create(ctx context.Context, job *models.Job) error
	Get(ctx context.Context, id string) (*models.Job, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	IncrementAttempts(ctx context.Context, id string) error
	SaveResult(ctx context.Context, id string, result datatypes.JSON, err string) error
	List(ctx context.Context, queue string) ([]models.Job, error)
}

type JobServiceInterface interface {
	CreateJob(ctx context.Context, dto *dto.JobCreateDTO) error
	GetJobByID(id string) (*models.Job, error)
	UpdateStatus(id string, status string) error
	IncrementAttempts(id string) error
	SaveResult(id string, result datatypes.JSON, err string) error
	ListJobs(queue string) ([]models.Job, error)
}
