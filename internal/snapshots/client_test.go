package snapshots

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// canned JSON helpers
const cannedList = `{"snapshots":[{"id":"s1","recorded_at":"2025-01-02T10:00:00Z","is_full_snapshot":true,"notes":"note","entries":[{"asset_id":"a1","quantity":"10.5","current_price":"150.00","current_value_override":null,"source":"MANUAL"}],"created_at":"2025-01-02T10:00:00Z"}],"total":1,"size":10,"offset":0}`
const cannedEmptyList = `{"snapshots":[],"total":0,"size":10,"offset":20}`
const cannedSnapshot = `{"id":"s1","recorded_at":"2025-01-02T10:00:00Z","is_full_snapshot":true,"notes":"note","entries":[{"asset_id":"a1","quantity":"10.5","current_price":"150.00","current_value_override":null,"source":"MANUAL"}],"created_at":"2025-01-02T10:00:00Z"}`

func TestClient_List_DefaultParams(t *testing.T) {
	var gotAuth string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/snapshots", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedList))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	res, err := c.List(context.Background(), "Bearer token-xyz", ListParams{})
	require.NoError(t, err)
	require.NotNil(t, res)

	// Verify parsed result.
	assert.Equal(t, 1, res.Total)
	require.Len(t, res.Snapshots, 1)
	assert.Equal(t, "s1", res.Snapshots[0].ID)
	assert.True(t, res.Snapshots[0].IsFullSnapshot)
	require.Len(t, res.Snapshots[0].Entries, 1)
	assert.Equal(t, "a1", res.Snapshots[0].Entries[0].AssetID)
	assert.Equal(t, "10.5", res.Snapshots[0].Entries[0].Quantity)

	// Verify forwarded auth.
	assert.Equal(t, "Bearer token-xyz", gotAuth)

	// Verify required query params.
	assert.Equal(t, "10", gotQuery.Get("size"))
	assert.Equal(t, "recorded_at", gotQuery.Get("sort"))
	assert.Equal(t, "desc", gotQuery.Get("order"))
	assert.Equal(t, "0", gotQuery.Get("offset"))

	// is_full_snapshot must NOT be present when nil.
	_, present := gotQuery["is_full_snapshot"]
	assert.False(t, present, "is_full_snapshot must not be sent when ListParams.IsFullSnapshot is nil")
}

func TestClient_List_IsFullSnapshotTrue(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedEmptyList))
	}))
	defer srv.Close()

	trueVal := true
	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{IsFullSnapshot: &trueVal})
	require.NoError(t, err)

	assert.Equal(t, "true", gotQuery.Get("is_full_snapshot"))
}

func TestClient_List_IsFullSnapshotFalse(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedEmptyList))
	}))
	defer srv.Close()

	falseVal := false
	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{IsFullSnapshot: &falseVal})
	require.NoError(t, err)

	assert.Equal(t, "false", gotQuery.Get("is_full_snapshot"))
}

func TestClient_List_Offset(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedEmptyList))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{Offset: 20})
	require.NoError(t, err)

	assert.Equal(t, "20", gotQuery.Get("offset"))
}

func TestClient_List_ForwardsAuthorization(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedEmptyList))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer secret", ListParams{})
	require.NoError(t, err)
	assert.Equal(t, "Bearer secret", gotAuth)
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

func TestClient_List_MalformedJSON(t *testing.T) {
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

func TestClient_GetSnapshot_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/snapshots/s1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedSnapshot))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	snap, err := c.GetSnapshot(context.Background(), "Bearer t", "s1")
	require.NoError(t, err)
	require.NotNil(t, snap)
	assert.Equal(t, "s1", snap.ID)
	assert.True(t, snap.IsFullSnapshot)
	assert.Equal(t, "2025-01-02T10:00:00Z", snap.RecordedAt)
	require.Len(t, snap.Entries, 1)
	assert.Equal(t, "a1", snap.Entries[0].AssetID)
	assert.Equal(t, "150.00", snap.Entries[0].CurrentPrice)
	assert.Equal(t, "", snap.Entries[0].CurrentValueOverride) // null → ""
}

func TestClient_GetSnapshot_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	snap, err := c.GetSnapshot(context.Background(), "Bearer t", "missing")
	require.Error(t, err)
	assert.Nil(t, snap)
	assert.True(t, errors.Is(err, ErrSnapshotNotFound))
}

func TestClient_GetSnapshot_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	snap, err := c.GetSnapshot(context.Background(), "Bearer t", "s1")
	require.Error(t, err)
	assert.Nil(t, snap)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_GetSnapshot_ForwardsAuthorization(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedSnapshot))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.GetSnapshot(context.Background(), "Bearer secret", "s1")
	require.NoError(t, err)
	assert.Equal(t, "Bearer secret", gotAuth)
}
