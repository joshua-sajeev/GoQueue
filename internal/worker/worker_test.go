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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestWorker_Process(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		job       *dto.JobDTO
		setupMock func(*mocks.JobRepoMock)
		wantErr   bool
	}{
		{
			name: "successful job processing updates DB to completed",
			job: &dto.JobDTO{
				ID:      123,
				Queue:   "email",
				Payload: mustMarshal(dto.SendEmailPayload{To: "test@example.com", Subject: "Test", Body: "Hello"}),
			},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("MarkCompleted", ctx, uint(123), mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed job triggers retry logic with atomic increment",
			job: &dto.JobDTO{
				ID:      456,
				Queue:   "unknown_queue",
				Payload: datatypes.JSON([]byte(`{}`)),
			},
			setupMock: func(m *mocks.JobRepoMock) {
				// Mock atomic increment (attempt 1/3)
				m.On("IncrementAttemptsAndGet", ctx, uint(456)).Return(1, 3, nil)
				m.On("RetryLater", ctx, uint(456), mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed job exceeding max retries marks as failed",
			job: &dto.JobDTO{
				ID:      789,
				Queue:   "unknown_queue",
				Payload: datatypes.JSON([]byte(`{}`)),
			},
			setupMock: func(m *mocks.JobRepoMock) {
				// Mock atomic increment (attempt 3/3)
				m.On("IncrementAttemptsAndGet", ctx, uint(789)).Return(3, 3, nil)
				m.On("SaveResult", ctx, uint(789), mock.Anything, mock.MatchedBy(func(errMsg string) bool {
					return len(errMsg) > 0
				})).Return(nil)
				m.On("UpdateStatus", ctx, uint(789), config.JobStatusFailed).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoMock := new(mocks.JobRepoMock)
			w := NewWorker(1, repoMock, []string{tt.job.Queue}, time.Minute, time.Second, 60*time.Second)

			if tt.setupMock != nil {
				tt.setupMock(repoMock)
			}

			w.process(ctx, tt.job)

			repoMock.AssertExpectations(t)
		})
	}
}

func TestWorker_HandleFailure(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		jobID          uint
		execErr        error
		setupMock      func(*mocks.JobRepoMock)
		assertMock     func(*testing.T, *mocks.JobRepoMock)
		wantRetryLater bool
		wantFailed     bool
	}{
		{
			name:    "first attempt schedules retry",
			jobID:   100,
			execErr: errors.New("temporary error"),
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("IncrementAttemptsAndGet", ctx, uint(100)).Return(1, 3, nil)
				m.On("RetryLater", ctx, uint(100), mock.Anything).Return(nil)
			},
			assertMock: func(t *testing.T, m *mocks.JobRepoMock) {
				m.AssertCalled(t, "RetryLater", ctx, uint(100), mock.Anything)
				m.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
			},
			wantRetryLater: true,
			wantFailed:     false,
		},
		{
			name:    "second attempt schedules retry",
			jobID:   101,
			execErr: errors.New("still failing"),
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("IncrementAttemptsAndGet", ctx, uint(101)).Return(2, 3, nil)
				m.On("RetryLater", ctx, uint(101), mock.Anything).Return(nil)
			},
			assertMock: func(t *testing.T, m *mocks.JobRepoMock) {
				m.AssertCalled(t, "RetryLater", ctx, uint(101), mock.Anything)
			},
			wantRetryLater: true,
			wantFailed:     false,
		},
		{
			name:    "max retries exceeded marks as failed",
			jobID:   200,
			execErr: errors.New("persistent error"),
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("IncrementAttemptsAndGet", ctx, uint(200)).Return(3, 3, nil)
				m.On("SaveResult", ctx, uint(200), mock.Anything, mock.MatchedBy(func(msg string) bool {
					return len(msg) > 0 && msg != ""
				})).Return(nil)
				m.On("UpdateStatus", ctx, uint(200), config.JobStatusFailed).Return(nil)
			},
			assertMock: func(t *testing.T, m *mocks.JobRepoMock) {
				m.AssertNotCalled(t, "RetryLater", mock.Anything, mock.Anything, mock.Anything)
				m.AssertCalled(t, "UpdateStatus", ctx, uint(200), config.JobStatusFailed)
			},
			wantRetryLater: false,
			wantFailed:     true,
		},
		{
			name:    "IncrementAttemptsAndGet failure logs error and returns early",
			jobID:   400,
			execErr: errors.New("test error"),
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("IncrementAttemptsAndGet", ctx, uint(400)).Return(0, 0, errors.New("db error"))
			},
			assertMock: func(t *testing.T, m *mocks.JobRepoMock) {
				m.AssertNotCalled(t, "RetryLater", mock.Anything, mock.Anything, mock.Anything)
				m.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
			},
			wantRetryLater: false,
			wantFailed:     false,
		},
		{
			name:    "retry scheduling failure is logged",
			jobID:   500,
			execErr: errors.New("test error"),
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("IncrementAttemptsAndGet", ctx, uint(500)).Return(1, 3, nil)
				m.On("RetryLater", ctx, uint(500), mock.Anything).Return(errors.New("retry scheduling failed"))
			},
			assertMock: func(t *testing.T, m *mocks.JobRepoMock) {
				m.AssertCalled(t, "RetryLater", ctx, uint(500), mock.Anything)
			},
			wantRetryLater: true,
			wantFailed:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoMock := new(mocks.JobRepoMock)
			w := NewWorker(1, repoMock, []string{"email"}, time.Minute, time.Second, 60*time.Second)

			job := &dto.JobDTO{
				ID:    tt.jobID,
				Queue: "email",
			}

			if tt.setupMock != nil {
				tt.setupMock(repoMock)
			}

			w.handleFailure(ctx, job, tt.execErr)

			repoMock.AssertExpectations(t)

			if tt.assertMock != nil {
				tt.assertMock(t, repoMock)
			}
		})
	}
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
			maxDelay: 72 * time.Minute, // 1h * 1.2
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
		assert.Greater(t, delay2, delay1*3/2, "delay should grow exponentially")
		assert.Greater(t, delay3, delay2*3/2, "delay should continue growing")
	})

	t.Run("respects maximum delay of 1 hour", func(t *testing.T) {
		delay := w.calculateBackoff(100)
		maxWithJitter := time.Hour + (time.Hour * 20 / 100) // 1h + 20% jitter
		assert.LessOrEqual(t, delay, maxWithJitter, "should not exceed max delay with jitter")
	})
}

