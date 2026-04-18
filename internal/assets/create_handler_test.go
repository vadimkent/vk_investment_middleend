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

type stubMutator struct {
	created   *Asset
	createErr error
	list      *ListResult
	listErr   error
	asset     *Asset
	assetErr  error
}

func (s *stubMutator) CreateAsset(_ context.Context, _ string, _ map[string]any) (*Asset, error) {
	return s.created, s.createErr
}
func (s *stubMutator) UpdateAsset(_ context.Context, _, _ string, _ map[string]any) (*Asset, error) {
	return nil, nil
}
func (s *stubMutator) DeleteAsset(_ context.Context, _, _ string, _ bool) error {
	return nil
}
func (s *stubMutator) GetAsset(_ context.Context, _, _ string) (*Asset, error) {
	return s.asset, s.assetErr
}
func (s *stubMutator) List(_ context.Context, _ string, _ ListParams) (*ListResult, error) {
	return s.list, s.listErr
}

func TestCreateHandler_HappyPath(t *testing.T) {
	sc := &stubMutator{
		created: &Asset{ID: "a1", Ticker: "TSLA"},
		list:    &ListResult{Assets: []Asset{{ID: "a1", Ticker: "TSLA"}}, Total: 1, Size: 10},
	}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{"ticker": "TSLA", "name": "Tesla", "asset_type": "STOCK", "currency": "USD"})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create?asset_type=STOCK&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var respBody map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Equal(t, "replace", respBody["action"])
	assert.Equal(t, "assets-root", respBody["target_id"])
	fb := respBody["feedback"].(map[string]any)
	assert.Equal(t, "snackbar", fb["type"])
	assert.Equal(t, "Asset created", fb["props"].(map[string]any)["message"])
}

func TestCreateHandler_ValidationError(t *testing.T) {
	sc := &stubMutator{createErr: &BackendValidationError{Code: "ASSET_ALREADY_EXISTS", Message: "Ticker already registered"}}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{"ticker": "AAPL"})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create?asset_type=&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code) // handler returns 200 with replace pointing at modal
	var respBody map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Equal(t, "replace", respBody["action"])
	assert.Equal(t, "assets-modal-slot", respBody["target_id"])
}

func TestCreateHandler_Unauthorized(t *testing.T) {
	sc := &stubMutator{createErr: ErrUnauthorized}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateHandler_BackendError(t *testing.T) {
	sc := &stubMutator{createErr: ErrBackend}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestCreateHandler_ListRefreshUnauthorizedAfterSuccess(t *testing.T) {
	sc := &stubMutator{
		created: &Asset{ID: "a1", Ticker: "TSLA"},
		listErr: ErrUnauthorized,
	}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{"ticker": "TSLA"})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateHandler_ListRefreshBackendErrorAfterSuccess(t *testing.T) {
	sc := &stubMutator{
		created: &Asset{ID: "a1", Ticker: "TSLA"},
		listErr: ErrBackend,
	}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{"ticker": "TSLA"})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
