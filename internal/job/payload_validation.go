package job

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/joshu-sajeev/goqueue/common"
	"github.com/joshu-sajeev/goqueue/middleware"
)

var validate = validator.New()

func validatePayload[T any](raw json.RawMessage) error {
	var payload T

	if err := json.Unmarshal(raw, &payload); err != nil {
		return common.APIError{
			Status:  http.StatusBadRequest,
			Message: "invalid payload format",
		}
	}

	if err := validate.Struct(payload); err != nil {
		return common.APIError{
			Status:  http.StatusBadRequest,
			Message: "payload validation failed",
			Fields:  middleware.FormatValidationErrors(err),
		}
	}

	return nil
}
