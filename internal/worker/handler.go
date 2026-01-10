package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/joshu-sajeev/goqueue/internal/dto"
	"gorm.io/datatypes"
)

// SendEmailHandler simulates sending an email
func SendEmailHandler(ctx context.Context, payload datatypes.JSON) (any, error) {
	var email dto.SendEmailPayload
	if err := json.Unmarshal(payload, &email); err != nil {
		return nil, fmt.Errorf("unmarshal email payload: %w", err)
	}

	// Simulate email sending delay
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	log.Printf("📧 Sent email to %s: %s", email.To, email.Subject)

	return map[string]any{
		"to":         email.To,
		"subject":    email.Subject,
		"sent_at":    time.Now().Format(time.RFC3339),
		"message_id": fmt.Sprintf("msg_%d", time.Now().Unix()),
	}, nil
}

// ProcessPaymentHandler simulates payment processing
func ProcessPaymentHandler(ctx context.Context, payload datatypes.JSON) (any, error) {
	var payment dto.ProcessPaymentPayload
	if err := json.Unmarshal(payload, &payment); err != nil {
		return nil, fmt.Errorf("unmarshal payment payload: %w", err)
	}

	// Simulate payment gateway delay
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	log.Printf("💳 Processed payment %s: %.2f %s", payment.PaymentID, payment.Amount, payment.Currency)

	return map[string]any{
		"payment_id":     payment.PaymentID,
		"status":         "completed",
		"amount":         payment.Amount,
		"currency":       payment.Currency,
		"transaction_id": fmt.Sprintf("txn_%d", time.Now().Unix()),
		"processed_at":   time.Now().Format(time.RFC3339),
	}, nil
}

// SendWebhookHandler sends an HTTP webhook
func SendWebhookHandler(ctx context.Context, payload datatypes.JSON) (any, error) {
	var webhook dto.SendWebhookPayload
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("unmarshal webhook payload: %w", err)
	}

	// Simulate network delay
	delay := time.Duration(webhook.Timeout) * time.Millisecond
	log.Printf("🔔 Simulating webhook to %s with delay %v ms", webhook.URL, delay)
	select {
	case <-time.After(delay):
		// Simulated successful response
	case <-ctx.Done():
		return nil, fmt.Errorf("webhook cancelled or timeout: %w", ctx.Err())
	}

	// Return a fake response
	return map[string]any{
		"url":          webhook.URL,
		"method":       webhook.Method,
		"status_code":  200,
		"response":     fmt.Sprintf("Simulated payload: %s", string(webhook.Body)),
		"delivered_at": time.Now().Format(time.RFC3339),
	}, nil
}
