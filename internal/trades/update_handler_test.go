package trades

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

// stubTradeFetcherUpdater is a fake that satisfies tradeFetcherUpdater. It
// records the PATCH body it was called with so tests can assert on the
// diff-only body the handler constructs.
type stubTradeFetcherUpdater struct {
	// GetTrade state.
	trade      *Trade
	getErr     error
	getCalls   int
	gotGetAuth string
	gotGetID   string

	// UpdateTrade state.
	updated      *Trade
	updErr       error
	updCalls     int
	gotUpdAuth   string
	gotUpdID     string
	gotUpdBody   map[string]any
}

func (s *stubTradeFetcherUpdater) GetTrade(_ context.Context, auth, id string) (*Trade, error) {
	s.getCalls++
	s.gotGetAuth = auth
	s.gotGetID = id
	return s.trade, s.getErr
}

func (s *stubTradeFetcherUpdater) UpdateTrade(_ context.Context, auth, id string, body map[string]any) (*Trade, error) {
	s.updCalls++
	s.gotUpdAuth = auth
	s.gotUpdID = id
	s.gotUpdBody = body
	return s.updated, s.updErr
}

func newUpdateHandlerRouter(h *UpdateHandler) *gin.Engine {
	r := gin.New()
	r.PATCH("/actions/trades/:id", h.Patch)
	return r
}

func originalTrade() *Trade {
	return &Trade{
		ID:           "t1",
		AssetID:      validAssetUUID,
		TradeType:    "BUY",
		Quantity:     "10",
		PricePerUnit: "100",
		Fees:         "1",
		Date:         "2024-03-15T00:00:00Z",
		Source:       "MANUAL",
		Notes:        "hello",
	}
}

func updatedTrade() *Trade {
	t := *originalTrade()
	t.Quantity = "20"
	return &t
}

func baseEditBody() map[string]any {
	return map[string]any{
		"asset_id":       validAssetUUID,
		"trade_type":     "BUY",
		"quantity":       "10",
		"price_per_unit": "100",
		"fees":           "1",
		"notes":          "hello",
	}
}

func patchUpdate(r *gin.Engine, id, rawQuery string, body map[string]any) *httptest.ResponseRecorder {
	raw, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/actions/trades/"+id+rawQuery, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateHandler_HappyPath_PartialDiff(t *testing.T) {
	fu := &stubTradeFetcherUpdater{trade: originalTrade(), updated: updatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL", Currency: "USD"}}}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	reqBody := baseEditBody()
	reqBody["quantity"] = "20" // only change

	w := patchUpdate(r, "t1", "", reqBody)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, fu.getCalls)
	assert.Equal(t, 1, fu.updCalls)
	assert.Equal(t, "Bearer tok", fu.gotUpdAuth)
	assert.Equal(t, "t1", fu.gotUpdID)

	// PATCH body must contain only the changed field.
	require.NotNil(t, fu.gotUpdBody)
	assert.Equal(t, "20", fu.gotUpdBody["quantity"])
	_, hasAssetID := fu.gotUpdBody["asset_id"]
	assert.False(t, hasAssetID, "asset_id unchanged → MUST be excluded")
	_, hasTradeType := fu.gotUpdBody["trade_type"]
	assert.False(t, hasTradeType)
	_, hasPrice := fu.gotUpdBody["price_per_unit"]
	assert.False(t, hasPrice)
	_, hasFees := fu.gotUpdBody["fees"]
	assert.False(t, hasFees)
	_, hasNotes := fu.gotUpdBody["notes"]
	assert.False(t, hasNotes)
	// Date and source are immutable — never in the body.
	_, hasDate := fu.gotUpdBody["date"]
	assert.False(t, hasDate)
	_, hasSource := fu.gotUpdBody["source"]
	assert.False(t, hasSource)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ScreenID, body["target_id"])
	assert.NotNil(t, body["tree"])
	assert.NotNil(t, body["feedback"], "success snackbar present")
}

func TestUpdateHandler_NoOp_NoPatchCall(t *testing.T) {
	fu := &stubTradeFetcherUpdater{trade: originalTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL", Currency: "USD"}}}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	// Form is identical to the original.
	w := patchUpdate(r, "t1", "", baseEditBody())

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, fu.getCalls)
	assert.Equal(t, 0, fu.updCalls, "no diff → no PATCH call")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ScreenID, body["target_id"])
	assert.NotNil(t, body["feedback"])
}

