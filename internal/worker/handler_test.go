package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestSendEmailHandler(t *testing.T) {
	tests := []struct {
		name      string
		payload   any
		ctx       context.Context
		wantErr   bool
		errMsg    string
		checkResp func(*testing.T, any)
	}{
		{
			name: "success with valid payload",
			payload: dto.SendEmailPayload{
				To:      "test@example.com",
				Subject: "Hello",
				Body:    "Test email",
			},
			ctx:     context.Background(),
			wantErr: false,
			checkResp: func(t *testing.T, resp any) {
				result := resp.(map[string]any)
				assert.Equal(t, "test@example.com", result["to"])
				assert.Equal(t, "Hello", result["subject"])
				assert.NotEmpty(t, result["message_id"])
			},
		},
		{
			name:    "error - missing 'to' field",
			payload: dto.SendEmailPayload{Subject: "Hello", Body: "Test"},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'to' field is required",
		},
		{
			name:    "error - missing 'subject' field",
			payload: dto.SendEmailPayload{To: "test@example.com", Body: "Test"},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'subject' field is required",
		},
		{
			name:    "error - missing 'body' field",
			payload: dto.SendEmailPayload{To: "test@example.com", Subject: "Hello"},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'body' field is required",
		},
		{
			name:    "error - invalid json",
			payload: "invalid-json",
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "unmarshal email payload",
		},
		{
			name: "error - context cancelled",
			payload: dto.SendEmailPayload{
				To:      "test@example.com",
				Subject: "Hello",
				Body:    "Test",
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data datatypes.JSON

			switch v := tt.payload.(type) {
			case string:
				data = datatypes.JSON(v)
			default:
				b, _ := json.Marshal(v)
				data = datatypes.JSON(b)
			}

			res, err := SendEmailHandler(tt.ctx, data)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			if tt.checkResp != nil {
				tt.checkResp(t, res)
			}
		})
	}
}

func TestProcessPaymentHandler(t *testing.T) {
	tests := []struct {
		name      string
		payload   dto.ProcessPaymentPayload
		ctx       context.Context
		wantErr   bool
		errMsg    string
		checkResp func(*testing.T, any)
	}{
		{
			name: "success with valid payload",
			payload: dto.ProcessPaymentPayload{
				PaymentID: "pay_123",
				UserID:    "user_456",
				Amount:    499.99,
				Currency:  "INR",
				Method:    "card",
			},
			ctx:     context.Background(),
			wantErr: false,
			checkResp: func(t *testing.T, resp any) {
				result := resp.(map[string]any)
				assert.Equal(t, "pay_123", result["payment_id"])
				assert.Equal(t, "completed", result["status"])
				assert.Equal(t, 499.99, result["amount"])
			},
		},
		{
			name: "error - missing payment_id",
			payload: dto.ProcessPaymentPayload{
				UserID:   "user_456",
				Amount:   100,
				Currency: "USD",
				Method:   "card",
			},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'payment_id' field is required",
		},
		{
			name: "error - missing user_id",
			payload: dto.ProcessPaymentPayload{
				PaymentID: "pay_123",
				Amount:    100,
				Currency:  "USD",
				Method:    "card",
			},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'user_id' field is required",
		},
		{
			name: "error - zero amount",
			payload: dto.ProcessPaymentPayload{
				PaymentID: "pay_123",
				UserID:    "user_456",
				Amount:    0,
				Currency:  "USD",
				Method:    "card",
			},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'amount' must be greater than 0",
		},
		{
			name: "error - negative amount",
			payload: dto.ProcessPaymentPayload{
				PaymentID: "pay_123",
				UserID:    "user_456",
				Amount:    -10,
				Currency:  "USD",
				Method:    "card",
			},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'amount' must be greater than 0",
		},
		{
			name: "error - missing currency",
			payload: dto.ProcessPaymentPayload{
				PaymentID: "pay_123",
				UserID:    "user_456",
				Amount:    100,
				Method:    "card",
			},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'currency' field is required",
		},
		{
			name: "error - missing method",
			payload: dto.ProcessPaymentPayload{
				PaymentID: "pay_123",
				UserID:    "user_456",
				Amount:    100,
				Currency:  "USD",
			},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'method' field is required",
		},
		{
			name: "error - context cancelled",
			payload: dto.ProcessPaymentPayload{
				PaymentID: "pay_123",
				UserID:    "user_456",
				Amount:    100,
				Currency:  "USD",
				Method:    "card",
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.payload)

			res, err := ProcessPaymentHandler(tt.ctx, datatypes.JSON(data))

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			if tt.checkResp != nil {
				tt.checkResp(t, res)
			}
		})
	}
}

func TestSendWebhookHandler(t *testing.T) {
	tests := []struct {
		name      string
		payload   dto.SendWebhookPayload
		ctx       context.Context
		wantErr   bool
		errMsg    string
		checkResp func(*testing.T, any)
	}{
		{
			name: "success with valid payload",
			payload: dto.SendWebhookPayload{
				URL:     "https://example.com/webhook",
				Method:  "POST",
				Body:    []byte(`{"event":"test"}`),
				Timeout: 50,
			},
			ctx:     context.Background(),
			wantErr: false,
			checkResp: func(t *testing.T, resp any) {
				result := resp.(map[string]any)
				assert.Equal(t, "https://example.com/webhook", result["url"])
				assert.Equal(t, "POST", result["method"])
				assert.Equal(t, 200, result["status_code"])
			},
		},
		{
			name: "error - missing url",
			payload: dto.SendWebhookPayload{
				Method:  "POST",
				Timeout: 10,
			},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'url' field is required",
		},
		{
			name: "error - missing method",
			payload: dto.SendWebhookPayload{
				URL:     "https://example.com",
				Timeout: 10,
			},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'method' field is required",
		},
		{
			name: "error - zero timeout",
			payload: dto.SendWebhookPayload{
				URL:     "https://example.com",
				Method:  "POST",
				Timeout: 0,
			},
			ctx:     context.Background(),
			wantErr: true,
			errMsg:  "'timeout' must be greater than 0",
		},
		{
			name: "error - context timeout",
			payload: dto.SendWebhookPayload{
				URL:     "https://example.com/webhook",
				Method:  "POST",
				Timeout: 200,
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()
				return ctx
			}(),
			wantErr: true,
			errMsg:  "cancelled or timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.payload)

			res, err := SendWebhookHandler(tt.ctx, datatypes.JSON(data))

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			if tt.checkResp != nil {
				tt.checkResp(t, res)
			}
		})
	}
}
