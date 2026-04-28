package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(h)
	c := NewClient(srv.URL, 90*time.Second)
	return c, srv.Close
}

func TestStartSession_PostsMultipartAndReturnsSession(t *testing.T) {
	var receivedFile, receivedHint string
	var receivedAuth string
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/import/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		receivedAuth = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer f.Close()
		b, _ := io.ReadAll(f)
		receivedFile = string(b)
		receivedHint = r.FormValue("hint")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id":"sess-1","status":"needs_review",
			"created_at":"2026-04-27T10:00:00Z","expires_at":"2026-04-27T11:00:00Z",
			"ai_summary":"Looks like Broker X export.",
			"assumptions":["amounts in USD"],
			"preview":{"assets":[],"trades":[],"snapshots":[]},
			"gaps":[],"gap_counts":{"blocking":0,"warnings":0}
		}`))
	})
	defer cleanup()

	sess, err := c.StartSession(context.Background(), "Bearer t", []byte("col1,col2\n1,2\n"), "text/csv", "broker x export")
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if sess.ID != "sess-1" || sess.Status != "needs_review" {
		t.Fatalf("unexpected session: %+v", sess)
	}
	if receivedFile != "col1,col2\n1,2\n" {
		t.Fatalf("file content not forwarded: got %q", receivedFile)
	}
	if receivedHint != "broker x export" {
		t.Fatalf("hint not forwarded: got %q", receivedHint)
	}
	if receivedAuth != "Bearer t" {
		t.Fatalf("authorization not forwarded: got %q", receivedAuth)
	}
}

func TestStartSession_Unauthorized(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer cleanup()
	_, err := c.StartSession(context.Background(), "", []byte("x"), "text/csv", "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestStartSession_BackendValidationError(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"IMPORT_FILE_TOO_LARGE","message":"File exceeds the 5 MB limit."}}`))
	})
	defer cleanup()
	_, err := c.StartSession(context.Background(), "", []byte("x"), "text/csv", "")
	var be *BackendError
	if !errors.As(err, &be) {
		t.Fatalf("expected *BackendError, got %v", err)
	}
	if be.Code != "IMPORT_FILE_TOO_LARGE" || be.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("unexpected backend error: %+v", be)
	}
}

// helper used by other tests
func newMultipart(t *testing.T, fieldName, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte(content))
	_ = w.Close()
	return &b, w.FormDataContentType()
}

func TestResolveGaps_PatchesAndReturnsSession(t *testing.T) {
	var got struct {
		Resolutions []GapResolution `json:"resolutions"`
	}
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/import/sessions/sess-1/gaps" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"sess-1","status":"ready",
			"created_at":"2026-04-27T10:00:00Z","expires_at":"2026-04-27T11:00:00Z",
			"ai_summary":"x","assumptions":[],
			"preview":{"assets":[],"trades":[],"snapshots":[]},
			"gaps":[],"gap_counts":{"blocking":0,"warnings":0}
		}`))
	})
	defer cleanup()

	sess, err := c.ResolveGaps(context.Background(), "", "sess-1", []GapResolution{{GapID: "g1", Value: "USD"}})
	if err != nil {
		t.Fatalf("ResolveGaps: %v", err)
	}
	if sess.Status != "ready" {
		t.Fatalf("expected status=ready, got %q", sess.Status)
	}
	if len(got.Resolutions) != 1 || got.Resolutions[0].GapID != "g1" || got.Resolutions[0].Value != "USD" {
		t.Fatalf("body not forwarded: %+v", got)
	}
}

func TestResolveGaps_NotFound(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()
	_, err := c.ResolveGaps(context.Background(), "", "sess-x", nil)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestConfirmSession_OK(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/import/sessions/sess-1/confirm" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assets_created":1,"trades_imported":2,"snapshots_imported":3,"warnings":0}`))
	})
	defer cleanup()
	res, err := c.ConfirmSession(context.Background(), "", "sess-1")
	if err != nil {
		t.Fatalf("ConfirmSession: %v", err)
	}
	if res.AssetsCreated != 1 || res.TradesImported != 2 || res.SnapshotsImported != 3 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestConfirmSession_NotFound(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	defer cleanup()
	_, err := c.ConfirmSession(context.Background(), "", "x")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestCancelSession_NoContent(t *testing.T) {
	called := false
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/import/sessions/sess-1" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()
	if err := c.CancelSession(context.Background(), "", "sess-1"); err != nil {
		t.Fatalf("CancelSession: %v", err)
	}
	if !called {
		t.Fatal("backend not called")
	}
}

func TestCancelSession_NotFoundIsNotError(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	defer cleanup()
	if err := c.CancelSession(context.Background(), "", "x"); err != nil {
		t.Fatalf("CancelSession should treat 404 as success, got %v", err)
	}
}

func TestExportStream_PassthroughHeaders(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/export" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="vk_tracker_export_2026-04-27.csv"`)
		_, _ = w.Write([]byte("col1,col2\n1,2\n"))
	})
	defer cleanup()

	resp, err := c.ExportStream(context.Background(), "Bearer t")
	if err != nil {
		t.Fatalf("ExportStream: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("Content-Disposition"); !strings.Contains(got, "vk_tracker_export_") {
		t.Fatalf("Content-Disposition not forwarded: %q", got)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "col1,col2\n1,2\n" {
		t.Fatalf("body mismatch: %q", body)
	}
}

func TestExportStream_Unauthorized(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	defer cleanup()
	_, err := c.ExportStream(context.Background(), "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRestore_PostsMultipartAndReturnsCounts(t *testing.T) {
	var receivedFile string
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/restore" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_ = r.ParseMultipartForm(10 << 20)
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer f.Close()
		b, _ := io.ReadAll(f)
		receivedFile = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"assets_imported":1,"assets_skipped":0,
			"trades_imported":2,"trades_skipped":1,
			"snapshots_imported":0,"snapshots_skipped":3,
			"snapshot_entries_imported":4,"snapshot_entries_skipped":5
		}`))
	})
	defer cleanup()

	res, err := c.Restore(context.Background(), "", []byte("col1\nA\n"))
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if receivedFile != "col1\nA\n" {
		t.Fatalf("file not forwarded: %q", receivedFile)
	}
	if res.SnapshotEntriesSkipped != 5 {
		t.Fatalf("unexpected counts: %+v", res)
	}
}

func TestRestore_BackendError(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"RESTORE_FILE_TOO_LARGE","message":"File exceeds the 10 MB limit."}}`))
	})
	defer cleanup()
	_, err := c.Restore(context.Background(), "", []byte("x"))
	var be *BackendError
	if !errors.As(err, &be) || be.Code != "RESTORE_FILE_TOO_LARGE" {
		t.Fatalf("expected RESTORE_FILE_TOO_LARGE, got %v", err)
	}
}
