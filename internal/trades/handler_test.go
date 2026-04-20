package trades

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
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

func init() {
	gin.SetMode(gin.TestMode)
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = i18n.Load(filepath.Join(repoRoot, "locales"))
}

// stubTradeFetcher and stubCatalogFetcher are handler-scoped stubs that mirror
// the fakes in get_usecase_test.go but live here so handler tests own their
// own capture state (Go tests are compiled together within a package, so the
// names differ from the get_usecase_test.go fakes).
type stubTradeFetcher struct {
	res     *ListResult
	err     error
	calls   int
	gotAuth string
	gotP    ListParams
}

func (s *stubTradeFetcher) List(_ context.Context, auth string, p ListParams) (*ListResult, error) {
	s.calls++
	s.gotAuth = auth
	s.gotP = p
	return s.res, s.err
}

type stubCatalogFetcher struct {
	res     []assetscatalog.Asset
	err     error
	calls   int
	gotAuth string
}

func (s *stubCatalogFetcher) List(_ context.Context, auth string) ([]assetscatalog.Asset, error) {
	s.calls++
	s.gotAuth = auth
	return s.res, s.err
}

func newRouterWithHandler(h *Handler) *gin.Engine {
	r := gin.New()
	r.GET("/screens/trades", h.Get)
	return r
}

const validAssetUUID = "11111111-2222-3333-4444-555555555555"

func TestHandler_Get_HappyPath(t *testing.T) {
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL", Currency: "USD"}}}
	h := NewHandler(NewGetUseCase(tf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/trades", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, tf.calls)
	assert.Equal(t, 1, cf.calls)
	assert.Equal(t, "Bearer token", tf.gotAuth)
	assert.Equal(t, "Bearer token", cf.gotAuth)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "trades-screen", body["id"])
}

func TestHandler_Get_ForwardsParams(t *testing.T) {
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewHandler(NewGetUseCase(tf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/trades?asset_id="+validAssetUUID+"&trade_type=BUY&offset=10", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, validAssetUUID, tf.gotP.AssetID)
	assert.Equal(t, "BUY", tf.gotP.TradeType)
	assert.Equal(t, 10, tf.gotP.Offset)
}

func TestHandler_Get_InvalidAssetID(t *testing.T) {
	h := NewHandler(NewGetUseCase(&stubTradeFetcher{}, &stubCatalogFetcher{}))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/trades?asset_id=not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok, "error must be an object")
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "invalid asset_id", errObj["message"])
}

func TestHandler_Get_InvalidTradeType(t *testing.T) {
	h := NewHandler(NewGetUseCase(&stubTradeFetcher{}, &stubCatalogFetcher{}))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/trades?trade_type=FOO", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "invalid trade_type", errObj["message"])
}

func TestHandler_Get_InvalidOffset(t *testing.T) {
	h := NewHandler(NewGetUseCase(&stubTradeFetcher{}, &stubCatalogFetcher{}))
	r := newRouterWithHandler(h)

	for _, val := range []string{"abc", "-1"} {
		t.Run("offset="+val, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/screens/trades?offset="+val, nil)
			req.Header.Set("Authorization", "Bearer token")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, "offset=%q", val)

			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			errObj, ok := body["error"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "BAD_REQUEST", errObj["code"])
			assert.Equal(t, "invalid offset", errObj["message"])
		})
	}
}

func TestHandler_Get_TradesUnauthorized(t *testing.T) {
	tf := &stubTradeFetcher{err: ErrUnauthorized}
	cf := &stubCatalogFetcher{}
	h := NewHandler(NewGetUseCase(tf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/trades", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestHandler_Get_CatalogUnauthorized(t *testing.T) {
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	h := NewHandler(NewGetUseCase(tf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/trades", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestHandler_Get_TradesBackendError(t *testing.T) {
	tf := &stubTradeFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewHandler(NewGetUseCase(tf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/trades", nil)
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

func TestHandler_Get_CatalogBackendError(t *testing.T) {
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrBackend}
	h := NewHandler(NewGetUseCase(tf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/trades", nil)
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

func TestHandler_Get_AcceptLanguageParsed(t *testing.T) {
	// Verify parseLang falls back to "en" on missing header, picks the base
	// language from "es-ES,es;q=0.9", and strips quality params.
	cases := []struct {
		header string
		want   string
	}{
		{"", "en"},
		{"es", "es"},
		{"es-ES,es;q=0.9", "es"},
		{"en-US", "en"},
	}
	for _, tc := range cases {
		t.Run("header="+tc.header, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/screens/trades", nil)
			if tc.header != "" {
				req.Header.Set("Accept-Language", tc.header)
			}
			// Build a fresh gin context that matches what parseLang reads.
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req
			got := parseLang(c)
			assert.Equal(t, tc.want, got)
		})
	}
}
