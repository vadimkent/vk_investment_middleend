package analysis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStartHandler_SuccessReplacesContentWithChat(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/start", NewStartHandler().Post)
	})

	form := url.Values{}
	form.Set("focus", "risk exposure")
	req := httptest.NewRequest(http.MethodPost, "/actions/analysis/start",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["action"] != "replace" || got["target_id"] != "analysis-content" {
		t.Fatalf("unexpected: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "analysis-chat") {
		t.Fatal("missing analysis_chat in tree")
	}
	if !strings.Contains(rec.Body.String(), `risk+exposure`) {
		t.Fatal("expected URL-encoded focus in initial_endpoint")
	}
}

func TestStartHandler_EmptyFocusOK(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/start", NewStartHandler().Post)
	})

	req := httptest.NewRequest(http.MethodPost, "/actions/analysis/start",
		strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"initial_endpoint":"/actions/analysis/stream"`) {
		t.Fatalf("expected stream endpoint without ?focus= when empty, got: %s", body)
	}
}

func TestStartHandler_FocusTooLongReplacesFormWithError(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/start", NewStartHandler().Post)
	})

	long := strings.Repeat("a", 501)
	form := url.Values{}
	form.Set("focus", long)
	req := httptest.NewRequest(http.MethodPost, "/actions/analysis/start",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"target_id":"analysis-start-form"`) {
		t.Fatalf("expected replace of analysis-start-form on validation error, got: %s", body)
	}
	if !strings.Contains(body, "Focus area is too long.") {
		t.Fatal("expected error message in tree")
	}
}
