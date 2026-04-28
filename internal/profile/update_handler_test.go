package profile

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

type stubProfileUpdater struct {
	res     *User
	err     error
	gotBody map[string]any
}

func (s *stubProfileUpdater) UpdateProfile(_ context.Context, _ string, body map[string]any) (*User, error) {
	s.gotBody = body
	return s.res, s.err
}

func newUpdateRouter(updater *stubProfileUpdater, cfg *stubCfg) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/profile/update", NewUpdateHandler(updater, cfg).Post)
	return r
}

func postJSON(t *testing.T, r http.Handler, path, jsonBody string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer t")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateHandler_Happy(t *testing.T) {
	updated := &User{ID: "u1", Email: "vadim@example.com", DisplayName: ptr("Vadim"), Preferences: Preferences{DefaultCurrency: ptr("EUR")}}
	upd := &stubProfileUpdater{res: updated}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)

	w := postJSON(t, r, "/actions/profile/update", `{"display_name":"Vadim","default_currency":"EUR"}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ProfileCardID, resp["target_id"])
	require.NotNil(t, resp["feedback"])
	assert.Equal(t, "Vadim", upd.gotBody["display_name"])
}

func TestUpdateHandler_EmptyDisplayNameSentAsNull(t *testing.T) {
	upd := &stubProfileUpdater{res: sampleUser()}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)

	w := postJSON(t, r, "/actions/profile/update", `{"display_name":"  ","default_currency":""}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, upd.gotBody["display_name"])
	prefs := upd.gotBody["preferences"].(map[string]any)
	assert.Nil(t, prefs["default_currency"])
}

func TestUpdateHandler_BackendValidationError_BannerInline(t *testing.T) {
	upd := &stubProfileUpdater{err: &BackendValidationError{Code: "INVALID_DISPLAY_NAME", Message: "too long"}}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)

	w := postJSON(t, r, "/actions/profile/update", `{"display_name":"x","default_currency":"USD"}`)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ProfileCardID, resp["target_id"])
	assert.Nil(t, resp["feedback"])
	assert.Contains(t, w.Body.String(), "profile-card-error")
}

func TestUpdateHandler_InvalidCurrencyError(t *testing.T) {
	upd := &stubProfileUpdater{err: &BackendValidationError{Code: "INVALID_CURRENCY", Message: "bad"}}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)

	w := postJSON(t, r, "/actions/profile/update", `{"default_currency":"XXX"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "profile-card-error")
}

func TestUpdateHandler_BadJSON_400(t *testing.T) {
	upd := &stubProfileUpdater{}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)
	w := postJSON(t, r, "/actions/profile/update", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHandler_BackendError_502(t *testing.T) {
	upd := &stubProfileUpdater{err: ErrBackend}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)
	w := postJSON(t, r, "/actions/profile/update", `{}`)
	require.Equal(t, http.StatusBadGateway, w.Code)
}
