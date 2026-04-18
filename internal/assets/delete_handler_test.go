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

type deleteStub struct {
	*stubMutator
	delErr   error
	delForce *bool
}

func (s *deleteStub) DeleteAsset(_ context.Context, _, _ string, force bool) error {
	s.delForce = &force
	return s.delErr
}

func TestDeleteHandler_HappyPath_NoForce(t *testing.T) {
	sc := &stubMutator{
		list:  &ListResult{Assets: []Asset{}, Total: 0, Size: 10},
		asset: &Asset{ID: "a1", Ticker: "AAPL"},
	}
	h := NewDeleteHandler(&deleteStub{stubMutator: sc})
	r := gin.New()
	r.DELETE("/actions/assets/:id", h.Delete)

	body, _ := json.Marshal(map[string]any{"force": false})
	req := httptest.NewRequest(http.MethodDelete, "/actions/assets/a1?asset_type=&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	fb := resp["feedback"].(map[string]any)
	assert.Equal(t, "Asset deleted", fb["props"].(map[string]any)["message"])
}

func TestDeleteHandler_HappyPath_Force(t *testing.T) {
	sc := &stubMutator{
		list:  &ListResult{Assets: []Asset{}, Total: 0, Size: 10},
		asset: &Asset{ID: "a1", Ticker: "AAPL"},
	}
	h := NewDeleteHandler(&deleteStub{stubMutator: sc})
	r := gin.New()
	r.DELETE("/actions/assets/:id", h.Delete)

	body, _ := json.Marshal(map[string]any{"force": true})
	req := httptest.NewRequest(http.MethodDelete, "/actions/assets/a1?asset_type=&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	fb := resp["feedback"].(map[string]any)
	assert.Equal(t, "Asset and associated data deleted", fb["props"].(map[string]any)["message"])
}

func TestDeleteHandler_AssetHasData_ReplacesModal(t *testing.T) {
	sc := &stubMutator{asset: &Asset{ID: "a1", Ticker: "AAPL"}}
	stub := &deleteStub{stubMutator: sc, delErr: &BackendValidationError{Code: "ASSET_HAS_DATA", Message: "Has data"}}
	h := NewDeleteHandler(stub)
	r := gin.New()
	r.DELETE("/actions/assets/:id", h.Delete)

	body, _ := json.Marshal(map[string]any{"force": false})
	req := httptest.NewRequest(http.MethodDelete, "/actions/assets/a1", bytes.NewReader(body))
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
