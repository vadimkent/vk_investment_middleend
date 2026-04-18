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

func TestDeleteModalHandler_HappyPath(t *testing.T) {
	a := &Asset{ID: "a1", Ticker: "AAPL"}
	h := NewDeleteModalHandler(&stubAssetFetcher{asset: a})
	r := gin.New()
	r.GET("/actions/assets/delete_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/delete_modal?id=a1", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	tree := body["tree"].(map[string]any)
	assert.Equal(t, "assets-delete-modal", tree["id"])
}

func TestDeleteModalHandler_NotFound(t *testing.T) {
	h := NewDeleteModalHandler(&stubAssetFetcher{err: ErrAssetNotFound})
	r := gin.New()
	r.GET("/actions/assets/delete_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/delete_modal?id=missing", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteModalHandler_MissingID(t *testing.T) {
	h := NewDeleteModalHandler(&stubAssetFetcher{})
	r := gin.New()
	r.GET("/actions/assets/delete_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/delete_modal", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
