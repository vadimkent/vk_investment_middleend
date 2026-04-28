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

type stubDeleter struct{ err error }

func (s *stubDeleter) DeleteAccount(_ context.Context, _, _ string) error { return s.err }

func newDeleteRouter(d *stubDeleter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/profile/delete_account", NewDeleteHandler(d).Post)
	return r
}

func TestDeleteHandler_Happy_LogoutResponse(t *testing.T) {
	r := newDeleteRouter(&stubDeleter{})
	w := postJSON(t, r, "/actions/profile/delete_account", `{"password":"pw"}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "logout", resp["action"])
	assert.Equal(t, "/screens/login", resp["target_id"])
	assert.Nil(t, resp["feedback"])
}

func TestDeleteHandler_InvalidCredentials_RemodalsWithError(t *testing.T) {
	r := newDeleteRouter(&stubDeleter{err: &BackendValidationError{Code: "INVALID_CREDENTIALS", Message: "wrong"}})
	w := postJSON(t, r, "/actions/profile/delete_account", `{"password":"x"}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ModalSlotID, resp["target_id"])
	assert.Contains(t, w.Body.String(), "delete-modal-error")
}

func TestDeleteHandler_MissingFields_RemodalsWithError(t *testing.T) {
	r := newDeleteRouter(&stubDeleter{err: &BackendValidationError{Code: "MISSING_FIELDS", Message: "missing"}})
	w := postJSON(t, r, "/actions/profile/delete_account", `{}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "delete-modal-error")
}

func TestDeleteHandler_BadJSON_400(t *testing.T) {
	r := newDeleteRouter(&stubDeleter{})
	w := postJSON(t, r, "/actions/profile/delete_account", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteHandler_BackendError_502(t *testing.T) {
	r := newDeleteRouter(&stubDeleter{err: ErrBackend})
	w := postJSON(t, r, "/actions/profile/delete_account", `{"password":"pw"}`)
	require.Equal(t, http.StatusBadGateway, w.Code)
}
