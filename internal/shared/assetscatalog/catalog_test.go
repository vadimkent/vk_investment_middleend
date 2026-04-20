package assetscatalog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func encodePage(t *testing.T, w http.ResponseWriter, assets []map[string]any, total, offset int) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{
		"assets": assets,
		"total":  total,
		"size":   100,
		"offset": offset,
	}); err != nil {
		t.Fatalf("encode page: %v", err)
	}
}

func TestCatalogListSinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("missing auth header: %q", r.Header.Get("Authorization"))
		}
		q := r.URL.Query()
		if q.Get("size") != "100" || q.Get("sort") != "ticker" || q.Get("order") != "desc" || q.Get("offset") != "0" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		encodePage(t, w, []map[string]any{
			{"id": "a", "ticker": "AAPL", "name": "Apple", "asset_type": "STOCK", "currency": "USD", "is_complex": false},
		}, 1, 0)
	}))
	defer srv.Close()

	c := NewCatalog(srv.URL, 2*time.Second)
	got, err := c.List(context.Background(), "Bearer tok")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 1 || got[0].Ticker != "AAPL" {
		t.Errorf("got %+v", got)
	}
}

func TestCatalogListMultiPage(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		q := r.URL.Query()
		offset, err := strconv.Atoi(q.Get("offset"))
		if err != nil {
			t.Fatalf("bad offset: %v", err)
		}
		expected := []int{0, 100, 200}
		if int(n-1) >= len(expected) {
			t.Fatalf("too many calls: %d", n)
		}
		if offset != expected[n-1] {
			t.Errorf("call %d: expected offset %d got %d", n, expected[n-1], offset)
		}
		if q.Get("size") != "100" || q.Get("sort") != "ticker" || q.Get("order") != "desc" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}

		pageCount := 100
		if offset == 200 {
			pageCount = 50
		}
		assets := make([]map[string]any, 0, pageCount)
		for i := 0; i < pageCount; i++ {
			assets = append(assets, map[string]any{
				"id":         "id-" + strconv.Itoa(offset+i),
				"ticker":     "T" + strconv.Itoa(offset+i),
				"name":       "N",
				"asset_type": "STOCK",
				"currency":   "USD",
				"is_complex": false,
			})
		}
		encodePage(t, w, assets, 250, offset)
	}))
	defer srv.Close()

	c := NewCatalog(srv.URL, 2*time.Second)
	got, err := c.List(context.Background(), "Bearer tok")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 250 {
		t.Errorf("expected 250 assets, got %d", len(got))
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("expected 3 backend calls, got %d", calls)
	}
	if got[0].Ticker != "T0" || got[249].Ticker != "T249" {
		t.Errorf("unexpected order: first=%s last=%s", got[0].Ticker, got[249].Ticker)
	}
}

func TestCatalogListUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewCatalog(srv.URL, 2*time.Second)
	got, err := c.List(context.Background(), "Bearer tok")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil slice, got %+v", got)
	}
}

func TestCatalogListBackendErrorFirstPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewCatalog(srv.URL, 2*time.Second)
	got, err := c.List(context.Background(), "Bearer tok")
	if !errors.Is(err, ErrBackend) {
		t.Errorf("expected ErrBackend, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil slice, got %+v", got)
	}
}

func TestCatalogListBackendErrorLaterPageDiscardsPartial(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if n == 1 {
			assets := make([]map[string]any, 0, 100)
			for i := 0; i < 100; i++ {
				assets = append(assets, map[string]any{
					"id":         "id-" + strconv.Itoa(offset+i),
					"ticker":     "T" + strconv.Itoa(offset+i),
					"name":       "N",
					"asset_type": "STOCK",
					"currency":   "USD",
					"is_complex": false,
				})
			}
			encodePage(t, w, assets, 200, offset)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewCatalog(srv.URL, 2*time.Second)
	got, err := c.List(context.Background(), "Bearer tok")
	if !errors.Is(err, ErrBackend) {
		t.Errorf("expected ErrBackend, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil slice (partial discarded), got len=%d", len(got))
	}
}

func TestCatalogListNoAuthHeaderWhenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h := r.Header.Get("Authorization"); h != "" {
			t.Errorf("expected no Authorization header, got %q", h)
		}
		encodePage(t, w, []map[string]any{}, 0, 0)
	}))
	defer srv.Close()

	c := NewCatalog(srv.URL, 2*time.Second)
	got, err := c.List(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %+v", got)
	}
}
