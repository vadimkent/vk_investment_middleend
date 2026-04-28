package imports

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolveGapsHandler_Success(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"gap_id":"g1"`) || !strings.Contains(string(body), `"value":"USD"`) {
			t.Fatalf("backend body did not include resolution: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"sess-1","status":"ready",
			"created_at":"x","expires_at":"y",
			"ai_summary":"x","assumptions":[],
			"preview":{"assets":[],"trades":[],"snapshots":[]},
			"gaps":[],"gap_counts":{"blocking":0,"warnings":0}
		}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/sessions/:id/resolve_gaps", NewResolveGapsHandler(mustClient(t, be.URL)).Post)
	})

	form := url.Values{}
	form.Set("resolutions[g1]", "USD")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/sessions/sess-1/resolve_gaps",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "import-modal-slot" {
		t.Fatalf("unexpected target_id: %v", got["target_id"])
	}
}

func TestResolveGapsHandler_SessionExpiredReplacesRoot(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/sessions/:id/resolve_gaps", NewResolveGapsHandler(mustClient(t, be.URL)).Post)
	})
	req := httptest.NewRequest(http.MethodPost, "/actions/import/sessions/sess-x/resolve_gaps",
		strings.NewReader("resolutions[g1]=USD"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "import-root" {
		t.Fatalf("expected target_id=import-root, got %v", got["target_id"])
	}
}
