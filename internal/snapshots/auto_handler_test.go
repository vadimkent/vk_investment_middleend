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

// stubSnapshotAutoCreator stubs AutoSnapshot calls.
type stubSnapshotAutoCreator struct {
	result  *AutoResult
	err     error
	calls   int
	gotAuth string
	gotNotes string
}

func (s *stubSnapshotAutoCreator) AutoSnapshot(_ context.Context, auth, notes string) (*AutoResult, error) {
	s.calls++
	s.gotAuth = auth
	s.gotNotes = notes
	return s.result, s.err
}

func newAutoHandlerRouter(h *AutoHandler) *gin.Engine {
	r := gin.New()
	r.POST("/actions/snapshots/auto", h.Post)
	return r
}

// postSnapshotsAuto issues a POST to /actions/snapshots/auto with optional query string.
func postSnapshotsAuto(r *gin.Engine, rawQuery string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/actions/snapshots/auto"+rawQuery, http.NoBody)
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Accept-Language", "en")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func okAutoResult() *AutoResult {
	return &AutoResult{
		Snapshot: Snapshot{
			ID:             "auto-snap-1",
			RecordedAt:     "2025-04-20T10:00:00Z",
			IsFullSnapshot: true,
		},
		Warnings: nil,
	}
}

func okAutoResultWithWarnings() *AutoResult {
	return &AutoResult{
		Snapshot: Snapshot{
			ID:             "auto-snap-2",
			RecordedAt:     "2025-04-20T10:00:00Z",
			IsFullSnapshot: true,
		},
		Warnings: []AutoWarning{
			{AssetID: "aaa-1", Ticker: "AAPL", Error: "rate_limited"},
			{AssetID: "bbb-2", Ticker: "TSLA", Error: "not_found"},
		},
	}
}

// Test 1: Happy path (no warnings) — ActionResponse with replace ScreenID, tree has list + wizard.
func TestAutoHandler_HappyPath_NoWarnings(t *testing.T) {
	ac := &stubSnapshotAutoCreator{result: okAutoResult()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: "aaa-1", Ticker: "AAPL", Currency: "USD"}}}
	uc := NewGetUseCase(sf, cf)
	h := NewAutoHandler(ac, uc, cf)
	r := newAutoHandlerRouter(h)

	w := postSnapshotsAuto(r, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, ac.calls)
	assert.Equal(t, "Bearer tok", ac.gotAuth)
	assert.Equal(t, "", ac.gotNotes, "notes must be empty — user fills in wizard")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ScreenID, body["target_id"])
	assert.NotNil(t, body["tree"], "tree must be present")
	assert.NotNil(t, body["feedback"], "success snackbar must be present")

	// Verify modal slot in tree contains the wizard.
	rawTree, _ := json.Marshal(body["tree"])
	assert.Contains(t, string(rawTree), ModalSlotID, "tree must contain the modal slot")
	assert.Contains(t, string(rawTree), WizardID, "tree must contain the edit wizard")

	// Verify banner variant in serialised tree.
	assert.Contains(t, string(rawTree), `"info"`, "wizard banner must use info variant")
}

// Test 2: Happy path with warnings — banner message includes failed tickers.
func TestAutoHandler_HappyPath_WithWarnings(t *testing.T) {
	ac := &stubSnapshotAutoCreator{result: okAutoResultWithWarnings()}
	sf := &stubSnapshotFetcher{res: &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	uc := NewGetUseCase(sf, cf)
	h := NewAutoHandler(ac, uc, cf)
	r := newAutoHandlerRouter(h)

	w := postSnapshotsAuto(r, "")

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])

	rawTree, _ := json.Marshal(body["tree"])
	treeStr := string(rawTree)

	// The banner message must mention the failed tickers.
	assert.Contains(t, treeStr, "AAPL", "banner must list AAPL warning")
	assert.Contains(t, treeStr, "TSLA", "banner must list TSLA warning")
}

// Test 3: NO_PRICE_PROVIDERS_CONFIGURED → feedback-only snackbar (warning), no tree replace.
func TestAutoHandler_NoProvidersConfigured(t *testing.T) {
	ac := &stubSnapshotAutoCreator{err: &BackendValidationError{Code: "NO_PRICE_PROVIDERS_CONFIGURED", Message: "no providers"}}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	uc := NewGetUseCase(sf, cf)
	h := NewAutoHandler(ac, uc, cf)
	r := newAutoHandlerRouter(h)

	w := postSnapshotsAuto(r, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, sf.calls, "list must not be fetched on terminal BE error")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	// Feedback-only: action is "none", no tree.
	assert.Equal(t, "none", body["action"])
	assert.Nil(t, body["tree"], "no tree on feedback-only response")
	assert.NotNil(t, body["feedback"], "feedback snackbar must be present")

	// Snackbar variant must be "warning".
	fbRaw, _ := json.Marshal(body["feedback"])
	assert.Contains(t, string(fbRaw), `"warning"`)
}

// Test 4: ALL_PROVIDERS_FAILED → feedback-only snackbar (error), no tree replace.
func TestAutoHandler_AllProvidersFailed(t *testing.T) {
	ac := &stubSnapshotAutoCreator{err: &BackendValidationError{Code: "ALL_PROVIDERS_FAILED", Message: "all failed"}}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	uc := NewGetUseCase(sf, cf)
	h := NewAutoHandler(ac, uc, cf)
	r := newAutoHandlerRouter(h)

	w := postSnapshotsAuto(r, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, sf.calls, "list must not be fetched on terminal BE error")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "none", body["action"])
	assert.Nil(t, body["tree"])
	assert.NotNil(t, body["feedback"])

	// Snackbar variant must be "error".
	fbRaw, _ := json.Marshal(body["feedback"])
	assert.Contains(t, string(fbRaw), `"error"`)
}

// Test 5: ErrUnauthorized → 401 redirect.
func TestAutoHandler_Unauthorized(t *testing.T) {
	ac := &stubSnapshotAutoCreator{err: ErrUnauthorized}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	uc := NewGetUseCase(sf, cf)
	h := NewAutoHandler(ac, uc, cf)
	r := newAutoHandlerRouter(h)

	w := postSnapshotsAuto(r, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

// Test 6: Generic backend error → 502.
func TestAutoHandler_GenericBackendError(t *testing.T) {
	ac := &stubSnapshotAutoCreator{err: ErrBackend}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	uc := NewGetUseCase(sf, cf)
	h := NewAutoHandler(ac, uc, cf)
	r := newAutoHandlerRouter(h)

	w := postSnapshotsAuto(r, "")

	assert.Equal(t, http.StatusBadGateway, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

// Test 7: Bad query param → 400.
func TestAutoHandler_BadQuery(t *testing.T) {
	ac := &stubSnapshotAutoCreator{}
	sf := &stubSnapshotFetcher{}
	cf := &stubCatalogFetcher{}
	uc := NewGetUseCase(sf, cf)
	h := NewAutoHandler(ac, uc, cf)
	r := newAutoHandlerRouter(h)

	w := postSnapshotsAuto(r, "?offset=-1")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, ac.calls, "auto creator must not be called on bad query")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

// Test 8: Success but uc (list refresh) fails → 502.
func TestAutoHandler_RefreshFails(t *testing.T) {
	ac := &stubSnapshotAutoCreator{result: okAutoResult()}
	sf := &stubSnapshotFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	uc := NewGetUseCase(sf, cf)
	h := NewAutoHandler(ac, uc, cf)
	r := newAutoHandlerRouter(h)

	w := postSnapshotsAuto(r, "")

	assert.Equal(t, http.StatusBadGateway, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}
