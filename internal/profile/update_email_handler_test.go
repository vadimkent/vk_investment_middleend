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

type stubEmailUpdater struct {
	err error
}

func (s *stubEmailUpdater) UpdateEmail(_ context.Context, _, _, _ string) error { return s.err }

func newEmailRouter(upd *stubEmailUpdater, me *stubMe) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/profile/update_email", NewUpdateEmailHandler(upd, me).Post)
	return r
}

func TestUpdateEmail_Happy_RebuildsCardWithNewEmail(t *testing.T) {
	updated := &User{ID: "u1", Email: "new@example.com"}
	me := &stubMe{res: updated}
	r := newEmailRouter(&stubEmailUpdater{}, me)
	w := postJSON(t, r, "/actions/profile/update_email", `{"new_email":"new@example.com","current_password":"pw"}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, EmailFormID, resp["target_id"])
	require.NotNil(t, resp["feedback"])
	assert.Contains(t, w.Body.String(), "new@example.com")
}

func TestUpdateEmail_InvalidCredentials_PreservesNewEmailClearsPassword(t *testing.T) {
	r := newEmailRouter(
		&stubEmailUpdater{err: &BackendValidationError{Code: "INVALID_CREDENTIALS", Message: "wrong"}},
		&stubMe{res: sampleUser()},
	)
	w := postJSON(t, r, "/actions/profile/update_email", `{"new_email":"preserved@x.y","current_password":"pw"}`)
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, `"default_value":"preserved@x.y"`)
	assert.Contains(t, body, "email-card-error")
	// The current_password field is rendered with empty default — InputFull omits
	// the default_value attr when empty. Confirm the *new_email* is the only
	// preserved input by checking we do NOT see a preserved password value.
	assert.NotContains(t, body, `"name":"current_password","default_value":"pw"`)
}

func TestUpdateEmail_EmailExists(t *testing.T) {
	r := newEmailRouter(
		&stubEmailUpdater{err: &BackendValidationError{Code: "EMAIL_ALREADY_EXISTS", Message: "in use"}},
		&stubMe{res: sampleUser()},
	)
	w := postJSON(t, r, "/actions/profile/update_email", `{"new_email":"taken@x.y","current_password":"pw"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "email-card-error")
}

func TestUpdateEmail_MissingFields(t *testing.T) {
	r := newEmailRouter(
		&stubEmailUpdater{err: &BackendValidationError{Code: "MISSING_FIELDS", Message: "missing"}},
		&stubMe{res: sampleUser()},
	)
	w := postJSON(t, r, "/actions/profile/update_email", `{}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "email-card-error")
}

func TestUpdateEmail_BadJSON_400(t *testing.T) {
	r := newEmailRouter(&stubEmailUpdater{}, &stubMe{res: sampleUser()})
	w := postJSON(t, r, "/actions/profile/update_email", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateEmail_BackendError_502(t *testing.T) {
	r := newEmailRouter(&stubEmailUpdater{err: ErrBackend}, &stubMe{res: sampleUser()})
	w := postJSON(t, r, "/actions/profile/update_email", `{}`)
	require.Equal(t, http.StatusBadGateway, w.Code)
}
