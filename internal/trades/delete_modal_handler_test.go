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

func newRouterWithDeleteModalHandler(h *DeleteModalHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/trades/delete_modal", h.Get)
	return r
}

func TestDeleteModalHandler_HappyPath(t *testing.T) {
	tg := &stubTradeGetter{trade: handlerSampleTrade()}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL", Currency: "USD"}}}
	h := NewDeleteModalHandler(tg, cf)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/delete_modal?id=t1&offset=10", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, tg.calls)
	assert.Equal(t, 1, cf.calls)
	assert.Equal(t, "t1", tg.gotID)
	assert.Equal(t, "Bearer token", tg.gotAuth)
	assert.Equal(t, "Bearer token", cf.gotAuth)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ModalSlotID, body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, ModalID, tree["id"])
}

func TestDeleteModalHandler_MissingID(t *testing.T) {
	tg := &stubTradeGetter{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteModalHandler(tg, cf)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/delete_modal", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, tg.calls)
	assert.Equal(t, 0, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "missing id", errObj["message"])
}

func TestDeleteModalHandler_InvalidQuery(t *testing.T) {
	tg := &stubTradeGetter{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteModalHandler(tg, cf)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/delete_modal?id=t1&offset=-1", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, tg.calls)
	assert.Equal(t, 0, cf.calls)
}

func TestDeleteModalHandler_TradeUnauthorized(t *testing.T) {
	tg := &stubTradeGetter{err: ErrUnauthorized}
	cf := &stubCatalogFetcher{}
	h := NewDeleteModalHandler(tg, cf)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/delete_modal?id=t1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 0, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestDeleteModalHandler_TradeNotFound(t *testing.T) {
	tg := &stubTradeGetter{err: ErrTradeNotFound}
	cf := &stubCatalogFetcher{}
	h := NewDeleteModalHandler(tg, cf)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/delete_modal?id=missing", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", errObj["code"])
}

func TestDeleteModalHandler_TradeBackendError(t *testing.T) {
	tg := &stubTradeGetter{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewDeleteModalHandler(tg, cf)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/delete_modal?id=t1", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 0, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

func TestDeleteModalHandler_CatalogUnauthorized(t *testing.T) {
	tg := &stubTradeGetter{trade: handlerSampleTrade()}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	h := NewDeleteModalHandler(tg, cf)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/delete_modal?id=t1", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 1, tg.calls)
	assert.Equal(t, 1, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestDeleteModalHandler_CatalogBackendError(t *testing.T) {
	tg := &stubTradeGetter{trade: handlerSampleTrade()}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrBackend}
	h := NewDeleteModalHandler(tg, cf)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/delete_modal?id=t1", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}
