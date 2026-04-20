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

func newRouterWithCreateModalHandler(h *CreateModalHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/trades/create_modal", h.Get)
	return r
}

func TestCreateModalHandler_HappyPath(t *testing.T) {
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL", Currency: "USD"}}}
	h := NewCreateModalHandler(cf)
	r := newRouterWithCreateModalHandler(h)

	req := httptest.NewRequest(http.MethodGet,
		"/actions/trades/create_modal?asset_id="+validAssetUUID+"&trade_type=BUY&offset=10", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, cf.calls)
	assert.Equal(t, "Bearer token", cf.gotAuth)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ModalSlotID, body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok, "tree must be present")
	assert.Equal(t, ModalID, tree["id"])
}

func TestCreateModalHandler_InvalidQuery(t *testing.T) {
	cf := &stubCatalogFetcher{}
	h := NewCreateModalHandler(cf)
	r := newRouterWithCreateModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/create_modal?offset=-1", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, cf.calls, "catalog must not be called when query is invalid")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

func TestCreateModalHandler_CatalogUnauthorized(t *testing.T) {
	cf := &stubCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	h := NewCreateModalHandler(cf)
	r := newRouterWithCreateModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/create_modal", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestCreateModalHandler_CatalogBackendError(t *testing.T) {
	cf := &stubCatalogFetcher{err: assetscatalog.ErrBackend}
	h := NewCreateModalHandler(cf)
	r := newRouterWithCreateModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/trades/create_modal", nil)
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
