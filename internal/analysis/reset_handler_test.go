package analysis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResetHandler_ReplacesContentWithStartState(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/analysis/reset", NewResetHandler().Get)
	})

	req := httptest.NewRequest(http.MethodGet, "/actions/analysis/reset", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["action"] != "replace" || got["target_id"] != "analysis-content" {
		t.Fatalf("unexpected: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "analysis-start-form") {
		t.Fatal("missing start form in reset response")
	}
}
