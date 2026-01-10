package pool

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/storage/postgres"
	"github.com/joshu-sajeev/goqueue/internal/worker"
)

type WorkerPool struct {
	workers      []*worker.Worker
	jobRepo      *postgres.JobRepository
	lockDuration time.Duration
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewWorkerPool(count int, repo *postgres.JobRepository, queues []string, dur time.Duration) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &WorkerPool{jobRepo: repo, lockDuration: dur, ctx: ctx, cancel: cancel}

	for i := 1; i <= count; i++ {
		p.workers = append(p.workers, worker.NewWorker(i, repo, queues, dur))
	}
	return p
}

func (p *WorkerPool) Start() {
	for _, w := range p.workers {
		w.Start(p.ctx)
	}

	p.wg.Add(1)
	go p.janitor()
}

func (p *WorkerPool) janitor() {
	defer p.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			stuck, _ := p.jobRepo.ListStuckJobs(p.ctx, p.lockDuration*2)
			for _, j := range stuck {
				log.Printf("Recovering stuck job %d", j.ID)
				p.jobRepo.Release(p.ctx, j.ID)
			}
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *WorkerPool) Stop() {
	p.cancel()
	for _, w := range p.workers {
		w.Stop()
	}
	p.wg.Wait()
}
