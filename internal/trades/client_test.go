package trades

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

func TestClient_List_ForwardsAuthAndDefaultParams(t *testing.T) {
	var gotAuth string
	var gotQuery map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/trades", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		q := r.URL.Query()
		gotQuery = map[string]string{
			"size":       q.Get("size"),
			"sort":       q.Get("sort"),
			"order":      q.Get("order"),
			"offset":     q.Get("offset"),
			"asset_id":   q.Get("asset_id"),
			"trade_type": q.Get("trade_type"),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"trades":[{"id":"t1","asset_id":"a1","trade_type":"BUY","quantity":"10","price_per_unit":"100.00","fees":"1.00","date":"2025-01-02","source":"MANUAL","notes":"note","created_at":"2025-01-02T10:00:00Z"}],"total":1,"size":10,"offset":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	res, err := c.List(context.Background(), "Bearer token-xyz", ListParams{})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.Total)
	require.Len(t, res.Trades, 1)
	assert.Equal(t, "t1", res.Trades[0].ID)
	assert.Equal(t, "a1", res.Trades[0].AssetID)
	assert.Equal(t, "BUY", res.Trades[0].TradeType)
	assert.Equal(t, "10", res.Trades[0].Quantity)
	assert.Equal(t, "100.00", res.Trades[0].PricePerUnit)

	assert.Equal(t, "Bearer token-xyz", gotAuth)
	assert.Equal(t, "10", gotQuery["size"])
	assert.Equal(t, "date", gotQuery["sort"])
	assert.Equal(t, "desc", gotQuery["order"])
	assert.Equal(t, "0", gotQuery["offset"])
	assert.Equal(t, "", gotQuery["asset_id"])
	assert.Equal(t, "", gotQuery["trade_type"])
}

func TestClient_List_WithFilters(t *testing.T) {
	var gotQuery map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotQuery = map[string]string{
			"size":       q.Get("size"),
			"sort":       q.Get("sort"),
			"order":      q.Get("order"),
			"offset":     q.Get("offset"),
			"asset_id":   q.Get("asset_id"),
			"trade_type": q.Get("trade_type"),
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"trades":[],"total":0,"size":10,"offset":20}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{AssetID: "uuid", TradeType: "SELL", Offset: 20})
	require.NoError(t, err)

	assert.Equal(t, "10", gotQuery["size"])
	assert.Equal(t, "date", gotQuery["sort"])
	assert.Equal(t, "desc", gotQuery["order"])
	assert.Equal(t, "20", gotQuery["offset"])
	assert.Equal(t, "uuid", gotQuery["asset_id"])
	assert.Equal(t, "SELL", gotQuery["trade_type"])
}

func TestClient_List_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	res, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_List_Backend5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	res, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_List_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	res, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_List_NoAuthHeaderWhenEmpty(t *testing.T) {
	var hadAuthHeader bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadAuthHeader = r.Header["Authorization"]
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"trades":[],"total":0,"size":10,"offset":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "", ListParams{})
	require.NoError(t, err)
	assert.False(t, hadAuthHeader, "Authorization header must not be set when authorization is empty")
}
