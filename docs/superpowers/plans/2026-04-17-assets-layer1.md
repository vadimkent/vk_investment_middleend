# Assets Layer 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `docs/superpowers/specs/2026-04-17-assets-layer1-design.md` — two new protected endpoints (`GET /screens/assets` and `GET /actions/assets/list`) that return an SDUI tree for the assets screen: paginated 6-column table, `asset_type` filter, prev/next pagination, and distinct empty states.

**Architecture:** New `internal/assets/` package mirroring `internal/portfolio/` layout (types → client → builder → use case → handler). Both handlers share a single `BuildAssetsSection` builder; the screen handler wraps it in `screen` + root `column`, while the list handler returns it directly in an `ActionResponse{replace}`. All reads are GETs — no POSTs in L1.

**Tech Stack:** Go, Gin, testify, existing `internal/components`, `internal/i18n`, `internal/auth`, `internal/shared`.

---

## File Structure

**Create:**

| File | Responsibility |
|---|---|
| `internal/assets/types.go` | `Asset`, `ListParams`, `ListResult` + `ParseListResponse([]byte) (*ListResult, error)` |
| `internal/assets/types_test.go` | parse happy path, nulls, empty, invalid JSON |
| `internal/assets/client.go` | HTTP client for `GET /v1/assets`; forwards `Authorization`; maps errors |
| `internal/assets/client_test.go` | forward header, params, 200/401/5xx paths |
| `internal/assets/builder.go` | `BuildScreen(result, params, lang) Component` + `BuildAssetsSection(result, params, lang) Component` + private sub-builders (table, filter, pagination, empty) |
| `internal/assets/builder_test.go` | section shape, filter default_value, pagination disabled states, empty state variants |
| `internal/assets/get_usecase.go` | `Execute(ctx, auth, params, lang) (*ListResult, Component, error)` — orchestrates client → builder |
| `internal/assets/get_usecase_test.go` | orchestration via fake client; surface `ErrUnauthorized` / `ErrBackend` |
| `internal/assets/handler.go` | `GET /screens/assets` — parses query, calls use case, emits full screen |
| `internal/assets/handler_test.go` | HTTP status + body assertions (happy path, 400, 401, 502) |
| `internal/assets/list_handler.go` | `GET /actions/assets/list` — parses query, calls use case, emits `ActionResponse{replace}` |
| `internal/assets/list_handler_test.go` | HTTP + action response assertions (happy path, 400, 401, 502) |

**Modify:**

- `locales/en.json`, `locales/es.json` — add top-level `assets` block.
- `internal/server/server.go` — register two protected routes.
- `spec/spec.md` — flip the assets row in the Screens table from "TBD" to the design doc reference (L1 only).

---

### Task 1: i18n keys

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

- [ ] **Step 1: Update `locales/en.json`**

Currently the JSON has a top-level `home` block as the last entry. Insert a new top-level `"assets"` block **before** the `home` block (so it sits with the other screen blocks). The `assets` block replaces nothing else — just add it.

Use Edit to insert, using this exact block (keep the preceding `}` on the `time` object and add a comma there before the new `assets` block):

```json
  "assets": {
    "title": "Assets",
    "filter": {
      "type": "Type",
      "type_any": "Any"
    },
    "col": {
      "ticker": "Ticker",
      "name": "Name",
      "type": "Type",
      "currency": "Currency",
      "complex": "Complex",
      "price_provider": "Price Provider"
    },
    "pagination": {
      "prev": "Previous",
      "next": "Next",
      "page_of": "Page {current} of {total}"
    },
    "empty_title": "No assets registered yet",
    "empty_subtitle": "Once you register assets, they will appear here.",
    "empty_filtered_title": "No assets match the filter",
    "empty_filtered_subtitle": "Try changing or clearing the filter."
  },
```

The containing file must remain valid JSON (commas between all top-level blocks except the last).

- [ ] **Step 2: Update `locales/es.json`** with the parallel Spanish block:

```json
  "assets": {
    "title": "Activos",
    "filter": {
      "type": "Tipo",
      "type_any": "Todos"
    },
    "col": {
      "ticker": "Ticker",
      "name": "Nombre",
      "type": "Tipo",
      "currency": "Moneda",
      "complex": "Complejo",
      "price_provider": "Proveedor"
    },
    "pagination": {
      "prev": "Anterior",
      "next": "Siguiente",
      "page_of": "Página {current} de {total}"
    },
    "empty_title": "Aún no hay activos",
    "empty_subtitle": "Cuando registres activos, aparecerán aquí.",
    "empty_filtered_title": "Ningún activo coincide con el filtro",
    "empty_filtered_subtitle": "Probá cambiar o limpiar el filtro."
  },
```

- [ ] **Step 3: Verify both files still parse**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && python3 -c "import json; json.load(open('locales/en.json')); json.load(open('locales/es.json')); print('ok')"`
Expected: `ok`.

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "i18n: add assets.* keys (title, filter, col, pagination, empty states)"
```

---

### Task 2: Domain types and parser

**Files:**
- Create: `internal/assets/types.go`
- Create: `internal/assets/types_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/types_test.go`:

