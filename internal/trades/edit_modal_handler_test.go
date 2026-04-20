package trades

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// stubTradeGetter is a file-scoped fake for the tradeGetter interface used
// by the edit/delete modal handlers.
type stubTradeGetter struct {
	trade   *Trade
	err     error
	calls   int
	gotAuth string
	gotID   string
}

func (s *stubTradeGetter) GetTrade(_ context.Context, auth, id string) (*Trade, error) {
	s.calls++
	s.gotAuth = auth
	s.gotID = id
	return s.trade, s.err
}

func newRouterWithEditModalHandler(h *EditModalHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/trades/edit_modal", h.Get)
	return r
}

func handlerSampleTrade() *Trade {
	return &Trade{
		ID:           "t1",
		AssetID:      validAssetUUID,
		TradeType:    "BUY",
		Quantity:     "10",
		PricePerUnit: "100",
		Fees:         "1",
		Date:         "2024-03-15T00:00:00Z",
		Source:       "manual",
		Notes:        "hello",
	}
}

func TestEditModalHandler_HappyPath(t *testing.T) {
	tg := &stubTradeGetter{trade: handlerSampleTrade()}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL", Currency: "USD"}}}
	h := NewEditModalHandler(tg, cf)
	r := newRouterWithEditModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/edit_modal?id=t1&offset=10", nil)
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

func TestEditModalHandler_MissingID(t *testing.T) {
	tg := &stubTradeGetter{}
	cf := &stubCatalogFetcher{}
	h := NewEditModalHandler(tg, cf)
	r := newRouterWithEditModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/edit_modal", nil)
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

func TestEditModalHandler_InvalidQuery(t *testing.T) {
	tg := &stubTradeGetter{}
	cf := &stubCatalogFetcher{}
	h := NewEditModalHandler(tg, cf)
	r := newRouterWithEditModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/edit_modal?id=t1&offset=-1", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, tg.calls)
	assert.Equal(t, 0, cf.calls)
}

func TestEditModalHandler_TradeUnauthorized(t *testing.T) {
	tg := &stubTradeGetter{err: ErrUnauthorized}
	cf := &stubCatalogFetcher{}
	h := NewEditModalHandler(tg, cf)
	r := newRouterWithEditModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/edit_modal?id=t1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 0, cf.calls, "catalog must not be called when trade fetch fails")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestEditModalHandler_TradeNotFound(t *testing.T) {
	tg := &stubTradeGetter{err: ErrTradeNotFound}
	cf := &stubCatalogFetcher{}
	h := NewEditModalHandler(tg, cf)
	r := newRouterWithEditModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/edit_modal?id=missing", nil)
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

func TestEditModalHandler_TradeBackendError(t *testing.T) {
	tg := &stubTradeGetter{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewEditModalHandler(tg, cf)
	r := newRouterWithEditModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/edit_modal?id=t1", nil)
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

func TestEditModalHandler_CatalogUnauthorized(t *testing.T) {
	tg := &stubTradeGetter{trade: handlerSampleTrade()}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	h := NewEditModalHandler(tg, cf)
	r := newRouterWithEditModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/edit_modal?id=t1", nil)
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

func TestEditModalHandler_CatalogBackendError(t *testing.T) {
	tg := &stubTradeGetter{trade: handlerSampleTrade()}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrBackend}
	h := NewEditModalHandler(tg, cf)
	r := newRouterWithEditModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/edit_modal?id=t1", nil)
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
