package job

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/joshu-sajeev/goqueue/common"
	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type JobService struct {
	repo JobRepoInterface
}

func NewJobService(repo JobRepoInterface) *JobService {
	return &JobService{repo: repo}
}

var _ JobServiceInterface = (*JobService)(nil)

// CreateJob validates job creation input, applies business rules,
// constructs a Job model, and persists it using the repository.
// It returns a typed API error for validation failures and an
// internal error for persistence failures.
func (s *JobService) CreateJob(ctx context.Context, dto *dto.JobCreateDTO) error {

	if err := ctx.Err(); err != nil {
		switch err {
		case context.Canceled:
			return common.Errf(http.StatusRequestTimeout, "request was canceled")
		case context.DeadlineExceeded:
			return common.Errf(http.StatusRequestTimeout, "request timeout")
		default:
			return common.Errf(http.StatusInternalServerError, "request failed")
		}
	}

	if !json.Valid(dto.Payload) {
		return common.Errf(http.StatusBadRequest, "payload must be valid JSON")
	}

	if !slices.Contains(config.AllowedQueues, dto.Queue) {
		return common.NewAPIError(
			http.StatusBadRequest,
			"invalid queue",
			map[string]any{
				"provided": dto.Queue,
				"allowed":  config.AllowedQueues,
			},
		)
	}

	if !slices.Contains(config.AllowedJobTypes, dto.Type) {
		return common.NewAPIError(
			http.StatusBadRequest,
			"invalid job type",
			map[string]any{
				"provided": dto.Type,
				"allowed":  config.AllowedJobTypes,
			},
		)
	}

	maxRetries := dto.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	job := models.Job{
		Queue:      dto.Queue,
		Type:       dto.Type,
		Payload:    datatypes.JSON(dto.Payload),
		Attempts:   0,
		MaxRetries: maxRetries,
		Status:     "pending",
	}

	if err := s.repo.Create(ctx, &job); err != nil {
		if errors.Is(err, context.Canceled) {
			return common.Errf(http.StatusRequestTimeout, "request was canceled")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return common.Errf(http.StatusRequestTimeout, "request timeout")
		}

		return common.Errf(
			http.StatusInternalServerError,
			"failed to add job to database: %v",
			err,
		)
	}

	return nil
}

// GetJobByID retrieves a job by its ID from the repository.
// It maps repository errors to appropriate API errors
// (e.g., not found, timeout, or internal failure).
func (s *JobService) GetJobByID(ctx context.Context, id uint) (*dto.JobResponseDTO, error) {
	if err := ctx.Err(); err != nil {
		return &dto.JobResponseDTO{}, common.Errf(
			http.StatusRequestTimeout,
			"request timed out",
		)
	}

	job, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			return &dto.JobResponseDTO{}, common.Errf(
				http.StatusRequestTimeout,
				"request timed out",
			)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) ||
			strings.Contains(err.Error(), "job not found") {
			return &dto.JobResponseDTO{}, common.Errf(
				http.StatusNotFound,
				"job not found",
			)
		}

		return &dto.JobResponseDTO{}, common.Errf(
			http.StatusInternalServerError,
			"failed to get job",
		)
	}

	return &dto.JobResponseDTO{
		ID:         job.ID,
		Queue:      job.Queue,
		Type:       job.Type,
		Payload:    json.RawMessage(job.Payload),
		Status:     job.Status,
		Attempts:   job.Attempts,
		MaxRetries: job.MaxRetries,
		Result:     json.RawMessage(job.Result),
		Error:      job.Error,
		CreatedAt:  job.CreatedAt,
		UpdatedAt:  job.UpdatedAt,
	}, nil
}

// UpdateStatus updates the status of a job identified by its ID.
// It validates request context, delegates the update to the repository,
// and maps repository or context errors to appropriate API errors
// (e.g., timeout or internal failure).
func (s *JobService) UpdateStatus(ctx context.Context, id uint, status string) error {
	if err := ctx.Err(); err != nil {
		return common.Errf(
			http.StatusRequestTimeout,
			"request timed out",
		)
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			return common.Errf(
				http.StatusRequestTimeout,
				"request timed out",
			)
		}

		return common.Errf(
			http.StatusInternalServerError,
			"failed to update job status",
		)
	}

	return nil
}

// IncrementAttempts increments the attempt counter for a job by one.
// It ensures request context validity before execution and maps
// repository or context errors to appropriate API errors.
func (s *JobService) IncrementAttempts(ctx context.Context, id uint) error {
	if err := ctx.Err(); err != nil {
		return common.Errf(
			http.StatusRequestTimeout,
			"request timed out",
		)
	}

	if err := s.repo.IncrementAttempts(ctx, id); err != nil {
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			return common.Errf(
				http.StatusRequestTimeout,
				"request timed out",
			)
		}

		return common.Errf(
			http.StatusInternalServerError,
			"failed to increment job attempts",
		)
	}

	return nil
}

// SaveResult persists the execution result and error message for a job.
// It validates request context, delegates persistence to the repository,
// and maps repository errors to appropriate API errors.
func (s *JobService) SaveResult(
	ctx context.Context,
	id uint,
	result datatypes.JSON,
	errMsg string,
) error {
	if err := ctx.Err(); err != nil {
		return common.Errf(
			http.StatusRequestTimeout,
			"request timed out",
		)
	}

	if err := s.repo.SaveResult(ctx, id, result, errMsg); err != nil {
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			return common.Errf(
				http.StatusRequestTimeout,
				"request timed out",
			)
		}

		return common.Errf(
			http.StatusInternalServerError,
			"failed to save job result",
		)
	}

	return nil
}

// ListJobs retrieves all jobs belonging to a specific queue.
// It validates request context, fetches jobs from the repository,
// and maps repository or context errors to appropriate API errors.
func (s *JobService) ListJobs(ctx context.Context, queue string) ([]models.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, common.Errf(
			http.StatusRequestTimeout,
			"request timed out",
		)
	}

	jobs, err := s.repo.List(ctx, queue)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			return nil, common.Errf(
				http.StatusRequestTimeout,
				"request timed out",
			)
		}

		return nil, common.Errf(
			http.StatusInternalServerError,
			"failed to list jobs",
		)
	}

	return jobs, nil
}
