package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/job"
	"gorm.io/datatypes"
)

type Worker struct {
	ID              int
	jobRepo         job.JobRepoInterface
	queues          []string
	lockDuration    time.Duration
	quit            chan struct{}
	PollInterval    time.Duration
	MaxPollInterval time.Duration
	wg              sync.WaitGroup
}

func NewWorker(id int, repo job.JobRepoInterface, queues []string, dur time.Duration, pi time.Duration, mi time.Duration) *Worker {
	return &Worker{ID: id, jobRepo: repo, queues: queues, lockDuration: dur, quit: make(chan struct{}), PollInterval: pi, MaxPollInterval: mi}
}

func (w *Worker) Start(ctx context.Context) {
	w.wg.Go(func() {

		delay := w.PollInterval
		if delay == 0 {
			delay = 1 * time.Second
		}

		maxDelay := w.MaxPollInterval
		if maxDelay == 0 {
			maxDelay = 60 * time.Second
		}

		currentDelay := delay

		for {
			job := w.pullJob(ctx)

			if job != nil {
				w.process(ctx, job)
				currentDelay = delay
			} else {
				currentDelay = min(currentDelay*2, maxDelay)
			}

			select {
			case <-time.After(currentDelay):
			case <-w.quit:
				return
			case <-ctx.Done():
				return
			}
		}
	})
}

func (w *Worker) Stop() {
	close(w.quit)
	w.wg.Wait() // This ensures the goroutine is DEAD before we continue
}

func (w *Worker) pullJob(ctx context.Context) *dto.JobDTO {
	for _, q := range w.queues {
		job, _ := w.jobRepo.AcquireNext(ctx, q, uint(w.ID), w.lockDuration)
		if job != nil {
			return job
		}
	}
	return nil
}

func (w *Worker) process(ctx context.Context, job *dto.JobDTO) {
	res, err := w.execute(ctx, job)

	if err != nil {
		w.handleFailure(ctx, job, err)
		return
	}

	b, _ := json.Marshal(res)
	w.jobRepo.MarkCompleted(ctx, job.ID, datatypes.JSON(b))
}

// CRITICAL FIX: Use atomic IncrementAttemptsAndGet to prevent race conditions
func (w *Worker) handleFailure(ctx context.Context, job *dto.JobDTO, execErr error) {
	// Atomically increment and get current values (prevents race conditions)
	currentAttempts, maxRetries, err := w.jobRepo.IncrementAttemptsAndGet(ctx, job.ID)
	if err != nil {
		log.Printf("Worker %d: Failed to increment attempts for job %d: %v", w.ID, job.ID, err)
		return
	}

	// Check if we've exceeded max retries
	if currentAttempts >= maxRetries {
		log.Printf("Worker %d: Job %d exceeded max retries (%d/%d). Marking as failed.",
			w.ID, job.ID, currentAttempts, maxRetries)

		// Mark as permanently failed
		errorMsg := fmt.Sprintf("Max retries exceeded. Last error: %v", execErr)
		w.jobRepo.SaveResult(ctx, job.ID, nil, errorMsg)
		w.jobRepo.UpdateStatus(ctx, job.ID, config.JobStatusFailed)
		return
	}

	// Calculate exponential backoff delay
	delay := w.calculateBackoff(currentAttempts)
	nextRun := time.Now().Add(delay)

	log.Printf("Worker %d: Job %d failed (attempt %d/%d). Retrying in %v. Error: %v",
		w.ID, job.ID, currentAttempts, maxRetries, delay, execErr)

	// Schedule retry (this now clears locks properly)
	if err := w.jobRepo.RetryLater(ctx, job.ID, nextRun); err != nil {
		log.Printf("Worker %d: Failed to schedule retry for job %d: %v", w.ID, job.ID, err)
	}
}

// calculateBackoff returns exponential backoff duration with jitter
func (w *Worker) calculateBackoff(attempts int) time.Duration {
	// Base delay: 10 seconds
	baseDelay := 10 * time.Second

	// Maximum delay: 1 hour
	maxDelay := 1 * time.Hour

	// Calculate exponential backoff: base * 2^(attempts-1)
	// attempts=1: 10s, attempts=2: 20s, attempts=3: 40s, attempts=4: 80s, etc.
	delay := float64(baseDelay) * math.Pow(2, float64(attempts-1))

	// Cap at max delay
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	// Add jitter (±20% randomness) to prevent thundering herd
	jitter := (rand.Float64() * 0.4) - 0.2 // Range: -0.2 to +0.2
	delay = delay * (1 + jitter)

	return time.Duration(delay)
}

func (w *Worker) execute(ctx context.Context, job *dto.JobDTO) (any, error) {
	queue := job.Queue
	if queue == "default" {
		queue = "email"
	}

	switch queue {
	case "email":
		return SendEmailHandler(ctx, job.Payload)
	case "payment":
		return ProcessPaymentHandler(ctx, job.Payload)
	case "webhook":
		return SendWebhookHandler(ctx, job.Payload)
	default:
		return nil, fmt.Errorf("unknown queue: %s", job.Queue)
	}
}
