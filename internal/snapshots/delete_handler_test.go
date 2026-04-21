package snapshots

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// stubSnapshotDeleter captures DeleteSnapshot calls for assertions. It
// satisfies the narrow snapshotDeleter interface the DeleteHandler depends on.
type stubSnapshotDeleter struct {
	err     error
	calls   int
	gotAuth string
	gotID   string
}

func (s *stubSnapshotDeleter) DeleteSnapshot(_ context.Context, auth, id string) error {
	s.calls++
	s.gotAuth = auth
	s.gotID = id
	return s.err
}

func newDeleteSnapshotRouter(h *DeleteHandler) *gin.Engine {
	r := gin.New()
	r.DELETE("/actions/snapshots/:id", h.Delete)
	return r
}

func deleteSnapshot(r *gin.Engine, id, rawQuery string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, "/actions/snapshots/"+id+rawQuery, nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const validSnapDeleteUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

// Test 1: Happy path — delete succeeds, screen tree rebuilt, snackbar present.
func TestDeleteHandler_Snapshots_HappyPath(t *testing.T) {
	sd := &stubSnapshotDeleter{}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewDeleteHandler(sd, NewGetUseCase(sf, cf))
	r := newDeleteSnapshotRouter(h)

	w := deleteSnapshot(r, validSnapDeleteUUID, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, sd.calls)
	assert.Equal(t, "Bearer tok", sd.gotAuth)
	assert.Equal(t, validSnapDeleteUUID, sd.gotID)

	// Screen rebuilt after successful delete.
	assert.Equal(t, 1, sf.calls)
	assert.Equal(t, 1, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ScreenID, body["target_id"])
	assert.NotNil(t, body["tree"])
	require.NotNil(t, body["feedback"], "success snackbar must be present")
}

// Test 2: Missing id (empty path param) → 400.
func TestDeleteHandler_Snapshots_MissingID(t *testing.T) {
	sd := &stubSnapshotDeleter{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(sd, NewGetUseCase(sf, cf))

	r := gin.New()
	r.DELETE("/actions/snapshots/", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/actions/snapshots/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sd.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "missing id", errObj["message"])
}

// Test 3: Invalid UUID → 400.
func TestDeleteHandler_Snapshots_InvalidUUID(t *testing.T) {
	sd := &stubSnapshotDeleter{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(sd, NewGetUseCase(sf, cf))
	r := newDeleteSnapshotRouter(h)

	w := deleteSnapshot(r, "not-a-uuid", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sd.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "invalid id", errObj["message"])
}

// Test 4: Bad query param → 400.
func TestDeleteHandler_Snapshots_BadQuery(t *testing.T) {
	sd := &stubSnapshotDeleter{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(sd, NewGetUseCase(sf, cf))
	r := newDeleteSnapshotRouter(h)

	w := deleteSnapshot(r, validSnapDeleteUUID, "?offset=abc")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sd.calls, "backend must not be called on bad query")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

// Test 5: ErrUnauthorized from delete → 401.
func TestDeleteHandler_Snapshots_Unauthorized(t *testing.T) {
	sd := &stubSnapshotDeleter{err: ErrUnauthorized}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(sd, NewGetUseCase(sf, cf))
	r := newDeleteSnapshotRouter(h)

	w := deleteSnapshot(r, validSnapDeleteUUID, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 0, sf.calls, "screen not rebuilt on unauthorized")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

// Test 6: ErrSnapshotNotFound → 404.
func TestDeleteHandler_Snapshots_NotFound(t *testing.T) {
	sd := &stubSnapshotDeleter{err: ErrSnapshotNotFound}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(sd, NewGetUseCase(sf, cf))
	r := newDeleteSnapshotRouter(h)

	w := deleteSnapshot(r, validSnapDeleteUUID, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, sf.calls, "screen not rebuilt on not-found")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", errObj["code"])
}

// Test 7: Generic delete error → 502.
func TestDeleteHandler_Snapshots_BackendError(t *testing.T) {
	sd := &stubSnapshotDeleter{err: ErrBackend}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(sd, NewGetUseCase(sf, cf))
	r := newDeleteSnapshotRouter(h)

	w := deleteSnapshot(r, validSnapDeleteUUID, "")

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 0, sf.calls, "screen not rebuilt on backend error")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

// Test 8: Refresh fails after successful delete → 502.
func TestDeleteHandler_Snapshots_ScreenRebuildFails(t *testing.T) {
	sd := &stubSnapshotDeleter{}
	sf := &stubSnapshotFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(sd, NewGetUseCase(sf, cf))
	r := newDeleteSnapshotRouter(h)

	w := deleteSnapshot(r, validSnapDeleteUUID, "")

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 1, sd.calls, "delete was called")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}
