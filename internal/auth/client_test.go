package auth

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

func TestClient_LoginSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/auth/login", r.URL.Path)
		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "a@b.com", body["email"])
		assert.Equal(t, "pw", body["password"])
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token":      "jwt-xyz",
			"expires_at": "2026-04-15T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	res, err := c.Login(context.Background(), "a@b.com", "pw")
	require.NoError(t, err)
	assert.Equal(t, "jwt-xyz", res.Token)
	assert.Equal(t, "2026-04-15T12:00:00Z", res.ExpiresAt)
}

func TestClient_LoginInvalidCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "INVALID_CREDENTIALS"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.Login(context.Background(), "a@b.com", "wrong")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCredentials))
}

func TestClient_RegisterSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/auth/register", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "u1", "email": "a@b.com"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	err := c.Register(context.Background(), "a@b.com", "pw")
	require.NoError(t, err)
}

func TestClient_RegisterDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "REGISTRATION_DISABLED"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	err := c.Register(context.Background(), "a@b.com", "pw")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRegistrationDisabled))
}

func TestClient_RegisterEmailExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "EMAIL_ALREADY_EXISTS"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	err := c.Register(context.Background(), "a@b.com", "pw")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmailAlreadyExists))
}
