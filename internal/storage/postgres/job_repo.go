package postgres

import (
	"context"
	"fmt"

	"github.com/joshu-sajeev/goqueue/internal/job"
	"github.com/joshu-sajeev/goqueue/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

var _ job.JobRepoInterface = (*JobRepository)(nil)

// Create inserts a new job record into the database using the
// provided context for cancellation and timeout propagation.
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
	if err := r.db.WithContext(ctx).Create(job).Error; err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

// TODO:
func (r *JobRepository) Get(ctx context.Context, id string) (*models.Job, error) {
	return nil, nil
}

// TODO:
func (r *JobRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return nil
}

// TODO:
func (r *JobRepository) IncrementAttempts(ctx context.Context, id string) error {
	return nil
}

// TODO:
func (r *JobRepository) SaveResult(ctx context.Context, id string, result datatypes.JSON, err string) error {
	return nil
}

// TODO:
func (r *JobRepository) List(ctx context.Context, queue string) ([]models.Job, error) {
	return nil, nil
}
