package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func postRegister(t *testing.T, body string, cli *Client) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/register", NewRegisterHandler(cli).Post)
	req := httptest.NewRequest("POST", "/actions/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRegisterHandler_Success(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "u1"})
	}))
	defer backend.Close()

	w := postRegister(t, `{"email":"a@b.com","password":"pw"}`, NewClient(backend.URL, 5*time.Second))
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "navigate", resp["action"])
	assert.Equal(t, "/login", resp["target_id"])
}

func TestRegisterHandler_Disabled(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer backend.Close()

	w := postRegister(t, `{"email":"a@b.com","password":"pw"}`, NewClient(backend.URL, 5*time.Second))
	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "REGISTRATION_DISABLED")
}

func TestRegisterHandler_EmailExists(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer backend.Close()

	w := postRegister(t, `{"email":"a@b.com","password":"pw"}`, NewClient(backend.URL, 5*time.Second))
	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "EMAIL_ALREADY_EXISTS")
}

func TestRegisterHandler_BadRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("backend should not be called on bad request")
	}))
	defer backend.Close()

	w := postRegister(t, `{"email":""}`, NewClient(backend.URL, 5*time.Second))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
