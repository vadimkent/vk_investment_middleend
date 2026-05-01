package analysis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newAnalysisClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	return NewClient(baseURL, 5*time.Second)
}

func TestStreamHandler_BypassesUpstreamSSE(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl := w.(http.Flusher)
		_, _ = w.Write([]byte("event: session\ndata: {\"session_id\":\"sess-1\"}\n\n"))
		fl.Flush()
		_, _ = w.Write([]byte("event: delta\ndata: {\"text\":\"hi\"}\n\n"))
		fl.Flush()
		_, _ = w.Write([]byte("event: done\ndata: {}\n\n"))
		fl.Flush()
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/analysis/stream", NewStreamHandler(newAnalysisClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/analysis/stream?focus=risk", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content-type: %q", got)
	}
	body := rec.Body.String()
	for _, want := range []string{"event: session", "event: delta", "event: done", "session_id"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in bypass body", want)
		}
	}
}

func TestStreamHandler_PreStreamRateLimitedEmitsErrorEvent(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"slow down"}}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/analysis/stream", NewStreamHandler(newAnalysisClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/analysis/stream", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `event: error`) || !strings.Contains(body, `"code":"RATE_LIMITED"`) {
		t.Fatalf("expected RATE_LIMITED SSE error, got: %s", body)
	}
}

func TestStreamHandler_Unauthorized401Envelope(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/analysis/stream", NewStreamHandler(newAnalysisClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/analysis/stream", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unauthorized") {
		t.Fatal("expected unauthorized envelope")
	}
}
