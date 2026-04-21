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

// stubSnapshotGetter is a file-scoped fake for the snapshotGetter interface
// used by the edit_wizard and delete_modal handlers.
type stubSnapshotGetter struct {
	snap    *Snapshot
	err     error
	calls   int
	gotAuth string
	gotID   string
}

func (s *stubSnapshotGetter) GetSnapshot(_ context.Context, auth, id string) (*Snapshot, error) {
	s.calls++
	s.gotAuth = auth
	s.gotID = id
	return s.snap, s.err
}

func newRouterWithEditWizardHandler(h *EditWizardHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/snapshots/edit_wizard", h.Get)
	return r
}

func handlerSampleSnapshot() *Snapshot {
	return &Snapshot{
		ID:             "11111111-2222-3333-4444-555555555555",
		RecordedAt:     "2024-03-15T00:00:00Z",
		IsFullSnapshot: true,
		Notes:          "test snapshot",
		Entries:        []Entry{},
	}
}

func TestEditWizardHandler_HappyPath(t *testing.T) {
	snapID := "11111111-2222-3333-4444-555555555555"
	sg := &stubSnapshotGetter{snap: handlerSampleSnapshot()}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: snapID, Ticker: "AAPL", Currency: "USD"}}}
	h := NewEditWizardHandler(sg, cf)
	r := newRouterWithEditWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/edit_wizard?id="+snapID+"&offset=10", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, sg.calls)
	assert.Equal(t, 1, cf.calls)
	assert.Equal(t, snapID, sg.gotID)
	assert.Equal(t, "Bearer token", sg.gotAuth)
	assert.Equal(t, "Bearer token", cf.gotAuth)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ModalSlotID, body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok, "tree must be present")
	assert.Equal(t, WizardID, tree["id"])
}

func TestEditWizardHandler_MissingID(t *testing.T) {
	sg := &stubSnapshotGetter{}
	cf := &stubCatalogFetcher{}
	h := NewEditWizardHandler(sg, cf)
	r := newRouterWithEditWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/edit_wizard", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sg.calls)
	assert.Equal(t, 0, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "missing id", errObj["message"])
}

func TestEditWizardHandler_InvalidID(t *testing.T) {
	sg := &stubSnapshotGetter{}
	cf := &stubCatalogFetcher{}
	h := NewEditWizardHandler(sg, cf)
	r := newRouterWithEditWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/edit_wizard?id=not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sg.calls)
	assert.Equal(t, 0, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

func TestEditWizardHandler_SnapshotNotFound(t *testing.T) {
	sg := &stubSnapshotGetter{err: ErrSnapshotNotFound}
	cf := &stubCatalogFetcher{}
	h := NewEditWizardHandler(sg, cf)
	r := newRouterWithEditWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/edit_wizard?id=11111111-2222-3333-4444-555555555555", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", errObj["code"])
}

func TestEditWizardHandler_SnapshotUnauthorized(t *testing.T) {
	sg := &stubSnapshotGetter{err: ErrUnauthorized}
	cf := &stubCatalogFetcher{}
	h := NewEditWizardHandler(sg, cf)
	r := newRouterWithEditWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/edit_wizard?id=11111111-2222-3333-4444-555555555555", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 0, cf.calls, "catalog must not be called when snapshot fetch fails")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestEditWizardHandler_SnapshotBackendError(t *testing.T) {
	sg := &stubSnapshotGetter{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewEditWizardHandler(sg, cf)
	r := newRouterWithEditWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/edit_wizard?id=11111111-2222-3333-4444-555555555555", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 0, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

func TestEditWizardHandler_CatalogUnauthorized(t *testing.T) {
	sg := &stubSnapshotGetter{snap: handlerSampleSnapshot()}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	h := NewEditWizardHandler(sg, cf)
	r := newRouterWithEditWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/edit_wizard?id=11111111-2222-3333-4444-555555555555", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 1, sg.calls)
	assert.Equal(t, 1, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestEditWizardHandler_CatalogBackendError(t *testing.T) {
	sg := &stubSnapshotGetter{snap: handlerSampleSnapshot()}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrBackend}
	h := NewEditWizardHandler(sg, cf)
	r := newRouterWithEditWizardHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/snapshots/edit_wizard?id=11111111-2222-3333-4444-555555555555", nil)
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
