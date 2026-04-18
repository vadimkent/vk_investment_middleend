package assets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

func init() {
	gin.SetMode(gin.TestMode)
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = i18n.Load(filepath.Join(repoRoot, "locales"))
}

type stubClient struct {
	res *ListResult
	err error
	got ListParams
}

func (s *stubClient) List(_ context.Context, _ string, p ListParams) (*ListResult, error) {
	s.got = p
	return s.res, s.err
}

func newRouterWithHandler(h *Handler) *gin.Engine {
	r := gin.New()
	r.GET("/screens/assets", h.Get)
	return r
}

func TestHandler_Get_HappyPath(t *testing.T) {
	sc := &stubClient{res: &ListResult{Assets: []Asset{{ID: "a1", Ticker: "AAPL"}}, Total: 1, Size: 10}}
	h := NewHandler(NewGetUseCase(sc))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/assets?asset_type=STOCK&offset=0", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "STOCK", sc.got.AssetType)
	assert.Equal(t, 0, sc.got.Offset)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "screen", body["type"])
	assert.Equal(t, "assets", body["id"])
}

func TestHandler_Get_InvalidAssetType(t *testing.T) {
	h := NewHandler(NewGetUseCase(&stubClient{}))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/assets?asset_type=BOGUS", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Get_InvalidOffset(t *testing.T) {
	h := NewHandler(NewGetUseCase(&stubClient{}))
	r := newRouterWithHandler(h)

	for _, val := range []string{"abc", "-5"} {
		req := httptest.NewRequest(http.MethodGet, "/screens/assets?offset="+val, nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, "offset=%q", val)
	}
}

func TestHandler_Get_Unauthorized(t *testing.T) {
	sc := &stubClient{err: ErrUnauthorized}
	h := NewHandler(NewGetUseCase(sc))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/assets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestHandler_Get_BackendError(t *testing.T) {
	sc := &stubClient{err: ErrBackend}
	h := NewHandler(NewGetUseCase(sc))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/assets", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
