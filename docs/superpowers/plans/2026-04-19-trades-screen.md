# Trades Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the full Trades SDUI screen (list + create/edit/delete modals) end-to-end, mirroring the Assets screen's structure but adapted to trades' two-filter catalog, computed Total column, and simpler delete flow.

**Architecture:** Mirror `internal/assets/` layer-by-layer. Three new pieces:
1. **`internal/shared/format/`** — extract money/quantity formatters from `internal/portfolio/format.go` so `portfolio` and `trades` both depend on `shared/format` (no cross-screen import).
2. **`internal/shared/assetscatalog/`** — full-asset-list helper (paged loop, `size=100`) consumed by trades (and later snapshots/import/analysis).
3. **`internal/trades/`** — the screen itself, handlers + builders + clients.

**Tech Stack:** Go · Gin · testify · stdlib `net/http/httptest`. Existing SDUI component library in `internal/components/`. Canonical specs: `spec/screens/trades.md` and `spec/shared/assets-catalog.md`.

---

## Conventions used throughout this plan

- **All new files live under `/Users/vadimkent/repos/vk_investment_middleend_v2/`.** Paths below are repo-relative.
- **TDD:** each behavior gets a failing test first, then the minimum code to pass, then a commit. Keep commits small — one logical unit per commit.
- **Reference files to emulate** (read these first when starting each task; they show the established pattern):
  - `internal/assets/types.go` / `client.go` / `mutate_client.go` / `get_usecase.go`
  - `internal/assets/builder.go` / `modal_builder.go`
  - `internal/assets/handler.go` / `list_handler.go` / `create_handler.go` / `update_handler.go` / `delete_handler.go`
  - `internal/assets/create_modal_handler.go` / `edit_modal_handler.go` / `delete_modal_handler.go`
  - `internal/assets/*_test.go` for test style
  - `internal/server/server.go` (lines 67–76) for wiring
  - `internal/portfolio/format.go` for the formatters being extracted
- **Commit message style:** Conventional Commits (`feat(trades): …`, `refactor(shared): …`, `test(trades): …`, `docs(spec): …`). **No Claude co-author trailer.**
- **Middleend restart after each phase:** after any phase that touches server/handlers/routes, kill the listener on `:8082` and run `./cli run` in background.

---

## Phase 0 — Preparation

### Task 0.1: Read existing code & specs

**Files to read (no changes):**
- `spec/screens/trades.md`
- `spec/shared/assets-catalog.md`
- `internal/assets/` (entire directory)
- `internal/portfolio/format.go`
- `internal/server/server.go`
- `internal/components/base.go`, `actions.go`
- `locales/en.json`, `locales/es.json` (the `assets.*` namespace as a reference for key shape)

- [ ] **Step 1: Read and internalize.** No code changes. The goal is to have the assets pattern fresh before starting.

---

## Phase 1 — Extract shared format helpers

### Task 1.1: Move formatters to `internal/shared/format/`

**Files:**
- Create: `internal/shared/format/format.go`
- Create: `internal/shared/format/format_test.go`
- Modify: `internal/portfolio/format.go` — delete the moved functions, leave portfolio-only helpers (`FormatRelativeTime`, `PnLPct`) in place.
- Modify: `internal/portfolio/builder.go` and any other portfolio files that call `FormatMoney` / `FormatSignedMoney` / `FormatQuantity` / `FormatSignedPercent` — update imports.

**Functions to move (keep signatures unchanged):**
- `FormatMoney(amount *float64, currency, lang string) string`
- `FormatSignedMoney(amount *float64, currency, lang string) string`
- `FormatQuantity(q *float64, lang string) string`
- `FormatSignedPercent(pct *float64, lang string) string`

Also move the private helpers these depend on: `currencyPrefix`, `formatDecimal`, `withThousands`, `absFloat`, `currencySymbols` map. Keep `FormatRelativeTime`, `PnLPct`, and `interp` in `internal/portfolio/format.go` (they are portfolio-specific).

- [ ] **Step 1: Write tests for `internal/shared/format/format_test.go` covering each public function.**

Use a small table-driven test per function. Cover: nil input → `"—"`; English locale → `,` thousands / `.` decimal; Spanish locale → `.` thousands / `,` decimal; known currency (USD, EUR, ARS) → symbol prefix; unknown currency → `CODE ` prefix; negative values; zero-value for signed variants.

```go
package format

import "testing"

func TestFormatMoney(t *testing.T) {
    cases := []struct {
        name     string
        amount   *float64
        currency string
        lang     string
        want     string
    }{
        {"nil returns dash", nil, "USD", "en", "—"},
        {"usd en", f64(1234.5), "USD", "en", "$1,234.50"},
        {"eur es", f64(1234.5), "EUR", "es", "€1.234,50"},
        {"unknown currency prefixes code", f64(10), "XYZ", "en", "XYZ 10.00"},
        {"zero", f64(0), "USD", "en", "$0.00"},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := FormatMoney(tc.amount, tc.currency, tc.lang)
            if got != tc.want {
                t.Errorf("got %q, want %q", got, tc.want)
            }
        })
    }
}

func f64(v float64) *float64 { return &v }
```

Repeat the same structure for `FormatSignedMoney`, `FormatQuantity`, `FormatSignedPercent`. For the signed variants, include a zero case and positive/negative cases to lock in the `+` / `-` prefix behavior.

- [ ] **Step 2: Run tests — expect compile failure.**

```bash
go test ./internal/shared/format/...
```
Expected: `no Go files` or `undefined: FormatMoney`.

- [ ] **Step 3: Create `internal/shared/format/format.go` with the moved code.**

Copy the four public functions and their private helpers from `internal/portfolio/format.go` into the new file with `package format`. Preserve comments. Imports needed: `"fmt"`, `"strconv"`, `"strings"`.

- [ ] **Step 4: Run tests — expect PASS.**

```bash
go test ./internal/shared/format/...
```
Expected: all PASS.

- [ ] **Step 5: Delete the moved functions from `internal/portfolio/format.go`.**

Remove `FormatMoney`, `FormatSignedMoney`, `FormatQuantity`, `FormatSignedPercent`, and their private helpers (`currencyPrefix`, `formatDecimal`, `withThousands`, `absFloat`, `currencySymbols`). Keep `FormatRelativeTime`, `PnLPct`, and `interp`.

- [ ] **Step 6: Update `internal/portfolio/builder.go` (and any other portfolio file that calls the moved functions) to import and use `format.FormatMoney` etc.**

Find all callers:
```bash
grep -rln "FormatMoney\|FormatSignedMoney\|FormatQuantity\|FormatSignedPercent" internal/portfolio/
```
In each file, add `format "github.com/project/vk-investment-middleend/internal/shared/format"` to imports and replace bare calls with `format.FormatMoney(...)` etc.

- [ ] **Step 7: Run the whole test suite.**

