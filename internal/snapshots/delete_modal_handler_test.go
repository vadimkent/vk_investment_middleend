package snapshots

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRouterWithDeleteModalHandler(h *DeleteModalHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/snapshots/delete_modal", h.Get)
	return r
}

func TestDeleteModalHandler_HappyPath(t *testing.T) {
	snapID := "11111111-2222-3333-4444-555555555555"
	sg := &stubSnapshotGetter{snap: handlerSampleSnapshot()}
	h := NewDeleteModalHandler(sg)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/delete_modal?id="+snapID+"&offset=10", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, sg.calls)
	assert.Equal(t, snapID, sg.gotID)
	assert.Equal(t, "Bearer token", sg.gotAuth)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ModalSlotID, body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok, "tree must be present")
	assert.Equal(t, DeleteModalID, tree["id"])
}

func TestDeleteModalHandler_MissingID(t *testing.T) {
	sg := &stubSnapshotGetter{}
	h := NewDeleteModalHandler(sg)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/delete_modal", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sg.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "missing id", errObj["message"])
}

func TestDeleteModalHandler_InvalidID(t *testing.T) {
	sg := &stubSnapshotGetter{}
	h := NewDeleteModalHandler(sg)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/delete_modal?id=not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sg.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

func TestDeleteModalHandler_SnapshotNotFound(t *testing.T) {
	sg := &stubSnapshotGetter{err: ErrSnapshotNotFound}
	h := NewDeleteModalHandler(sg)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/delete_modal?id=11111111-2222-3333-4444-555555555555", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", errObj["code"])
}

func TestDeleteModalHandler_SnapshotUnauthorized(t *testing.T) {
	sg := &stubSnapshotGetter{err: ErrUnauthorized}
	h := NewDeleteModalHandler(sg)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/delete_modal?id=11111111-2222-3333-4444-555555555555", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestDeleteModalHandler_SnapshotBackendError(t *testing.T) {
	sg := &stubSnapshotGetter{err: ErrBackend}
	h := NewDeleteModalHandler(sg)
	r := newRouterWithDeleteModalHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/delete_modal?id=11111111-2222-3333-4444-555555555555", nil)
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
