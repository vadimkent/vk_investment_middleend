package profile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/screens/profile", h.Get)
	return r
}

func TestHandler_Get_Happy(t *testing.T) {
	uc := NewGetUseCase(&stubMe{res: sampleUser()}, &stubCfg{res: sampleConfig()})
	r := newRouter(NewHandler(uc))
	req := httptest.NewRequest(http.MethodGet, "/screens/profile", nil)
	req.Header.Set("Authorization", "Bearer t")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, ScreenID, body["id"])
}

func TestHandler_Get_MeUnauthorized_RedirectsToLogin(t *testing.T) {
	uc := NewGetUseCase(&stubMe{err: ErrUnauthorized}, &stubCfg{})
	r := newRouter(NewHandler(uc))
	req := httptest.NewRequest(http.MethodGet, "/screens/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/screens/login", body["redirect"])
}

func TestHandler_Get_ConfigBackendError_502(t *testing.T) {
	uc := NewGetUseCase(&stubMe{res: sampleUser()}, &stubCfg{err: ErrBackend})
	r := newRouter(NewHandler(uc))
	req := httptest.NewRequest(http.MethodGet, "/screens/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj := body["error"].(map[string]any)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}