```bash
go test ./...
```
Expected: all PASS (portfolio tests still pass because behavior is unchanged; the new format tests pass; all other tests untouched).

- [ ] **Step 8: Run lint + build.**

```bash
make lint && make build
```
Expected: clean.

- [ ] **Step 9: Commit.**

```bash
git add internal/shared/format/ internal/portfolio/format.go internal/portfolio/builder.go
# add any other portfolio files touched
git commit -m "refactor(shared): extract money/quantity formatters to internal/shared/format"
```

---

## Phase 2 — Assets Catalog shared package

### Task 2.1: Define catalog types

**Files:**
- Create: `internal/shared/assetscatalog/types.go`
- Create: `internal/shared/assetscatalog/types_test.go`

- [ ] **Step 1: Write `types_test.go` with a `ParseListResponse` test.**

```go
package assetscatalog

import "testing"

func TestParseListResponse(t *testing.T) {
    body := []byte(`{"assets":[{"id":"a","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD","is_complex":false}],"total":1,"size":100,"offset":0}`)
    r, err := ParseListResponse(body)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if r.Total != 1 || len(r.Assets) != 1 || r.Assets[0].Ticker != "AAPL" || r.Assets[0].IsComplex {
        t.Errorf("parsed wrong: %+v", r)
    }
}
```

- [ ] **Step 2: Run — expect FAIL (no package yet).**

```bash
go test ./internal/shared/assetscatalog/...
```

- [ ] **Step 3: Create `types.go`.**

```go
package assetscatalog

import "encoding/json"

// Asset is the minimum surface downstream screens need from the catalog.
// Additional fields returned by the backend pass through untouched via rawAsset.
type Asset struct {
    ID        string
    Ticker    string
    Name      string
    AssetType string
    Currency  string
    IsComplex bool
}

type rawAsset struct {
    ID        string `json:"id"`
    Ticker    string `json:"ticker"`
    Name      string `json:"name"`
    AssetType string `json:"asset_type"`
    Currency  string `json:"currency"`
    IsComplex bool   `json:"is_complex"`
}

type rawListResponse struct {
    Assets []rawAsset `json:"assets"`
    Total  int        `json:"total"`
    Size   int        `json:"size"`
    Offset int        `json:"offset"`
}

// ListPage is one backend page result.
type ListPage struct {
    Assets []Asset
    Total  int
    Size   int
    Offset int
}

// ParseListResponse parses a single /v1/assets page body.
func ParseListResponse(body []byte) (*ListPage, error) {
    var r rawListResponse
    if err := json.Unmarshal(body, &r); err != nil {
        return nil, err
    }
    out := &ListPage{Total: r.Total, Size: r.Size, Offset: r.Offset}
    out.Assets = make([]Asset, 0, len(r.Assets))
    for _, ra := range r.Assets {
        out.Assets = append(out.Assets, Asset(ra))
    }
    return out, nil
}
```

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/shared/assetscatalog/types.go internal/shared/assetscatalog/types_test.go
git commit -m "feat(assetscatalog): add Asset/ListPage types and parser"
```

### Task 2.2: Implement the paged catalog client

**Files:**
- Create: `internal/shared/assetscatalog/catalog.go`
- Create: `internal/shared/assetscatalog/catalog_test.go`

- [ ] **Step 1: Write failing tests for `Catalog.List`.**

Use `httptest.NewServer` with a handler that returns different pages based on `?offset=`. Cover:
1. Single page (total < 100, 1 backend call).
2. Multi-page (total 250 → 3 backend calls with offsets 0, 100, 200).
3. `401` on the first page → returns `ErrUnauthorized`.
4. `500` on the first page → returns `ErrBackend`.
5. `500` on a later page after first succeeded → returns `ErrBackend`, no partial result.
6. `Authorization` header forwarded on every page.

Test file skeleton:

```go
package assetscatalog

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

func TestCatalogListSinglePage(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Authorization") != "Bearer tok" {
            t.Errorf("missing auth header")
        }
        if r.URL.Query().Get("size") != "100" || r.URL.Query().Get("sort") != "ticker" || r.URL.Query().Get("order") != "desc" {
            t.Errorf("unexpected query: %s", r.URL.RawQuery)
        }
        _ = json.NewEncoder(w).Encode(map[string]any{
            "assets": []map[string]any{{"id": "a", "ticker": "AAPL", "name": "Apple", "asset_type": "STOCK", "currency": "USD", "is_complex": false}},
            "total":  1, "size": 100, "offset": 0,
        })
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
    calls := 0
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        calls++
        off := r.URL.Query().Get("offset")
        total := 250
        var assets []map[string]any
        switch off {
        case "0":
            for i := 0; i < 100; i++ {
                assets = append(assets, map[string]any{"id": fmt.Sprintf("a%d", i), "ticker": "T1", "currency": "USD"})
            }
        case "100":
            for i := 100; i < 200; i++ {
                assets = append(assets, map[string]any{"id": fmt.Sprintf("a%d", i), "ticker": "T2", "currency": "USD"})
            }
        case "200":
            for i := 200; i < 250; i++ {
                assets = append(assets, map[string]any{"id": fmt.Sprintf("a%d", i), "ticker": "T3", "currency": "USD"})
            }
        default:
            t.Fatalf("unexpected offset %s", off)
        }
        _ = json.NewEncoder(w).Encode(map[string]any{
            "assets": assets, "total": total, "size": 100, "offset": mustAtoi(off),
        })
    }))
    defer srv.Close()
    c := NewCatalog(srv.URL, 2*time.Second)
    got, err := c.List(context.Background(), "")
    if err != nil {
        t.Fatalf("err: %v", err)
    }
    if calls != 3 {
        t.Errorf("expected 3 calls, got %d", calls)
    }
    if len(got) != 250 {
        t.Errorf("expected 250 assets, got %d", len(got))
    }
}

func TestCatalogList401(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusUnauthorized)
    }))
    defer srv.Close()
    c := NewCatalog(srv.URL, 2*time.Second)
    _, err := c.List(context.Background(), "")
    if !errors.Is(err, ErrUnauthorized) {
        t.Errorf("expected ErrUnauthorized, got %v", err)
    }
}

func TestCatalogList500(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
    }))
    defer srv.Close()
    c := NewCatalog(srv.URL, 2*time.Second)
    _, err := c.List(context.Background(), "")
    if !errors.Is(err, ErrBackend) {
        t.Errorf("expected ErrBackend, got %v", err)
    }
}

func TestCatalogListLaterPageFails(t *testing.T) {
    calls := 0
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        calls++
        if calls == 1 {
            _ = json.NewEncoder(w).Encode(map[string]any{
                "assets": make([]map[string]any, 100), "total": 250, "size": 100, "offset": 0,
            })
            return
        }
        w.WriteHeader(http.StatusInternalServerError)
    }))
    defer srv.Close()
    c := NewCatalog(srv.URL, 2*time.Second)
    _, err := c.List(context.Background(), "")
    if !errors.Is(err, ErrBackend) {
        t.Errorf("expected ErrBackend, got %v", err)
    }
}