```go
package assets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseListResponse_AllFieldsSet(t *testing.T) {
	raw := []byte(`{
	  "assets":[
	    {
	      "id":"a1","ticker":"AAPL","name":"Apple Inc.","asset_type":"STOCK","currency":"USD",
	      "is_complex":false,"price_provider":"TWELVE_DATA","external_ticker":"AAPL",
	      "created_at":"2024-01-10T10:00:00Z"
	    }
	  ],
	  "total":42,"size":10,"offset":0
	}`)

	r, err := ParseListResponse(raw)
	require.NoError(t, err)
	require.Len(t, r.Assets, 1)
	assert.Equal(t, 42, r.Total)
	assert.Equal(t, 10, r.Size)
	assert.Equal(t, 0, r.Offset)

	a := r.Assets[0]
	assert.Equal(t, "a1", a.ID)
	assert.Equal(t, "AAPL", a.Ticker)
	assert.Equal(t, "Apple Inc.", a.Name)
	assert.Equal(t, "STOCK", a.AssetType)
	assert.Equal(t, "USD", a.Currency)
	assert.False(t, a.IsComplex)
	require.NotNil(t, a.PriceProvider)
	assert.Equal(t, "TWELVE_DATA", *a.PriceProvider)
}

func TestParseListResponse_NullPriceProviderAndComplex(t *testing.T) {
	raw := []byte(`{
	  "assets":[
	    {
	      "id":"a2","ticker":"HOUSE","name":"Apartment","asset_type":"REAL_ESTATE","currency":"USD",
	      "is_complex":true,"price_provider":null,"external_ticker":null,
	      "created_at":"2024-01-11T10:00:00Z"
	    }
	  ],
	  "total":1,"size":10,"offset":0
	}`)

	r, err := ParseListResponse(raw)
	require.NoError(t, err)
	require.Len(t, r.Assets, 1)

	a := r.Assets[0]
	assert.True(t, a.IsComplex)
	assert.Nil(t, a.PriceProvider)
}

func TestParseListResponse_EmptyAssets(t *testing.T) {
	raw := []byte(`{"assets":[],"total":0,"size":10,"offset":0}`)
	r, err := ParseListResponse(raw)
	require.NoError(t, err)
	assert.Empty(t, r.Assets)
	assert.Equal(t, 0, r.Total)
}

func TestParseListResponse_InvalidJSON(t *testing.T) {
	_, err := ParseListResponse([]byte(`not json`))
	require.Error(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/assets/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement types**

Create `internal/assets/types.go`:

```go
package assets

import "encoding/json"

// Asset is the middleend domain representation of a backend asset.
type Asset struct {
	ID            string
	Ticker        string
	Name          string
	AssetType     string
	Currency      string
	IsComplex     bool
	PriceProvider *string
}

// ListParams captures the query parameters accepted by both asset endpoints.
type ListParams struct {
	AssetType string // "" means no filter; otherwise one of STOCK/ETF/CRYPTO/BOND
	Offset    int    // non-negative; 0 when unset
}

// ListResult wraps the parsed backend list response.
type ListResult struct {
	Assets []Asset
	Total  int
	Size   int
	Offset int
}

type rawAsset struct {
	ID            string  `json:"id"`
	Ticker        string  `json:"ticker"`
	Name          string  `json:"name"`
	AssetType     string  `json:"asset_type"`
	Currency      string  `json:"currency"`
	IsComplex     bool    `json:"is_complex"`
	PriceProvider *string `json:"price_provider"`
}

type rawListResponse struct {
	Assets []rawAsset `json:"assets"`
	Total  int        `json:"total"`
	Size   int        `json:"size"`
	Offset int        `json:"offset"`
}

