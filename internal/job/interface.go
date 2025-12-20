package job

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/models"
	"gorm.io/datatypes"
)

// JobRepoInterface defines the contract for job repository operations.
type JobRepoInterface interface {
	Create(ctx context.Context, job *models.Job) error
	Get(ctx context.Context, id uint) (*models.Job, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	IncrementAttempts(ctx context.Context, id uint) error
	SaveResult(ctx context.Context, id uint, result datatypes.JSON, err string) error
	List(ctx context.Context, queue string) ([]models.Job, error)
}

// JobServiceInterface defines the contract for job business logic operations.
type JobServiceInterface interface {
	CreateJob(ctx context.Context, dto *dto.JobCreateDTO) error
	GetJobByID(ctx context.Context, id uint) (*dto.JobResponseDTO, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	IncrementAttempts(ctx context.Context, id uint) error
	SaveResult(ctx context.Context, id uint, result datatypes.JSON, err string) error
	ListJobs(ctx context.Context, queue string) ([]dto.JobResponseDTO, error)
}

// JobHandlerInterface defines the contract for HTTP request handlers.
type JobHandlerInterface interface {
	Create(c *gin.Context)
	Get(c *gin.Context)
	Update(c *gin.Context)
	Increment(c *gin.Context)
	Save(c *gin.Context)
	List(c *gin.Context)
}
