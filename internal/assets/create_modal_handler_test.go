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

func TestCreateModalHandler_HappyPath(t *testing.T) {
	h := NewCreateModalHandler()
	r := gin.New()
	r.GET("/actions/assets/create_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/create_modal?asset_type=STOCK&offset=10", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, "assets-modal-slot", body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "modal", tree["type"])
	assert.Equal(t, "assets-create-modal", tree["id"])
}

func TestCreateModalHandler_InvalidParams(t *testing.T) {
	h := NewCreateModalHandler()
	r := gin.New()
	r.GET("/actions/assets/create_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/create_modal?asset_type=BOGUS", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