// ParseListResponse parses the backend GET /v1/assets body into a ListResult.
func ParseListResponse(body []byte) (*ListResult, error) {
	var r rawListResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := &ListResult{Total: r.Total, Size: r.Size, Offset: r.Offset}
	out.Assets = make([]Asset, 0, len(r.Assets))
	for _, ra := range r.Assets {
		out.Assets = append(out.Assets, Asset{
			ID:            ra.ID,
			Ticker:        ra.Ticker,
			Name:          ra.Name,
			AssetType:     ra.AssetType,
			Currency:      ra.Currency,
			IsComplex:     ra.IsComplex,
			PriceProvider: ra.PriceProvider,
		})
	}
	return out, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/assets/... -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/assets/types.go internal/assets/types_test.go
git commit -m "feat(assets): domain types + ParseListResponse"
```

---

### Task 3: Backend client

**Files:**
- Create: `internal/assets/client.go`
- Create: `internal/assets/client_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/client_test.go`:

```go
package assets

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_List_ForwardsAuthAndParams(t *testing.T) {
	var gotAuth string
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/assets", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[{"id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD","is_complex":false,"price_provider":"TWELVE_DATA"}],"total":1,"size":10,"offset":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	res, err := c.List(context.Background(), "Bearer token-xyz", ListParams{AssetType: "STOCK", Offset: 10})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.Total)
	assert.Equal(t, "Bearer token-xyz", gotAuth)

	// The query must include size, sort, order, asset_type, offset.
	assert.Contains(t, gotQuery, "size=10")
	assert.Contains(t, gotQuery, "sort=ticker")
	assert.Contains(t, gotQuery, "order=desc")
	assert.Contains(t, gotQuery, "asset_type=STOCK")
	assert.Contains(t, gotQuery, "offset=10")
}

func TestClient_List_OmitsAssetTypeWhenEmpty(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets":[],"total":0,"size":10,"offset":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.NoError(t, err)

	assert.NotContains(t, gotQuery, "asset_type=")
	assert.Contains(t, gotQuery, "offset=0")
}

func TestClient_List_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_List_Backend5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_List_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.List(context.Background(), "Bearer t", ListParams{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/assets/... -v`
Expected: FAIL — `NewClient`, `ErrUnauthorized`, `ErrBackend` undefined.

- [ ] **Step 3: Implement the client**

Create `internal/assets/client.go`:

```go
package assets

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

// Client talks to the backend /v1/assets endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// List calls GET /v1/assets with the caller's Authorization header forwarded
// verbatim. Always sends size=10, sort=ticker, order=desc. Sends asset_type
// and offset when set. Returns ErrUnauthorized on 401, ErrBackend on 5xx,
// network errors, or malformed JSON.
func (c *Client) List(ctx context.Context, authorization string, p ListParams) (*ListResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/assets", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("size", "10")
	q.Set("sort", "ticker")
	q.Set("order", "desc")
	q.Set("offset", strconv.Itoa(p.Offset))
	if p.AssetType != "" {
		q.Set("asset_type", p.AssetType)
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

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/assets/... -v`
Expected: PASS (5 client tests + 4 types tests = 9 total).

- [ ] **Step 5: Commit**

```bash
git add internal/assets/client.go internal/assets/client_test.go
git commit -m "feat(assets): HTTP client for GET /v1/assets with auth + params forwarding"
```

---

### Task 4: SDUI builder — filter, table, pagination, empty state

**Files:**
- Create: `internal/assets/builder.go`
- Create: `internal/assets/builder_test.go`

The builder exposes two public entry points:

- `BuildScreen(result *ListResult, params ListParams, lang string) components.Component` — wraps the section in a `screen` + `assets-root` column.
- `BuildAssetsSection(result *ListResult, params ListParams, lang string) components.Component` — the replaceable subtree used by both handlers.

Internally it composes: filter form → (table + pagination) when `len(result.Assets) > 0`, else (filter form → empty column).

- [ ] **Step 1: Write the failing tests**

Create `internal/assets/builder_test.go`:

```go
package assets

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

func TestMain(m *testing.M) {
	// Load locales for i18n.T resolution.
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile -> internal/assets/builder_test.go; walk up to repo root.
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = i18n.Load(filepath.Join(repoRoot, "locales"))
	m.Run()
}

func findByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if got := findByID(c.Children[i], id); got != nil {
			return got
		}
	}
	return nil
}

func sampleAsset(id, ticker string, isComplex bool, provider *string) Asset {
	return Asset{
		ID: id, Ticker: ticker, Name: "Name-" + ticker,
		AssetType: "STOCK", Currency: "USD",
		IsComplex: isComplex, PriceProvider: provider,
	}
}

func TestBuildScreen_ShapeAndTitle(t *testing.T) {
	provider := "TWELVE_DATA"
	result := &ListResult{
		Assets: []Asset{sampleAsset("a1", "AAPL", false, &provider)},
		Total:  1, Size: 10, Offset: 0,
	}
	tree := BuildScreen(result, ListParams{}, "en")

	assert.Equal(t, "screen", tree.Type)
	assert.Equal(t, "assets", tree.ID)
	assert.Equal(t, "Assets", tree.Props["title"])

	root := findByID(tree, "assets-root")
	require.NotNil(t, root)
	assert.Equal(t, "column", root.Type)

	section := findByID(tree, "assets-section")
	require.NotNil(t, section)
	assert.Equal(t, "column", section.Type)
}

func TestBuildAssetsSection_FilterSelectAction(t *testing.T) {
	result := &ListResult{Assets: []Asset{}, Total: 0, Size: 10, Offset: 0}
	section := BuildAssetsSection(result, ListParams{AssetType: "STOCK"}, "en")

	sel := findByID(section, "asset-type-select")
	require.NotNil(t, sel)
	assert.Equal(t, "select", sel.Type)
	assert.Equal(t, "STOCK", sel.Props["default_value"])

	require.Len(t, sel.Actions, 1)
	act := sel.Actions[0]
	assert.Equal(t, "change", act.Trigger)
	assert.Equal(t, "submit", act.Type)
	assert.Equal(t, "GET", act.Method)
	assert.Equal(t, "/actions/assets/list", act.Endpoint)
	assert.Equal(t, "assets-filter-form", act.TargetID)
	assert.Equal(t, "section", act.Loading)
}

func TestBuildAssetsSection_FilterSelectOptions(t *testing.T) {
	section := BuildAssetsSection(&ListResult{Size: 10}, ListParams{}, "en")
	sel := findByID(section, "asset-type-select")
	require.NotNil(t, sel)

	opts, _ := json.Marshal(sel.Props["options"])
	var parsed []components.SelectOption
	require.NoError(t, json.Unmarshal(opts, &parsed))
	require.Len(t, parsed, 5)
	assert.Equal(t, "", parsed[0].Value)
	assert.Equal(t, "Any", parsed[0].Label)
	assert.Equal(t, "STOCK", parsed[1].Value)
	assert.Equal(t, "ETF", parsed[2].Value)
	assert.Equal(t, "CRYPTO", parsed[3].Value)
	assert.Equal(t, "BOND", parsed[4].Value)
}

func TestBuildAssetsSection_TableColumnsAndRows(t *testing.T) {
	provider := "TWELVE_DATA"
	result := &ListResult{
		Assets: []Asset{
			sampleAsset("a1", "AAPL", false, &provider),
			sampleAsset("a2", "HOUSE", true, nil),
			sampleAsset("a3", "TSLA", false, nil),
		},
		Total: 3, Size: 10, Offset: 0,
	}
	section := BuildAssetsSection(result, ListParams{}, "en")
	table := findByID(section, "assets-table")
	require.NotNil(t, table)
	assert.Equal(t, "table", table.Type)

	colsRaw, _ := json.Marshal(table.Props["columns"])
	var cols []components.TableColumn
	require.NoError(t, json.Unmarshal(colsRaw, &cols))
	require.Len(t, cols, 6)
	assert.Equal(t, []string{"ticker", "name", "type", "currency", "complex", "price_provider"},
		[]string{cols[0].ID, cols[1].ID, cols[2].ID, cols[3].ID, cols[4].ID, cols[5].ID})

	require.Len(t, table.Children, 3)

	// Row 1: AAPL, not complex, provider set.
	r1 := table.Children[0]
	require.Equal(t, "table_row", r1.Type)
	require.Len(t, r1.Children, 6)
	assert.Equal(t, "AAPL", r1.Children[0].Props["content"])
	assert.Equal(t, "—", r1.Children[4].Props["content"]) // complex=false renders "—"
	assert.Equal(t, "TWELVE_DATA", r1.Children[5].Props["content"])

	// Row 2: HOUSE, complex=true, provider null.
	r2 := table.Children[1]
	assert.Equal(t, "✓", r2.Children[4].Props["content"])
	assert.Equal(t, "—", r2.Children[5].Props["content"]) // complex -> dash

	// Row 3: TSLA, not complex, provider null.
	r3 := table.Children[2]
	assert.Equal(t, "—", r3.Children[4].Props["content"])
	assert.Equal(t, "—", r3.Children[5].Props["content"])
}

func TestBuildAssetsSection_PaginationOmittedWhenTotalFits(t *testing.T) {
	result := &ListResult{
		Assets: []Asset{sampleAsset("a1", "AAPL", false, nil)},
		Total:  1, Size: 10, Offset: 0,
	}
	section := BuildAssetsSection(result, ListParams{}, "en")
	assert.Nil(t, findByID(section, "assets-pagination"))
}

func TestBuildAssetsSection_PaginationFirstPage(t *testing.T) {
	assets := make([]Asset, 10)
	for i := range assets {
		assets[i] = sampleAsset("a", "T", false, nil)
	}
	result := &ListResult{Assets: assets, Total: 25, Size: 10, Offset: 0}
	section := BuildAssetsSection(result, ListParams{AssetType: "STOCK"}, "en")

	pag := findByID(section, "assets-pagination")
	require.NotNil(t, pag)

	prev := findByID(*pag, "pagination-prev")
	require.NotNil(t, prev)
	assert.Equal(t, true, prev.Props["disabled"])

	next := findByID(*pag, "pagination-next")
	require.NotNil(t, next)
	assert.NotEqual(t, true, next.Props["disabled"])
	require.Len(t, next.Actions, 1)
	assert.Equal(t, "reload", next.Actions[0].Type)
	assert.Contains(t, next.Actions[0].Endpoint, "asset_type=STOCK")
	assert.Contains(t, next.Actions[0].Endpoint, "offset=10")
	assert.Equal(t, "assets-section", next.Actions[0].TargetID)

	info := findByID(*pag, "pagination-info")
	require.NotNil(t, info)
	assert.Equal(t, "Page 1 of 3", info.Props["content"])
}

func TestBuildAssetsSection_PaginationLastPage(t *testing.T) {
	assets := make([]Asset, 5)
	for i := range assets {
		assets[i] = sampleAsset("a", "T", false, nil)
	}
	result := &ListResult{Assets: assets, Total: 25, Size: 10, Offset: 20}
	section := BuildAssetsSection(result, ListParams{}, "en")
	pag := findByID(section, "assets-pagination")
	require.NotNil(t, pag)

	prev := findByID(*pag, "pagination-prev")
	require.NotNil(t, prev)
	assert.NotEqual(t, true, prev.Props["disabled"])
	require.Len(t, prev.Actions, 1)
	assert.Contains(t, prev.Actions[0].Endpoint, "offset=10")

	next := findByID(*pag, "pagination-next")
	require.NotNil(t, next)
	assert.Equal(t, true, next.Props["disabled"])

	info := findByID(*pag, "pagination-info")
	require.NotNil(t, info)
	assert.Equal(t, "Page 3 of 3", info.Props["content"])
}

func TestBuildAssetsSection_EmptyNoFilter(t *testing.T) {
	result := &ListResult{Assets: []Asset{}, Total: 0, Size: 10, Offset: 0}
	section := BuildAssetsSection(result, ListParams{}, "en")

	// Filter form still present.
	assert.NotNil(t, findByID(section, "assets-filter-form"))
	// No table, no pagination.
	assert.Nil(t, findByID(section, "assets-table"))
	assert.Nil(t, findByID(section, "assets-pagination"))

	empty := findByID(section, "assets-empty")
	require.NotNil(t, empty)
	title := findByID(*empty, "empty-title")
	sub := findByID(*empty, "empty-subtitle")
	require.NotNil(t, title)
	require.NotNil(t, sub)
	assert.Equal(t, "No assets registered yet", title.Props["content"])
	assert.Equal(t, "Once you register assets, they will appear here.", sub.Props["content"])
}

func TestBuildAssetsSection_EmptyWithFilter(t *testing.T) {
	result := &ListResult{Assets: []Asset{}, Total: 0, Size: 10, Offset: 0}
	section := BuildAssetsSection(result, ListParams{AssetType: "STOCK"}, "en")

	title := findByID(section, "empty-title")
	sub := findByID(section, "empty-subtitle")
	require.NotNil(t, title)
	require.NotNil(t, sub)
	assert.Equal(t, "No assets match the filter", title.Props["content"])
	assert.Equal(t, "Try changing or clearing the filter.", sub.Props["content"])
}

func TestBuildScreen_SpanishTitle(t *testing.T) {
	result := &ListResult{Assets: []Asset{}, Total: 0, Size: 10, Offset: 0}
	tree := BuildScreen(result, ListParams{}, "es")
	assert.Equal(t, "Activos", tree.Props["title"])
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/assets/... -v`
Expected: FAIL — `BuildScreen`, `BuildAssetsSection` undefined.

- [ ] **Step 3: Implement the builder**

Create `internal/assets/builder.go`:

```go
package assets

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildScreen returns the full SDUI tree for GET /screens/assets.
func BuildScreen(result *ListResult, params ListParams, lang string) components.Component {
	section := BuildAssetsSection(result, params, lang)
	root := components.ColumnWithGap("assets-root", "lg", section)
	return components.Screen("assets", i18n.T(lang, "assets.title"), root)
}

// BuildAssetsSection returns the replaceable subtree shared by both handlers.
func BuildAssetsSection(result *ListResult, params ListParams, lang string) components.Component {
	children := []components.Component{buildFilterForm(params, lang)}

	if len(result.Assets) == 0 {
		children = append(children, buildEmpty(params, lang))
	} else {
		children = append(children, buildTable(result.Assets, lang))
		if result.Total > result.Size {
			children = append(children, buildPagination(result, params, lang))
		}
	}

	return components.ColumnWithGap("assets-section", "sm", children...)
}

func buildFilterForm(params ListParams, lang string) components.Component {
	opts := []components.SelectOption{
		{Value: "", Label: i18n.T(lang, "assets.filter.type_any")},
		{Value: "STOCK", Label: "STOCK"},
		{Value: "ETF", Label: "ETF"},
		{Value: "CRYPTO", Label: "CRYPTO"},
		{Value: "BOND", Label: "BOND"},
	}
	sel := components.Component{
		Type: "select",
		ID:   "asset-type-select",
		Props: map[string]any{
			"name":          "asset_type",
			"label":         i18n.T(lang, "assets.filter.type"),
			"default_value": params.AssetType,
			"options":       opts,
		},
		Actions: []components.Action{
			{
				Trigger:  "change",
				Type:     "submit",
				Method:   "GET",
				Endpoint: "/actions/assets/list",
				TargetID: "assets-filter-form",
				Loading:  "section",
			},
		},
	}
	filler := components.Spacer("filter-spacer", "none")
	row := components.Row("assets-filter-row", []string{"240px", "1fr"}, sel, filler)
	return components.Form("assets-filter-form", row)
}

func buildTable(assets []Asset, lang string) components.Component {
	cols := []components.TableColumn{
		{ID: "ticker", Header: i18n.T(lang, "assets.col.ticker"), Width: "120px"},
		{ID: "name", Header: i18n.T(lang, "assets.col.name"), Width: "1fr"},
		{ID: "type", Header: i18n.T(lang, "assets.col.type"), Width: "100px"},
		{ID: "currency", Header: i18n.T(lang, "assets.col.currency"), Width: "100px"},
		{ID: "complex", Header: i18n.T(lang, "assets.col.complex"), Width: "100px", Align: "center"},
		{ID: "price_provider", Header: i18n.T(lang, "assets.col.price_provider"), Width: "160px"},
	}
	rows := make([]components.Component, 0, len(assets))
	for _, a := range assets {
		rows = append(rows, buildRow(a))
	}
	return components.Table("assets-table", cols, rows...)
}

func buildRow(a Asset) components.Component {
	cell := func(id, content string) components.Component {
		return components.Text(id, content, "sm", "normal")
	}
	ticker := components.Text("asset-"+a.ID+"-ticker", strings.ToUpper(a.Ticker), "sm", "bold")
	complexCell := "—"
	if a.IsComplex {
		complexCell = "✓"
	}
	providerCell := "—"
	if !a.IsComplex && a.PriceProvider != nil {
		providerCell = *a.PriceProvider
	}
	return components.TableRow("asset-"+a.ID,
		ticker,
		cell("asset-"+a.ID+"-name", a.Name),
		cell("asset-"+a.ID+"-type", a.AssetType),
		cell("asset-"+a.ID+"-currency", strings.ToUpper(a.Currency)),
		cell("asset-"+a.ID+"-complex", complexCell),
		cell("asset-"+a.ID+"-price_provider", providerCell),
	)
}

func buildPagination(result *ListResult, params ListParams, lang string) components.Component {
	size := result.Size
	if size <= 0 {
		size = 10
	}
	currentPage := (result.Offset / size) + 1
	totalPages := (result.Total + size - 1) / size

	prevOffset := result.Offset - size
	if prevOffset < 0 {
		prevOffset = 0
	}
	nextOffset := result.Offset + size

	prev := paginationButton("pagination-prev", i18n.T(lang, "assets.pagination.prev"),
		paginationURL(params.AssetType, prevOffset), result.Offset == 0)
	next := paginationButton("pagination-next", i18n.T(lang, "assets.pagination.next"),
		paginationURL(params.AssetType, nextOffset), result.Offset+size >= result.Total)

	infoText := renderPageOf(i18n.T(lang, "assets.pagination.page_of"), currentPage, totalPages)
	info := components.TextStyled("pagination-info", infoText, "sm", "normal", "", "muted", "", "")

	row := components.Row("assets-pagination", []string{"auto", "1fr", "auto"}, prev, info, next)
	row.Props["gap"] = "md"
	return row
}

func paginationButton(id, label, endpoint string, disabled bool) components.Component {
	btn := components.ButtonFull(id, label, "", "secondary", "ghost",
		components.Reload(endpoint, "assets-section"),
	)
	if disabled {
		btn.Props["disabled"] = true
	}
	return btn
}

func paginationURL(assetType string, offset int) string {
	v := url.Values{}
	if assetType != "" {
		v.Set("asset_type", assetType)
	}
	v.Set("offset", strconv.Itoa(offset))
	return "/actions/assets/list?" + v.Encode()
}

func renderPageOf(template string, current, total int) string {
	s := strings.ReplaceAll(template, "{current}", fmt.Sprintf("%d", current))
	s = strings.ReplaceAll(s, "{total}", fmt.Sprintf("%d", total))
	return s
}

func buildEmpty(params ListParams, lang string) components.Component {
	titleKey := "assets.empty_title"
	subKey := "assets.empty_subtitle"
	if params.AssetType != "" {
		titleKey = "assets.empty_filtered_title"
		subKey = "assets.empty_filtered_subtitle"
	}
	title := components.Text("empty-title", i18n.T(lang, titleKey), "lg", "bold")
	sub := components.TextStyled("empty-subtitle", i18n.T(lang, subKey), "md", "normal", "", "muted", "", "")
	return components.ColumnWithGap("assets-empty", "xs", title, sub)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/assets/... -v`
Expected: PASS (all builder tests + existing 9 = 19 total).

- [ ] **Step 5: Commit**

```bash
git add internal/assets/builder.go internal/assets/builder_test.go
git commit -m "feat(assets): SDUI builder — filter, table, pagination, empty state"
```

---

### Task 5: Use case orchestration

**Files:**
- Create: `internal/assets/get_usecase.go`
- Create: `internal/assets/get_usecase_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/get_usecase_test.go`:

```go
package assets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	res     *ListResult
	err     error
	gotAuth string
	gotP    ListParams
}

func (f *fakeClient) List(_ context.Context, auth string, p ListParams) (*ListResult, error) {
	f.gotAuth = auth
	f.gotP = p
	return f.res, f.err
}

func TestUseCase_Execute_HappyPath(t *testing.T) {
	fc := &fakeClient{res: &ListResult{Assets: []Asset{{ID: "a1", Ticker: "AAPL"}}, Total: 1, Size: 10}}
	uc := NewGetUseCase(fc)

	tree, err := uc.Execute(context.Background(), "Bearer x", ListParams{AssetType: "STOCK", Offset: 0}, "en")
	require.NoError(t, err)
	assert.Equal(t, "screen", tree.Type)
	assert.Equal(t, "assets", tree.ID)
	assert.Equal(t, "Bearer x", fc.gotAuth)
	assert.Equal(t, "STOCK", fc.gotP.AssetType)
}

func TestUseCase_Execute_UnauthorizedPropagates(t *testing.T) {
	fc := &fakeClient{err: ErrUnauthorized}
	uc := NewGetUseCase(fc)

	_, err := uc.Execute(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestUseCase_ExecuteSection_ReturnsSubtree(t *testing.T) {
	fc := &fakeClient{res: &ListResult{Assets: []Asset{{ID: "a1", Ticker: "AAPL"}}, Total: 1, Size: 10}}
	uc := NewGetUseCase(fc)

	section, err := uc.ExecuteSection(context.Background(), "Bearer x", ListParams{}, "en")
	require.NoError(t, err)
	assert.Equal(t, "column", section.Type)
	assert.Equal(t, "assets-section", section.ID)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/assets/... -v`
Expected: FAIL — `NewGetUseCase`, `Execute`, `ExecuteSection` undefined.

- [ ] **Step 3: Implement the use case**

Create `internal/assets/get_usecase.go`:

```go
package assets

import (
	"context"

	"github.com/project/vk-investment-middleend/internal/components"
)

// assetFetcher is the narrow client interface the use case depends on.
type assetFetcher interface {
	List(ctx context.Context, authorization string, p ListParams) (*ListResult, error)
}

type GetUseCase struct {
	client assetFetcher
}

func NewGetUseCase(client assetFetcher) *GetUseCase {
	return &GetUseCase{client: client}
}

// Execute fetches and returns the full screen tree.
func (uc *GetUseCase) Execute(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
	res, err := uc.client.List(ctx, authorization, p)
	if err != nil {
		return components.Component{}, err
	}
	return BuildScreen(res, p, lang), nil
}

// ExecuteSection fetches and returns only the replaceable assets-section subtree.
func (uc *GetUseCase) ExecuteSection(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
	res, err := uc.client.List(ctx, authorization, p)
	if err != nil {
		return components.Component{}, err
	}
	return BuildAssetsSection(res, p, lang), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/assets/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/assets/get_usecase.go internal/assets/get_usecase_test.go
git commit -m "feat(assets): GetUseCase orchestrating client → builder"
```

---

### Task 6: Screen handler

**Files:**
- Create: `internal/assets/handler.go`
- Create: `internal/assets/handler_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/handler_test.go`:

```go
package assets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

func init() {
	gin.SetMode(gin.TestMode)
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = i18n.Load(filepath.Join(repoRoot, "locales"))
}

type stubClient struct {
	res *ListResult
	err error
	got ListParams
}

func (s *stubClient) List(_ context.Context, _ string, p ListParams) (*ListResult, error) {
	s.got = p
	return s.res, s.err
}

func newRouterWithHandler(h *Handler) *gin.Engine {
	r := gin.New()
	r.GET("/screens/assets", h.Get)
	return r
}

func TestHandler_Get_HappyPath(t *testing.T) {
	sc := &stubClient{res: &ListResult{Assets: []Asset{{ID: "a1", Ticker: "AAPL"}}, Total: 1, Size: 10}}
	h := NewHandler(NewGetUseCase(sc))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/assets?asset_type=STOCK&offset=0", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "STOCK", sc.got.AssetType)
	assert.Equal(t, 0, sc.got.Offset)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "screen", body["type"])
	assert.Equal(t, "assets", body["id"])
}

func TestHandler_Get_InvalidAssetType(t *testing.T) {
	h := NewHandler(NewGetUseCase(&stubClient{}))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/assets?asset_type=BOGUS", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Get_InvalidOffset(t *testing.T) {
	h := NewHandler(NewGetUseCase(&stubClient{}))
	r := newRouterWithHandler(h)

	for _, val := range []string{"abc", "-5"} {
		req := httptest.NewRequest(http.MethodGet, "/screens/assets?offset="+val, nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, "offset=%q", val)
	}
}

func TestHandler_Get_Unauthorized(t *testing.T) {
	sc := &stubClient{err: ErrUnauthorized}
	h := NewHandler(NewGetUseCase(sc))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/assets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/login", body["redirect"])
}

func TestHandler_Get_BackendError(t *testing.T) {
	sc := &stubClient{err: ErrBackend}
	h := NewHandler(NewGetUseCase(sc))
	r := newRouterWithHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/screens/assets", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/assets/... -v`
Expected: FAIL — `NewHandler`, `Handler.Get` undefined.

- [ ] **Step 3: Implement the handler**

Create `internal/assets/handler.go`:

```go
package assets

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
)

type Handler struct {
	uc *GetUseCase
}

func NewHandler(uc *GetUseCase) *Handler {
	return &Handler{uc: uc}
}

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
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load assets"}})
		return
	}
	c.JSON(http.StatusOK, tree)
}

// parseListParams extracts and validates asset_type and offset from the query.
func parseListParams(c *gin.Context) (ListParams, error) {
	p := ListParams{}
	at := c.Query("asset_type")
	if at != "" {
		switch at {
		case "STOCK", "ETF", "CRYPTO", "BOND":
			p.AssetType = at
		default:
			return p, errors.New("invalid asset_type")
		}
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

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/assets/... -v`
Expected: PASS (5 handler tests + 14 prior = 19).

- [ ] **Step 5: Commit**

```bash
git add internal/assets/handler.go internal/assets/handler_test.go
git commit -m "feat(assets): GET /screens/assets handler with query param validation"
```

---

### Task 7: List action handler

**Files:**
- Create: `internal/assets/list_handler.go`
- Create: `internal/assets/list_handler_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/list_handler_test.go`:

```go
package assets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRouterWithListHandler(h *ListHandler) *gin.Engine {
	r := gin.New()
	r.GET("/actions/assets/list", h.Get)
	return r
}

func TestListHandler_Get_ReturnsReplaceActionResponse(t *testing.T) {
	sc := &stubClient{res: &ListResult{Assets: []Asset{{ID: "a1", Ticker: "AAPL"}}, Total: 1, Size: 10}}
	h := NewListHandler(NewGetUseCase(sc))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/list?asset_type=STOCK&offset=0", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, "assets-section", body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok, "tree must be present")
	assert.Equal(t, "column", tree["type"])
	assert.Equal(t, "assets-section", tree["id"])
}

func TestListHandler_Get_InvalidAssetType(t *testing.T) {
	h := NewListHandler(NewGetUseCase(&stubClient{}))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/list?asset_type=BOGUS", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListHandler_Get_Unauthorized(t *testing.T) {
	sc := &stubClient{err: ErrUnauthorized}
	h := NewListHandler(NewGetUseCase(sc))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListHandler_Get_BackendError(t *testing.T) {
	sc := &stubClient{err: ErrBackend}
	h := NewListHandler(NewGetUseCase(sc))
	r := newRouterWithListHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/list", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/assets/... -v`
Expected: FAIL — `NewListHandler`, `ListHandler.Get` undefined.

- [ ] **Step 3: Implement the list handler**

Create `internal/assets/list_handler.go`:

```go
package assets

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type ListHandler struct {
	uc *GetUseCase
}

func NewListHandler(uc *GetUseCase) *ListHandler {
	return &ListHandler{uc: uc}
}

func (h *ListHandler) Get(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	section, err := h.uc.ExecuteSection(c.Request.Context(), auth, params, lang)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load assets"}})
		return
	}
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "assets-section",
		Tree:     &section,
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/assets/... -v`
Expected: PASS (4 list handler tests + 19 prior = 23).

- [ ] **Step 5: Commit**

```bash
git add internal/assets/list_handler.go internal/assets/list_handler_test.go
git commit -m "feat(assets): GET /actions/assets/list handler returning ActionResponse{replace}"
```

---

### Task 8: Route registration

**Files:**
- Modify: `internal/server/server.go`

- [ ] **Step 1: Add the import**

Edit `internal/server/server.go`. Add `"github.com/project/vk-investment-middleend/internal/assets"` to the imports list, alphabetized (between `auth` and `config`).

- [ ] **Step 2: Register the routes**

In `setupRoutes`, after the `protected.GET("/actions/portfolio/live_data", ...)` line, add:

```go
	assetsClient := assets.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	assetsUC := assets.NewGetUseCase(assetsClient)
	protected.GET("/screens/assets", assets.NewHandler(assetsUC).Get)
	protected.GET("/actions/assets/list", assets.NewListHandler(assetsUC).Get)
```

- [ ] **Step 3: Run build + tests**

Run: `go build ./... && go test ./... -count=1`
Expected: no build errors; all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/server/server.go
git commit -m "feat(server): register /screens/assets and /actions/assets/list"
```

---

### Task 9: Spec index update

**Files:**
- Modify: `spec/spec.md`

- [ ] **Step 1: Flip the Assets row in the Screens table**

Open `spec/spec.md`. Find the row:

```
| Assets | `screens/assets.md` — TBD |
```

Replace it with:

```
| Assets | `../docs/superpowers/specs/2026-04-17-assets-layer1-design.md` — L1: list + paginación + filtros |
```

- [ ] **Step 2: Commit**

```bash
git add spec/spec.md
git commit -m "docs(spec): link assets layer 1 design from spec index"
```

---

### Task 10: Smoke test via running server

**Files:**
- (no files — live verification)

- [ ] **Step 1: Build the CLI**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && make build`
Expected: `bin/server` produced without errors.

- [ ] **Step 2: Restart the middleend**

Kill any process on :8082 and start fresh in background per memory guidance:

```bash
pkill -f "go run ./cmd/server" || true
pkill -f "bin/server" || true
./cli run &
```

Wait ~2s, then verify:

```bash
curl -s http://localhost:8082/health
```

Expected: `{"status":"healthy","service":"vk-investment-middleend"}`.

- [ ] **Step 3: Hit `/screens/assets` without auth**

```bash
curl -s -o /tmp/assets_noauth.json -w "%{http_code}\n" http://localhost:8082/screens/assets
cat /tmp/assets_noauth.json
```

Expected: HTTP `401` and body containing `"unauthorized"` and `"redirect":"/login"`.

- [ ] **Step 4: Hit `/screens/assets` with a valid JWT (if available locally)**

Using a token from the login flow or the backend's test fixtures, verify:

```bash
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/assets | jq '.type, .id, .props.title'
```

Expected output:

```
"screen"
"assets"
"Assets"
```

If no local backend is running, skip this step and document it in the PR description instead.

- [ ] **Step 5: Run the full Go test suite once more**

Run: `go test ./... -count=1`
Expected: all tests pass.

- [ ] **Step 6: Lint**

Run: `make lint`
Expected: no lint errors.

- [ ] **Step 7: Leave the server running**

Per project convention, the middleend stays running on `:8082` after each change so the frontend has a live target to hit. Do not kill the process at the end of this task.

No commit — this task is verification only.

---

## Self-Review Notes

Coverage checklist against `docs/superpowers/specs/2026-04-17-assets-layer1-design.md`:

- [x] `GET /screens/assets` endpoint — Task 6.
- [x] `GET /actions/assets/list` endpoint — Task 7.
- [x] Auth forwarding to BE — Task 3.
- [x] Downstream call with `size=10&sort=ticker&order=desc` — Task 3.
- [x] Screen title via i18n — Tasks 1, 4.
- [x] `assets-section` subtree structure — Task 4.
- [x] Filter select with `default_value`, options, `submit`+`GET`+`loading:section` action — Task 4.
- [x] Table with 6 columns in order — Task 4.
- [x] Cell rendering: uppercase ticker, `✓/—` for complex, `—` for null/complex provider — Task 4.
- [x] Pagination omitted when `total <= size` — Task 4.
- [x] Prev disabled at `offset==0`, Next disabled at `offset+size >= total` — Task 4.
- [x] Pagination `reload` URLs bake in `asset_type` + target offset — Task 4.
- [x] `page_of` i18n with `{current}/{total}` substitution — Task 4.
- [x] Empty state (no filter) keys — Task 4.
- [x] Empty state (filtered) keys — Task 4.
- [x] 400 on invalid `asset_type` — Tasks 6, 7.
- [x] 400 on invalid `offset` — Tasks 6, 7.
- [x] 401 on unauthorized — Tasks 6, 7.
- [x] 502 on backend error — Tasks 6, 7.
- [x] en/es i18n — Task 1.
- [x] Route registration — Task 8.
