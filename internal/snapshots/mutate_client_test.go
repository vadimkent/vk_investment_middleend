package snapshots

import (
	"bytes"
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

// snapshotBody is a minimal but valid snapshot JSON for test servers.
const snapshotBody = `{"id":"s1","recorded_at":"2026-01-01","is_full_snapshot":true,"notes":"","entries":[],"created_at":"2026-01-01T00:00:00Z"}`

// ─── CreateSnapshot ───────────────────────────────────────────────────────────

func TestMutate_CreateSnapshot_HappyPath(t *testing.T) {
	var gotMethod, gotPath, gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(snapshotBody))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	body := map[string]any{"recorded_at": "2026-01-01", "is_full_snapshot": true}
	snap, err := c.CreateSnapshot(context.Background(), "Bearer tok", body)
	require.NoError(t, err)
	assert.Equal(t, "s1", snap.ID)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/v1/snapshots", gotPath)
	assert.Equal(t, "application/json", gotCT)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &parsed))
	assert.Equal(t, "2026-01-01", parsed["recorded_at"])
}

func TestMutate_CreateSnapshot_ValidationError_FutureDated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"FUTURE_DATED_SNAPSHOT","message":"Cannot record a future-dated snapshot"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateSnapshot(context.Background(), "Bearer tok", map[string]any{})
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be), "want BackendValidationError, got %T", err)
	assert.Equal(t, "FUTURE_DATED_SNAPSHOT", be.Code)
	assert.Equal(t, "Cannot record a future-dated snapshot", be.Message)
}

func TestMutate_CreateSnapshot_ValidationError_ConflictingValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"CONFLICTING_SNAPSHOT_VALUE","message":"Conflicting value"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateSnapshot(context.Background(), "Bearer tok", map[string]any{})
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "CONFLICTING_SNAPSHOT_VALUE", be.Code)
	assert.Equal(t, "Conflicting value", be.Message)
}

func TestMutate_CreateSnapshot_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateSnapshot(context.Background(), "", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

// ─── AutoSnapshot ─────────────────────────────────────────────────────────────

func TestMutate_AutoSnapshot_HappyPath_WithWarnings(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/snapshots/auto", r.URL.Path)
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"snapshot":` + snapshotBody + `,"warnings":[{"asset_id":"a1","ticker":"AAPL","error":"provider timeout"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	result, err := c.AutoSnapshot(context.Background(), "Bearer tok", "hi")
	require.NoError(t, err)
	assert.Equal(t, "s1", result.Snapshot.ID)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, "a1", result.Warnings[0].AssetID)
	assert.Equal(t, "AAPL", result.Warnings[0].Ticker)
	assert.Equal(t, "provider timeout", result.Warnings[0].Error)

	// notes="hi" → body must contain {"notes":"hi"}
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &parsed))
	assert.Equal(t, "hi", parsed["notes"])
}

func TestMutate_AutoSnapshot_EmptyNotes_SendsEmptyBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"snapshot":` + snapshotBody + `}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	result, err := c.AutoSnapshot(context.Background(), "Bearer tok", "")
	require.NoError(t, err)
	assert.Equal(t, "s1", result.Snapshot.ID)

	// empty notes → body must be exactly {} (no "notes" key)
	assert.Equal(t, []byte("{}"), bytes.TrimSpace(gotBody))
}

func TestMutate_AutoSnapshot_NoWarningsInResponse_ReturnsNilWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"snapshot":` + snapshotBody + `}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	result, err := c.AutoSnapshot(context.Background(), "Bearer tok", "")
	require.NoError(t, err)
	assert.Nil(t, result.Warnings)
}

func TestMutate_AutoSnapshot_422_NoPriceProviders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"NO_PRICE_PROVIDERS_CONFIGURED","message":"No providers"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.AutoSnapshot(context.Background(), "Bearer tok", "")
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be), "want BackendValidationError, got %T", err)
	assert.Equal(t, "NO_PRICE_PROVIDERS_CONFIGURED", be.Code)
}

func TestMutate_AutoSnapshot_502_WithStructuredCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"code":"ALL_PROVIDERS_FAILED","message":"All providers failed"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.AutoSnapshot(context.Background(), "Bearer tok", "")
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be), "want BackendValidationError for 502 with code, got %T", err)
	assert.Equal(t, "ALL_PROVIDERS_FAILED", be.Code)
	assert.Equal(t, "All providers failed", be.Message)
}

func TestMutate_AutoSnapshot_502_PlainText_ReturnsErrBackend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("Bad Gateway"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.AutoSnapshot(context.Background(), "Bearer tok", "")
	require.Error(t, err)
	var be *BackendValidationError
	assert.False(t, errors.As(err, &be), "should not be BackendValidationError for unstructured 502, got %T", err)
	assert.True(t, errors.Is(err, ErrBackend))
}

// ─── UpdateSnapshot ───────────────────────────────────────────────────────────

func TestMutate_UpdateSnapshot_HappyPath(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"s1","recorded_at":"2026-01-01","is_full_snapshot":true,"notes":"updated","entries":[],"created_at":"2026-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	snap, err := c.UpdateSnapshot(context.Background(), "Bearer tok", "s1", map[string]any{"notes": "updated"})
	require.NoError(t, err)
	assert.Equal(t, "updated", snap.Notes)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, "/v1/snapshots/s1", gotPath)
}

func TestMutate_UpdateSnapshot_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"SNAPSHOT_NOT_FOUND","message":"not found"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.UpdateSnapshot(context.Background(), "Bearer tok", "missing", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSnapshotNotFound))
}

func TestMutate_UpdateSnapshot_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"DUPLICATE_SNAPSHOT_ENTRY","message":"Duplicate"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.UpdateSnapshot(context.Background(), "Bearer tok", "s1", map[string]any{})
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "DUPLICATE_SNAPSHOT_ENTRY", be.Code)
}

// ─── DeleteSnapshot ───────────────────────────────────────────────────────────

func TestMutate_DeleteSnapshot_HappyPath(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteSnapshot(context.Background(), "Bearer tok", "s1")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/v1/snapshots/s1", gotPath)
}

func TestMutate_DeleteSnapshot_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteSnapshot(context.Background(), "Bearer tok", "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSnapshotNotFound))
}
