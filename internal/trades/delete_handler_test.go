package trades

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTradeDeleter captures DeleteTrade calls for assertions. It satisfies
// the narrow tradeDeleter interface the DeleteHandler depends on.
type stubTradeDeleter struct {
	err     error
	calls   int
	gotAuth string
	gotID   string
}

func (s *stubTradeDeleter) DeleteTrade(_ context.Context, auth, id string) error {
	s.calls++
	s.gotAuth = auth
	s.gotID = id
	return s.err
}

func newDeleteHandlerRouter(h *DeleteHandler) *gin.Engine {
	r := gin.New()
	r.DELETE("/actions/trades/:id", h.Delete)
	return r
}

func deleteTrade(r *gin.Engine, id, rawQuery string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, "/actions/trades/"+id+rawQuery, nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDeleteHandler_HappyPath(t *testing.T) {
	td := &stubTradeDeleter{}
	tf := &stubTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(td, NewGetUseCase(tf, cf))
	r := newDeleteHandlerRouter(h)

	w := deleteTrade(r, "t1", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, td.calls)
	assert.Equal(t, "Bearer tok", td.gotAuth)
	assert.Equal(t, "t1", td.gotID)

	// Screen rebuilt after successful delete.
	assert.Equal(t, 1, tf.calls)
	assert.Equal(t, 1, cf.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, ScreenID, body["target_id"])
	assert.NotNil(t, body["tree"])
	require.NotNil(t, body["feedback"], "success snackbar present")
}

func TestDeleteHandler_MissingID(t *testing.T) {
	// Mount on a path that doesn't expose :id so c.Param("id") is "".
	td := &stubTradeDeleter{}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(td, NewGetUseCase(tf, cf))

	r := gin.New()
	r.DELETE("/actions/trades/", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/actions/trades/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, td.calls)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "missing id", errObj["message"])
}

func TestDeleteHandler_BadQuery(t *testing.T) {
	td := &stubTradeDeleter{}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(td, NewGetUseCase(tf, cf))
	r := newDeleteHandlerRouter(h)

	w := deleteTrade(r, "t1", "?offset=abc")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, td.calls, "backend must not be called on bad query")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
}

func TestDeleteHandler_Unauthorized(t *testing.T) {
	td := &stubTradeDeleter{err: ErrUnauthorized}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(td, NewGetUseCase(tf, cf))
	r := newDeleteHandlerRouter(h)

	w := deleteTrade(r, "t1", "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 0, tf.calls, "screen not rebuilt on unauthorized")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestDeleteHandler_NotFound(t *testing.T) {
	td := &stubTradeDeleter{err: ErrTradeNotFound}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(td, NewGetUseCase(tf, cf))
	r := newDeleteHandlerRouter(h)

	w := deleteTrade(r, "t1", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, tf.calls, "screen not rebuilt on not-found")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", errObj["code"])
}

func TestDeleteHandler_BackendError(t *testing.T) {
	td := &stubTradeDeleter{err: ErrBackend}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(td, NewGetUseCase(tf, cf))
	r := newDeleteHandlerRouter(h)

	w := deleteTrade(r, "t1", "")

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 0, tf.calls, "screen not rebuilt on backend error")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

func TestDeleteHandler_BackendValidationErrorFallsThroughTo502(t *testing.T) {
	// Trades delete has no modal-replay flow — a *BackendValidationError on
	// delete is exceptional and must surface as 502 BACKEND_ERROR.
	td := &stubTradeDeleter{err: &BackendValidationError{Code: "CONFLICT", Message: "cannot delete"}}
	tf := &stubTradeFetcher{}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(td, NewGetUseCase(tf, cf))
	r := newDeleteHandlerRouter(h)

	w := deleteTrade(r, "t1", "")

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 0, tf.calls, "screen not rebuilt on validation error")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
	// No modal-replay response shape (no target_id == ModalSlotID, no tree).
	assert.NotContains(t, w.Body.String(), "\"action\":\"replace\"")
}

func TestDeleteHandler_ScreenRebuildFails(t *testing.T) {
	// Delete succeeds but screen rebuild (trades list) fails → 502.
	td := &stubTradeDeleter{}
	tf := &stubTradeFetcher{err: ErrBackend}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(td, NewGetUseCase(tf, cf))
	r := newDeleteHandlerRouter(h)

	w := deleteTrade(r, "t1", "")

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 1, td.calls, "delete was called")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}

func TestDeleteHandler_ScreenRebuildUnauthorized(t *testing.T) {
	// Delete succeeds but screen rebuild returns unauthorized → 401.
	td := &stubTradeDeleter{}
	tf := &stubTradeFetcher{err: ErrUnauthorized}
	cf := &stubCatalogFetcher{}
	h := NewDeleteHandler(td, NewGetUseCase(tf, cf))
	r := newDeleteHandlerRouter(h)

	w := deleteTrade(r, "t1", "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 1, td.calls, "delete was called")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}