func TestWorker_Execute(t *testing.T) {
	ctx := context.Background()
	w := NewWorker(1, nil, []string{"email"}, time.Minute, time.Second, 60*time.Second)

	tests := []struct {
		name    string
		job     *dto.JobDTO
		wantErr bool
		errMsg  string
	}{
		{
			name: "executes email handler",
			job: &dto.JobDTO{
				ID:      1,
				Queue:   "email",
				Payload: mustMarshal(dto.SendEmailPayload{To: "test@example.com", Subject: "Test", Body: "Hello"}),
			},
			wantErr: false,
		},
		{
			name: "executes payment handler",
			job: &dto.JobDTO{
				ID:      2,
				Queue:   "payment",
				Payload: mustMarshal(dto.ProcessPaymentPayload{PaymentID: "pay_123", UserID: "user_456", Amount: 100.50, Currency: "USD", Method: "card"}),
			},
			wantErr: false,
		},
		{
			name: "executes webhook handler",
			job: &dto.JobDTO{
				ID:      3,
				Queue:   "webhook",
				Payload: mustMarshal(dto.SendWebhookPayload{URL: "https://example.com/webhook", Method: "POST", Body: json.RawMessage(`{"event":"test"}`), Timeout: 10}),
			},
			wantErr: false,
		},
		{
			name: "default queue routes to email",
			job: &dto.JobDTO{
				ID:      4,
				Queue:   "default",
				Payload: mustMarshal(dto.SendEmailPayload{To: "test@example.com", Subject: "Test", Body: "Hello"}),
			},
			wantErr: false,
		},
		{
			name: "unknown queue returns error",
			job: &dto.JobDTO{
				ID:      5,
				Queue:   "unknown",
				Payload: datatypes.JSON([]byte(`{}`)),
			},
			wantErr: true,
			errMsg:  "unknown queue",
		},
		// NOTE: Handlers don't validate payloads - they process whatever JSON they receive
		// These would succeed with empty/zero values
		{
			name: "malformed JSON returns error",
			job: &dto.JobDTO{
				ID:      6,
				Queue:   "email",
				Payload: datatypes.JSON([]byte(`{invalid json}`)),
			},
			wantErr: true,
			errMsg:  "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := w.execute(ctx, tt.job)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestWorker_PullJob(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		queues    []string
		setupMock func(*mocks.JobRepoMock)
		wantJob   bool
		wantJobID uint
	}{
		{
			name:   "pulls from first available queue",
			queues: []string{"default", "payment"},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("AcquireNext", ctx, "default", uint(1), time.Minute).Return(nil, nil)
				m.On("AcquireNext", ctx, "payment", uint(1), time.Minute).Return(&dto.JobDTO{ID: 1}, nil)
			},
			wantJob:   true,
			wantJobID: 1,
		},
		{
			name:   "returns nil when no jobs available",
			queues: []string{"default", "payment"},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("AcquireNext", ctx, "default", uint(1), time.Minute).Return(nil, nil)
				m.On("AcquireNext", ctx, "payment", uint(1), time.Minute).Return(nil, nil)
			},
			wantJob: false,
		},
		{
			name:   "pulls from first queue if available",
			queues: []string{"email", "payment"},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("AcquireNext", ctx, "email", uint(1), time.Minute).Return(&dto.JobDTO{ID: 42}, nil)
			},
			wantJob:   true,
			wantJobID: 42,
		},
		{
			name:   "handles single queue",
			queues: []string{"webhook"},
			setupMock: func(m *mocks.JobRepoMock) {
				m.On("AcquireNext", ctx, "webhook", uint(1), time.Minute).Return(&dto.JobDTO{ID: 99}, nil)
			},
			wantJob:   true,
			wantJobID: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoMock := new(mocks.JobRepoMock)
			w := NewWorker(1, repoMock, tt.queues, time.Minute, time.Second, 60*time.Second)

			if tt.setupMock != nil {
				tt.setupMock(repoMock)
			}

			job := w.pullJob(ctx)

			if tt.wantJob {
				assert.NotNil(t, job)
				assert.Equal(t, tt.wantJobID, job.ID)
			} else {
				assert.Nil(t, job)
			}

			repoMock.AssertExpectations(t)
		})
	}
}

