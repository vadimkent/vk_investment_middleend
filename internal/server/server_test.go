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
