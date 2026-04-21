package snapshots

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

	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// stubSnapshotUpdater captures UpdateSnapshot calls.
type stubSnapshotUpdater struct {
	snapshot *Snapshot
	err      error
	calls    int
	gotAuth  string
	gotID    string
	gotBody  map[string]any
}

func (s *stubSnapshotUpdater) UpdateSnapshot(_ context.Context, auth, id string, body map[string]any) (*Snapshot, error) {
	s.calls++
	s.gotAuth = auth
	s.gotID = id
	s.gotBody = body
	return s.snapshot, s.err
}

const validSnapUUID = "11111111-2222-3333-4444-555555555555"

func newUpdateHandlerSnapshotRouter(h *UpdateHandler) *gin.Engine {
	r := gin.New()
	r.PATCH("/actions/snapshots/:id", h.Patch)
	return r
}

// patchSnapshot issues a PATCH to /actions/snapshots/:id with the given query and JSON body.
func patchSnapshot(r *gin.Engine, id, rawQuery string, body map[string]any) *httptest.ResponseRecorder {
	raw, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/actions/snapshots/"+id+rawQuery, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// originalSnapshot returns a sample snapshot with two entries for diff tests.
func originalSnapshot() *Snapshot {
	return &Snapshot{
		ID:         validSnapUUID,
		RecordedAt: "2025-06-01T00:00:00Z",
		Notes:      "original note",
		Entries: []Entry{
			{
				AssetID:      "aaaa-price-asset",
				CurrentPrice: "100.00",
			},
			{
				AssetID:              "bbbb-override-asset",
				CurrentValueOverride: "5000.00",
			},
		},
	}
}

// baseEditBody returns a wizard submission identical to originalSnapshot — no diff.
func baseEditSnapshotBody() map[string]any {
	return map[string]any{
		"notes":                                        "original note",
		"entries[aaaa-price-asset].mode":               "price",
		"entries[aaaa-price-asset].current_price":      "100.00",
		"entries[aaaa-price-asset].current_value_override": "",
		"entries[bbbb-override-asset].mode":                "override",
		"entries[bbbb-override-asset].current_price":       "",
		"entries[bbbb-override-asset].current_value_override": "5000.00",
	}
}

func newUpdatedSnapshot() *Snapshot {
	s := *originalSnapshot()
	s.Entries[0].CurrentPrice = "200.00"
	return &s
}

// Test 1: Happy path — fetch + update + refresh produce success response.
func TestUpdateHandler_Snapshots_HappyPath(t *testing.T) {
	su := &stubSnapshotUpdater{snapshot: newUpdatedSnapshot()}
	sg := &stubSnapshotGetter{snap: originalSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validSnapUUID, Ticker: "AAPL", Currency: "USD"}}}
	uc := NewGetUseCase(sf, cf)
	h := NewUpdateHandler(su, sg, uc, cf)
	r := newUpdateHandlerSnapshotRouter(h)

	body := baseEditSnapshotBody()
	body["entries[aaaa-price-asset].current_price"] = "200.00" // changed

	w := patchSnapshot(r, validSnapUUID, "", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, sg.calls, "should fetch original")
	assert.Equal(t, 1, su.calls, "should call UpdateSnapshot")
	assert.Equal(t, "Bearer tok", su.gotAuth)
	assert.Equal(t, validSnapUUID, su.gotID)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ScreenID, resp["target_id"])
	assert.NotNil(t, resp["tree"])
	assert.NotNil(t, resp["feedback"], "success snackbar must be present")
}

// Test 2: Missing id → 400.
func TestUpdateHandler_Snapshots_MissingID(t *testing.T) {
	su := &stubSnapshotUpdater{}
	sg := &stubSnapshotGetter{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)

	// Mount without :id param so c.Param("id") == "".
	r := gin.New()
	r.PATCH("/actions/snapshots/", h.Patch)

	raw, _ := json.Marshal(baseEditSnapshotBody())
	req := httptest.NewRequest(http.MethodPatch, "/actions/snapshots/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sg.calls)
	assert.Equal(t, 0, su.calls)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errObj, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "missing id", errObj["message"])
}

