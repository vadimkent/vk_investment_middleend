package profile

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(handler http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	return NewClient(srv.URL, 2*time.Second), srv
}

func decodeJSON(t *testing.T, r *http.Request, v any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(r.Body).Decode(v))
}

func TestClient_GetMe_Happy(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/me", r.URL.Path)
		assert.Equal(t, "Bearer t", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"u1","email":"a@b.c","display_name":"Vadim","preferences":{"default_currency":"USD"},"created_at":"2026-01-01T00:00:00Z"}`))
	})
	defer srv.Close()

	me, err := c.GetMe(context.Background(), "Bearer t")
	require.NoError(t, err)
	assert.Equal(t, "u1", me.ID)
	require.NotNil(t, me.DisplayName)
	assert.Equal(t, "Vadim", *me.DisplayName)
	require.NotNil(t, me.Preferences.DefaultCurrency)
	assert.Equal(t, "USD", *me.Preferences.DefaultCurrency)
}

func TestClient_GetMe_Unauthorized(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	_, err := c.GetMe(context.Background(), "Bearer t")
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_UpdateProfile_ForwardsBodyAndAuth(t *testing.T) {
	var got map[string]any
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/v1/user/me", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		decodeJSON(t, r, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"u1","email":"a@b.c","preferences":{}}`))
	})
	defer srv.Close()

	body := map[string]any{"display_name": "Vadim", "preferences": map[string]any{"default_currency": "USD"}}
	_, err := c.UpdateProfile(context.Background(), "Bearer t", body)
	require.NoError(t, err)
	assert.Equal(t, "Vadim", got["display_name"])
}

func TestClient_UpdateProfile_ValidationError(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"INVALID_DISPLAY_NAME","message":"too long"}}`))
	})
	defer srv.Close()

	_, err := c.UpdateProfile(context.Background(), "Bearer t", map[string]any{})
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "INVALID_DISPLAY_NAME", be.Code)
}

func TestClient_UpdateEmail_Happy(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/me/email", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"u1","email":"new@b.c"}`))
	})
	defer srv.Close()

	err := c.UpdateEmail(context.Background(), "Bearer t", "new@b.c", "pw")
	require.NoError(t, err)
}

func TestClient_UpdateEmail_BE401IsValidationError(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"INVALID_CREDENTIALS","message":"wrong"}}`))
	})
	defer srv.Close()

	err := c.UpdateEmail(context.Background(), "Bearer t", "n@x.y", "pw")
	var be *BackendValidationError
	require.True(t, errors.As(err, &be), "got %v", err)
	assert.Equal(t, "INVALID_CREDENTIALS", be.Code)
}

func TestClient_ChangePassword_Returns204(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/me/password", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	err := c.ChangePassword(context.Background(), "Bearer t", "old", "new12345")
	require.NoError(t, err)
}

func TestClient_DeleteAccount_Returns204(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/me", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	err := c.DeleteAccount(context.Background(), "Bearer t", "pw")
	require.NoError(t, err)
}
