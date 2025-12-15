package common

import "fmt"

type APIError struct {
	Status  int            `json:"-"`
	Message string         `json:"error"`
	Fields  map[string]any `json:"fields,omitempty"`
}

func (e APIError) Error() string {
	return e.Message
}

func Errf(status int, format string, args ...any) APIError {
	return APIError{Status: status, Message: fmt.Sprintf(format, args...)}
}

// NewAPIError creates an APIError with status, message, and optional fields
func NewAPIError(status int, message string, fields map[string]any) APIError {
	return APIError{
		Status:  status,
		Message: message,
		Fields:  fields,
	}
}