// Test 3: Invalid UUID → 400.
func TestUpdateHandler_Snapshots_InvalidUUID(t *testing.T) {
	su := &stubSnapshotUpdater{}
	sg := &stubSnapshotGetter{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	w := patchSnapshot(r, "not-a-uuid", "", baseEditSnapshotBody())

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sg.calls)
	assert.Equal(t, 0, su.calls)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errObj, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

// Test 4: Bad query → 400.
func TestUpdateHandler_Snapshots_BadQuery(t *testing.T) {
	su := &stubSnapshotUpdater{}
	sg := &stubSnapshotGetter{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	w := patchSnapshot(r, validSnapUUID, "?offset=-1", baseEditSnapshotBody())

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sg.calls)
	assert.Equal(t, 0, su.calls)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errObj, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

// Test 5: Invalid JSON body → 400.
func TestUpdateHandler_Snapshots_InvalidJSONBody(t *testing.T) {
	su := &stubSnapshotUpdater{}
	sg := &stubSnapshotGetter{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	req := httptest.NewRequest(http.MethodPatch, "/actions/snapshots/"+validSnapUUID, bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, su.calls)
}

// Test 6: Original fetch returns 404 → 404 response.
func TestUpdateHandler_Snapshots_FetchNotFound(t *testing.T) {
	su := &stubSnapshotUpdater{}
	sg := &stubSnapshotGetter{err: ErrSnapshotNotFound}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	w := patchSnapshot(r, validSnapUUID, "", baseEditSnapshotBody())

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, su.calls)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errObj, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", errObj["code"])
}

// Test 7: Original fetch unauthorized → 401.
func TestUpdateHandler_Snapshots_FetchUnauthorized(t *testing.T) {
	su := &stubSnapshotUpdater{}
	sg := &stubSnapshotGetter{err: ErrUnauthorized}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	w := patchSnapshot(r, validSnapUUID, "", baseEditSnapshotBody())

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 0, su.calls)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "unauthorized", resp["error"])
	assert.Equal(t, "/login", resp["redirect"])
}

// Test 8: Original fetch backend error → 502.
func TestUpdateHandler_Snapshots_FetchBackendError(t *testing.T) {
	su := &stubSnapshotUpdater{}
	sg := &stubSnapshotGetter{err: ErrBackend}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	w := patchSnapshot(r, validSnapUUID, "", baseEditSnapshotBody())

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 0, su.calls)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errObj, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

// Test 9: No diff → no UpdateSnapshot call, success snackbar returned.
func TestUpdateHandler_Snapshots_NoDiff_NoUpdateCall(t *testing.T) {
	su := &stubSnapshotUpdater{}
	sg := &stubSnapshotGetter{snap: originalSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	// Identical to original.
	w := patchSnapshot(r, validSnapUUID, "", baseEditSnapshotBody())

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, sg.calls)
	assert.Equal(t, 0, su.calls, "no diff → UpdateSnapshot must NOT be called")

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ScreenID, resp["target_id"])
	assert.NotNil(t, resp["feedback"], "success snackbar must be present even for no-op")
}

