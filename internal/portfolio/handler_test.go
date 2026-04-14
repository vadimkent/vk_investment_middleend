package portfolio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFetcher struct {
	positions []Position
	err       error
	gotAuth   string
}

func (s *stubFetcher) GetPositions(ctx context.Context, auth string) ([]Position, error) {
	s.gotAuth = auth
	return s.positions, s.err
}

func setupHandlerRouter(f positionsFetcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(NewGetUseCase(f))
	r.GET("/screens/portfolio", h.Get)
	return r
}

func TestHandler_ForwardsAuthorizationAndReturnsScreen(t *testing.T) {
	v := 100.0
	f := &stubFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Name: "Apple", Currency: "USD", CurrentValue: &v}}}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Bearer abc", f.gotAuth)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "screen", body["type"])
	assert.Equal(t, "portfolio", body["id"])
}

func TestHandler_BackendUnauthorizedReturns401WithRedirect(t *testing.T) {
	f := &stubFetcher{err: ErrUnauthorized}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"unauthorized"`)
	assert.Contains(t, w.Body.String(), `"redirect":"/screens/login"`)
}

func TestHandler_BackendErrorReturns502(t *testing.T) {
	f := &stubFetcher{err: ErrBackend}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "BACKEND_ERROR")
}

func TestHandler_UsesAcceptLanguage(t *testing.T) {
	v := 100.0
	f := &stubFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Name: "Apple", Currency: "USD", CurrentValue: &v}}}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer x")
	req.Header.Set("Accept-Language", "es")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Portafolio")
}

func TestHandler_NowIsSet(t *testing.T) {
	snap := time.Now().Add(-24 * time.Hour)
	v := 100.0
	f := &stubFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Currency: "USD", CurrentValue: &v, LastSnapshotAt: &snap}}}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "1 days ago")
}
