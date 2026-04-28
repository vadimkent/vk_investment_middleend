package imports

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func mustClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	return NewClient(baseURL, 90*time.Second)
}

func multipartBody(t *testing.T, fileContent, hint string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "broker.csv")
	_, _ = fw.Write([]byte(fileContent))
	if hint != "" {
		_ = w.WriteField("hint", hint)
	}
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

func TestAnalyzeHandler_Success(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id":"sess-1","status":"ready",
			"created_at":"2026-04-27T10:00:00Z","expires_at":"2026-04-27T11:00:00Z",
			"ai_summary":"x","assumptions":[],
			"preview":{"assets":[],"trades":[],"snapshots":[]},
			"gaps":[],"gap_counts":{"blocking":0,"warnings":0}
		}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/analyze", NewAnalyzeHandler(mustClient(t, be.URL)).Post)
	})

	body, ct := multipartBody(t, "col1\n1\n", "broker x")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/analyze", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["action"] != "replace" || got["target_id"] != "import-modal-slot" {
		t.Fatalf("unexpected action response: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "import-review-modal") {
		t.Fatal("missing review modal in response tree")
	}
}

func TestAnalyzeHandler_ReplacesCardOnBackendError(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"IMPORT_FILE_TOO_LARGE","message":"File exceeds 5 MB."}}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/analyze", NewAnalyzeHandler(mustClient(t, be.URL)).Post)
	})

	body, ct := multipartBody(t, "x", "h")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/analyze", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "ai-import-card" {
		t.Fatalf("expected target_id ai-import-card, got %v", got["target_id"])
	}
	if !strings.Contains(rec.Body.String(), "File exceeds 5 MB.") {
		t.Fatal("expected backend error message in tree")
	}
	if !errors.Is(nil, errors.New("")) { // keep import alive in case of refactor
		_ = errors.New("")
	}
}
