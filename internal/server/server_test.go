package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/config"
)

func TestHealthHandler(t *testing.T) {
	cfg := &config.Config{Port: 8080, BackendURL: "http://localhost:8080"}
	srv := New(cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "healthy", body["status"])
}

func TestHomeScreenHandler(t *testing.T) {
	cfg := &config.Config{Port: 8080, BackendURL: "http://localhost:8080"}
	srv := New(cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/screens/home", nil)
	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var screen map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &screen)
	require.NoError(t, err)
	assert.Equal(t, "screen", screen["type"])
	assert.Equal(t, "home", screen["id"])
}
