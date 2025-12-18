package job

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/joshu-sajeev/goqueue/common"
	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/models"
	"gorm.io/datatypes"
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
		return common.Errf(http.StatusInternalServerError, "failed to add job to database: %v", err)
	}

	return nil
}

// TODO:
func (s *JobService) GetJobByID(ctx context.Context, id uint) (*models.Job, error) {
	return nil, nil
}

// TODO:
func (s *JobService) UpdateStatus(ctx context.Context, id uint, status string) error {
	return nil
}

// TODO:
func (s *JobService) IncrementAttempts(ctx context.Context, id uint) error {
	return nil
}

// TODO:
func (s *JobService) SaveResult(ctx context.Context, id uint, result datatypes.JSON, err string) error {
	return nil
}

// TODO:
func (s *JobService) ListJobs(ctx context.Context, queue string) ([]models.Job, error) {
	return nil, nil
}
