package analysis

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(h)
	c := NewClient(srv.URL, 5*time.Second)
	return c, srv.Close
}

func TestStreamSession_PostsAndReturnsLiveResponse(t *testing.T) {
	var receivedAuth, receivedCT string
	var receivedBody []byte
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/analysis/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		receivedAuth = r.Header.Get("Authorization")
		receivedCT = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: session\ndata: {\"session_id\":\"sess-1\"}\n\n"))
	})
	defer cleanup()

	resp, err := c.StreamSession(context.Background(), "Bearer t", "risk")
	if err != nil {
		t.Fatalf("StreamSession: %v", err)
	}
	defer resp.Body.Close()

	if receivedAuth != "Bearer t" {
		t.Fatalf("authorization not forwarded: %q", receivedAuth)
	}
	if receivedCT != "application/json" {
		t.Fatalf("content-type: %q", receivedCT)
	}
	if !strings.Contains(string(receivedBody), `"focus":"risk"`) {
		t.Fatalf("focus body: %q", receivedBody)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "session_id") {
		t.Fatalf("body bypass: %q", body)
	}
}

func TestStreamSession_EmptyFocusSendsEmptyJSON(t *testing.T) {
	var receivedBody []byte
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	resp, err := c.StreamSession(context.Background(), "", "")
	if err != nil {
		t.Fatalf("StreamSession: %v", err)
	}
	resp.Body.Close()
	if string(receivedBody) != `{}` && string(receivedBody) != `{"focus":""}` {
		t.Fatalf("expected empty-focus body to be {} or {\"focus\":\"\"}, got: %q", receivedBody)
	}
}

func TestStreamSession_Unauthorized(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer cleanup()
	_, err := c.StreamSession(context.Background(), "", "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestStreamSession_BackendError(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"ANALYSIS_FOCUS_TOO_LONG","message":"too long"}}`))
	})
	defer cleanup()
	_, err := c.StreamSession(context.Background(), "", "")
	var be *BackendError
	if !errors.As(err, &be) {
		t.Fatalf("expected *BackendError, got %v", err)
	}
	if be.Code != "ANALYSIS_FOCUS_TOO_LONG" || be.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("unexpected: %+v", be)
	}
}

func TestStreamSession_RateLimited(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"slow down"}}`))
	})
	defer cleanup()
	_, err := c.StreamSession(context.Background(), "", "")
	var be *BackendError
	if !errors.As(err, &be) || be.HTTPStatus != http.StatusTooManyRequests {
		t.Fatalf("expected 429 BackendError, got %v", err)
	}
}

func TestAddMessage_PostsAndReturnsLiveResponse(t *testing.T) {
	var receivedBody []byte
	var receivedPath string
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: delta\ndata: {\"text\":\"hi\"}\n\n"))
	})
	defer cleanup()

	resp, err := c.AddMessage(context.Background(), "", "sess-1", "hello")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	defer resp.Body.Close()
	if receivedPath != "/v1/analysis/sessions/sess-1/messages" {
		t.Fatalf("path: %q", receivedPath)
	}
	if !strings.Contains(string(receivedBody), `"content":"hello"`) {
		t.Fatalf("body: %q", receivedBody)
	}
}

func TestAddMessage_SessionNotFound(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()
	_, err := c.AddMessage(context.Background(), "", "sess-x", "x")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
