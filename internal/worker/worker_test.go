package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/internal/mocks"
	"github.com/joshu-sajeev/goqueue/internal/models"
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
		payload, _ := json.Marshal(dto.SendEmailPayload{To: "test@test.com", Subject: "Hi", Body: "Test"})

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

	t.Run("Successful job processing updates DB to completed", func(t *testing.T) {
		repoMock := new(mocks.JobRepoMock)
		w := NewWorker(1, repoMock, []string{"email"}, time.Minute, time.Second, 60*time.Second)

		payload, _ := json.Marshal(dto.SendEmailPayload{
			To:      "test@example.com",
			Subject: "Test",
			Body:    "Hello",
		})

		job := &dto.JobDTO{
			ID:      123,
			Queue:   "email",
			Payload: datatypes.JSON(payload),
		}

		repoMock.On("MarkCompleted", ctx, uint(123), mock.Anything).Return(nil)

		w.process(ctx, job)

		repoMock.AssertExpectations(t)
	})

	t.Run("Failed job triggers retry logic", func(t *testing.T) {
		repoMock := new(mocks.JobRepoMock)
		w := NewWorker(1, repoMock, []string{"unknown_queue"}, time.Minute, time.Second, 60*time.Second)

		job := &dto.JobDTO{
			ID:      456,
			Queue:   "unknown_queue",
			Payload: datatypes.JSON([]byte(`{}`)),
		}

		repoMock.On("Get", ctx, uint(456)).Return(&models.Job{
			ID:         456,
			Attempts:   0,
			MaxRetries: 3,
		}, nil)

		repoMock.On("IncrementAttempts", ctx, uint(456)).Return(nil)
		repoMock.On("RetryLater", ctx, uint(456), mock.Anything).Return(nil)

		w.process(ctx, job)

		repoMock.AssertExpectations(t)
	})

	t.Run("Failed job exceeding max retries marks as failed", func(t *testing.T) {
		repoMock := new(mocks.JobRepoMock)
		w := NewWorker(1, repoMock, []string{"email"}, time.Minute, time.Second, 60*time.Second)

		job := &dto.JobDTO{
			ID:      789,
			Queue:   "unknown_queue",
			Payload: datatypes.JSON([]byte(`{}`)),
		}

		repoMock.On("Get", ctx, uint(789)).Return(&models.Job{
			ID:         789,
			Attempts:   2,
			MaxRetries: 3,
		}, nil)

		repoMock.On("IncrementAttempts", ctx, uint(789)).Return(nil)
		repoMock.On("SaveResult", ctx, uint(789), mock.Anything, mock.MatchedBy(func(errMsg string) bool {
			return len(errMsg) > 0
		})).Return(nil)
		repoMock.On("UpdateStatus", ctx, uint(789), config.JobStatusFailed).Return(nil)

		w.process(ctx, job)

		repoMock.AssertExpectations(t)
	})
}

