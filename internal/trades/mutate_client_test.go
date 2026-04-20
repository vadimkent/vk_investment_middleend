package trades

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetTrade_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/trades/t1", r.URL.Path)
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"t1","asset_id":"a1","trade_type":"BUY","quantity":"10","price_per_unit":"100.00","fees":"1.00","date":"2025-01-01","source":"MANUAL","notes":"n","created_at":"2025-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	tr, err := c.GetTrade(context.Background(), "Bearer tok", "t1")
	require.NoError(t, err)
	assert.Equal(t, "t1", tr.ID)
	assert.Equal(t, "a1", tr.AssetID)
	assert.Equal(t, "BUY", tr.TradeType)
	assert.Equal(t, "10", tr.Quantity)
}

func TestClient_GetTrade_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.GetTrade(context.Background(), "", "t1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_GetTrade_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.GetTrade(context.Background(), "Bearer tok", "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTradeNotFound))
}

func TestClient_GetTrade_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"INSUFFICIENT_QUANTITY","message":"Insufficient"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.GetTrade(context.Background(), "Bearer tok", "t1")
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be), "want BackendValidationError, got %T", err)
	assert.Equal(t, "INSUFFICIENT_QUANTITY", be.Code)
	assert.Equal(t, "Insufficient", be.Message)
}

func TestClient_GetTrade_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.GetTrade(context.Background(), "Bearer tok", "t1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_CreateTrade_ForwardsBody(t *testing.T) {
	var gotBody []byte
	var gotMethod, gotPath, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"t2","asset_id":"a1","trade_type":"BUY","quantity":"5","price_per_unit":"10.00","fees":"0","date":"2025-02-01","source":"MANUAL","notes":"","created_at":"2025-02-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	body := map[string]any{
		"asset_id":       "a1",
		"trade_type":     "BUY",
		"quantity":       "5",
		"price_per_unit": "10.00",
		"fees":           "0",
		"date":           "2025-02-01",
	}
	tr, err := c.CreateTrade(context.Background(), "Bearer tok", body)
	require.NoError(t, err)
	assert.Equal(t, "t2", tr.ID)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/v1/trades", gotPath)
	assert.Equal(t, "application/json", gotCT)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &parsed))
	assert.Equal(t, "a1", parsed["asset_id"])
	assert.Equal(t, "BUY", parsed["trade_type"])
	assert.Equal(t, "5", parsed["quantity"])
	assert.Equal(t, "10.00", parsed["price_per_unit"])
	assert.Equal(t, "0", parsed["fees"])
	assert.Equal(t, "2025-02-01", parsed["date"])
}

func TestClient_CreateTrade_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateTrade(context.Background(), "", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_CreateTrade_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"INSUFFICIENT_QUANTITY","message":"Insufficient"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateTrade(context.Background(), "Bearer tok", map[string]any{})
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "INSUFFICIENT_QUANTITY", be.Code)
	assert.Equal(t, "Insufficient", be.Message)
}

func TestClient_CreateTrade_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateTrade(context.Background(), "Bearer tok", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_UpdateTrade_PATCH(t *testing.T) {
	var gotMethod, gotPath, gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"t1","asset_id":"a1","trade_type":"BUY","quantity":"10","price_per_unit":"100.00","fees":"1.00","date":"2025-01-01","source":"MANUAL","notes":"updated","created_at":"2025-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	body := map[string]any{"notes": "updated"}
	tr, err := c.UpdateTrade(context.Background(), "Bearer tok", "t1", body)
	require.NoError(t, err)
	assert.Equal(t, "updated", tr.Notes)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, "/v1/trades/t1", gotPath)
	assert.Equal(t, "application/json", gotCT)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &parsed))
	assert.Equal(t, "updated", parsed["notes"])
}

func TestClient_UpdateTrade_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.UpdateTrade(context.Background(), "Bearer tok", "missing", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTradeNotFound))
}

func TestClient_UpdateTrade_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"INSUFFICIENT_QUANTITY","message":"Insufficient"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.UpdateTrade(context.Background(), "Bearer tok", "t1", map[string]any{})
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "INSUFFICIENT_QUANTITY", be.Code)
}

func TestClient_UpdateTrade_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.UpdateTrade(context.Background(), "", "t1", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_DeleteTrade_HappyPath(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteTrade(context.Background(), "Bearer tok", "t1")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/v1/trades/t1", gotPath)
}

func TestClient_DeleteTrade_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteTrade(context.Background(), "", "t1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_DeleteTrade_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteTrade(context.Background(), "Bearer tok", "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTradeNotFound))
}

func TestClient_DeleteTrade_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"TRADE_LOCKED","message":"Locked"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteTrade(context.Background(), "Bearer tok", "t1")
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "TRADE_LOCKED", be.Code)
	assert.Equal(t, "Locked", be.Message)
}

func TestClient_DeleteTrade_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteTrade(context.Background(), "Bearer tok", "t1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_parseValidationError_MalformedFallsBackToErrBackend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.GetTrade(context.Background(), "Bearer tok", "t1")
	require.Error(t, err)
	var be *BackendValidationError
	assert.False(t, errors.As(err, &be), "should not be BackendValidationError, got %T", err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_BackendValidationError_ErrorString(t *testing.T) {
	e := &BackendValidationError{Code: "X", Message: "Y"}
	assert.Equal(t, "backend validation: X: Y", e.Error())
}
