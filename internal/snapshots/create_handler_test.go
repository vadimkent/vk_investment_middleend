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

// stubSnapshotCreator captures CreateSnapshot calls for assertions.
type stubSnapshotCreator struct {
	snapshot *Snapshot
	err      error
	calls    int
	gotAuth  string
	gotBody  map[string]any
}

func (s *stubSnapshotCreator) CreateSnapshot(_ context.Context, auth string, body map[string]any) (*Snapshot, error) {
	s.calls++
	s.gotAuth = auth
	s.gotBody = body
	return s.snapshot, s.err
}

func newCreateSnapshotHandlerRouter(h *CreateHandler) *gin.Engine {
	r := gin.New()
	r.POST("/actions/snapshots/create", h.Post)
	return r
}

func newCreatedSnapshot() *Snapshot {
	return &Snapshot{
		ID:         "snap-1",
		RecordedAt: "2025-06-01T00:00:00Z",
	}
}

// postSnapshotCreate issues a POST to /actions/snapshots/create with the given query and JSON body.
func postSnapshotCreate(r *gin.Engine, rawQuery string, body map[string]any) *httptest.ResponseRecorder {
	raw, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/actions/snapshots/create"+rawQuery, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// baseCreateBody returns a minimal valid wizard submission body.
func baseCreateBody() map[string]any {
	return map[string]any{
		"recorded_at": "2025-06-01T12:00",
		"notes":       "",
	}
}

// Test 1: Happy path — creates snapshot, rebuilds screen, returns ActionResponse with tree + snackbar.
func TestCreateHandler_Snapshots_HappyPath(t *testing.T) {
	sc := &stubSnapshotCreator{snapshot: newCreatedSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: "11111111-2222-3333-4444-555555555555", Ticker: "AAPL", Currency: "USD"}}}
	uc := NewGetUseCase(sf, cf)
	h := NewCreateHandler(sc, uc, cf)
	r := newCreateSnapshotHandlerRouter(h)

	w := postSnapshotCreate(r, "", baseCreateBody())

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, sc.calls)
	assert.Equal(t, "Bearer tok", sc.gotAuth)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ScreenID, body["target_id"])
	assert.NotNil(t, body["tree"])
	assert.NotNil(t, body["feedback"], "success snackbar must be present")
}

// Test 2: Bad query param → 400.
func TestCreateHandler_Snapshots_BadQuery(t *testing.T) {
	sc := &stubSnapshotCreator{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	w := postSnapshotCreate(r, "?offset=-1", baseCreateBody())

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sc.calls, "backend must not be called on bad query")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

// Test 3: Invalid JSON body → 400.
func TestCreateHandler_Snapshots_InvalidJSONBody(t *testing.T) {
	sc := &stubSnapshotCreator{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)

	req := httptest.NewRequest(http.MethodPost, "/actions/snapshots/create", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := newCreateSnapshotHandlerRouter(h)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, sc.calls)
}

// Test 4: BackendValidationError with code FUTURE_DATED_SNAPSHOT → wizard rebuilt at "info" step.
func TestCreateHandler_Snapshots_ValidationError_FutureDated(t *testing.T) {
	sc := &stubSnapshotCreator{err: &BackendValidationError{Code: "FUTURE_DATED_SNAPSHOT", Message: "Cannot be future dated"}}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: "11111111-2222-3333-4444-555555555555", Ticker: "AAPL", Currency: "USD"}}}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	w := postSnapshotCreate(r, "", baseCreateBody())

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, sc.calls)
	assert.Equal(t, 1, cf.calls, "catalog re-fetched to rebuild wizard")
	assert.Equal(t, 0, sf.calls, "snapshot list NOT fetched on validation error")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ModalSlotID, body["target_id"])
	assert.Contains(t, w.Body.String(), "Cannot be future dated", "inline error must appear in wizard")
	// initial_step_id must be "info" for FUTURE_DATED_SNAPSHOT.
	assert.Contains(t, w.Body.String(), `"initial_step_id":"info"`)
}

// Test 5: BackendValidationError with code CONFLICTING_SNAPSHOT_VALUE → wizard rebuilt at "summary" step.
func TestCreateHandler_Snapshots_ValidationError_OtherCode(t *testing.T) {
	sc := &stubSnapshotCreator{err: &BackendValidationError{Code: "CONFLICTING_SNAPSHOT_VALUE", Message: "Conflicting value"}}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	w := postSnapshotCreate(r, "", baseCreateBody())

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, ModalSlotID, body["target_id"])
	assert.Contains(t, w.Body.String(), "Conflicting value")
	assert.Contains(t, w.Body.String(), `"initial_step_id":"summary"`)
}

