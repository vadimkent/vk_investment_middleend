package portfolio

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubIncludeClosedFetcher struct {
	positions        []Position
	err              error
	gotAuth          string
	gotIncludeClosed bool
	called           bool
}

func (s *stubIncludeClosedFetcher) GetPositions(ctx context.Context, auth string, includeClosed bool) ([]Position, error) {
	s.called = true
	s.gotAuth = auth
	s.gotIncludeClosed = includeClosed
	return s.positions, s.err
}

func setupIncludeClosedRouter(f positionsFetcherWithInclude) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/portfolio/include_closed", NewIncludeClosedHandler(f).Post)
	return r
}

func doPost(t *testing.T, r *gin.Engine, body string, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/actions/portfolio/include_closed", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestIncludeClosedHandler_SuccessTrueReturnsReplaceActionResponse(t *testing.T) {
	v := 100.0
	f := &stubIncludeClosedFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Currency: "USD", CurrentValue: &v}}}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{"include_closed":true}`, "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "positions-table-card", resp["target_id"])
	tree, ok := resp["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "card", tree["type"])
	assert.Equal(t, "positions-table-card", tree["id"])

	assert.True(t, f.called)
	assert.Equal(t, "Bearer tok", f.gotAuth)
	assert.True(t, f.gotIncludeClosed)
}

func TestIncludeClosedHandler_SuccessFalsePassesFalse(t *testing.T) {
	v := 100.0
	f := &stubIncludeClosedFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Currency: "USD", CurrentValue: &v}}}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{"include_closed":false}`, "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, f.gotIncludeClosed)
}

func TestIncludeClosedHandler_MalformedJSONReturns400(t *testing.T) {
	f := &stubIncludeClosedFetcher{}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `not json`, "Bearer tok")
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "BAD_REQUEST")
	assert.False(t, f.called)
}

func TestIncludeClosedHandler_MissingFieldReturns400(t *testing.T) {
	f := &stubIncludeClosedFetcher{}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{}`, "Bearer tok")
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, f.called)
}

func TestIncludeClosedHandler_BackendUnauthorizedReturns401WithRedirect(t *testing.T) {
	f := &stubIncludeClosedFetcher{err: ErrUnauthorized}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{"include_closed":true}`, "Bearer x")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"unauthorized"`)
	assert.Contains(t, w.Body.String(), `"redirect":"/screens/login"`)
}

func TestIncludeClosedHandler_BackendErrorReturns502(t *testing.T) {
	f := &stubIncludeClosedFetcher{err: ErrBackend}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{"include_closed":true}`, "Bearer x")
	require.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "BACKEND_ERROR")
}
