package dto

import "encoding/json"

type SendWebhookPayload struct {
	URL     string            `json:"url" validate:"required,url"`
	Method  string            `json:"method" validate:"required,oneof=POST PUT PATCH"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body" validate:"required"`
	Timeout int               `json:"timeout" validate:"gte=1,lte=30"`
}
