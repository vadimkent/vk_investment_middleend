package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func postLogin(t *testing.T, body string, cli *Client) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/login", NewLoginHandler(cli).Post)
	req := httptest.NewRequest("POST", "/actions/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLoginHandler_Success(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "jwt-xyz", "expires_at": "2026-04-15T12:00:00Z"})
	}))
	defer backend.Close()

	w := postLogin(t, `{"email":"a@b.com","password":"pw"}`, NewClient(backend.URL, 5*time.Second))

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "navigate", resp["action"])
	assert.Equal(t, "/shell", resp["target_id"])

	auth, ok := resp["auth"].(map[string]any)
	require.True(t, ok, "auth should be present, got: %s", w.Body.String())
	assert.Equal(t, "jwt-xyz", auth["token"])
	assert.Equal(t, "2026-04-15T12:00:00Z", auth["expires_at"])
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer backend.Close()

	w := postLogin(t, `{"email":"a@b.com","password":"nope"}`, NewClient(backend.URL, 5*time.Second))

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "none", resp["action"])
	assert.Nil(t, resp["auth"])
	assert.NotNil(t, resp["feedback"])
}

func TestLoginHandler_BadRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("backend should not be called on bad request")
	}))
	defer backend.Close()

	w := postLogin(t, `not-json`, NewClient(backend.URL, 5*time.Second))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginHandler_MissingFields(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("backend should not be called when fields are missing")
	}))
	defer backend.Close()

	w := postLogin(t, `{"email":""}`, NewClient(backend.URL, 5*time.Second))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginHandler_NavigateFeedbackSnackbar(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "t", "expires_at": "x"})
	}))
	defer backend.Close()

	w := postLogin(t, `{"email":"a@b.com","password":"pw"}`, NewClient(backend.URL, 5*time.Second))
	assert.True(t, strings.Contains(w.Body.String(), `"type":"snackbar"`))
}
