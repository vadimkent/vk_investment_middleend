package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

func TestMain(m *testing.M) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = i18n.Load(filepath.Join(repoRoot, "locales"))
	m.Run()
}

type fakeRegistrar struct {
	gotEmail, gotPassword string
	err                   error
}

func (f *fakeRegistrar) Register(_ context.Context, email, password string) error {
	f.gotEmail, f.gotPassword = email, password
	return f.err
}

func postRegisterFake(t *testing.T, body string, reg *fakeRegistrar) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/register", NewRegisterHandler(reg).Post)
	req := httptest.NewRequest("POST", "/actions/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRegisterHandler_Success(t *testing.T) {
	reg := &fakeRegistrar{}
	w := postRegisterFake(t, `{"email":"a@b.com","password":"longpass1","confirm_password":"longpass1"}`, reg)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "navigate", resp["action"])
	assert.Equal(t, "/screens/login", resp["target_id"])
}

func TestRegisterHandler_StripsConfirmPassword(t *testing.T) {
	reg := &fakeRegistrar{}
	w := postRegisterFake(t, `{"email":"a@b.com","password":"longpass1","confirm_password":"longpass1"}`, reg)
	require.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "a@b.com", reg.gotEmail)
	assert.Equal(t, "longpass1", reg.gotPassword)
	// confirm_password must not reach the backend registrar
	assert.NotEqual(t, "confirm_password", reg.gotPassword)
}

func TestRegisterHandler_PasswordTooShort(t *testing.T) {
	reg := &fakeRegistrar{}
	w := postRegisterFake(t, `{"email":"a@b.com","password":"short","confirm_password":"short"}`, reg)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "register-form", resp["target_id"])
	assert.Contains(t, w.Body.String(), "Please check the form")
}

func TestRegisterHandler_Mismatch(t *testing.T) {
	reg := &fakeRegistrar{}
	w := postRegisterFake(t, `{"email":"a@b.com","password":"longpass1","confirm_password":"different1"}`, reg)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
}

func TestRegisterHandler_EmailAlreadyExists(t *testing.T) {
	reg := &fakeRegistrar{err: ErrEmailAlreadyExists}
	w := postRegisterFake(t, `{"email":"a@b.com","password":"longpass1","confirm_password":"longpass1"}`, reg)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Contains(t, w.Body.String(), "already exists")
}

func TestRegisterHandler_RegistrationDisabled(t *testing.T) {
	reg := &fakeRegistrar{err: ErrRegistrationDisabled}
	w := postRegisterFake(t, `{"email":"a@b.com","password":"longpass1","confirm_password":"longpass1"}`, reg)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Contains(t, w.Body.String(), "disabled")
	assert.Contains(t, w.Body.String(), `"disabled":true`)
}

func TestRegisterHandler_Transient(t *testing.T) {
	reg := &fakeRegistrar{err: errors.New("boom")}
	w := postRegisterFake(t, `{"email":"a@b.com","password":"longpass1","confirm_password":"longpass1"}`, reg)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Contains(t, w.Body.String(), "snackbar")
}

func TestRegisterHandler_EmailAlreadyExists_EmitsSnackbar(t *testing.T) {
	reg := &fakeRegistrar{err: ErrEmailAlreadyExists}
	w := postRegisterFake(t, `{"email":"a@b.com","password":"longpass1","confirm_password":"longpass1"}`, reg)
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, `"type":"snackbar"`)
	assert.Contains(t, body, `"variant":"error"`)
	assert.Contains(t, body, "already exists")
}
