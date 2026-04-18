package assets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRouterWithListHandler(h *ListHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/assets/list", h.Get)
	return r
}

func TestListHandler_Get_ReturnsReplaceActionResponse(t *testing.T) {
	sc := &stubClient{res: &ListResult{Assets: []Asset{{ID: "a1", Ticker: "AAPL"}}, Total: 1, Size: 10}}
	h := NewListHandler(NewGetUseCase(sc))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/list?asset_type=STOCK&offset=0", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, "assets-section", body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok, "tree must be present")
	assert.Equal(t, "column", tree["type"])
	assert.Equal(t, "assets-section", tree["id"])
}

func TestListHandler_Get_InvalidAssetType(t *testing.T) {
	h := NewListHandler(NewGetUseCase(&stubClient{}))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/list?asset_type=BOGUS", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok, "error must be an object")
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "invalid asset_type", errObj["message"])
}

func TestListHandler_Get_InvalidOffset(t *testing.T) {
	h := NewListHandler(NewGetUseCase(&stubClient{}))
	r := newRouterWithListHandler(h)

	for _, val := range []string{"abc", "-5"} {
		req := httptest.NewRequest(http.MethodGet, "/actions/assets/list?offset="+val, nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, "offset=%q", val)

		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		errObj, ok := body["error"].(map[string]any)
		require.True(t, ok, "error must be an object")
		assert.Equal(t, "BAD_REQUEST", errObj["code"])
		assert.Equal(t, "invalid offset", errObj["message"])
	}
}

func TestListHandler_Get_Unauthorized(t *testing.T) {
	sc := &stubClient{err: ErrUnauthorized}
	h := NewListHandler(NewGetUseCase(sc))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestListHandler_Get_BackendError(t *testing.T) {
	sc := &stubClient{err: ErrBackend}
	h := NewListHandler(NewGetUseCase(sc))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/list", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