func TestWorker_HandleFailure(t *testing.T) {
	ctx := context.Background()

	t.Run("First attempt schedules retry", func(t *testing.T) {
		repoMock := new(mocks.JobRepoMock)
		w := NewWorker(1, repoMock, []string{"email"}, time.Minute, time.Second, 60*time.Second)

		job := &dto.JobDTO{
			ID:    100,
			Queue: "email",
		}

		repoMock.On("Get", ctx, uint(100)).Return(&models.Job{
			ID:         100,
			Attempts:   0,
			MaxRetries: 3,
		}, nil)

		repoMock.On("IncrementAttempts", ctx, uint(100)).Return(nil)
		repoMock.On("RetryLater", ctx, uint(100), mock.Anything).Return(nil)

		w.handleFailure(ctx, job, errors.New("test error"))

		repoMock.AssertExpectations(t)
		repoMock.AssertCalled(t, "RetryLater", ctx, uint(100), mock.Anything)
	})

	t.Run("Max retries exceeded marks as failed", func(t *testing.T) {
		repoMock := new(mocks.JobRepoMock)
		w := NewWorker(1, repoMock, []string{"email"}, time.Minute, time.Second, 60*time.Second)

		job := &dto.JobDTO{
			ID:    200,
			Queue: "email",
		}

		repoMock.On("Get", ctx, uint(200)).Return(&models.Job{
			ID:         200,
			Attempts:   2,
			MaxRetries: 3,
		}, nil)

		repoMock.On("IncrementAttempts", ctx, uint(200)).Return(nil)
		repoMock.On("SaveResult", ctx, uint(200), mock.Anything, mock.MatchedBy(func(msg string) bool {
			return len(msg) > 0 && msg != ""
		})).Return(nil)
		repoMock.On("UpdateStatus", ctx, uint(200), config.JobStatusFailed).Return(nil)

		w.handleFailure(ctx, job, errors.New("persistent error"))

		repoMock.AssertExpectations(t)
		repoMock.AssertNotCalled(t, "RetryLater", mock.Anything, mock.Anything, mock.Anything)
		repoMock.AssertCalled(t, "UpdateStatus", ctx, uint(200), config.JobStatusFailed)
	})

	t.Run("Get failure logs error and returns early", func(t *testing.T) {
		repoMock := new(mocks.JobRepoMock)
		w := NewWorker(1, repoMock, []string{"email"}, time.Minute, time.Second, 60*time.Second)

		job := &dto.JobDTO{
			ID:    300,
			Queue: "email",
		}

		repoMock.On("Get", ctx, uint(300)).Return(nil, errors.New("db error"))

		w.handleFailure(ctx, job, errors.New("test error"))

		repoMock.AssertExpectations(t)
		repoMock.AssertNotCalled(t, "IncrementAttempts", mock.Anything, mock.Anything)
	})

	t.Run("IncrementAttempts failure logs error and returns early", func(t *testing.T) {
		repoMock := new(mocks.JobRepoMock)
		w := NewWorker(1, repoMock, []string{"email"}, time.Minute, time.Second, 60*time.Second)

		job := &dto.JobDTO{
			ID:    400,
			Queue: "email",
		}

		repoMock.On("Get", ctx, uint(400)).Return(&models.Job{
			ID:         400,
			Attempts:   0,
			MaxRetries: 3,
		}, nil)

		repoMock.On("IncrementAttempts", ctx, uint(400)).Return(errors.New("increment error"))

		w.handleFailure(ctx, job, errors.New("test error"))

		repoMock.AssertExpectations(t)
		repoMock.AssertNotCalled(t, "RetryLater", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestWorker_CalculateBackoff(t *testing.T) {
	w := NewWorker(1, nil, []string{"email"}, time.Minute, time.Second, 60*time.Second)

	tests := []struct {
		name     string
		attempts int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{
			name:     "attempt 1",
			attempts: 1,
			minDelay: 8 * time.Second,  // 10s * 0.8 (with -20% jitter)
			maxDelay: 12 * time.Second, // 10s * 1.2 (with +20% jitter)
		},
		{
			name:     "attempt 2",
			attempts: 2,
			minDelay: 16 * time.Second, // 20s * 0.8
			maxDelay: 24 * time.Second, // 20s * 1.2
		},
		{
			name:     "attempt 3",
			attempts: 3,
			minDelay: 32 * time.Second, // 40s * 0.8
			maxDelay: 48 * time.Second, // 40s * 1.2
		},
		{
			name:     "attempt 4",
			attempts: 4,
			minDelay: 64 * time.Second, // 80s * 0.8
			maxDelay: 96 * time.Second, // 80s * 1.2
		},
		{
			name:     "attempt 10 (should cap at 1 hour)",
			attempts: 10,
			minDelay: 48 * time.Minute, // 1h * 0.8
			maxDelay: 72 * time.Minute, // 1h * 1.2 (but capped at 1h, so effectively 1h * 1.2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to account for randomness
			for range 10 {
				delay := w.calculateBackoff(tt.attempts)
				assert.GreaterOrEqual(t, delay, tt.minDelay, "delay should be >= min")
				assert.LessOrEqual(t, delay, tt.maxDelay, "delay should be <= max")
			}
		})
	}

	t.Run("backoff grows exponentially", func(t *testing.T) {
		delay1 := w.calculateBackoff(1)
		delay2 := w.calculateBackoff(2)
		delay3 := w.calculateBackoff(3)

		// Even with jitter, delay2 should be roughly 2x delay1
		// We'll check that delay2 is at least 1.5x delay1 (accounting for jitter)
		assert.Greater(t, delay2, delay1*3/2, "delay should grow exponentially")
		assert.Greater(t, delay3, delay2*3/2, "delay should continue growing")
	})

	t.Run("respects maximum delay of 1 hour", func(t *testing.T) {
		// Very large attempt count
		delay := w.calculateBackoff(100)
		maxWithJitter := time.Hour + (time.Hour * 20 / 100) // 1h + 20% jitter
		assert.LessOrEqual(t, delay, maxWithJitter, "should not exceed max delay with jitter")
	})
}

func TestWorker_PullJob(t *testing.T) {
	ctx := context.Background()
	repoMock := new(mocks.JobRepoMock)
	w := NewWorker(1, repoMock, []string{"default", "payment"}, time.Minute, time.Second, 60*time.Second)

	t.Run("Pulls from first available queue", func(t *testing.T) {
		repoMock.On("AcquireNext", ctx, "default", uint(1), time.Minute).Return(nil, nil)
		repoMock.On("AcquireNext", ctx, "payment", uint(1), time.Minute).Return(&dto.JobDTO{ID: 1}, nil)

		job := w.pullJob(ctx)

		assert.NotNil(t, job)
		assert.Equal(t, uint(1), job.ID)
		repoMock.AssertExpectations(t)
	})

	t.Run("Returns nil when no jobs available", func(t *testing.T) {
		repoMock := new(mocks.JobRepoMock)
		w := NewWorker(1, repoMock, []string{"default", "payment"}, time.Minute, time.Second, 60*time.Second)

		repoMock.On("AcquireNext", ctx, "default", uint(1), time.Minute).Return(nil, nil)
		repoMock.On("AcquireNext", ctx, "payment", uint(1), time.Minute).Return(nil, nil)

		job := w.pullJob(ctx)

		assert.Nil(t, job)
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
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("worker did not stop after context cancel")
	}

	repoMock.AssertExpectations(t)
}

func TestWorker_Execute(t *testing.T) {
	ctx := context.Background()
	w := NewWorker(1, nil, []string{"email"}, time.Minute, time.Second, 60*time.Second)

	t.Run("executes email handler", func(t *testing.T) {
		payload, _ := json.Marshal(dto.SendEmailPayload{
			To:      "test@example.com",
			Subject: "Test",
			Body:    "Hello",
		})

		job := &dto.JobDTO{
			ID:      1,
			Queue:   "email",
			Payload: datatypes.JSON(payload),
		}

		result, err := w.execute(ctx, job)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("executes payment handler", func(t *testing.T) {
		payload, _ := json.Marshal(dto.ProcessPaymentPayload{
			PaymentID: "pay_123",
			UserID:    "user_456",
			Amount:    100.50,
			Currency:  "USD",
			Method:    "card",
		})

		job := &dto.JobDTO{
			ID:      2,
			Queue:   "payment",
			Payload: datatypes.JSON(payload),
		}

		result, err := w.execute(ctx, job)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("executes webhook handler", func(t *testing.T) {
		payload, _ := json.Marshal(dto.SendWebhookPayload{
			URL:     "https://example.com/webhook",
			Method:  "POST",
			Body:    json.RawMessage(`{"event":"test"}`),
			Timeout: 10,
		})

		job := &dto.JobDTO{
			ID:      3,
			Queue:   "webhooks",
			Payload: datatypes.JSON(payload),
		}

		result, err := w.execute(ctx, job)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("default queue routes to email", func(t *testing.T) {
		payload, _ := json.Marshal(dto.SendEmailPayload{
			To:      "test@example.com",
			Subject: "Test",
			Body:    "Hello",
		})

		job := &dto.JobDTO{
			ID:      4,
			Queue:   "default",
			Payload: datatypes.JSON(payload),
		}

		result, err := w.execute(ctx, job)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("unknown queue returns error", func(t *testing.T) {
		job := &dto.JobDTO{
			ID:      5,
			Queue:   "unknown",
			Payload: datatypes.JSON([]byte(`{}`)),
		}

		result, err := w.execute(ctx, job)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unknown queue")
	})
}
