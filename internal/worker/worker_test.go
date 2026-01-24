package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestWorker_Start(t *testing.T) {
	repoMock := new(mocks.JobRepoMock)
	w := NewWorker(1, repoMock, []string{"default"}, time.Second, time.Second, 60*time.Second)

	w.PollInterval = 1 * time.Millisecond
	w.MaxPollInterval = 2 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	t.Run("Worker pulls jobs until stopped", func(t *testing.T) {
		payload, _ := json.Marshal(dto.SendEmailPayload{To: "test@test.com", Subject: "Hi"})

		repoMock.On("AcquireNext", mock.Anything, "default", uint(1), mock.Anything).
			Return(&dto.JobDTO{
				ID:      1,
				Queue:   "default",
				Payload: datatypes.JSON(payload),
			}, nil).Once()

		repoMock.On("MarkCompleted", mock.Anything, uint(1), mock.Anything).
			Return(nil).Once()

		repoMock.On("AcquireNext", mock.Anything, "default", uint(1), mock.Anything).
			Return(nil, nil)

		w.Start(ctx)
		time.Sleep(200 * time.Millisecond)
		w.Stop()

		repoMock.AssertExpectations(t)
	})
}
func TestWorker_Process(t *testing.T) {
	ctx := context.Background()
	repoMock := new(mocks.JobRepoMock)
	queues := []string{"email"}
	dur := 1 * time.Minute

	w := NewWorker(1, repoMock, queues, dur, time.Second, 60*time.Second)

	t.Run("Successful job processing updates DB to completed", func(t *testing.T) {
		job := &dto.JobDTO{
			ID:      123,
			Queue:   "email",
			Payload: []byte(`{"to": "test@example.com"}`),
		}

		repoMock.On("MarkCompleted", ctx, uint(123), mock.Anything).Return(nil)

		w.process(ctx, job)

		repoMock.AssertExpectations(t)
	})

	t.Run("Failed job triggers RetryLater", func(t *testing.T) {
		job := &dto.JobDTO{
			ID:      456,
			Queue:   "unknown_queue",
			Payload: []byte(`{}`),
		}

		repoMock.On("RetryLater", ctx, uint(456), mock.Anything).Return(nil)

		w.process(ctx, job)

		repoMock.AssertExpectations(t)
	})
}

func TestWorker_PullJob(t *testing.T) {
	ctx := context.Background()
	repoMock := new(mocks.JobRepoMock)
	w := NewWorker(1, repoMock, []string{"high-priority", "low-priority"}, time.Minute, time.Second, 60*time.Second)

	t.Run("Pulls from first available queue", func(t *testing.T) {
		repoMock.On("AcquireNext", ctx, "high-priority", uint(1), time.Minute).Return(nil, nil)
		repoMock.On("AcquireNext", ctx, "low-priority", uint(1), time.Minute).Return(&dto.JobDTO{ID: 1}, nil)

		job := w.pullJob(ctx)

		assert.NotNil(t, job)
		assert.Equal(t, uint(1), job.ID)
		repoMock.AssertExpectations(t)
	})
}

func TestNewWorker(t *testing.T) {
	repoMock := new(mocks.JobRepoMock)

	w := NewWorker(
		1,
		repoMock,
		[]string{"default"},
		5*time.Second,
		2*time.Second,
		10*time.Second,
	)

	require.NotNil(t, w)
	assert.Equal(t, 1, w.ID)
	assert.Equal(t, repoMock, w.jobRepo)
	assert.Equal(t, []string{"default"}, w.queues)
	assert.Equal(t, 5*time.Second, w.lockDuration)
	assert.Equal(t, 2*time.Second, w.PollInterval)
	assert.Equal(t, 10*time.Second, w.MaxPollInterval)
	assert.NotNil(t, w.quit)
}

func TestWorkerStartStopsOnContextCancel(t *testing.T) {
	repoMock := new(mocks.JobRepoMock)

	repoMock.
		On(
			"AcquireNext",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).
		Return(nil, nil)

	w := NewWorker(
		1,
		repoMock,
		[]string{"default"},
		5*time.Second,
		10*time.Millisecond,
		50*time.Millisecond,
	)

	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)

	time.Sleep(30 * time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("worker did not stop after context cancel")
	}

	repoMock.AssertExpectations(t)
}
