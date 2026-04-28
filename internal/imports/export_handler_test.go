package imports

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExportHandler_StreamsBackendResponse(t *testing.T) {
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="vk_tracker_export_2026-04-27.csv"`)
		_, _ = w.Write([]byte("col1,col2\nA,B\n"))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/import/export", NewExportHandler(mustClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/import/export", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "vk_tracker_export_") {
		t.Fatalf("Content-Disposition not forwarded: %q", got)
	}
	if rec.Body.String() != "col1,col2\nA,B\n" {
		t.Fatalf("body mismatch: %q", rec.Body.String())
	}
}

func TestExportHandler_Unauthorized_Redirects(t *testing.T) {
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/import/export", NewExportHandler(mustClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/import/export", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected Location=/login, got %q", loc)
	}
}
