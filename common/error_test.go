package common

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      APIError
		expected string
	}{
		{
			name:     "returns message",
			err:      APIError{Status: http.StatusBadRequest, Message: "invalid input"},
			expected: "invalid input",
		},
		{
			name:     "empty message",
			err:      APIError{Status: http.StatusInternalServerError, Message: ""},
			expected: "",
		},
		{
			name:     "message with special characters",
			err:      APIError{Status: http.StatusNotFound, Message: "job 42: not found"},
			expected: "job 42: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestErrf(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		format          string
		args            []any
		expectedStatus  int
		expectedMessage string
		expectedFields  map[string]any
	}{
		{
			name:            "no format args",
			status:          http.StatusBadRequest,
			format:          "invalid request",
			args:            nil,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
			expectedFields:  nil,
		},
		{
			name:            "single format arg",
			status:          http.StatusNotFound,
			format:          "job %d not found",
			args:            []any{42},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "job 42 not found",
			expectedFields:  nil,
		},
		{
			name:            "multiple format args",
			status:          http.StatusInternalServerError,
			format:          "failed to %s job %d: %s",
			args:            []any{"update", 7, "db error"},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "failed to update job 7: db error",
			expectedFields:  nil,
		},
		{
			name:            "string format arg",
			status:          http.StatusUnauthorized,
			format:          "queue %q is not allowed",
			args:            []any{"secret"},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: `queue "secret" is not allowed`,
			expectedFields:  nil,
		},
		{
			name:            "returns APIError implementing error interface",
			status:          http.StatusBadRequest,
			format:          "bad input: %v",
			args:            []any{"missing field"},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "bad input: missing field",
			expectedFields:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error = Errf(tt.status, tt.format, tt.args...)

			apiErr, ok := err.(APIError)
			assert.True(t, ok, "Errf should return an APIError")
			assert.Equal(t, tt.expectedStatus, apiErr.Status)
			assert.Equal(t, tt.expectedMessage, apiErr.Message)
			assert.Equal(t, tt.expectedFields, apiErr.Fields)
			assert.Equal(t, tt.expectedMessage, err.Error())
		})
	}
}

func TestNewAPIError(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		message         string
		fields          map[string]any
		expectedStatus  int
		expectedMessage string
		expectedFields  map[string]any
	}{
		{
			name:            "nil fields",
			status:          http.StatusBadRequest,
			message:         "validation failed",
			fields:          nil,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "validation failed",
			expectedFields:  nil,
		},
		{
			name:    "with fields",
			status:  http.StatusBadRequest,
			message: "invalid queue",
			fields: map[string]any{
				"provided": "unknown",
				"allowed":  []string{"default", "email"},
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid queue",
			expectedFields: map[string]any{
				"provided": "unknown",
				"allowed":  []string{"default", "email"},
			},
		},
		{
			name:            "empty fields map",
			status:          http.StatusUnprocessableEntity,
			message:         "unprocessable",
			fields:          map[string]any{},
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "unprocessable",
			expectedFields:  map[string]any{},
		},
		{
			name:    "500 with nested fields",
			status:  http.StatusInternalServerError,
			message: "internal error",
			fields: map[string]any{
				"component": "database",
				"details":   map[string]any{"code": 500},
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "internal error",
			expectedFields: map[string]any{
				"component": "database",
				"details":   map[string]any{"code": 500},
			},
		},
		{
			name:            "implements error interface",
			status:          http.StatusForbidden,
			message:         "forbidden",
			fields:          nil,
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
			expectedFields:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error = NewAPIError(tt.status, tt.message, tt.fields)

			apiErr, ok := err.(APIError)
			assert.True(t, ok, "NewAPIError should return an APIError")
			assert.Equal(t, tt.expectedStatus, apiErr.Status)
			assert.Equal(t, tt.expectedMessage, apiErr.Message)
			assert.Equal(t, tt.expectedFields, apiErr.Fields)
			assert.Equal(t, tt.expectedMessage, err.Error())
		})
	}
}
