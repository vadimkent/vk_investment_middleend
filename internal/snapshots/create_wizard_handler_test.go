package snapshots

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

func newRouterWithCreateWizardHandler(h *CreateWizardHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/snapshots/create_wizard", h.Get)
	return r
}

func TestCreateWizardHandler_HappyPath(t *testing.T) {
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: "11111111-2222-3333-4444-555555555555", Ticker: "AAPL", Currency: "USD"}}}
	h := NewCreateWizardHandler(cf)
	r := newRouterWithCreateWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/create_wizard?offset=10", nil)
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
	assert.Equal(t, WizardID, tree["id"])
}

func TestCreateWizardHandler_InvalidQuery(t *testing.T) {
	cf := &stubCatalogFetcher{}
	h := NewCreateWizardHandler(cf)
	r := newRouterWithCreateWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/create_wizard?offset=-1", nil)
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

func TestCreateWizardHandler_CatalogUnauthorized(t *testing.T) {
	cf := &stubCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	h := NewCreateWizardHandler(cf)
	r := newRouterWithCreateWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/create_wizard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestCreateWizardHandler_CatalogBackendError(t *testing.T) {
	cf := &stubCatalogFetcher{err: assetscatalog.ErrBackend}
	h := NewCreateWizardHandler(cf)
	r := newRouterWithCreateWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/create_wizard", nil)
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