func mustAtoi(s string) int { var n int; _, _ = fmt.Sscanf(s, "%d", &n); return n }
```

- [ ] **Step 2: Run — expect FAIL (`Catalog` not defined).**

```bash
go test ./internal/shared/assetscatalog/...
```

- [ ] **Step 3: Implement `catalog.go`.**

```go
package assetscatalog

import (
    "context"
    "errors"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "time"
)

var (
    ErrUnauthorized = errors.New("backend unauthorized")
    ErrBackend      = errors.New("backend error")
)

const pageSize = 100

type Catalog struct {
    baseURL    string
    httpClient *http.Client
}

func NewCatalog(baseURL string, timeout time.Duration) *Catalog {
    return &Catalog{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// List fetches every asset across all backend pages. Pages are fetched
// sequentially until offset+size >= total. See spec/shared/assets-catalog.md.
func (c *Catalog) List(ctx context.Context, authorization string) ([]Asset, error) {
    offset := 0
    var all []Asset
    for {
        page, err := c.fetchPage(ctx, authorization, offset)
        if err != nil {
            return nil, err
        }
        all = append(all, page.Assets...)
        if offset+pageSize >= page.Total {
            return all, nil
        }
        offset += pageSize
    }
}

func (c *Catalog) fetchPage(ctx context.Context, authorization string, offset int) (*ListPage, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/assets", nil)
    if err != nil {
        return nil, err
    }
    q := req.URL.Query()
    q.Set("size", strconv.Itoa(pageSize))
    q.Set("sort", "ticker")
    q.Set("order", "desc")
    q.Set("offset", strconv.Itoa(offset))
    req.URL.RawQuery = q.Encode()
    if authorization != "" {
        req.Header.Set("Authorization", authorization)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrBackend, err)
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
    }
    switch resp.StatusCode {
    case http.StatusOK:
        page, err := ParseListResponse(body)
        if err != nil {
            return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
        }
        return page, nil
    case http.StatusUnauthorized:
        return nil, ErrUnauthorized
    default:
        return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
    }
}
```

- [ ] **Step 4: Run — expect PASS.**

```bash
go test ./internal/shared/assetscatalog/... -v
```

- [ ] **Step 5: Commit.**

```bash
git add internal/shared/assetscatalog/catalog.go internal/shared/assetscatalog/catalog_test.go
git commit -m "feat(assetscatalog): add paged List helper for cross-screen asset lookups"
```

---

## Phase 3 — Trades domain types and clients

### Task 3.1: Trade types

**Files:**
- Create: `internal/trades/types.go`
- Create: `internal/trades/types_test.go`

- [ ] **Step 1: Write failing test for `ParseListResponse`.**

```go
package trades

import "testing"

func TestParseListResponse(t *testing.T) {
    body := []byte(`{"trades":[{"id":"t1","asset_id":"a1","trade_type":"BUY","quantity":"10","price_per_unit":"100","fees":"0","date":"2024-01-10T10:00:00Z","source":"MANUAL","notes":"n","created_at":"2024-01-10T10:00:00Z"}],"total":1,"size":10,"offset":0}`)
    r, err := ParseListResponse(body)
    if err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if r.Total != 1 || len(r.Trades) != 1 {
        t.Fatalf("parsed wrong: %+v", r)
    }
    got := r.Trades[0]
    if got.ID != "t1" || got.AssetID != "a1" || got.TradeType != "BUY" || got.Quantity != "10" || got.Source != "MANUAL" {
        t.Errorf("unexpected trade: %+v", got)
    }
}
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Create `types.go`.**

```go
package trades

import "encoding/json"

// Trade is the middleend domain representation of a backend trade.
// Money-like fields (Quantity, PricePerUnit, Fees) are kept as strings to
// preserve decimal precision; formatting happens in the builder with the
// asset's currency via internal/shared/format.
type Trade struct {
    ID           string
    AssetID      string
    TradeType    string // "BUY" or "SELL"
    Quantity     string
    PricePerUnit string
    Fees         string
    Date         string // RFC3339 from backend
    Source       string // "MANUAL" or "IMPORT"
    Notes        string
    CreatedAt    string
}

// ListParams captures the query parameters for the trades list.
type ListParams struct {
    AssetID   string // "" = no filter; otherwise a UUID
    TradeType string // "" = no filter; otherwise "BUY" or "SELL"
    Offset    int
}

type ListResult struct {
    Trades []Trade
    Total  int
    Size   int
    Offset int
}

type rawTrade struct {
    ID           string `json:"id"`
    AssetID      string `json:"asset_id"`
    TradeType    string `json:"trade_type"`
    Quantity     string `json:"quantity"`
    PricePerUnit string `json:"price_per_unit"`
    Fees         string `json:"fees"`
    Date         string `json:"date"`
    Source       string `json:"source"`
    Notes        string `json:"notes"`
    CreatedAt    string `json:"created_at"`
}

type rawListResponse struct {
    Trades []rawTrade `json:"trades"`
    Total  int        `json:"total"`
    Size   int        `json:"size"`
    Offset int        `json:"offset"`
}

func ParseListResponse(body []byte) (*ListResult, error) {
    var r rawListResponse
    if err := json.Unmarshal(body, &r); err != nil {
        return nil, err
    }
    out := &ListResult{Total: r.Total, Size: r.Size, Offset: r.Offset}
    out.Trades = make([]Trade, 0, len(r.Trades))
    for _, rt := range r.Trades {
        out.Trades = append(out.Trades, Trade(rt))
    }
    return out, nil
}
```

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/types.go internal/trades/types_test.go
git commit -m "feat(trades): add Trade/ListParams/ListResult types and parser"
```

### Task 3.2: Trades list client

**Files:**
- Create: `internal/trades/client.go`
- Create: `internal/trades/client_test.go`

Mirror `internal/assets/client.go` exactly, but:
- Endpoint: `/v1/trades`
- Fixed query: `size=10`, `sort=date`, `order=desc`
- Optional: `asset_id`, `trade_type`, `offset`

- [ ] **Step 1: Write failing tests.**

Cover: 200 happy path (verifies forwarded `Authorization` and full query string with both filters set and unset), 401 → `ErrUnauthorized`, 500 → `ErrBackend`, malformed JSON → `ErrBackend`. Use `httptest.NewServer` exactly as `internal/assets/client_test.go` does.

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement `client.go` mirroring `internal/assets/client.go`.**

```go
package trades

import (
    "context"
    "errors"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "time"
)

var (
    ErrUnauthorized = errors.New("backend unauthorized")
    ErrBackend      = errors.New("backend error")
)

type Client struct {
    baseURL    string
    httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
    return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

func (c *Client) List(ctx context.Context, authorization string, p ListParams) (*ListResult, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/trades", nil)
    if err != nil {
        return nil, err
    }
    q := req.URL.Query()
    q.Set("size", "10")
    q.Set("sort", "date")
    q.Set("order", "desc")
    q.Set("offset", strconv.Itoa(p.Offset))
    if p.AssetID != "" {
        q.Set("asset_id", p.AssetID)
    }
    if p.TradeType != "" {
        q.Set("trade_type", p.TradeType)
    }
    req.URL.RawQuery = q.Encode()
    if authorization != "" {
        req.Header.Set("Authorization", authorization)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrBackend, err)
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
    }
    switch resp.StatusCode {
    case http.StatusOK:
        res, err := ParseListResponse(body)
        if err != nil {
            return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
        }
        return res, nil
    case http.StatusUnauthorized:
        return nil, ErrUnauthorized
    default:
        return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
    }
}
```

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/client.go internal/trades/client_test.go
git commit -m "feat(trades): add list client with asset_id/trade_type filters"
```

### Task 3.3: Trades mutate client (Get/Create/Update/Delete)

**Files:**
- Create: `internal/trades/mutate_client.go`
- Create: `internal/trades/mutate_client_test.go`

Mirror `internal/assets/mutate_client.go`. Reuse its `BackendValidationError` shape (define `ErrTradeNotFound`, add `BackendValidationError` local to this package — do NOT import from `assets`).

- [ ] **Step 1: Write failing tests.**

Cover for each verb (GetTrade, CreateTrade, UpdateTrade, DeleteTrade):
- Happy path (verifies body + auth + method + status).
- `401` → `ErrUnauthorized`.
- `404` on Get/Delete → `ErrTradeNotFound`.
- `422` with `{"error":{"code":"INSUFFICIENT_QUANTITY","message":"..."}}` → returns `*BackendValidationError` with the code populated.
- `5xx` → `ErrBackend`.

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement `mutate_client.go`.**

Copy the shape of `internal/assets/mutate_client.go`, replacing `Asset` with `Trade`, `ErrAssetNotFound` with `ErrTradeNotFound`, and the path prefix with `/v1/trades`. `DeleteTrade` takes only `id` (no `force`). Keep `BackendValidationError` and `parseValidationError` local (they have no cross-package significance).

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/mutate_client.go internal/trades/mutate_client_test.go
git commit -m "feat(trades): add mutate client (get/create/update/delete)"
```

---

## Phase 4 — Builders

### Task 4.1: Screen + list subtree builder (skeleton, empty state, filters, pagination)

**Files:**
- Create: `internal/trades/builder.go`
- Create: `internal/trades/builder_test.go`

**Entry points (to mirror assets):**
- `BuildScreen(res *ListResult, catalog []assetscatalog.Asset, p ListParams, lang string) components.Component` — full screen: header + list region + empty modal slot.
- `BuildTradesSection(res *ListResult, catalog []assetscatalog.Asset, p ListParams, lang string) components.Component` — list region subtree only (target for partial replace).

**Design rules** (per `spec/screens/trades.md`):
- List region is a `Column` with a fixed `trades-section` id so partial replace works.
- Two filters live on one `Row`: an asset `Select` (options: `Any` + one per catalog asset labeled by ticker) and a `trade_type` `Select` (options: `All` with empty value, `BUY`, `SELL`). Changing either triggers `GET /actions/trades/list` with the updated query.
- Pagination omitted when `total <= size`.
- Empty-state copy differs when a filter is active: check if `p.AssetID != ""` or `p.TradeType != ""`.
- `Total = Quantity × PricePerUnit` is computed here in Go; parse the decimal strings with `strconv.ParseFloat`. If parsing fails (shouldn't happen with backend data), render `"—"` for that cell.
- `Fees == "0"` → render `"—"`, otherwise `format.FormatMoney`.
- Asset lookup: build a `map[string]assetscatalog.Asset` keyed by `ID` once; fall back to the raw `asset_id` UUID if not found.

- [ ] **Step 1: Write table-driven test for `BuildScreen` covering:**
  1. Full list with 3 trades (verify table has 9 data columns + actions cell; Total is computed).
  2. Empty, no filter → `trades.empty_title` / `trades.empty_subtitle`.
  3. Empty, with filter → `trades.empty_filtered_title` / `trades.empty_filtered_subtitle`; filters stay visible.
  4. Pagination: `total=25`, `size=10`, `offset=10` → page 2 of 3, Prev enabled, Next enabled.
  5. Pagination hidden when `total <= size`.
  6. `Fees == "0"` renders as `"—"`.
  7. Missing asset in catalog → renders `asset_id` UUID as fallback.

Use a helper to crawl the component tree and assert on specific nodes (find by id / by type). Look at `internal/assets/builder_test.go` for the crawler helper style.

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement `builder.go`.**

Structure:
```go
package trades

import (
    "strconv"
    "strings"

    "github.com/project/vk-investment-middleend/internal/components"
    "github.com/project/vk-investment-middleend/internal/i18n"
    "github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
    "github.com/project/vk-investment-middleend/internal/shared/format"
)

const (
    ScreenID           = "trades-screen"
    SectionID          = "trades-section"
    ModalSlotID        = "trades-modal-slot"
)

func BuildScreen(res *ListResult, catalog []assetscatalog.Asset, p ListParams, lang string) components.Component {
    // header ("Trades" + New Trade button) + section + modal slot
    // see internal/assets/builder.go BuildScreen for the exact layout idiom
}

func BuildTradesSection(res *ListResult, catalog []assetscatalog.Asset, p ListParams, lang string) components.Component {
    // Column { id: SectionID, children: [filters, table | empty, pagination?] }
}

// helpers
func buildFilters(catalog []assetscatalog.Asset, p ListParams, lang string) components.Component { ... }
func buildTable(trades []Trade, byID map[string]assetscatalog.Asset, lang string) components.Component { ... }
func buildRow(t Trade, byID map[string]assetscatalog.Asset, lang string) components.Component { ... }
func buildEmpty(p ListParams, lang string) components.Component { ... }
func buildPagination(p ListParams, total, size int, lang string) components.Component { ... }

func totalString(qty, ppu, currency, lang string) string {
    q, err1 := strconv.ParseFloat(qty, 64)
    v, err2 := strconv.ParseFloat(ppu, 64)
    if err1 != nil || err2 != nil {
        return "—"
    }
    total := q * v
    return format.FormatMoney(&total, currency, lang)
}

func feesString(fees, currency, lang string) string {
    if fees == "" || fees == "0" {
        return "—"
    }
    v, err := strconv.ParseFloat(fees, 64)
    if err != nil {
        return "—"
    }
    return format.FormatMoney(&v, currency, lang)
}

func quantityString(qty, lang string) string {
    v, err := strconv.ParseFloat(qty, 64)
    if err != nil {
        return "—"
    }
    return format.FormatQuantity(&v, lang)
}

func priceString(ppu, currency, lang string) string {
    v, err := strconv.ParseFloat(ppu, 64)
    if err != nil {
        return "—"
    }
    return format.FormatMoney(&v, currency, lang)
}

func truncateNotes(s string) string {
    if len(s) <= 40 {
        return s
    }
    return s[:40] + "…"
}

func dateOnly(rfc3339 string) string {
    if len(rfc3339) >= 10 {
        return rfc3339[:10]
    }
    return rfc3339
}
```

Concrete layout rules (follow `internal/assets/builder.go` for the matching idioms):
- **Filters row:** asset `Select` (options = `{label: i18n.T(lang, "trades.filter.asset_any"), value: ""}` + one per catalog asset `{label: a.Ticker, value: a.ID}`). `trade_type` `Select` with three options (label `trades.filter.type_all` / value `""`, `trades.filter.type_buy` / `"BUY"`, `trades.filter.type_sell` / `"SELL"`). Both controls have an `on_change` action pointing at `/actions/trades/list` with the updated query, preserving the other filter's current value in the query string.
- **Table header row:** i18n keys `trades.col.date`, `trades.col.asset`, `trades.col.type`, `trades.col.quantity`, `trades.col.price`, `trades.col.total`, `trades.col.fees`, `trades.col.source`, `trades.col.notes`, plus an empty Actions header.
- **Data row:** 10 cells in the same order. `Type` uses `components.Badge` with a `green`/`red` variant. `Source` uses `Badge` with a neutral variant.
- **Actions cell:** two icon buttons. Edit → `GET /actions/trades/edit_modal?id=<t.ID>` replacing `ModalSlotID`. Delete → `GET /actions/trades/delete_modal?id=<t.ID>` replacing `ModalSlotID`.
- **Pagination:** identical math to assets (`page = offset/size + 1`, `total_pages = ceil(total/size)`). Each button's action URL carries `asset_id`, `trade_type`, and the new `offset`.

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/builder.go internal/trades/builder_test.go
git commit -m "feat(trades): add screen + list subtree builder"
```

### Task 4.2: Modal builder (create / edit / delete)

**Files:**
- Create: `internal/trades/modal_builder.go`
- Create: `internal/trades/modal_builder_test.go`

**Entry points:**
- `BuildCreateModal(catalog []assetscatalog.Asset, p ListParams, lang string, inlineError string) components.Component`
- `BuildEditModal(t Trade, catalog []assetscatalog.Asset, p ListParams, lang string, inlineError string) components.Component`
- `BuildDeleteModal(t Trade, catalog []assetscatalog.Asset, p ListParams, lang string, inlineError string) components.Component`

**Rules:**
- All three modals have a root Modal with id `trades-modal` (wrapped in a `Column` with id `ModalSlotID` by the handler — match the assets pattern: `internal/assets/modal_builder.go`).
- **Create form fields (7):** `asset_id` (Select, options = catalog filtered to `!IsComplex`, labeled by ticker), `trade_type` (Select: BUY/SELL), `quantity` (Input text, required, min regex `^[0-9]+(\.[0-9]+)?$`), `price_per_unit` (Input text, required), `fees` (Input text, default "0"), `date` (Input date, required, `max` = today's date computed in Go), `notes` (Textarea, max 500).
- **Edit form fields:** same but `date` and `source` are static text (rendered via `components.Text` with a label + value pattern — see `internal/assets/modal_builder.go` for how immutable fields look). `asset_id` select still excludes complex assets but defaults to the current trade's asset.
- **Delete modal body:** single confirmation message interpolated with the trade's type / quantity / ticker / date.
- **Submit URLs** carry `asset_id`, `trade_type`, and `offset` from `p` so post-mutation refresh preserves the list context.
- **Inline error** (if non-empty): render as a prominent `Text` at the top of the modal's form body. The handler passes this on `422`.
- If the catalog is empty (no assets exist), the create modal shows a disabled submit button and an info `Text` with key `trades.form.no_assets_hint`.

- [ ] **Step 1: Write tests.**

Cover:
1. Create modal renders all 7 fields with correct i18n labels and the current list context in the submit URL.
2. Create modal excludes complex assets from the `asset_id` options (include one complex + one non-complex catalog entry; assert only the non-complex appears).
3. Create modal with empty catalog: submit is disabled, hint text is present.
4. Create modal with `inlineError = "Invalid quantity"` renders the error at the top.
5. Edit modal shows `date` and `source` as static labeled text (not as inputs) and pre-populates all mutable fields from the given `Trade`.
6. Delete modal interpolates the trade's `TradeType`, `Quantity`, ticker (resolved via catalog), and date.

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement `modal_builder.go`.**

Mirror `internal/assets/modal_builder.go`'s idioms. Use `time.Now().Format("2006-01-02")` for the `max` date attribute.

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/modal_builder.go internal/trades/modal_builder_test.go
git commit -m "feat(trades): add create/edit/delete modal builders"
```

---

## Phase 5 — Use case and handlers

### Task 5.1: GetUseCase

**Files:**
- Create: `internal/trades/get_usecase.go`
- Create: `internal/trades/get_usecase_test.go`

Mirror `internal/assets/get_usecase.go`, but `Execute` takes both a trades list fetcher and a catalog provider.

```go
package trades

import (
    "context"

    "github.com/project/vk-investment-middleend/internal/components"
    "github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

type tradeFetcher interface {
    List(ctx context.Context, authorization string, p ListParams) (*ListResult, error)
}

type catalogFetcher interface {
    List(ctx context.Context, authorization string) ([]assetscatalog.Asset, error)
}

type GetUseCase struct {
    client  tradeFetcher
    catalog catalogFetcher
}

func NewGetUseCase(client tradeFetcher, catalog catalogFetcher) *GetUseCase {
    return &GetUseCase{client: client, catalog: catalog}
}

func (uc *GetUseCase) Execute(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
    res, err := uc.client.List(ctx, authorization, p)
    if err != nil {
        return components.Component{}, err
    }
    cat, err := uc.catalog.List(ctx, authorization)
    if err != nil {
        return components.Component{}, err
    }
    return BuildScreen(res, cat, p, lang), nil
}

func (uc *GetUseCase) ExecuteSection(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
    res, err := uc.client.List(ctx, authorization, p)
    if err != nil {
        return components.Component{}, err
    }
    cat, err := uc.catalog.List(ctx, authorization)
    if err != nil {
        return components.Component{}, err
    }
    return BuildTradesSection(res, cat, p, lang), nil
}
```

**Error mapping:** in the handler layer, if either call returns `ErrUnauthorized` (from trades or catalog) → `401` redirect; otherwise → `502 BACKEND_ERROR`.

- [ ] **Step 1: Write tests** with two stub fetchers (trades + catalog). Cover happy path, trades-401, catalog-401, trades-5xx, catalog-5xx. Assert the returned Component has id `ScreenID` (or `SectionID` for `ExecuteSection`).

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement.**

- [ ] **Step 4: Run — PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/get_usecase.go internal/trades/get_usecase_test.go
git commit -m "feat(trades): add GetUseCase composing list client + asset catalog"
```

### Task 5.2: Screen + list handlers

**Files:**
- Create: `internal/trades/handler.go`
- Create: `internal/trades/handler_test.go`
- Create: `internal/trades/list_handler.go`
- Create: `internal/trades/list_handler_test.go`

`handler.go` handles `GET /screens/trades`. `list_handler.go` handles `GET /actions/trades/list`. Both parse the same three query params (`asset_id`, `trade_type`, `offset`).

- [ ] **Step 1: Write a shared `parseListParams` test first.** Cover: empty → zero value; `asset_id` accepts only valid UUIDs (parsed via `github.com/google/uuid`), anything else → 400; `trade_type` accepts exactly `"BUY"` or `"SELL"`, anything else → 400; `offset` must be non-negative integer, anything else → 400.

**`asset_id` validation uses `uuid.Parse`** from `github.com/google/uuid` (already in `go.sum` as an indirect dep — promote to direct in this task). The parsed value is not needed; we only use it to reject malformed input before forwarding to the backend.

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement `handler.go` mirroring `internal/assets/handler.go`.**

Key differences from assets:
- Parse three params instead of two.
- Use `GetUseCase.Execute` (returns full screen).

```go
package trades

import (
    "errors"
    "net/http"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "github.com/project/vk-investment-middleend/internal/shared"
    "github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

type Handler struct{ uc *GetUseCase }

func NewHandler(uc *GetUseCase) *Handler { return &Handler{uc: uc} }

func (h *Handler) Get(c *gin.Context) {
    params, err := parseListParams(c)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
        return
    }
    auth := c.GetHeader("Authorization")
    lang := parseLang(c)

    tree, err := h.uc.Execute(c.Request.Context(), auth, params, lang)
    if err != nil {
        if errors.Is(err, ErrUnauthorized) || errors.Is(err, assetscatalog.ErrUnauthorized) {
            shared.RespondUnauthorized(c, "/login")
            return
        }
        c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load trades"}})
        return
    }
    c.JSON(http.StatusOK, tree)
}

func parseListParams(c *gin.Context) (ListParams, error) {
    p := ListParams{}
    if v := c.Query("asset_id"); v != "" {
        if _, err := uuid.Parse(v); err != nil {
            return p, errors.New("invalid asset_id")
        }
        p.AssetID = v
    }
    if v := c.Query("trade_type"); v != "" {
        if v != "BUY" && v != "SELL" {
            return p, errors.New("invalid trade_type")
        }
        p.TradeType = v
    }
    if raw := c.Query("offset"); raw != "" {
        n, err := strconv.Atoi(raw)
        if err != nil || n < 0 {
            return p, errors.New("invalid offset")
        }
        p.Offset = n
    }
    return p, nil
}

func parseLang(c *gin.Context) string {
    header := c.GetHeader("Accept-Language")
    if header == "" {
        return "en"
    }
    parts := strings.SplitN(header, ",", 2)
    lang := strings.SplitN(parts[0], "-", 2)[0]
    lang = strings.SplitN(lang, ";", 2)[0]
    return strings.TrimSpace(lang)
}
```

- [ ] **Step 4: Run — PASS.**

- [ ] **Step 5: Implement `list_handler.go`.**

```go
package trades

import (
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"

    "github.com/project/vk-investment-middleend/internal/shared"
    "github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

type ListHandler struct{ uc *GetUseCase }

func NewListHandler(uc *GetUseCase) *ListHandler { return &ListHandler{uc: uc} }

func (h *ListHandler) Get(c *gin.Context) {
    params, err := parseListParams(c)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
        return
    }
    auth := c.GetHeader("Authorization")
    lang := parseLang(c)
    tree, err := h.uc.ExecuteSection(c.Request.Context(), auth, params, lang)
    if err != nil {
        if errors.Is(err, ErrUnauthorized) || errors.Is(err, assetscatalog.ErrUnauthorized) {
            shared.RespondUnauthorized(c, "/login")
            return
        }
        c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load trades"}})
        return
    }
    c.JSON(http.StatusOK, components.ActionResponse{
        Action:   "replace",
        TargetID: SectionID,
        Tree:     tree,
    })
}
```

Note: import `components` (adjust the top of the file accordingly).

- [ ] **Step 6: Test — cover success, 400 on bad query, 401, 502.**

- [ ] **Step 7: Run — PASS.**

- [ ] **Step 8: Commit.**

```bash
git add internal/trades/handler.go internal/trades/handler_test.go internal/trades/list_handler.go internal/trades/list_handler_test.go
git commit -m "feat(trades): add screen + list handlers with query validation"
```

### Task 5.3: Modal GET handlers

**Files:**
- Create: `internal/trades/create_modal_handler.go` + test
- Create: `internal/trades/edit_modal_handler.go` + test
- Create: `internal/trades/delete_modal_handler.go` + test

Dependencies:
- `CreateModalHandler` — only needs the catalog (empty-form render).
- `EditModalHandler` — needs both a trade-by-id fetcher (`mutateClient.GetTrade`) and the catalog.
- `DeleteModalHandler` — same as Edit (needs the trade for the confirmation interpolation + catalog for the ticker).

Each handler parses the current list context (`asset_id`, `trade_type`, `offset`) via `parseListParams` and passes it through to the modal builder so the submit URL preserves it. `id` param is required for edit/delete; 400 if missing; 404 if trade not found.

All three respond with an `ActionResponse{Action: "replace", TargetID: ModalSlotID, Tree: <modal>}`.

- [ ] **Step 1: Write tests.** For edit_modal: test 200 happy path (stub `GetTrade` returns a trade, stub catalog returns 2 assets), 400 on missing id, 404 when client returns `ErrTradeNotFound`, 401, 502.

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement each handler** mirroring `internal/assets/edit_modal_handler.go`.

- [ ] **Step 4: Run — PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/create_modal_handler.go internal/trades/create_modal_handler_test.go \
       internal/trades/edit_modal_handler.go internal/trades/edit_modal_handler_test.go \
       internal/trades/delete_modal_handler.go internal/trades/delete_modal_handler_test.go
git commit -m "feat(trades): add create/edit/delete modal GET handlers"
```

### Task 5.4: Create handler (POST /actions/trades/create)

**Files:**
- Create: `internal/trades/create_handler.go` + test

**Flow:**
1. Parse list context from query (`asset_id`, `trade_type`, `offset`).
2. Parse form body: `asset_id`, `trade_type`, `quantity`, `price_per_unit`, `fees`, `date`, `notes`.
3. Build backend request body: include only non-empty fields, always inject `source: "MANUAL"`, default `fees` to `"0"` if empty. `date` is sent as the form's `YYYY-MM-DD` string — backend accepts this or a full RFC3339; verify which by looking at `be_specs/api/trades.md`. Per spec the backend request example uses full RFC3339, but the list spec accepts the date-only form too; **send `YYYY-MM-DDT00:00:00Z`** to be unambiguous.
4. Call `mutateClient.CreateTrade(ctx, auth, body)`.
5. On success: re-run the list with the preserved context via `GetUseCase.Execute`, return `ActionResponse{Action: "replace", TargetID: ScreenID, Tree, Feedback: snackbar("trades.create.success")}`.
6. On `*BackendValidationError`: fetch the catalog (for the modal's asset select), re-render `BuildCreateModal(catalog, p, lang, inlineError = err.Message)`, return `ActionResponse{Action: "replace", TargetID: ModalSlotID, Tree}`.
7. On `ErrUnauthorized`: `401` redirect.
8. On `ErrBackend` or any other error: `502`.

- [ ] **Step 1: Write tests.** Stub `tradeMutator` + `catalogFetcher` + `tradeFetcher`. Cover:
  - Happy path: POST form → backend CreateTrade called with correct body (incl. `source: "MANUAL"` and `fees: "0"` default and date formatted as RFC3339) → response replaces ScreenID with success snackbar.
  - Validation error: backend returns `INSUFFICIENT_QUANTITY` → response replaces ModalSlotID with the create modal and the inline error message.
  - 401 → redirect.
  - 502 on `ErrBackend`.

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement.** Use `internal/assets/create_handler.go` as the template. The shape is identical; only the field list, source injection, and date formatting differ.

- [ ] **Step 4: Run — PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/create_handler.go internal/trades/create_handler_test.go
git commit -m "feat(trades): add create handler (POST /actions/trades/create)"
```

### Task 5.5: Update handler (PATCH /actions/trades/:id)

**Files:**
- Create: `internal/trades/update_handler.go` + test

**Flow:**
1. Parse list context from query.
2. Parse form body. Immutable fields (`date`, `source`) must not be sent — if they appear in the body, silently ignore them.
3. Fetch the original trade via `mutateClient.GetTrade(ctx, auth, id)` to compute the diff.
4. Build a patch body containing only fields whose submitted value differs from the fetched original: `asset_id`, `trade_type`, `quantity`, `price_per_unit`, `fees`, `notes`. If nothing changed, skip the PATCH and proceed to refresh (no-op update — still success).
5. Call `mutateClient.UpdateTrade(ctx, auth, id, patch)`.
6. On success / validation error / 401 / 502: same shape as create. Success snackbar key `trades.edit.success`. Validation error re-renders the Edit modal (use `BuildEditModal` with the fetched trade and the inline error) — not the Create modal.

- [ ] **Step 1: Write tests.** Cover happy path with partial diff, no-op update (same values → no PATCH call), validation error replays Edit modal, 404 on original trade fetch, 401, 502.

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement.**

- [ ] **Step 4: Run — PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/update_handler.go internal/trades/update_handler_test.go
git commit -m "feat(trades): add update handler with diff-only PATCH body"
```

### Task 5.6: Delete handler (DELETE /actions/trades/:id)

**Files:**
- Create: `internal/trades/delete_handler.go` + test

**Flow:**
1. Parse list context from query.
2. Call `mutateClient.DeleteTrade(ctx, auth, id)`.
3. On success: re-run the list with the preserved context, return `replace` of `ScreenID` + success snackbar (`trades.delete.success`).
4. On `ErrTradeNotFound`: `404`.
5. On `*BackendValidationError`: should not normally happen for delete (no force flag), but if it does, surface as a `502` with the code for now.
6. `ErrUnauthorized` / `ErrBackend` / other: standard.

- [ ] **Step 1: Write tests.** Happy path, 404, 401, 502.

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement** mirroring `internal/assets/delete_handler.go` but without the force-delete two-stage flow.

- [ ] **Step 4: Run — PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/trades/delete_handler.go internal/trades/delete_handler_test.go
git commit -m "feat(trades): add delete handler"
```

---

## Phase 6 — i18n and wiring

### Task 6.1: Add i18n keys

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

All keys required by the spec (`spec/screens/trades.md` §i18n keys). Follow the existing `assets.*` nesting style.

- [ ] **Step 1: Write a smoke test in `internal/trades/builder_test.go` (or a new `i18n_keys_test.go`) that renders the screen in both `en` and `es` and asserts no rendered string is the raw key itself.**

```go
func TestAllI18nKeysResolved(t *testing.T) {
    for _, lang := range []string{"en", "es"} {
        tree := BuildScreen(&ListResult{Trades: []Trade{sampleTrade()}, Total: 1, Size: 10}, sampleCatalog(), ListParams{}, lang)
        strings := collectAllStrings(tree)
        for _, s := range strings {
            if strings.HasPrefix(s, "trades.") {
                t.Errorf("[%s] unresolved i18n key rendered: %q", lang, s)
            }
        }
    }
}
```

`collectAllStrings` recursively walks the component tree collecting every string attribute's value. Write this helper in the test file.

- [ ] **Step 2: Run — expect FAIL (keys not in locale files yet).**

- [ ] **Step 3: Add all required keys to `locales/en.json`.**

Required keys (spec §i18n):
- `trades.title`, `trades.new`
- `trades.empty_title`, `trades.empty_subtitle`, `trades.empty_filtered_title`, `trades.empty_filtered_subtitle`
- `trades.filter.asset`, `trades.filter.asset_any`, `trades.filter.type`, `trades.filter.type_all`, `trades.filter.type_buy`, `trades.filter.type_sell`
- `trades.col.{date,asset,type,quantity,price,total,fees,source,notes}`
- `trades.type.{buy,sell}`, `trades.source.{manual,import}`
- `trades.pagination.{prev,next,page_of}`
- `trades.create.{title,submit,success}`
- `trades.edit.{title,submit,success}`
- `trades.delete.{title,confirm,submit,success}`
- `trades.form.{asset,trade_type,quantity,price_per_unit,fees,date,notes,notes_placeholder,no_assets_hint,date_readonly,source_readonly}`

Write English copy for each. Interpolation: `trades.edit.title` = `"Edit {date} · {ticker}"`; `trades.pagination.page_of` = `"Page {current} of {total}"`; `trades.delete.confirm` = `"Delete this {type} of {quantity} {ticker} on {date}? This will affect AVCO calculations."`.

- [ ] **Step 4: Add the mirror keys to `locales/es.json`** with Spanish translations.

- [ ] **Step 5: Run — expect PASS.**

- [ ] **Step 6: Commit.**

```bash
git add locales/en.json locales/es.json internal/trades/
git commit -m "feat(trades): add en/es i18n keys for the trades screen"
```

### Task 6.2: Wire into the server

**Files:**
- Modify: `internal/server/server.go`

- [ ] **Step 1: Read `internal/server/server.go` lines 1–100 to see where `assets` is wired** and where the catalog should live (ideally constructed once per server and reused).

- [ ] **Step 2: Add catalog construction + trades wiring.** After the assets block:

```go
catalog := assetscatalog.NewCatalog(s.cfg.BackendURL, s.cfg.RequestTimeout)

tradesClient := trades.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
tradesUC := trades.NewGetUseCase(tradesClient, catalog)
protected.GET("/screens/trades", trades.NewHandler(tradesUC).Get)
protected.GET("/actions/trades/list", trades.NewListHandler(tradesUC).Get)
protected.GET("/actions/trades/create_modal", trades.NewCreateModalHandler(catalog).Get)
protected.GET("/actions/trades/edit_modal", trades.NewEditModalHandler(tradesClient, catalog).Get)
protected.GET("/actions/trades/delete_modal", trades.NewDeleteModalHandler(tradesClient, catalog).Get)
protected.POST("/actions/trades/create", trades.NewCreateHandler(tradesClient, tradesUC, catalog).Post)
protected.PATCH("/actions/trades/:id", trades.NewUpdateHandler(tradesClient, tradesUC, catalog).Patch)
protected.DELETE("/actions/trades/:id", trades.NewDeleteHandler(tradesClient, tradesUC).Delete)
```

Required imports: add `"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"` and `"github.com/project/vk-investment-middleend/internal/trades"`.

**Note:** the exact constructor signatures depend on what the handlers ended up needing; adjust as required (e.g. if `CreateHandler` only needs `mutateClient` + catalog + list fetcher, pass those).

- [ ] **Step 3: Run `make build` — expect clean compile.**

- [ ] **Step 4: Run `make test` — expect all PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/server/server.go
git commit -m "feat(server): wire trades screen + actions routes"
```

---

## Phase 7 — End-to-end verification

### Task 7.1: Smoke test the running server

- [ ] **Step 1: Restart the middleend.**

```bash
lsof -ti :8082 | xargs -r kill
./cli run
```

Wait for `Listening on :8082` or similar, then verify:

```bash
curl -s http://localhost:8082/health
```

- [ ] **Step 2: Hit each trades endpoint with a valid JWT.**

Prerequisite: obtain a JWT (the user has a flow for this; ask if unclear — don't guess).

For each of:
- `GET /screens/trades`
- `GET /screens/trades?asset_id=<real-uuid>&trade_type=BUY&offset=0`
- `GET /actions/trades/list?trade_type=SELL`
- `GET /actions/trades/create_modal`
- `GET /actions/trades/edit_modal?id=<real-trade-uuid>`
- `GET /actions/trades/delete_modal?id=<real-trade-uuid>`

Verify:
1. HTTP 200.
2. The response body is a valid SDUI component tree (has `type`, `props`, `children`).
3. No raw `trades.*` strings appear in the rendered text (i18n resolves).
4. The asset filter dropdown has one option per user asset.

- [ ] **Step 3: Verify unauthorized path.**

```bash
curl -si http://localhost:8082/screens/trades
```
Expect `HTTP/1.1 401` and body `{"error":"unauthorized","redirect":"/login"}`.

- [ ] **Step 4: Verify validation error paths.**

```bash
curl -si 'http://localhost:8082/screens/trades?trade_type=BOGUS' -H 'Authorization: Bearer <valid>'
```
Expect `400 BAD_REQUEST`.

```bash
curl -si 'http://localhost:8082/screens/trades?asset_id=not-a-uuid' -H 'Authorization: Bearer <valid>'
```
Expect `400 BAD_REQUEST`.

```bash
curl -si 'http://localhost:8082/screens/trades?offset=-1' -H 'Authorization: Bearer <valid>'
```
Expect `400 BAD_REQUEST`.

- [ ] **Step 5: If the SDUI frontend is running, open the Trades route and manually exercise:**
  - Load the screen; verify table renders.
  - Change the asset filter; verify the list re-fetches and `offset` resets.
  - Change the BUY/SELL toggle; same.
  - Paginate next/prev.
  - Open Create modal, submit a valid trade; verify snackbar + list refresh.
  - Submit an invalid trade (e.g. SELL with quantity larger than held); verify inline error.
  - Open Edit modal on an existing trade; change quantity; verify success.
  - Open Delete modal; confirm; verify removal + snackbar.

If the FE is not available to test manually, state that explicitly in the task report.

- [ ] **Step 6: No commit for this task** — it's pure verification. Any bugs found go back into a new task in this plan or a follow-up commit with a clear `fix(trades): …` message.

### Task 7.2: Update the canonical spec if implementation diverged

Per the project's SDD rule, the spec in `spec/` must match shipped behavior.

- [ ] **Step 1: Re-read `spec/screens/trades.md` and `spec/shared/assets-catalog.md`.**

- [ ] **Step 2: If anything in the implementation differs from the spec** (e.g. you ended up sending the date as `YYYY-MM-DD` without the time suffix, or you chose a different fallback for missing-asset tickers), either:
   - Change the code to match the spec, OR
   - Update the spec to match the code and commit with `docs(spec): …`.

- [ ] **Step 3: If nothing diverged, no action needed.**

---

## Self-review checklist

Before handing this plan off, confirm:

- [ ] Every `spec/screens/trades.md` requirement in the Acceptance Criteria section is covered by a task.
- [ ] The `spec/shared/assets-catalog.md` acceptance criteria are covered by Task 2.2.
- [ ] No placeholder text like "TBD" or "similar to assets — figure it out" in any task.
- [ ] Type names and function signatures used in later tasks match earlier definitions (e.g. `Catalog.List(ctx, auth) ([]Asset, error)`, `NewGetUseCase(client, catalog)`).
- [ ] Each task ends with a single, focused commit. Commit messages use Conventional Commits with no Claude co-author trailer.

## Architectural decisions (confirmed 2026-04-19)

1. **Extract `internal/shared/format/`** — Phase 1 refactors portfolio's formatters into a shared package; portfolio behavior unchanged. ✓ approved.
2. **New package `internal/shared/assetscatalog/`** — separate from `internal/assets/` so `trades` (and future screens) don't transitively depend on the Assets screen's concrete types. ✓ approved.
3. **`Select` for the `trade_type` filter** — `radio_group` is form-scoped per the SDUI spec; `Select` is consistent with the Assets filter idiom and the only reasonable primitive today. Spec (`spec/screens/trades.md`) updated to match. ✓ approved.
4. **Date sent to backend as `YYYY-MM-DDT00:00:00Z`** — unambiguous RFC3339 from the form's date input. ✓ approved.
5. **`uuid.Parse` from `github.com/google/uuid`** for `asset_id` query validation — lib already in `go.sum` as indirect; promoted to direct in Task 5.2. ✓ approved.
