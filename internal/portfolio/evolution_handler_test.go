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

type stubEvolutionFetcher struct {
	points   []EvolutionPoint
	err      error
	gotAuth  string
	gotQuery EvolutionQuery
	called   bool
}

func (s *stubEvolutionFetcher) GetEvolution(ctx context.Context, auth string, q EvolutionQuery) ([]EvolutionPoint, error) {
	s.called = true
	s.gotAuth = auth
	s.gotQuery = q
	return s.points, s.err
}

func setupEvolutionRouter(f evolutionFetcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/actions/portfolio/evolution", NewEvolutionHandler(f).Get)
	return r
}

func doGet(t *testing.T, r *gin.Engine, query string, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/actions/portfolio/evolution?"+query, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestEvolutionHandler_SuccessReturnsReplaceActionResponse(t *testing.T) {
	f := &stubEvolutionFetcher{points: []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
	}}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=3m&mode=abs&currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "chart-value-over-time-card", resp["target_id"])
	tree, ok := resp["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "card", tree["type"])
	assert.Equal(t, "chart-value-over-time-card", tree["id"])

	assert.True(t, f.called)
	assert.Equal(t, "Bearer tok", f.gotAuth)
	assert.Equal(t, 100, f.gotQuery.Points)
	assert.Equal(t, "", f.gotQuery.Currency) // currency filtering happens in the builder, not the BE call
	require.NotNil(t, f.gotQuery.From)
}

func TestEvolutionHandler_AllTimeframeOmitsFrom(t *testing.T) {
	f := &stubEvolutionFetcher{}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=abs&currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, f.gotQuery.From)
}

func TestEvolutionHandler_InvalidTimeframeReturns400(t *testing.T) {
	f := &stubEvolutionFetcher{}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=xxx&mode=abs&currency=USD", "Bearer tok")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, f.called)
}

func TestEvolutionHandler_InvalidModeReturns400(t *testing.T) {
	f := &stubEvolutionFetcher{}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=yolo&currency=USD", "Bearer tok")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, f.called)
}

func TestEvolutionHandler_DefaultsAppliedWhenOmitted(t *testing.T) {
	f := &stubEvolutionFetcher{}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, f.gotQuery.From)
	assert.Equal(t, "", f.gotQuery.Currency)
}

func TestEvolutionHandler_BackendUnauthorizedReturns401WithRedirect(t *testing.T) {
	f := &stubEvolutionFetcher{err: ErrUnauthorized}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=abs&currency=USD", "Bearer x")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"unauthorized"`)
	assert.Contains(t, w.Body.String(), `"redirect":"/screens/login"`)
}

func TestEvolutionHandler_BackendErrorReturns502(t *testing.T) {
	f := &stubEvolutionFetcher{err: ErrBackend}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=abs&currency=USD", "Bearer x")
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "BACKEND_ERROR")
}

func TestEvolutionHandler_PctWithNoCostDataShowsEmptyMessage(t *testing.T) {
	// EvolutionPoint has no total_cost. pct mode should surface the no-cost
	// empty state.
	f := &stubEvolutionFetcher{points: []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
	}}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=pct&currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "No cost data available")
}
