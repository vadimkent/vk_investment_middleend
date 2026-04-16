package portfolio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAllocationFetcher struct {
	positions []Position
	err       error
	gotAuth   string
	called    bool
}

func (s *stubAllocationFetcher) GetPositions(ctx context.Context, auth string, includeClosed, live, refresh bool) (*PortfolioResponse, error) {
	s.called = true
	s.gotAuth = auth
	return &PortfolioResponse{Positions: s.positions}, s.err
}

func setupAllocationRouter(f allocationFetcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/actions/portfolio/allocation", NewAllocationHandler(f).Get)
	return r
}

func allocationGet(t *testing.T, r *gin.Engine, query, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/actions/portfolio/allocation?"+query, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAllocationHandler_SuccessReturnsReplaceActionResponse(t *testing.T) {
	v := 1000.0
	f := &stubAllocationFetcher{positions: []Position{
		{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
	}}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "group_by=asset&currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "allocation-section", resp["target_id"])
	tree, ok := resp["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "column", tree["type"])
	assert.Equal(t, "allocation-section", tree["id"])

	assert.True(t, f.called)
	assert.Equal(t, "Bearer tok", f.gotAuth)
}

func TestAllocationHandler_DefaultsGroupByAsset(t *testing.T) {
	v := 1000.0
	f := &stubAllocationFetcher{positions: []Position{
		{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
	}}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	// The tree should contain the asset group-by button selected solid.
	assert.Contains(t, w.Body.String(), `"allocation-group-by-asset"`)
}

func TestAllocationHandler_InvalidGroupByReturns400(t *testing.T) {
	f := &stubAllocationFetcher{}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "group_by=xxx&currency=USD", "Bearer tok")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, f.called)
}

func TestAllocationHandler_BackendUnauthorizedReturns401WithRedirect(t *testing.T) {
	f := &stubAllocationFetcher{err: ErrUnauthorized}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "group_by=asset&currency=USD", "Bearer x")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"unauthorized"`)
	assert.Contains(t, w.Body.String(), `"redirect":"/screens/login"`)
}

func TestAllocationHandler_BackendErrorReturns502(t *testing.T) {
	f := &stubAllocationFetcher{err: ErrBackend}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "group_by=asset&currency=USD", "Bearer x")
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "BACKEND_ERROR")
}
