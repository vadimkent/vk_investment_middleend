package profile

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPasswordChanger struct {
	called bool
	err    error
}

func (s *stubPasswordChanger) ChangePassword(_ context.Context, _, _, _ string) error {
	s.called = true
	return s.err
}

func newPasswordRouter(c *stubPasswordChanger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/profile/change_password", NewChangePasswordHandler(c).Post)
	return r
}

func TestChangePassword_Happy(t *testing.T) {
	pc := &stubPasswordChanger{}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"old","new_password":"newPassword!","confirm_password":"newPassword!"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, pc.called)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, PasswordFormID, resp["target_id"])
	require.NotNil(t, resp["feedback"])
}

func TestChangePassword_MissingFields_NoBECall(t *testing.T) {
	pc := &stubPasswordChanger{}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"","new_password":"","confirm_password":""}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, pc.called, "BE must not be called when middleend validation fails")
	assert.Contains(t, w.Body.String(), "password-card-error")
}

func TestChangePassword_PartialMissing_NoBECall(t *testing.T) {
	pc := &stubPasswordChanger{}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"a","new_password":"b","confirm_password":""}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, pc.called)
	assert.Contains(t, w.Body.String(), "password-card-error")
}

func TestChangePassword_DoNotMatch_NoBECall(t *testing.T) {
	pc := &stubPasswordChanger{}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"a","new_password":"b","confirm_password":"c"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, pc.called)
	assert.Contains(t, w.Body.String(), "password-card-error")
}

func TestChangePassword_BEInvalidCredentials(t *testing.T) {
	pc := &stubPasswordChanger{err: &BackendValidationError{Code: "INVALID_CREDENTIALS", Message: "wrong"}}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"old","new_password":"newPassword!","confirm_password":"newPassword!"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, pc.called)
	assert.Contains(t, w.Body.String(), "password-card-error")
}

func TestChangePassword_BEInvalidPassword(t *testing.T) {
	pc := &stubPasswordChanger{err: &BackendValidationError{Code: "INVALID_PASSWORD", Message: "too short"}}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"old","new_password":"abc","confirm_password":"abc"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, pc.called)
	assert.Contains(t, w.Body.String(), "password-card-error")
}

func TestChangePassword_BadJSON_400(t *testing.T) {
	pc := &stubPasswordChanger{}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChangePassword_BackendError_502(t *testing.T) {
	pc := &stubPasswordChanger{err: ErrBackend}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"a","new_password":"b","confirm_password":"b"}`)
	require.Equal(t, http.StatusBadGateway, w.Code)
}
