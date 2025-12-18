package postgres

import (
	"context"
	"errors"
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

// Create inserts a new job record into the database. It uses the provided
// context for cancellation and timeout propagation. Returns an error if the
// database operation fails.
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
	if err := r.db.WithContext(ctx).Create(job).Error; err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

// Get retrieves a single job record by its ID. Returns the job if found,
// or an error if the job doesn't exist or the database query fails.
func (r *JobRepository) Get(ctx context.Context, id uint) (*models.Job, error) {
	var job models.Job
	if err := r.db.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("job not found: %w", err)
		}
		return nil, fmt.Errorf("get job: %w", err)
	}
	return &job, nil
}

// UpdateStatus updates the status field of a job identified by id.
// Common statuses include "pending", "processing", "completed", and "failed".
// Returns an error if the database operation fails.
func (r *JobRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	if err := r.db.WithContext(ctx).Model(&models.Job{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// IncrementAttempts increments the attempts counter for a job by one.
// Uses gorm.Expr to safely increment atomically at the database level,
// preventing race conditions in concurrent environments. Returns an error
// if the database operation fails.
func (r *JobRepository) IncrementAttempts(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Model(&models.Job{}).
		Where("id = ?", id).
		Update("attempts", gorm.Expr("attempts + ?", 1)).Error; err != nil {
		return fmt.Errorf("increment attempts: %w", err)
	}
	return nil
}

// SaveResult persists the result and error message for a completed job.
// Both fields are updated atomically in a single operation. Use this to
// store job execution results after the job has finished processing.
// Returns an error if the database operation fails.
func (r *JobRepository) SaveResult(ctx context.Context, id uint, result datatypes.JSON, errMsg string) error {
	if err := r.db.WithContext(ctx).Model(&models.Job{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"result": result,
			"error":  errMsg,
		}).Error; err != nil {
		return fmt.Errorf("save result: %w", err)
	}
	return nil
}

// List retrieves all jobs belonging to a specific queue. Useful for
// fetching pending or processing jobs for a job worker. Returns a slice
// of jobs or an error if the database query fails.
func (r *JobRepository) List(ctx context.Context, queue string) ([]models.Job, error) {
	var jobs []models.Job
	if err := r.db.WithContext(ctx).
		Where("queue = ?", queue).
		Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	return jobs, nil
}
