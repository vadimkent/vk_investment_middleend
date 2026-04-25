package profile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValidationError_KnownCode(t *testing.T) {
	body := []byte(`{"error":{"code":"INVALID_DISPLAY_NAME","message":"too long"}}`)
	err := parseValidationError(body)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "INVALID_DISPLAY_NAME", be.Code)
	assert.Equal(t, "too long", be.Message)
}

func TestParseValidationError_Malformed(t *testing.T) {
	err := parseValidationError([]byte(`not json`))
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestParseValidationError_NoCode(t *testing.T) {
	err := parseValidationError([]byte(`{"error":{}}`))
	assert.True(t, errors.Is(err, ErrBackend))
}
