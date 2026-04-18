package assets

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetAsset_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/assets/a1", r.URL.Path)
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD","is_complex":false,"price_provider":"TWELVE_DATA","external_ticker":"AAPL"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	a, err := c.GetAsset(context.Background(), "Bearer tok", "a1")
	require.NoError(t, err)
	assert.Equal(t, "AAPL", a.Ticker)
	require.NotNil(t, a.PriceProvider)
	assert.Equal(t, "TWELVE_DATA", *a.PriceProvider)
	require.NotNil(t, a.ExternalTicker)
	assert.Equal(t, "AAPL", *a.ExternalTicker)
}

func TestClient_GetAsset_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.GetAsset(context.Background(), "Bearer tok", "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAssetNotFound))
}

func TestClient_CreateAsset_ForwardsBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/assets", r.URL.Path)
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"a2","ticker":"TSLA","name":"Tesla","asset_type":"STOCK","currency":"USD","is_complex":false}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	body := map[string]any{"ticker": "TSLA", "name": "Tesla", "asset_type": "STOCK", "currency": "USD"}
	a, err := c.CreateAsset(context.Background(), "Bearer tok", body)
	require.NoError(t, err)
	assert.Equal(t, "TSLA", a.Ticker)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &parsed))
	assert.Equal(t, "TSLA", parsed["ticker"])
}

func TestClient_CreateAsset_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"ASSET_ALREADY_EXISTS","message":"Asset exists"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateAsset(context.Background(), "Bearer tok", map[string]any{})
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be), "want BackendValidationError, got %T", err)
	assert.Equal(t, "ASSET_ALREADY_EXISTS", be.Code)
}

func TestClient_UpdateAsset_PATCH(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		assert.Equal(t, "/v1/assets/a1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"a1","ticker":"AAPL","name":"Apple Inc","asset_type":"STOCK","currency":"USD","is_complex":false}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	body := map[string]any{"name": "Apple Inc"}
	a, err := c.UpdateAsset(context.Background(), "Bearer tok", "a1", body)
	require.NoError(t, err)
	assert.Equal(t, "Apple Inc", a.Name)
	assert.Equal(t, http.MethodPatch, gotMethod)
}

func TestClient_DeleteAsset_PassesForce(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteAsset(context.Background(), "Bearer tok", "a1", true)
	require.NoError(t, err)
	assert.Contains(t, gotQuery, "force=true")
}

func TestClient_DeleteAsset_AssetHasData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"ASSET_HAS_DATA","message":"Has data"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteAsset(context.Background(), "Bearer tok", "a1", false)
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "ASSET_HAS_DATA", be.Code)
}

func TestClient_CreateAsset_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateAsset(context.Background(), "", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_CreateAsset_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateAsset(context.Background(), "Bearer tok", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

// silence "imported and not used: strings" if the file above doesn't otherwise use it.
var _ = strings.TrimSpace
