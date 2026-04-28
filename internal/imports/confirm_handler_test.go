package imports

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConfirmHandler_Success(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assets_created":1,"trades_imported":2,"snapshots_imported":3,"warnings":0}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/sessions/:id/confirm", NewConfirmHandler(mustClient(t, be.URL)).Post)
	})
	req := httptest.NewRequest(http.MethodPost, "/actions/import/sessions/sess-1/confirm", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "import-root" {
		t.Fatalf("expected target_id=import-root, got %v", got["target_id"])
	}
	if !strings.Contains(rec.Body.String(), `"feedback"`) {
		t.Fatal("expected snackbar feedback")
	}
}
