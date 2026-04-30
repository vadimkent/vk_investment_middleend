package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Port:             8081,
		BackendURL:       "http://localhost:9999",
		RequestTimeout:   5 * time.Second,
		JWTSecret:        "test-secret",
		JWTLeewaySeconds: 0,
	}
}

func TestServer_HealthIsPublic(t *testing.T) {
	s := New(testConfig())
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_ShellRequiresAuth(t *testing.T) {
	s := New(testConfig())
	req := httptest.NewRequest("GET", "/shell", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServer_HomeScreenRequiresAuth(t *testing.T) {
	s := New(testConfig())
	req := httptest.NewRequest("GET", "/screens/home", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServer_LoginActionIsPublic(t *testing.T) {
	s := New(testConfig())
	req := httptest.NewRequest("POST", "/actions/login", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	require.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestServer_RegisterActionIsPublic(t *testing.T) {
	s := New(testConfig())
	req := httptest.NewRequest("POST", "/actions/register", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	require.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestServer_LoginScreenIsPublic(t *testing.T) {
	s := New(testConfig())
	req := httptest.NewRequest("GET", "/screens/login", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_RegisterScreenIsPublic(t *testing.T) {
	s := New(testConfig())
	req := httptest.NewRequest("GET", "/screens/register", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"register"`)
}

func TestRouter_HasProfileRoutes(t *testing.T) {
	s := New(testConfig())
	routes := s.router.Routes()
	wanted := map[string]bool{
		"GET /screens/profile":                  false,
		"POST /actions/profile/update":          false,
		"POST /actions/profile/update_email":    false,
		"POST /actions/profile/change_password": false,
		"GET /actions/profile/delete_modal":     false,
		"POST /actions/profile/delete_account":  false,
	}
	for _, ri := range routes {
		key := ri.Method + " " + ri.Path
		if _, ok := wanted[key]; ok {
			wanted[key] = true
		}
	}
	for k, found := range wanted {
		assert.Truef(t, found, "route missing: %s", k)
	}
}