func TestWorker_Start(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocks.JobRepoMock)
		timeout   time.Duration
		wantCalls int
	}{
		{
			name: "worker pulls jobs until stopped",
			setupMock: func(m *mocks.JobRepoMock) {
				payload, _ := json.Marshal(dto.SendEmailPayload{To: "test@test.com", Subject: "Hi", Body: "Test"})

				m.On("AcquireNext", mock.Anything, "default", uint(1), mock.Anything).
					Return(&dto.JobDTO{
						ID:      1,
						Queue:   "default",
						Payload: datatypes.JSON(payload),
					}, nil).Once()

				m.On("MarkCompleted", mock.Anything, uint(1), mock.Anything).
					Return(nil).Once()

				m.On("AcquireNext", mock.Anything, "default", uint(1), mock.Anything).
					Return(nil, nil)
			},
			timeout: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoMock := new(mocks.JobRepoMock)
			w := NewWorker(1, repoMock, []string{"default"}, time.Second, time.Millisecond, 2*time.Millisecond)

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			if tt.setupMock != nil {
				tt.setupMock(repoMock)
			}

			w.Start(ctx)
			time.Sleep(200 * time.Millisecond)
			w.Stop()

			repoMock.AssertExpectations(t)
		})
	}
}

func TestWorker_StartStopsOnContextCancel(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		wantPanic bool
	}{
		{
			name:      "worker stops on context cancellation",
			timeout:   30 * time.Millisecond,
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoMock := new(mocks.JobRepoMock)
			repoMock.On("AcquireNext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, nil)

			w := NewWorker(1, repoMock, []string{"default"}, 5*time.Second, 10*time.Millisecond, 50*time.Millisecond)

			ctx, cancel := context.WithCancel(context.Background())
			w.Start(ctx)

			time.Sleep(tt.timeout)
			cancel()

			done := make(chan struct{})
			go func() {
				w.wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Success
			case <-time.After(time.Second):
				t.Fatal("worker did not stop after context cancel")
			}

			repoMock.AssertExpectations(t)
		})
	}
}

func TestNewWorker(t *testing.T) {
	tests := []struct {
		name            string
		id              int
		queues          []string
		lockDuration    time.Duration
		pollInterval    time.Duration
		maxPollInterval time.Duration
	}{
		{
			name:            "creates worker with valid config",
			id:              1,
			queues:          []string{"default"},
			lockDuration:    5 * time.Second,
			pollInterval:    2 * time.Second,
			maxPollInterval: 10 * time.Second,
		},
		{
			name:            "creates worker with multiple queues",
			id:              2,
			queues:          []string{"email", "payment", "webhook"},
			lockDuration:    time.Minute,
			pollInterval:    time.Second,
			maxPollInterval: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoMock := new(mocks.JobRepoMock)

			w := NewWorker(tt.id, repoMock, tt.queues, tt.lockDuration, tt.pollInterval, tt.maxPollInterval)

			require.NotNil(t, w)
			assert.Equal(t, tt.id, w.ID)
			assert.Equal(t, repoMock, w.jobRepo)
			assert.Equal(t, tt.queues, w.queues)
			assert.Equal(t, tt.lockDuration, w.lockDuration)
			assert.Equal(t, tt.pollInterval, w.PollInterval)
			assert.Equal(t, tt.maxPollInterval, w.MaxPollInterval)
			assert.NotNil(t, w.quit)
		})
	}
}

// Helper function to marshal payloads for tests
func mustMarshal(v any) datatypes.JSON {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return datatypes.JSON(b)
}
