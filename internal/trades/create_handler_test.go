package trades

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// stubTradeCreator captures CreateTrade calls for assertions.
type stubTradeCreator struct {
	trade   *Trade
	err     error
	calls   int
	gotAuth string
	gotBody map[string]any
}

func (s *stubTradeCreator) CreateTrade(_ context.Context, auth string, body map[string]any) (*Trade, error) {
	s.calls++
	s.gotAuth = auth
	s.gotBody = body
	return s.trade, s.err
}

func newCreateHandlerRouter(h *CreateHandler) *gin.Engine {
	r := gin.New()
	r.POST("/actions/trades/create", h.Post)
	return r
}

func newCreatedTrade() *Trade {
	return &Trade{
		ID:        "new-trade",
		AssetID:   validAssetUUID,
		TradeType: "BUY",
	}
}

func baseForm() url.Values {
	v := url.Values{}
	v.Set("asset_id", validAssetUUID)
	v.Set("trade_type", "BUY")
	v.Set("quantity", "10")
	v.Set("price_per_unit", "100")
	v.Set("fees", "")
	v.Set("date", "2024-01-15")
	v.Set("notes", "")
	return v
}

func postCreate(r *gin.Engine, rawQuery string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/actions/trades/create"+rawQuery, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateHandler_HappyPath(t *testing.T) {
	tc := &stubTradeCreator{trade: newCreatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL", Currency: "USD"}}}
	uc := NewGetUseCase(tf, cf)
	h := NewCreateHandler(tc, uc, cf)
	r := newCreateHandlerRouter(h)

	w := postCreate(r, "", baseForm())

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, tc.calls)
	assert.Equal(t, "Bearer tok", tc.gotAuth)
	// Backend body shape assertions.
	assert.Equal(t, validAssetUUID, tc.gotBody["asset_id"])
	assert.Equal(t, "BUY", tc.gotBody["trade_type"])
	assert.Equal(t, "10", tc.gotBody["quantity"])
	assert.Equal(t, "100", tc.gotBody["price_per_unit"])
	assert.Equal(t, "0", tc.gotBody["fees"], "fees default to \"0\" when empty")
	assert.Equal(t, "2024-01-15T00:00:00Z", tc.gotBody["date"], "date converted to RFC3339")
	assert.Equal(t, "MANUAL", tc.gotBody["source"], "source always injected by middleend")
	_, hasNotes := tc.gotBody["notes"]
	assert.False(t, hasNotes, "notes omitted when empty")

	// Response shape.
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ScreenID, body["target_id"])
	assert.NotNil(t, body["tree"])
	assert.NotNil(t, body["feedback"], "success snackbar present")

	// Screen rebuild called both the trades list and the catalog.
	assert.Equal(t, 1, tf.calls)
	assert.Equal(t, 1, cf.calls)
}

func TestCreateHandler_FeesDefault(t *testing.T) {
	tc := &stubTradeCreator{trade: newCreatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	form := baseForm()
	form.Set("fees", "")
	w := postCreate(r, "", form)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "0", tc.gotBody["fees"])
}

func TestCreateHandler_FeesProvided(t *testing.T) {
	tc := &stubTradeCreator{trade: newCreatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	form := baseForm()
	form.Set("fees", "1.50")
	w := postCreate(r, "", form)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1.50", tc.gotBody["fees"])
}

func TestCreateHandler_NotesOmittedWhenEmpty(t *testing.T) {
	tc := &stubTradeCreator{trade: newCreatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	form := baseForm()
	form.Set("notes", "")
	w := postCreate(r, "", form)

	require.Equal(t, http.StatusOK, w.Code)
	_, hasNotes := tc.gotBody["notes"]
	assert.False(t, hasNotes)
}

func TestCreateHandler_NotesIncludedWhenPresent(t *testing.T) {
	tc := &stubTradeCreator{trade: newCreatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	form := baseForm()
	form.Set("notes", "hello")
	w := postCreate(r, "", form)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "hello", tc.gotBody["notes"])
}

func TestCreateHandler_SourceAlwaysInjected(t *testing.T) {
	tc := &stubTradeCreator{trade: newCreatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	form := baseForm()
	// Attempt to spoof source via form — the handler must ignore it.
	form.Set("source", "IMPORTED")
	w := postCreate(r, "", form)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "MANUAL", tc.gotBody["source"], "source must never come from the form")
}

func TestCreateHandler_DateConversion(t *testing.T) {
	tc := &stubTradeCreator{trade: newCreatedTrade()}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{}}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	form := baseForm()
	form.Set("date", "2024-03-10")
	w := postCreate(r, "", form)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2024-03-10T00:00:00Z", tc.gotBody["date"])
}

func TestCreateHandler_BadQuery(t *testing.T) {
	tc := &stubTradeCreator{}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	w := postCreate(r, "?offset=-1", baseForm())

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, tc.calls, "backend must not be called on bad query")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

func TestCreateHandler_ValidationError(t *testing.T) {
	tc := &stubTradeCreator{err: &BackendValidationError{Code: "INSUFFICIENT_QUANTITY", Message: "Not enough"}}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{res: []assetscatalog.Asset{{ID: validAssetUUID, Ticker: "AAPL", Currency: "USD"}}}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	w := postCreate(r, "", baseForm())

	require.Equal(t, http.StatusOK, w.Code) // modal replay on validation error
	assert.Equal(t, 1, tc.calls)
	assert.Equal(t, 1, cf.calls, "catalog re-fetched to rebuild modal with asset options")
	assert.Equal(t, 0, tf.calls, "trades list NOT fetched on validation error")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ModalSlotID, body["target_id"])
	// Inline error embedded in modal tree.
	assert.Contains(t, w.Body.String(), "Not enough")
}

func TestCreateHandler_ValidationErrorCatalogRefetchFails(t *testing.T) {
	// When the catalog re-fetch fails during modal rebuild, we surface a 502.
	tc := &stubTradeCreator{err: &BackendValidationError{Code: "INSUFFICIENT_QUANTITY", Message: "Not enough"}}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{err: assetscatalog.ErrBackend}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	w := postCreate(r, "", baseForm())

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestCreateHandler_Unauthorized(t *testing.T) {
	tc := &stubTradeCreator{err: ErrUnauthorized}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	w := postCreate(r, "", baseForm())

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestCreateHandler_BackendError(t *testing.T) {
	tc := &stubTradeCreator{err: ErrBackend}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	w := postCreate(r, "", baseForm())

	assert.Equal(t, http.StatusBadGateway, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

func TestCreateHandler_ScreenRebuildFails(t *testing.T) {
	// Create succeeds but screen rebuild fails -> 502.
	tc := &stubTradeCreator{trade: newCreatedTrade()}
	tf := &stubTradeFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	w := postCreate(r, "", baseForm())

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestCreateHandler_ScreenRebuildUnauthorized(t *testing.T) {
	// Create succeeds but screen rebuild returns unauthorized -> 401.
	tc := &stubTradeCreator{trade: newCreatedTrade()}
	tf := &stubTradeFetcher{err: ErrUnauthorized}
	cf := &stubCatalogFetcher{}
	h := NewCreateHandler(tc, NewGetUseCase(tf, cf), cf)
	r := newCreateHandlerRouter(h)

	w := postCreate(r, "", baseForm())

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
