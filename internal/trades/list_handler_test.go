package trades

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

func newRouterWithListHandler(h *ListHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/trades/list", h.Get)
	return r
}

func TestListHandler_Get_ReturnsReplaceActionResponse(t *testing.T) {
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL"}}}
	h := NewListHandler(NewGetUseCase(tf, cf))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/list?trade_type=BUY&offset=0", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, "trades-section", body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok, "tree must be present")
	assert.Equal(t, "trades-section", tree["id"])
}

func TestListHandler_Get_InvalidQuery(t *testing.T) {
	h := NewListHandler(NewGetUseCase(&stubTradeFetcher{}, &stubCatalogFetcher{}))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/list?asset_id=not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

func TestListHandler_Get_Unauthorized(t *testing.T) {
	tf := &stubTradeFetcher{err: ErrUnauthorized}
	cf := &stubCatalogFetcher{}
	h := NewListHandler(NewGetUseCase(tf, cf))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestListHandler_Get_CatalogUnauthorized(t *testing.T) {
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	h := NewListHandler(NewGetUseCase(tf, cf))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListHandler_Get_BackendError(t *testing.T) {
	tf := &stubTradeFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewListHandler(NewGetUseCase(tf, cf))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/list", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
