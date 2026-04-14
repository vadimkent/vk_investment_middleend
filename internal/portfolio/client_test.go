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
	positions, err := c.GetPositions(context.Background(), "Bearer abc", false)
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
	_, err := c.GetPositions(context.Background(), "Bearer bad", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_GetPositions_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetPositions(context.Background(), "Bearer x", false)
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
	_, err := c.GetPositions(context.Background(), "Bearer x", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_GetEvolutionLast_ForwardsAuthAndQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/portfolio/evolution", r.URL.Path)
		assert.Equal(t, "2", r.URL.Query().Get("last"))
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"evolution":[{"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"1000.00","currency":"USD"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	pts, err := c.GetEvolutionLast(context.Background(), "Bearer tok", 2)
	require.NoError(t, err)
	require.Len(t, pts, 1)
	assert.Equal(t, "s1", pts[0].SnapshotID)
}

func TestClient_GetEvolutionLast_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolutionLast(context.Background(), "Bearer x", 2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_GetEvolutionLast_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolutionLast(context.Background(), "Bearer x", 2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_GetEvolutionLast_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolutionLast(context.Background(), "Bearer x", 2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_GetPositions_ForwardsIncludeClosed(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"positions":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetPositions(context.Background(), "Bearer t", true)
	require.NoError(t, err)
	assert.Equal(t, "include_closed=true", gotQuery)

	_, err = c.GetPositions(context.Background(), "Bearer t", false)
	require.NoError(t, err)
	assert.Equal(t, "include_closed=false", gotQuery)
}

func TestClient_GetEvolution_WithFromPointsCurrency(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/portfolio/evolution", r.URL.Path)
		gotQuery = r.URL.RawQuery
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"evolution":[{"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"1000.00","currency":"USD"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	pts, err := c.GetEvolution(context.Background(), "Bearer tok", EvolutionQuery{From: &from, Points: 100, Currency: "USD"})
	require.NoError(t, err)
	require.Len(t, pts, 1)
	assert.Contains(t, gotQuery, "from=2026-01-01T00%3A00%3A00Z")
	assert.Contains(t, gotQuery, "points=100")
	assert.Contains(t, gotQuery, "currency=USD")
}

func TestClient_GetEvolution_OmitsUnsetParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"evolution":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolution(context.Background(), "Bearer t", EvolutionQuery{Points: 100})
	require.NoError(t, err)
	assert.NotContains(t, gotQuery, "from=")
	assert.NotContains(t, gotQuery, "currency=")
	assert.Contains(t, gotQuery, "points=100")
}

func TestClient_GetEvolution_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolution(context.Background(), "Bearer bad", EvolutionQuery{Points: 100})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_GetEvolution_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolution(context.Background(), "Bearer x", EvolutionQuery{Points: 100})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}
