package assets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAssetFetcher struct {
	asset *Asset
	err   error
}

func (s *stubAssetFetcher) GetAsset(_ context.Context, _ string, _ string) (*Asset, error) {
	return s.asset, s.err
}

func TestEditModalHandler_HappyPath(t *testing.T) {
	a := &Asset{ID: "a1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD"}
	h := NewEditModalHandler(&stubAssetFetcher{asset: a})
	r := gin.New()
	r.GET("/actions/assets/edit_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/edit_modal?id=a1", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	tree := body["tree"].(map[string]any)
	assert.Equal(t, "assets-edit-modal", tree["id"])
}

func TestEditModalHandler_MissingID(t *testing.T) {
	h := NewEditModalHandler(&stubAssetFetcher{})
	r := gin.New()
	r.GET("/actions/assets/edit_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/edit_modal", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEditModalHandler_NotFound(t *testing.T) {
	h := NewEditModalHandler(&stubAssetFetcher{err: ErrAssetNotFound})
	r := gin.New()
	r.GET("/actions/assets/edit_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/edit_modal?id=missing", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEditModalHandler_Unauthorized(t *testing.T) {
	h := NewEditModalHandler(&stubAssetFetcher{err: ErrUnauthorized})
	r := gin.New()
	r.GET("/actions/assets/edit_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/edit_modal?id=a1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
