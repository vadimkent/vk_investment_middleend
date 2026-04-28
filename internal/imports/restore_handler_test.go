package imports

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRestoreHandler_Success(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"assets_imported":1,"assets_skipped":0,
			"trades_imported":2,"trades_skipped":1,
			"snapshots_imported":0,"snapshots_skipped":3,
			"snapshot_entries_imported":4,"snapshot_entries_skipped":5
		}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/restore", NewRestoreHandler(mustClient(t, be.URL)).Post)
	})
	body, ct := multipartBody(t, "col\nA\n", "")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/restore", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "restore-card" {
		t.Fatalf("expected restore-card target, got %v", got["target_id"])
	}
	if !strings.Contains(rec.Body.String(), "Restored successfully") {
		t.Fatal("missing success copy")
	}
}

func TestRestoreHandler_BackendErrorReplacesIdle(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"RESTORE_FILE_TOO_LARGE","message":"File exceeds 10 MB."}}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/restore", NewRestoreHandler(mustClient(t, be.URL)).Post)
	})
	body, ct := multipartBody(t, "x", "")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/restore", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "File exceeds 10 MB.") {
		t.Fatal("expected backend error message in response")
	}
}

func TestRestoreIdleHandler_ReturnsIdleCard(t *testing.T) {
	loadTestLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/import/restore_idle", NewRestoreIdleHandler().Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/import/restore_idle", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":"restore-card"`) {
		t.Fatal("missing restore-card in response")
	}
}