// Test 10: Notes change only — PATCH body contains only notes.
func TestUpdateHandler_Snapshots_NotesChangeOnly(t *testing.T) {
	su := &stubSnapshotUpdater{snapshot: originalSnapshot()}
	sg := &stubSnapshotGetter{snap: originalSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	body := baseEditSnapshotBody()
	body["notes"] = "updated note"

	w := patchSnapshot(r, validSnapUUID, "", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, su.calls)

	// PATCH body must have notes only — no entries key.
	require.NotNil(t, su.gotBody)
	assert.Equal(t, "updated note", su.gotBody["notes"])
	_, hasEntries := su.gotBody["entries"]
	assert.False(t, hasEntries, "unchanged entries must be absent from PATCH body")
}

// Test 11: Existing entry price change — PATCH body entries contains exactly that entry.
func TestUpdateHandler_Snapshots_EntryPriceChanged(t *testing.T) {
	su := &stubSnapshotUpdater{snapshot: newUpdatedSnapshot()}
	sg := &stubSnapshotGetter{snap: originalSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	body := baseEditSnapshotBody()
	body["entries[aaaa-price-asset].current_price"] = "200.00" // changed; override asset unchanged

	w := patchSnapshot(r, validSnapUUID, "", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, su.calls)

	entries := extractBeEntries(t, su.gotBody)
	require.Len(t, entries, 1, "only changed entry should be in diff")
	e := entries[0]
	assert.Equal(t, "aaaa-price-asset", e["asset_id"])
	assert.Equal(t, "200.00", e["current_price"])

	_, hasNotes := su.gotBody["notes"]
	assert.False(t, hasNotes, "unchanged notes must be absent")
}

// Test 12: New entry — asset not in original, submitted with values → included in PATCH.
func TestUpdateHandler_Snapshots_NewEntry(t *testing.T) {
	su := &stubSnapshotUpdater{snapshot: originalSnapshot()}
	sg := &stubSnapshotGetter{snap: originalSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	body := baseEditSnapshotBody()
	// Add a brand new entry not in the original snapshot.
	body["entries[cccc-new-asset].mode"] = "price"
	body["entries[cccc-new-asset].current_price"] = "300.00"
	body["entries[cccc-new-asset].current_value_override"] = ""

	w := patchSnapshot(r, validSnapUUID, "", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, su.calls)

	entries := extractBeEntries(t, su.gotBody)
	require.Len(t, entries, 1, "only the new entry should be in diff")
	e := entries[0]
	assert.Equal(t, "cccc-new-asset", e["asset_id"])
	assert.Equal(t, "300.00", e["current_price"])
}

// Test 13: BackendValidationError → ActionResponse with edit wizard, summary step, inline error.
func TestUpdateHandler_Snapshots_BackendValidationError(t *testing.T) {
	su := &stubSnapshotUpdater{
		err: &BackendValidationError{Code: "DUPLICATE_SNAPSHOT_ENTRY", Message: "duplicate entry"},
	}
	sg := &stubSnapshotGetter{snap: originalSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{
		{ID: "aaaa-price-asset", Ticker: "AAPL", Currency: "USD"},
	}}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	body := baseEditSnapshotBody()
	body["entries[aaaa-price-asset].current_price"] = "200.00" // force diff so update is called

	w := patchSnapshot(r, validSnapUUID, "", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, su.calls)
	assert.Equal(t, 1, cf.calls, "catalog re-fetched to rebuild wizard")
	assert.Equal(t, 0, sf.calls, "list NOT fetched on validation error")

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ModalSlotID, resp["target_id"])
	// Error message must appear in the tree.
	assert.Contains(t, w.Body.String(), "duplicate entry")
	// initial_step_id must be "summary" for edit-mode validation errors.
	assert.Contains(t, w.Body.String(), `"initial_step_id":"summary"`)
	// Must be an edit wizard (not create).
	assert.Contains(t, w.Body.String(), `"mode":"edit"`)
}

// Test 14: ErrUnauthorized from update → 401.
func TestUpdateHandler_Snapshots_UpdateUnauthorized(t *testing.T) {
	su := &stubSnapshotUpdater{err: ErrUnauthorized}
	sg := &stubSnapshotGetter{snap: originalSnapshot()}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	body := baseEditSnapshotBody()
	body["notes"] = "changed" // force diff

	w := patchSnapshot(r, validSnapUUID, "", body)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "unauthorized", resp["error"])
	assert.Equal(t, "/login", resp["redirect"])
}

// Test 15: Generic error from update → 502.
func TestUpdateHandler_Snapshots_UpdateGenericError(t *testing.T) {
	su := &stubSnapshotUpdater{err: ErrBackend}
	sg := &stubSnapshotGetter{snap: originalSnapshot()}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(su, sg, NewGetUseCase(sf, cf), cf)
	r := newUpdateHandlerSnapshotRouter(h)

	body := baseEditSnapshotBody()
	body["notes"] = "changed" // force diff

	w := patchSnapshot(r, validSnapUUID, "", body)

	assert.Equal(t, http.StatusBadGateway, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errObj, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}
