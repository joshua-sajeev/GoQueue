package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/dto"
	"gorm.io/datatypes"
)

func TestSendEmailHandler(t *testing.T) {
	tests := []struct {
		name      string
		payload   any
		ctx       context.Context
		expectErr bool
	}{
		{
			name: "success",
			payload: dto.SendEmailPayload{
				To:      "test@example.com",
				Subject: "Hello",
				Body:    "Test email",
			},
			ctx:       context.Background(),
			expectErr: false,
		},
		{
			name:      "invalid json",
			payload:   "invalid-json",
			ctx:       context.Background(),
			expectErr: true,
		},
		{
			name: "context cancelled",
			payload: dto.SendEmailPayload{
				To:      "test@example.com",
				Subject: "Hello",
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			expectErr: true,
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

			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.expectErr {
				result := res.(map[string]any)
				if result["to"] == "" {
					t.Errorf("expected 'to' field in result")
				}
			}
		})
	}
}

func TestProcessPaymentHandler(t *testing.T) {
	tests := []struct {
		name      string
		payload   dto.ProcessPaymentPayload
		ctx       context.Context
		expectErr bool
	}{
		{
			name: "success",
			payload: dto.ProcessPaymentPayload{
				PaymentID: "pay_123",
				Amount:    499.99,
				Currency:  "INR",
			},
			ctx:       context.Background(),
			expectErr: false,
		},
		{
			name: "context cancelled",
			payload: dto.ProcessPaymentPayload{
				PaymentID: "pay_123",
				Amount:    100,
				Currency:  "USD",
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.payload)

			res, err := ProcessPaymentHandler(tt.ctx, datatypes.JSON(data))

			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.expectErr {
				result := res.(map[string]any)
				if result["status"] != "completed" {
					t.Errorf("expected status completed, got %v", result["status"])
				}
			}
		})
	}
}

func TestSendWebhookHandler(t *testing.T) {
	tests := []struct {
		name      string
		payload   dto.SendWebhookPayload
		ctx       context.Context
		expectErr bool
	}{
		{
			name: "success",
			payload: dto.SendWebhookPayload{
				URL:     "https://example.com/webhook",
				Method:  "POST",
				Body:    []byte(`{"event":"test"}`),
				Timeout: 50,
			},
			ctx:       context.Background(),
			expectErr: false,
		},
		{
			name: "context timeout",
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
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.payload)

			_, err := SendWebhookHandler(tt.ctx, datatypes.JSON(data))

			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
