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

func TestActionResponse_WithAuth(t *testing.T) {
	resp := NavigateResponse("/shell", nil).WithAuth("tok-1", "2026-04-15T12:00:00Z")
	require.NotNil(t, resp.Auth)
	assert.Equal(t, "tok-1", resp.Auth.Token)
	assert.Equal(t, "2026-04-15T12:00:00Z", resp.Auth.ExpiresAt)
	assert.Equal(t, "navigate", resp.Action)
	assert.Equal(t, "/shell", resp.TargetID)
}

func TestActionResponse_ExpiresAtOmittedWhenEmpty(t *testing.T) {
	resp := ActionResponse{Action: "navigate", Auth: &AuthInfo{Token: "t"}}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(b), "expires_at"), "expires_at should be omitted, got: %s", b)
}

func TestActionResponse_AuthSerialized(t *testing.T) {
	resp := ActionResponse{
		Action:   "navigate",
		TargetID: "/shell",
		Auth:     &AuthInfo{Token: "t", ExpiresAt: "2026-04-15T12:00:00Z"},
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"auth":{"token":"t","expires_at":"2026-04-15T12:00:00Z"}`)
}

func TestToggleSidebar_ReturnsClickToggleAction(t *testing.T) {
	action := ToggleSidebar()
	assert.Equal(t, "click", action.Trigger)
	assert.Equal(t, "toggle_sidebar", action.Type)
	assert.Empty(t, action.URL)
	assert.Empty(t, action.Endpoint)
	assert.Empty(t, action.TargetID)
}

func TestToggleSidebar_JSONShape(t *testing.T) {
	b, err := json.Marshal(ToggleSidebar())
	require.NoError(t, err)
	assert.Equal(t, `{"trigger":"click","type":"toggle_sidebar"}`, string(b))
}
