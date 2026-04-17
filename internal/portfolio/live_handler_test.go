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

type stubLiveFetcher struct {
	resp       *PortfolioResponse
	posErr     error
	evo        []EvolutionPoint
	evoErr     error
	gotAuth    string
	gotLive    bool
	gotRefresh bool
}

func (s *stubLiveFetcher) GetPositions(ctx context.Context, auth string, includeClosed, live, refresh bool) (*PortfolioResponse, error) {
	s.gotAuth = auth
	s.gotLive = live
	s.gotRefresh = refresh
	return s.resp, s.posErr
}

func (s *stubLiveFetcher) GetEvolutionLast(ctx context.Context, auth string, n int) ([]EvolutionPoint, error) {
	return s.evo, s.evoErr
}

func setupLiveRouter(f liveFetcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/actions/portfolio/live_data", NewLiveHandler(f).Get)
	return r
}

func liveGet(t *testing.T, r *gin.Engine, query, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/actions/portfolio/live_data?"+query, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLiveHandler_ToggleOnReturnsLiveDataSection(t *testing.T) {
	v := 1000.0
	f := &stubLiveFetcher{resp: &PortfolioResponse{
		Positions: []Position{
			{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
		},
		IsLive: true,
	}}
	r := setupLiveRouter(f)

	w := liveGet(t, r, "live=true", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "live-data-section", resp["target_id"])
	tree, ok := resp["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "column", tree["type"])
	assert.Equal(t, "live-data-section", tree["id"])

	assert.Equal(t, "Bearer tok", f.gotAuth)
	assert.True(t, f.gotLive)
}

func TestLiveHandler_ToggleOffReturnsStandardDataSection(t *testing.T) {
	v := 1000.0
	f := &stubLiveFetcher{resp: &PortfolioResponse{
		Positions: []Position{
			{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
		},
		IsLive: false,
	}}
	r := setupLiveRouter(f)

	w := liveGet(t, r, "live=false", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "live-data-section", resp["target_id"])

	assert.False(t, f.gotLive)
}

func TestLiveHandler_RefreshParam(t *testing.T) {
	v := 1000.0
	f := &stubLiveFetcher{resp: &PortfolioResponse{
		Positions: []Position{
			{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
		},
		IsLive: true,
	}}
	r := setupLiveRouter(f)

	w := liveGet(t, r, "live=true&refresh=true", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	assert.True(t, f.gotLive)
	assert.True(t, f.gotRefresh)
}

func TestLiveHandler_DefaultsToStandard(t *testing.T) {
	v := 1000.0
	f := &stubLiveFetcher{resp: &PortfolioResponse{
		Positions: []Position{
			{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
		},
		IsLive: false,
	}}
	r := setupLiveRouter(f)

	w := liveGet(t, r, "", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	assert.False(t, f.gotLive)
	assert.False(t, f.gotRefresh)
}

func TestLiveHandler_BackendUnauthorized401(t *testing.T) {
	f := &stubLiveFetcher{posErr: ErrUnauthorized}
	r := setupLiveRouter(f)

	w := liveGet(t, r, "live=true", "Bearer x")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"unauthorized"`)
	assert.Contains(t, w.Body.String(), `"redirect":"/login"`)
}

func TestLiveHandler_BackendError502(t *testing.T) {
	f := &stubLiveFetcher{posErr: ErrBackend}
	r := setupLiveRouter(f)

	w := liveGet(t, r, "live=true", "Bearer x")
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "BACKEND_ERROR")
}

func TestLiveHandler_EvolutionUnauthorizedFallback(t *testing.T) {
	// When evolution fetch fails with unauthorized, the handler should still
	// return 401 (not tolerate the error like other backend errors).
	v := 1000.0
	f := &stubLiveFetcher{
		resp: &PortfolioResponse{
			Positions: []Position{
				{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
			},
			IsLive: true,
		},
		evoErr: ErrUnauthorized,
	}
	r := setupLiveRouter(f)

	w := liveGet(t, r, "live=true", "Bearer x")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLiveHandler_EvolutionBackendErrorTolerated(t *testing.T) {
	// When evolution fetch fails with a backend error (non-auth), the handler
	// should tolerate and continue with nil evolution data.
	v := 1000.0
	f := &stubLiveFetcher{
		resp: &PortfolioResponse{
			Positions: []Position{
				{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
			},
			IsLive: true,
		},
		evoErr: ErrBackend,
	}
	r := setupLiveRouter(f)

	w := liveGet(t, r, "live=true", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
}
