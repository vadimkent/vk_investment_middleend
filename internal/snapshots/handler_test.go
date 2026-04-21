package snapshots

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

// stubSnapshotFetcher and stubCatalogFetcher are handler-scoped stubs. They
// mirror the fakes in get_usecase_test.go but have distinct names so both
// files compile together in the same package without conflicts.
type stubSnapshotFetcher struct {
	res     *ListResult
	err     error
	calls   int
	gotAuth string
	gotP    ListParams
}

func (s *stubSnapshotFetcher) List(_ context.Context, auth string, p ListParams) (*ListResult, error) {
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
	r.GET("/screens/snapshots", h.Get)
	return r
}

func TestHandler_Get_HappyPath(t *testing.T) {
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: "11111111-2222-3333-4444-555555555555", Ticker: "AAPL", Currency: "USD"}}}
	h := NewHandler(NewGetUseCase(sf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/snapshots", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, sf.calls)
	assert.Equal(t, 1, cf.calls)
	assert.Equal(t, "Bearer token", sf.gotAuth)
	assert.Equal(t, "Bearer token", cf.gotAuth)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, ScreenID, body["id"])
}

func TestHandler_Get_ForwardsParams(t *testing.T) {
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewHandler(NewGetUseCase(sf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/snapshots?is_full_snapshot=true&offset=10", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, sf.gotP.IsFullSnapshot)
	assert.True(t, *sf.gotP.IsFullSnapshot)
	assert.Equal(t, 10, sf.gotP.Offset)
}

func TestHandler_Get_IsFullSnapshotFalse(t *testing.T) {
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewHandler(NewGetUseCase(sf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/snapshots?is_full_snapshot=false", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, sf.gotP.IsFullSnapshot)
	assert.False(t, *sf.gotP.IsFullSnapshot)
}

func TestHandler_Get_IsFullSnapshotAbsent_NilParam(t *testing.T) {
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewHandler(NewGetUseCase(sf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/snapshots", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, sf.gotP.IsFullSnapshot, "absent filter must be nil")
}

func TestHandler_Get_InvalidIsFullSnapshot(t *testing.T) {
	h := NewHandler(NewGetUseCase(&stubSnapshotFetcher{}, &stubCatalogFetcher{}))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/snapshots?is_full_snapshot=maybe", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok, "error must be an object")
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "invalid is_full_snapshot", errObj["message"])
}

func TestHandler_Get_InvalidOffset(t *testing.T) {
	h := NewHandler(NewGetUseCase(&stubSnapshotFetcher{}, &stubCatalogFetcher{}))
	r := newRouterWithHandler(h)

	for _, val := range []string{"abc", "-1"} {
		t.Run("offset="+val, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/screens/snapshots?offset="+val, nil)
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

func TestHandler_Get_SnapshotUnauthorized(t *testing.T) {
	sf := &stubSnapshotFetcher{err: ErrUnauthorized}
	cf := &stubCatalogFetcher{}
	h := NewHandler(NewGetUseCase(sf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/snapshots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestHandler_Get_CatalogUnauthorized(t *testing.T) {
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	h := NewHandler(NewGetUseCase(sf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/snapshots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestHandler_Get_BackendError(t *testing.T) {
	sf := &stubSnapshotFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewHandler(NewGetUseCase(sf, cf))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/snapshots", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
	assert.Equal(t, "could not load snapshots", errObj["message"])
}

func TestHandler_Get_AcceptLanguageParsed(t *testing.T) {
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
			req := httptest.NewRequest(http.MethodGet, "/screens/snapshots", nil)
			if tc.header != "" {
				req.Header.Set("Accept-Language", tc.header)
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req
			got := parseLang(c)
			assert.Equal(t, tc.want, got)
		})
	}
}
