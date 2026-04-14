package login

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/screens/login", NewHandler().Get)
	return r
}

func TestHandler_Returns200WithoutAuth(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/screens/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ReturnsLoginScreen(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/screens/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "screen", body["type"])
	assert.Equal(t, "login", body["id"])
}

func TestHandler_UsesAcceptLanguage(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/screens/login", nil)
	req.Header.Set("Accept-Language", "es")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Iniciar sesión")
}

func TestHandler_DefaultsToEnglishWhenNoAcceptLanguage(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/screens/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, w.Body.String(), "Log in")
}
