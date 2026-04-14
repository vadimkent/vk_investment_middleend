package portfolio

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

func TestClient_GetPositions_ForwardsAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/portfolio", r.URL.Path)
		assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"positions":[{"asset_id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD","quantity":"1","avg_cost":"100","total_cost":"100","current_price":"110","current_value":"110","unrealized_pnl":"10","realized_pnl":"0","last_snapshot_at":"2024-06-01T10:00:00Z"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	positions, err := c.GetPositions(context.Background(), "Bearer abc")
	require.NoError(t, err)
	require.Len(t, positions, 1)
	assert.Equal(t, "AAPL", positions[0].Ticker)
}

func TestClient_GetPositions_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetPositions(context.Background(), "Bearer bad")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_GetPositions_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetPositions(context.Background(), "Bearer x")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_GetPositions_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetPositions(context.Background(), "Bearer x")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}
