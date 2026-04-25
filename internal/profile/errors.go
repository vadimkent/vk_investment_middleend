package profile

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Sentinel errors emitted by the profile clients.
var (
	ErrUnauthorized = errors.New("backend unauthorized")
	ErrBackend      = errors.New("backend error")
)

// BackendValidationError is returned for the 4xx codes the BE documents:
// INVALID_DISPLAY_NAME, INVALID_CURRENCY, MISSING_FIELDS, INVALID_CREDENTIALS,
// EMAIL_ALREADY_EXISTS, INVALID_PASSWORD.
type BackendValidationError struct {
	Code    string
	Message string
}

func (e *BackendValidationError) Error() string {
	return fmt.Sprintf("backend validation: %s: %s", e.Code, e.Message)
}

// parseValidationError pulls {"error":{"code":"...","message":"..."}} out of a
// 4xx body. If the body is unrecognised, it returns a generic error wrapped
// around ErrBackend so the caller still maps to 502.
func parseValidationError(body []byte) error {
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.Error.Code == "" {
		return fmt.Errorf("%w: malformed validation error", ErrBackend)
	}
	return &BackendValidationError{Code: env.Error.Code, Message: env.Error.Message}
}