// Test 6: ErrUnauthorized during create → 401.
func TestCreateHandler_Snapshots_Unauthorized(t *testing.T) {
	sc := &stubSnapshotCreator{err: ErrUnauthorized}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	w := postSnapshotCreate(r, "", baseCreateBody())

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

// Test 7: Other backend error during create → 502.
func TestCreateHandler_Snapshots_BackendError(t *testing.T) {
	sc := &stubSnapshotCreator{err: ErrBackend}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	w := postSnapshotCreate(r, "", baseCreateBody())

	assert.Equal(t, http.StatusBadGateway, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

// Test 8: Success but refresh (uc.Execute) fails → 502.
func TestCreateHandler_Snapshots_RefreshFails(t *testing.T) {
	sc := &stubSnapshotCreator{snapshot: newCreatedSnapshot()}
	sf := &stubSnapshotFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	w := postSnapshotCreate(r, "", baseCreateBody())

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

// Test 9a: Entry with mode=price + current_price → BE body has current_price only.
func TestCreateHandler_Snapshots_EntryMode_Price(t *testing.T) {
	sc := &stubSnapshotCreator{snapshot: newCreatedSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	body := baseCreateBody()
	body["entries[asset-1].mode"] = "price"
	body["entries[asset-1].current_price"] = "123.45"
	body["entries[asset-1].current_value_override"] = ""

	w := postSnapshotCreate(r, "", body)
	require.Equal(t, http.StatusOK, w.Code)

	entries := extractBeEntries(t, sc.gotBody)
	require.Len(t, entries, 1)
	e := entries[0]
	assert.Equal(t, "asset-1", e["asset_id"])
	assert.Equal(t, "123.45", e["current_price"])
	_, hasOverride := e["current_value_override"]
	assert.False(t, hasOverride, "current_value_override must be absent in price mode")
}

// Test 9b: Entry with mode=override + current_value_override → BE body has current_value_override only.
func TestCreateHandler_Snapshots_EntryMode_Override(t *testing.T) {
	sc := &stubSnapshotCreator{snapshot: newCreatedSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	body := baseCreateBody()
	body["entries[asset-2].mode"] = "override"
	body["entries[asset-2].current_price"] = ""
	body["entries[asset-2].current_value_override"] = "5000.00"

	w := postSnapshotCreate(r, "", body)
	require.Equal(t, http.StatusOK, w.Code)

	entries := extractBeEntries(t, sc.gotBody)
	require.Len(t, entries, 1)
	e := entries[0]
	assert.Equal(t, "asset-2", e["asset_id"])
	assert.Equal(t, "5000.00", e["current_value_override"])
	_, hasPrice := e["current_price"]
	assert.False(t, hasPrice, "current_price must be absent in override mode")
}

// Test 10: Complex asset (no mode, only current_value_override) → BE body has current_value_override.
func TestCreateHandler_Snapshots_EntryMode_Complex(t *testing.T) {
	sc := &stubSnapshotCreator{snapshot: newCreatedSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	body := baseCreateBody()
	// Complex asset: no mode field submitted, only override.
	body["entries[complex-1].current_value_override"] = "9999.99"

	w := postSnapshotCreate(r, "", body)
	require.Equal(t, http.StatusOK, w.Code)

	entries := extractBeEntries(t, sc.gotBody)
	require.Len(t, entries, 1)
	e := entries[0]
	assert.Equal(t, "complex-1", e["asset_id"])
	assert.Equal(t, "9999.99", e["current_value_override"])
	_, hasPrice := e["current_price"]
	assert.False(t, hasPrice)
}

// Test 11: Entry with all fields empty → dropped from BE body entirely.
func TestCreateHandler_Snapshots_EntrySkipped_AllEmpty(t *testing.T) {
	sc := &stubSnapshotCreator{snapshot: newCreatedSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	body := baseCreateBody()
	body["entries[skipped-1].mode"] = ""
	body["entries[skipped-1].current_price"] = ""
	body["entries[skipped-1].current_value_override"] = ""

	w := postSnapshotCreate(r, "", body)
	require.Equal(t, http.StatusOK, w.Code)

	entries := extractBeEntries(t, sc.gotBody)
	assert.Empty(t, entries, "skipped entry must not appear in BE body")
}

// Test 12a: notes omitted from BE body when empty.
func TestCreateHandler_Snapshots_NotesOmittedWhenEmpty(t *testing.T) {
	sc := &stubSnapshotCreator{snapshot: newCreatedSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	body := baseCreateBody()
	body["notes"] = ""
	w := postSnapshotCreate(r, "", body)
	require.Equal(t, http.StatusOK, w.Code)

	_, hasNotes := sc.gotBody["notes"]
	assert.False(t, hasNotes, "notes must be omitted when empty")
}

// Test 12b: notes included in BE body when non-empty.
func TestCreateHandler_Snapshots_NotesIncludedWhenPresent(t *testing.T) {
	sc := &stubSnapshotCreator{snapshot: newCreatedSnapshot()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(sc, NewGetUseCase(sf, cf), cf)
	r := newCreateSnapshotHandlerRouter(h)

	body := baseCreateBody()
	body["notes"] = "some note"
	w := postSnapshotCreate(r, "", body)
	require.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "some note", sc.gotBody["notes"])
}

// extractBeEntries pulls the "entries" slice out of the BE body and converts
// each element to map[string]any for assertions.
func extractBeEntries(t *testing.T, beBody map[string]any) []map[string]any {
	t.Helper()
	raw, ok := beBody["entries"]
	if !ok {
		return nil
	}
	slice, ok := raw.([]map[string]any)
	if ok {
		return slice
	}
	// Type may come back as []any when built via interface{}.
	anySlice, ok := raw.([]any)
	if !ok {
		t.Fatalf("entries is not a slice: %T", raw)
	}
	result := make([]map[string]any, 0, len(anySlice))
	for _, item := range anySlice {
		m, ok := item.(map[string]any)
		require.True(t, ok)
		result = append(result, m)
	}
	return result
}
