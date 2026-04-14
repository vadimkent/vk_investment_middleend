package components

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionResponse_AuthOmittedWhenNil(t *testing.T) {
	resp := ActionResponse{Action: "refresh"}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(b), "auth"), "auth should be omitted, got: %s", b)
}

func TestActionResponse_AuthSerialized(t *testing.T) {
	resp := ActionResponse{
		Action:   "navigate",
		TargetID: "/shell",
		Auth:     &AuthPayload{Token: "t", ExpiresAt: "2026-04-15T12:00:00Z"},
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"auth":{"token":"t","expires_at":"2026-04-15T12:00:00Z"}`)
}
