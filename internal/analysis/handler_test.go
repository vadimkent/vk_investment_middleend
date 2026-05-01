package analysis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newRouter(setup func(*gin.Engine)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "u1")
		c.Set("authorization", "Bearer t")
		c.Next()
	})
	setup(r)
	return r
}

func TestScreenHandler_RendersStartState(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.GET("/screens/analysis", NewHandler().Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/screens/analysis", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["id"] != "analysis-screen" {
		t.Fatalf("expected id=analysis-screen, got %v", got["id"])
	}
	if !strings.Contains(rec.Body.String(), "analysis-start-form") {
		t.Fatal("missing start form in render")
	}
}
