package analysis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMessagesHandler_BypassesUpstreamSSE(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/analysis/sessions/sess-1/messages" {
			t.Fatalf("unexpected upstream path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl := w.(http.Flusher)
		_, _ = w.Write([]byte("event: delta\ndata: {\"text\":\"hello\"}\n\n"))
		fl.Flush()
		_, _ = w.Write([]byte("event: done\ndata: {}\n\n"))
		fl.Flush()
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/sessions/:id/messages", NewMessagesHandler(newAnalysisClient(t, be.URL)).Post)
	})

	req := httptest.NewRequest(http.MethodPost,
		"/actions/analysis/sessions/sess-1/messages",
		strings.NewReader(`{"content":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: delta") || !strings.Contains(body, "hello") {
		t.Fatalf("missing bypass content: %s", body)
	}
}

func TestMessagesHandler_BadRequestOnMissingContent(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		// Use a stub client: no upstream call expected since we error out early.
		r.POST("/actions/analysis/sessions/:id/messages", NewMessagesHandler(newAnalysisClient(t, "http://example.invalid")).Post)
	})
	req := httptest.NewRequest(http.MethodPost,
		"/actions/analysis/sessions/sess-1/messages",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
}

func TestMessagesHandler_SessionNotFoundEmitsErrorEvent(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/sessions/:id/messages", NewMessagesHandler(newAnalysisClient(t, be.URL)).Post)
	})

	req := httptest.NewRequest(http.MethodPost,
		"/actions/analysis/sessions/sess-x/messages",
		strings.NewReader(`{"content":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := rec.Body.String()
	// 404 from BE → ErrSessionNotFound → handleStreamError synthesizes an
	// SSE error event; component will treat it as terminal via its
	// terminal_error_codes config.
	if !strings.Contains(body, `event: error`) {
		t.Fatalf("expected SSE error event for 404, got: %s", body)
	}
}
