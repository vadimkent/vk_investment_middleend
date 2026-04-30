package register

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandler_GetReturns200WithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/screens/register", NewHandler().Get)

	req := httptest.NewRequest(http.MethodGet, "/screens/register", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, `"id":"register"`))
	assert.False(t, strings.Contains(body, `"id":"nav_header"`), "shell slot must not be present")
}

func TestHandler_RespectsAcceptLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/screens/register", NewHandler().Get)

	req := httptest.NewRequest(http.MethodGet, "/screens/register", nil)
	req.Header.Set("Accept-Language", "es-AR,es;q=0.9")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Spanish title should appear somewhere in the body
	assert.True(t, strings.Contains(w.Body.String(), "Crear cuenta"))
}
