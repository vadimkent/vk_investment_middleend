package imports

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

func TestScreenHandler_Get_RendersRoot(t *testing.T) {
	loadTestLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.GET("/screens/import", NewHandler().Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/screens/import", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["id"] != "import-screen" {
		t.Fatalf("expected import-screen, got %v", got["id"])
	}
	if !strings.Contains(rec.Body.String(), "ai-import-card") {
		t.Fatal("missing ai-import-card in render")
	}
}
