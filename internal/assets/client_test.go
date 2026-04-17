package assets

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

func TestClient_List_ForwardsAuthAndParams(t *testing.T) {
	var gotAuth string
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/assets", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[{"id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD","is_complex":false,"price_provider":"TWELVE_DATA"}],"total":1,"size":10,"offset":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	res, err := c.List(context.Background(), "Bearer token-xyz", ListParams{AssetType: "STOCK", Offset: 10})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.Total)
	assert.Equal(t, "Bearer token-xyz", gotAuth)

	// The query must include size, sort, order, asset_type, offset.
	assert.Contains(t, gotQuery, "size=10")
	assert.Contains(t, gotQuery, "sort=ticker")
	assert.Contains(t, gotQuery, "order=desc")
	assert.Contains(t, gotQuery, "asset_type=STOCK")
	assert.Contains(t, gotQuery, "offset=10")
}

func TestClient_List_OmitsAssetTypeWhenEmpty(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[],"total":0,"size":10,"offset":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.NoError(t, err)

	assert.NotContains(t, gotQuery, "asset_type=")
	assert.Contains(t, gotQuery, "offset=0")
}

func TestClient_List_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_List_Backend5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_List_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}
