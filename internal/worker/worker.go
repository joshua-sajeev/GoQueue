package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/job"
	"gorm.io/datatypes"
)

type JobRepository interface {
	AcquireNext(ctx context.Context, queue string, workerID uint, lockDuration time.Duration) (*dto.JobDTO, error)
	RetryLater(ctx context.Context, id uint, availableAt time.Time) error
	MarkCompleted(ctx context.Context, id uint, result datatypes.JSON) error
}

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
		nextRun := time.Now().Add(10 * time.Second)
		w.jobRepo.RetryLater(ctx, job.ID, nextRun)
		return
	}

	b, _ := json.Marshal(res)
	w.jobRepo.MarkCompleted(ctx, job.ID, datatypes.JSON(b))
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
	case "webhooks":
		return SendWebhookHandler(ctx, job.Payload)
	default:
		return nil, fmt.Errorf("unknown queue: %s", job.Queue)
	}
}
