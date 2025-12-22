package dto

import (
	"encoding/json"
	"time"
)

type JobCreateDTO struct {
	Queue       string          `json:"queue" validate:"required"`
	Type        string          `json:"type" validate:"required"`
	Payload     json.RawMessage `json:"payload" validate:"required"`
	MaxRetries  int             `json:"max_retries" validate:"gte=0,lte=20"`
	AvailableAt *time.Time      `json:"available_at,omitempty"`
}

type JobResponseDTO struct {
	ID         uint            `json:"id"`
	Queue      string          `json:"queue"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Status     string          `json:"status"`
	Attempts   int             `json:"attempts"`
	MaxRetries int             `json:"max_retries"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}
