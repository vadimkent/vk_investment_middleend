package profile

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigClient_Get_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/config", r.URL.Path)
		assert.Equal(t, "Bearer t", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"asset_types":[],"currencies":["USD","EUR","ARS"],"price_providers":[],"sources":[]}`))
	}))
	defer srv.Close()

	c := NewConfigClient(srv.URL, 2*time.Second)
	cfg, err := c.GetConfig(context.Background(), "Bearer t")
	require.NoError(t, err)
	assert.Equal(t, []string{"USD", "EUR", "ARS"}, cfg.Currencies)
}

func TestConfigClient_Get_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewConfigClient(srv.URL, 2*time.Second)
	_, err := c.GetConfig(context.Background(), "Bearer t")
	assert.True(t, errors.Is(err, ErrUnauthorized))
}
