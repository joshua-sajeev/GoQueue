package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"gorm.io/datatypes"
)

type Worker struct {
	ID           int
	jobRepo      *postgres.JobRepository
	queues       []string
	lockDuration time.Duration
	quit         chan struct{}
}

func NewWorker(id int, repo *postgres.JobRepository, queues []string, dur time.Duration) *Worker {
	return &Worker{ID: id, jobRepo: repo, queues: queues, lockDuration: dur, quit: make(chan struct{})}
}

func (w *Worker) Start(ctx context.Context) {
	go func() {
		currentDelay := 1 * time.Second
		maxDelay := 60 * time.Second

		for {
			job := w.pullJob(ctx)

			if job != nil {
				w.process(ctx, job)
				currentDelay = 1 * time.Second
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
	}()
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

func (w *Worker) Stop() { close(w.quit) }
