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

func newRouterWithListHandler(h *ListHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/snapshots/list", h.Get)
	return r
}

func TestListHandler_Get_ReturnsReplaceActionResponse(t *testing.T) {
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: "11111111-2222-3333-4444-555555555555", Ticker: "AAPL", Currency: "USD"}}}
	h := NewListHandler(NewGetUseCase(sf, cf))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/list?is_full_snapshot=true&offset=0", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, SectionID, body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok, "tree must be present and non-null")
	assert.Equal(t, SectionID, tree["id"])
}

func TestListHandler_Get_InvalidQuery(t *testing.T) {
	h := NewListHandler(NewGetUseCase(&stubSnapshotFetcher{}, &stubCatalogFetcher{}))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/list?is_full_snapshot=maybe", nil)
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

func TestListHandler_Get_InvalidOffset(t *testing.T) {
	h := NewListHandler(NewGetUseCase(&stubSnapshotFetcher{}, &stubCatalogFetcher{}))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/list?offset=-5", nil)
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
	sf := &stubSnapshotFetcher{err: ErrUnauthorized}
	cf := &stubCatalogFetcher{}
	h := NewListHandler(NewGetUseCase(sf, cf))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestListHandler_Get_CatalogUnauthorized(t *testing.T) {
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	h := NewListHandler(NewGetUseCase(sf, cf))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestListHandler_Get_BackendError(t *testing.T) {
	sf := &stubSnapshotFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewListHandler(NewGetUseCase(sf, cf))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/list", nil)
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
