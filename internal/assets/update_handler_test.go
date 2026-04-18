package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type updateStub struct {
	*stubMutator
	updated *Asset
	updErr  error
}

func (s *updateStub) UpdateAsset(_ context.Context, _, _ string, _ map[string]any) (*Asset, error) {
	return s.updated, s.updErr
}

func TestUpdateHandler_HappyPath(t *testing.T) {
	sc := &stubMutator{list: &ListResult{Assets: []Asset{}, Total: 0, Size: 10}}
	h := NewUpdateHandler(&updateStub{stubMutator: sc, updated: &Asset{ID: "a1", Ticker: "AAPL"}})
	r := gin.New()
	r.PATCH("/actions/assets/:id", h.Patch)

	body, _ := json.Marshal(map[string]any{"name": "Apple Inc"})
	req := httptest.NewRequest(http.MethodPatch, "/actions/assets/a1?asset_type=&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "assets-root", resp["target_id"])
	fb := resp["feedback"].(map[string]any)
	assert.Equal(t, "Asset updated", fb["props"].(map[string]any)["message"])
}

func TestUpdateHandler_ValidationError_ReplacesModalSlot(t *testing.T) {
	sc := &stubMutator{asset: &Asset{ID: "a1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD"}}
	h := NewUpdateHandler(&updateStub{stubMutator: sc, updErr: &BackendValidationError{Code: "INVALID_PRICE_PROVIDER", Message: "bad provider"}})

	r := gin.New()
	r.PATCH("/actions/assets/:id", h.Patch)

	body, _ := json.Marshal(map[string]any{"name": "x"})
	req := httptest.NewRequest(http.MethodPatch, "/actions/assets/a1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "assets-modal-slot", resp["target_id"])
}