func TestUpdateHandler_DateSilentlyIgnored(t *testing.T) {
	fu := &stubTradeFetcherUpdater{trade: originalTrade(), updated: updatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	body := baseEditBody()
	body["quantity"] = "20"
	body["date"] = "2099-01-01"

	w := patchUpdate(r, "t1", "", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, fu.updCalls)
	_, hasDate := fu.gotUpdBody["date"]
	assert.False(t, hasDate, "date must be silently ignored — never in PATCH body")
}

func TestUpdateHandler_SourceSilentlyIgnored(t *testing.T) {
	fu := &stubTradeFetcherUpdater{trade: originalTrade(), updated: updatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	body := baseEditBody()
	body["quantity"] = "20"
	body["source"] = "IMPORT"

	w := patchUpdate(r, "t1", "", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, fu.updCalls)
	_, hasSource := fu.gotUpdBody["source"]
	assert.False(t, hasSource, "source must be silently ignored — never in PATCH body")
}

func TestUpdateHandler_FeesCanonicalization_EmptyVsZero(t *testing.T) {
	orig := originalTrade()
	orig.Fees = ""
	fu := &stubTradeFetcherUpdater{trade: orig}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	body := baseEditBody()
	body["fees"] = "0" // canonical form of empty

	w := patchUpdate(r, "t1", "", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, fu.updCalls, "empty vs '0' → no diff, no PATCH")
}

func TestUpdateHandler_FeesCanonicalization_ZeroVsEmpty(t *testing.T) {
	orig := originalTrade()
	orig.Fees = "0"
	fu := &stubTradeFetcherUpdater{trade: orig}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	body := baseEditBody()
	body["fees"] = "" // canonicalizes to "0"

	w := patchUpdate(r, "t1", "", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, fu.updCalls, "'0' vs empty → no diff, no PATCH")
}

func TestUpdateHandler_NotesEmptyEquivalence(t *testing.T) {
	orig := originalTrade()
	orig.Notes = ""
	fu := &stubTradeFetcherUpdater{trade: orig}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	body := baseEditBody()
	body["notes"] = "" // both empty

	w := patchUpdate(r, "t1", "", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, fu.updCalls, "notes both empty → no diff")
}

func TestUpdateHandler_MissingID(t *testing.T) {
	// Route /actions/trades/ without an id → gin will 404. Exercise the
	// explicit missing-id branch via a direct gin.Context with no id.
	fu := &stubTradeFetcherUpdater{}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)

	r := gin.New()
	// Mount on a path that doesn't expose :id so c.Param("id") is "".
	r.PATCH("/actions/trades/", h.Patch)

	raw, _ := json.Marshal(baseEditBody())
	req := httptest.NewRequest(http.MethodPatch, "/actions/trades/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, fu.getCalls)
	assert.Equal(t, 0, fu.updCalls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "missing id", errObj["message"])
}

func TestUpdateHandler_BadQuery(t *testing.T) {
	fu := &stubTradeFetcherUpdater{}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	w := patchUpdate(r, "t1", "?offset=-1", baseEditBody())

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, fu.getCalls)
	assert.Equal(t, 0, fu.updCalls)
}

func TestUpdateHandler_GetTradeUnauthorized(t *testing.T) {
	fu := &stubTradeFetcherUpdater{getErr: ErrUnauthorized}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	w := patchUpdate(r, "t1", "", baseEditBody())

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 0, fu.updCalls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestUpdateHandler_GetTradeNotFound(t *testing.T) {
	fu := &stubTradeFetcherUpdater{getErr: ErrTradeNotFound}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	w := patchUpdate(r, "t1", "", baseEditBody())

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, fu.updCalls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", errObj["code"])
}

func TestUpdateHandler_GetTradeBackendError(t *testing.T) {
	fu := &stubTradeFetcherUpdater{getErr: ErrBackend}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	w := patchUpdate(r, "t1", "", baseEditBody())

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 0, fu.updCalls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

func TestUpdateHandler_UpdateValidationError(t *testing.T) {
	fu := &stubTradeFetcherUpdater{
		trade:  originalTrade(),
		updErr: &BackendValidationError{Code: "INSUFFICIENT_QUANTITY", Message: "Not enough"},
	}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL", Currency: "USD"}}}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	reqBody := baseEditBody()
	reqBody["quantity"] = "20" // force a diff

	w := patchUpdate(r, "t1", "", reqBody)

	require.Equal(t, http.StatusOK, w.Code) // modal replay
	assert.Equal(t, 1, fu.updCalls)
	assert.Equal(t, 1, cf.calls, "catalog re-fetched for modal replay")
	assert.Equal(t, 0, tf.calls, "trades list NOT fetched on validation error")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ModalSlotID, body["target_id"])
	// Inline error embedded in the Edit modal tree.
	assert.Contains(t, w.Body.String(), "Not enough")
	// Confirm this is the Edit modal (not Create) by looking for an Edit-only
	// field id (the quantity input uses trades-edit-quantity in BuildEditModal).
	assert.Contains(t, w.Body.String(), "trades-edit-")
}

func TestUpdateHandler_UpdateUnauthorized(t *testing.T) {
	fu := &stubTradeFetcherUpdater{trade: originalTrade(), updErr: ErrUnauthorized}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	reqBody := baseEditBody()
	reqBody["quantity"] = "20"

	w := patchUpdate(r, "t1", "", reqBody)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestUpdateHandler_UpdateBackendError(t *testing.T) {
	fu := &stubTradeFetcherUpdater{trade: originalTrade(), updErr: ErrBackend}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	reqBody := baseEditBody()
	reqBody["quantity"] = "20"

	w := patchUpdate(r, "t1", "", reqBody)

	assert.Equal(t, http.StatusBadGateway, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

func TestUpdateHandler_ValidationErrorCatalogRefetchFails(t *testing.T) {
	fu := &stubTradeFetcherUpdater{
		trade:  originalTrade(),
		updErr: &BackendValidationError{Code: "INSUFFICIENT_QUANTITY", Message: "Not enough"},
	}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrBackend}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	reqBody := baseEditBody()
	reqBody["quantity"] = "20"

	w := patchUpdate(r, "t1", "", reqBody)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestUpdateHandler_ScreenRebuildFails(t *testing.T) {
	// Update succeeds but screen rebuild (trades list) fails → 502.
	fu := &stubTradeFetcherUpdater{trade: originalTrade(), updated: updatedTrade()}
	tf := &stubTradeFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewUpdateHandler(fu, NewGetUseCase(tf, cf), cf)
	r := newUpdateHandlerRouter(h)

	reqBody := baseEditBody()
	reqBody["quantity"] = "20"

	w := patchUpdate(r, "t1", "", reqBody)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
