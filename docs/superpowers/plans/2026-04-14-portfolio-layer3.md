# Portfolio Layer 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `spec/screens/portfolio/03-include-closed.md` — add a checkbox above the positions table that, on change, partially replaces `positions-table-card` via `POST /actions/portfolio/include_closed`.

**Architecture:** Extend the client to accept `includeClosed`, extract the existing table-builder into an exported `BuildPositionsTable`, insert a checkbox+form row in `BuildScreen`, implement a new action handler that re-fetches with `include_closed=<value>` and returns an `ActionResponse{replace}` targeting `positions-table-card`. The summary row is left untouched by the action.

**Tech Stack:** Go, Gin, testify, existing `internal/components`, `internal/portfolio`, `internal/auth`.

---

## File Structure

**Modify:**

| File | Change |
|---|---|
| `internal/portfolio/client.go` | `GetPositions` gains `includeClosed bool` parameter |
| `internal/portfolio/client_test.go` | update existing tests + new coverage for `includeClosed=true` |
| `internal/portfolio/get_usecase.go` | pass `false` explicitly |
| `internal/portfolio/get_usecase_test.go` | update `fakeFetcher.GetPositions` signature |
| `internal/portfolio/handler_test.go` | update `stubFetcher.GetPositions` signature |
| `internal/portfolio/builder.go` | extract `BuildPositionsTable`, insert checkbox row in `BuildScreen` |
| `internal/portfolio/builder_test.go` | assert new checkbox row + submit action |
| `locales/en.json`, `locales/es.json` | add `portfolio.include_closed` key |
| `internal/server/server.go` | register protected `POST /actions/portfolio/include_closed` |

**Create:**

| File | Responsibility |
|---|---|
| `internal/portfolio/include_closed_handler.go` | Gin handler for `POST /actions/portfolio/include_closed` |
| `internal/portfolio/include_closed_handler_test.go` | covers success, bad body, BE 401, BE 5xx |

---

### Task 1: Extend `Client.GetPositions` with `includeClosed`

**Files:**
- Modify: `internal/portfolio/client.go`
- Modify: `internal/portfolio/client_test.go`
- Modify: `internal/portfolio/get_usecase.go`
- Modify: `internal/portfolio/get_usecase_test.go`
- Modify: `internal/portfolio/handler_test.go`

- [ ] **Step 1: Update the existing client test**

Open `internal/portfolio/client_test.go`. Find **every call** to `c.GetPositions(ctx, "...")`. Each has two args today; add `false` as the third arg so the signature change compiles.

Specifically these four tests:
- `TestClient_GetPositions_ForwardsAuthorization`
- `TestClient_GetPositions_Unauthorized`
- `TestClient_GetPositions_BackendError`
- `TestClient_GetPositions_MalformedJSON`

Change each `c.GetPositions(context.Background(), "Bearer abc")` to `c.GetPositions(context.Background(), "Bearer abc", false)` (preserve the exact auth string per test). Same for `"Bearer bad"` and `"Bearer x"`.

Also append this new test at the end of the file:

```go
func TestClient_GetPositions_ForwardsIncludeClosed(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"positions":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetPositions(context.Background(), "Bearer t", true)
	require.NoError(t, err)
	assert.Equal(t, "include_closed=true", gotQuery)

	_, err = c.GetPositions(context.Background(), "Bearer t", false)
	require.NoError(t, err)
	assert.Equal(t, "include_closed=false", gotQuery)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/portfolio/... -run TestClient_GetPositions -v`
Expected: FAIL — `GetPositions` takes 2 args, tests pass 3.

- [ ] **Step 3: Update `GetPositions` signature**

In `internal/portfolio/client.go`, replace the `GetPositions` function body with:

```go
// GetPositions calls GET /v1/portfolio with the caller's Authorization header
// forwarded verbatim and an include_closed query param. Returns
// ErrUnauthorized on 401, ErrBackend on 5xx or malformed response.
func (c *Client) GetPositions(ctx context.Context, authorization string, includeClosed bool) ([]Position, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/portfolio", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("include_closed", strconv.FormatBool(includeClosed))
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
		positions, err := ParsePositions(body)
		if err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return positions, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

Add `"strconv"` to the import block if it is not already there.

- [ ] **Step 4: Update the use case**

In `internal/portfolio/get_usecase.go`, find the line:

```go
		p, err := uc.client.GetPositions(gctx, authorization)
```

Replace with:

```go
		p, err := uc.client.GetPositions(gctx, authorization, false)
```

And update the `portfolioFetcher` interface in the same file:

```go
type portfolioFetcher interface {
	GetPositions(ctx context.Context, authorization string, includeClosed bool) ([]Position, error)
	GetEvolutionLast(ctx context.Context, authorization string, n int) ([]EvolutionPoint, error)
}
```

- [ ] **Step 5: Update `fakeFetcher` in `get_usecase_test.go`**

In `internal/portfolio/get_usecase_test.go`, replace the `GetPositions` method on `fakeFetcher`:

```go
func (f *fakeFetcher) GetPositions(ctx context.Context, auth string, includeClosed bool) ([]Position, error) {
	f.gotAuthP = auth
	f.gotIncludeClosed = includeClosed
	return f.positions, f.posErr
}
```

And add the field to the struct declaration:

```go
type fakeFetcher struct {
	positions        []Position
	evolution        []EvolutionPoint
	posErr           error
	evoErr           error
	gotAuthP         string
	gotAuthE         string
	gotLastN         int
	gotIncludeClosed bool
}
```

Add a new test after `TestGetUseCase_FetchesBothInParallel`:

```go
func TestGetUseCase_PassesIncludeClosedFalse(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}}}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.False(t, f.gotIncludeClosed)
}
```

- [ ] **Step 6: Update `stubFetcher` in `handler_test.go`**

In `internal/portfolio/handler_test.go`, replace the `GetPositions` method on `stubFetcher`:

```go
func (s *stubFetcher) GetPositions(ctx context.Context, auth string, includeClosed bool) ([]Position, error) {
	s.gotAuth = auth
	return s.positions, s.err
}
```

If `stubFetcher` also has `GetEvolutionLast`, leave it alone. If not, the compile error will surface and that method should already exist from layer 2 — check it and leave as-is.

- [ ] **Step 7: Run the full suite**

Run: `go test ./... -count=1`
Expected: all tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/portfolio/client.go internal/portfolio/client_test.go internal/portfolio/get_usecase.go internal/portfolio/get_usecase_test.go internal/portfolio/handler_test.go
git commit -m "feat(portfolio): client.GetPositions accepts includeClosed"
```

---

### Task 2: Extract `BuildPositionsTable`

**Files:**
- Modify: `internal/portfolio/builder.go`
- Modify: `internal/portfolio/builder_test.go`

- [ ] **Step 1: Add a failing test for the extracted function**

Append to `internal/portfolio/builder_test.go`:

```go
func TestBuildPositionsTable_ReturnsCardWithExpectedID(t *testing.T) {
	ps := samplePositions()
	card := BuildPositionsTable(ps, "en", time.Now())

	assert.Equal(t, "card", card.Type)
	assert.Equal(t, "positions-table-card", card.ID)

	header := findDescendantByID(card, "positions-header")
	require.NotNil(t, header)
	body := findDescendantByID(card, "positions-body")
	require.NotNil(t, body)
	assert.Equal(t, "list", body.Type)
	assert.Len(t, body.Children, len(ps))
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/portfolio/... -run TestBuildPositionsTable -v`
Expected: FAIL — `BuildPositionsTable` undefined.

- [ ] **Step 3: Extract the function**

In `internal/portfolio/builder.go`, find the existing `buildTable` function:

```go
func buildTable(ps []Position, lang string, now time.Time) components.Component {
	headerCells := make([]components.Component, 0, 11)
	for i, key := range columnKeys {
		cell := components.Text("col-"+columnShortID(i), i18n.T(lang, key), "sm", "bold")
		headerCells = append(headerCells, cell)
	}
	header := components.Row("positions-header", columnWidths, headerCells...)

	listChildren := make([]components.Component, 0, len(ps))
	for _, p := range ps {
		listChildren = append(listChildren, buildPositionItem(p, lang, now))
	}
	body := components.List("positions-body", listChildren...)

	inner := components.ColumnWithGap("positions-table", "sm", header, body)
	return components.Card("positions-table-card", inner)
}
```

Rename it to `BuildPositionsTable` (exported, same body). Then replace the call site inside `BuildScreen`:

```go
	table := buildTable(positions, lang, now)
```

with:

```go
	table := BuildPositionsTable(positions, lang, now)
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS (new test + existing).

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/builder.go internal/portfolio/builder_test.go
git commit -m "refactor(portfolio): export BuildPositionsTable for reuse"
```

---

### Task 3: i18n key + checkbox row in `BuildScreen`

**Files:**
- Modify: `locales/en.json`, `locales/es.json`
- Modify: `internal/portfolio/builder.go`
- Modify: `internal/portfolio/builder_test.go`

- [ ] **Step 1: Add the i18n key to `locales/en.json`**

Open `locales/en.json`. Find the `"portfolio"` object and the line `"open_positions": "Open Positions",`. Add immediately after that line:

```json
    "include_closed": "Include closed positions",
```

- [ ] **Step 2: Add the i18n key to `locales/es.json`**

Open `locales/es.json`. Find the line `"open_positions": "Posiciones abiertas",`. Add immediately after:

```json
    "include_closed": "Incluir posiciones cerradas",
```

- [ ] **Step 3: Write failing tests**

Append to `internal/portfolio/builder_test.go`:

```go
func TestBuildScreen_IncludeClosedFormPresent(t *testing.T) {
	s := BuildScreen(samplePositions(), nil, "en", time.Now())

	form := findDescendantByID(s, "include-closed-form")
	require.NotNil(t, form)
	assert.Equal(t, "form", form.Type)

	cb := findDescendantByID(s, "include-closed-checkbox")
	require.NotNil(t, cb)
	assert.Equal(t, "checkbox", cb.Type)
	assert.Equal(t, "include_closed", cb.Props["name"])
	assert.Equal(t, "Include closed positions", cb.Props["label"])

	require.Len(t, cb.Actions, 1)
	a := cb.Actions[0]
	assert.Equal(t, "change", a.Trigger)
	assert.Equal(t, "submit", a.Type)
	assert.Equal(t, "/actions/portfolio/include_closed", a.Endpoint)
	assert.Equal(t, "POST", a.Method)
	assert.Equal(t, "include-closed-form", a.TargetID)
}

func TestBuildScreen_CheckboxOutsidePositionsTableCard(t *testing.T) {
	s := BuildScreen(samplePositions(), nil, "en", time.Now())
	tableCard := findDescendantByID(s, "positions-table-card")
	require.NotNil(t, tableCard)
	assert.Nil(t, findDescendantByID(*tableCard, "include-closed-checkbox"))
	assert.Nil(t, findDescendantByID(*tableCard, "include-closed-form"))
}

func TestBuildScreen_IncludeClosedLocalizedEs(t *testing.T) {
	s := BuildScreen(samplePositions(), nil, "es", time.Now())
	cb := findDescendantByID(s, "include-closed-checkbox")
	require.NotNil(t, cb)
	assert.Equal(t, "Incluir posiciones cerradas", cb.Props["label"])
}

func TestBuildScreen_EmptyHasNoIncludeClosedForm(t *testing.T) {
	s := BuildScreen(nil, nil, "en", time.Now())
	assert.Nil(t, findDescendantByID(s, "include-closed-form"))
}
```

- [ ] **Step 4: Run tests to verify failure**

Run: `go test ./internal/portfolio/... -run TestBuildScreen_IncludeClosed -v`
Expected: FAIL — no `include-closed-form` / `include-closed-checkbox` nodes.

- [ ] **Step 5: Implement the checkbox row in `BuildScreen`**

In `internal/portfolio/builder.go`, locate the current `BuildScreen` body:

```go
func BuildScreen(positions []Position, evolution []EvolutionPoint, lang string, now time.Time) components.Component {
	if len(positions) == 0 {
		return BuildEmpty(lang)
	}

	metrics := ComputeMetrics(positions, evolution)
	summary := buildSummaryRow(metrics, lang)
	table := BuildPositionsTable(positions, lang, now)

	root := components.ColumnWithGap("portfolio-root", "lg", summary, table)
	return components.Screen("portfolio", i18n.T(lang, "portfolio.title"), root)
}
```

Replace with:

```go
func BuildScreen(positions []Position, evolution []EvolutionPoint, lang string, now time.Time) components.Component {
	if len(positions) == 0 {
		return BuildEmpty(lang)
	}

	metrics := ComputeMetrics(positions, evolution)
	summary := buildSummaryRow(metrics, lang)
	controls := buildIncludeClosedForm(lang)
	table := BuildPositionsTable(positions, lang, now)

	root := components.ColumnWithGap("portfolio-root", "lg", summary, controls, table)
	return components.Screen("portfolio", i18n.T(lang, "portfolio.title"), root)
}

func buildIncludeClosedForm(lang string) components.Component {
	checkbox := components.Component{
		Type: "checkbox",
		ID:   "include-closed-checkbox",
		Props: map[string]any{
			"name":  "include_closed",
			"label": i18n.T(lang, "portfolio.include_closed"),
		},
		Actions: []components.Action{
			{
				Trigger:  "change",
				Type:     "submit",
				Endpoint: "/actions/portfolio/include_closed",
				Method:   "POST",
				TargetID: "include-closed-form",
			},
		},
	}
	return components.Form("include-closed-form", checkbox)
}
```

- [ ] **Step 6: Run full suite**

Run: `go test ./... -count=1`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/portfolio/builder.go internal/portfolio/builder_test.go locales/en.json locales/es.json
git commit -m "feat(portfolio): include-closed form + checkbox in portfolio screen"
```

---

### Task 4: `POST /actions/portfolio/include_closed` handler

**Files:**
- Create: `internal/portfolio/include_closed_handler.go`
- Create: `internal/portfolio/include_closed_handler_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/include_closed_handler_test.go`:

```go
package portfolio

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubIncludeClosedFetcher struct {
	positions        []Position
	err              error
	gotAuth          string
	gotIncludeClosed bool
	called           bool
}

func (s *stubIncludeClosedFetcher) GetPositions(ctx context.Context, auth string, includeClosed bool) ([]Position, error) {
	s.called = true
	s.gotAuth = auth
	s.gotIncludeClosed = includeClosed
	return s.positions, s.err
}

func setupIncludeClosedRouter(f positionsFetcherWithInclude) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/portfolio/include_closed", NewIncludeClosedHandler(f).Post)
	return r
}

func doPost(t *testing.T, r *gin.Engine, body string, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/actions/portfolio/include_closed", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestIncludeClosedHandler_SuccessTrueReturnsReplaceActionResponse(t *testing.T) {
	v := 100.0
	f := &stubIncludeClosedFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Currency: "USD", CurrentValue: &v}}}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{"include_closed":true}`, "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "positions-table-card", resp["target_id"])
	tree, ok := resp["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "card", tree["type"])
	assert.Equal(t, "positions-table-card", tree["id"])

	assert.True(t, f.called)
	assert.Equal(t, "Bearer tok", f.gotAuth)
	assert.True(t, f.gotIncludeClosed)
}

func TestIncludeClosedHandler_SuccessFalsePassesFalse(t *testing.T) {
	v := 100.0
	f := &stubIncludeClosedFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Currency: "USD", CurrentValue: &v}}}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{"include_closed":false}`, "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, f.gotIncludeClosed)
}

func TestIncludeClosedHandler_MalformedJSONReturns400(t *testing.T) {
	f := &stubIncludeClosedFetcher{}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `not json`, "Bearer tok")
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "BAD_REQUEST")
	assert.False(t, f.called)
}

func TestIncludeClosedHandler_MissingFieldReturns400(t *testing.T) {
	f := &stubIncludeClosedFetcher{}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{}`, "Bearer tok")
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, f.called)
}

func TestIncludeClosedHandler_BackendUnauthorizedReturns401WithRedirect(t *testing.T) {
	f := &stubIncludeClosedFetcher{err: ErrUnauthorized}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{"include_closed":true}`, "Bearer x")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"unauthorized"`)
	assert.Contains(t, w.Body.String(), `"redirect":"/screens/login"`)
}

func TestIncludeClosedHandler_BackendErrorReturns502(t *testing.T) {
	f := &stubIncludeClosedFetcher{err: ErrBackend}
	r := setupIncludeClosedRouter(f)

	w := doPost(t, r, `{"include_closed":true}`, "Bearer x")
	require.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "BACKEND_ERROR")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/portfolio/... -run TestIncludeClosedHandler -v`
Expected: FAIL — `NewIncludeClosedHandler` / `positionsFetcherWithInclude` undefined.

- [ ] **Step 3: Implement the handler**

Create `internal/portfolio/include_closed_handler.go`:

```go
package portfolio

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// positionsFetcherWithInclude is the narrow interface the include-closed
// handler needs. Satisfied by *Client.
type positionsFetcherWithInclude interface {
	GetPositions(ctx context.Context, authorization string, includeClosed bool) ([]Position, error)
}

type IncludeClosedHandler struct {
	client positionsFetcherWithInclude
}

func NewIncludeClosedHandler(client positionsFetcherWithInclude) *IncludeClosedHandler {
	return &IncludeClosedHandler{client: client}
}

type includeClosedRequest struct {
	IncludeClosed *bool `json:"include_closed"`
}

func (h *IncludeClosedHandler) Post(c *gin.Context) {
	var req includeClosedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid request body"}})
		return
	}
	if req.IncludeClosed == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "include_closed is required"}})
		return
	}

	auth := c.GetHeader("Authorization")
	positions, err := h.client.GetPositions(c.Request.Context(), auth, *req.IncludeClosed)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load portfolio"}})
		return
	}
	SortPositions(positions)

	lang := parseLang(c)
	tree := BuildPositionsTable(positions, lang, time.Now())
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "positions-table-card",
		Tree:     &tree,
	})
}

// parseLang is declared in handler.go; avoid duplicate definition. A simple
// fallback is provided here so tests in isolation still work.
func parseLangForInclude(c *gin.Context) string {
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

Note: the reference to `parseLang(c)` requires the existing function from `handler.go` — same package. Leave `parseLangForInclude` as dead code is wasteful; remove it and keep only the `parseLang(c)` call. Final handler:

```go
package portfolio

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type positionsFetcherWithInclude interface {
	GetPositions(ctx context.Context, authorization string, includeClosed bool) ([]Position, error)
}

type IncludeClosedHandler struct {
	client positionsFetcherWithInclude
}

func NewIncludeClosedHandler(client positionsFetcherWithInclude) *IncludeClosedHandler {
	return &IncludeClosedHandler{client: client}
}

type includeClosedRequest struct {
	IncludeClosed *bool `json:"include_closed"`
}

func (h *IncludeClosedHandler) Post(c *gin.Context) {
	var req includeClosedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid request body"}})
		return
	}
	if req.IncludeClosed == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "include_closed is required"}})
		return
	}

	auth := c.GetHeader("Authorization")
	positions, err := h.client.GetPositions(c.Request.Context(), auth, *req.IncludeClosed)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load portfolio"}})
		return
	}
	SortPositions(positions)

	lang := parseLang(c)
	tree := BuildPositionsTable(positions, lang, time.Now())
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "positions-table-card",
		Tree:     &tree,
	})
}

// The `context` import is used by positionsFetcherWithInclude's signature.
var _ = context.Background
```

Remove the trailing `var _ = context.Background` if `context` ends up being used transitively (it is via the interface parameter — the compiler should accept it).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/include_closed_handler.go internal/portfolio/include_closed_handler_test.go
git commit -m "feat(portfolio): POST /actions/portfolio/include_closed handler"
```

---

### Task 5: Wire the route

**Files:**
- Modify: `internal/server/server.go`

- [ ] **Step 1: Add the protected route**

In `internal/server/server.go`, locate the existing portfolio wiring block:

```go
	portfolioClient := portfolio.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	portfolioHandler := portfolio.NewHandler(portfolio.NewGetUseCase(portfolioClient))
	protected.GET("/screens/portfolio", portfolioHandler.Get)
```

Append after the last line in that block:

```go
	protected.POST("/actions/portfolio/include_closed", portfolio.NewIncludeClosedHandler(portfolioClient).Post)
```

- [ ] **Step 2: Run the full suite**

Run: `go test ./... -count=1`
Expected: all tests pass.

- [ ] **Step 3: Build and lint**

Run: `./cli build 2>&1 | tail -1 && ./cli lint 2>&1 | tail -1`
Expected: both `"status":"success"`.

- [ ] **Step 4: Smoke-test end-to-end**

Run:

```bash
cd /Users/vadimkent/repos/vk_investment_middleend_v2
lsof -ti:8082 | xargs kill -9 2>/dev/null; sleep 1
./cli run >/tmp/srv.log 2>&1 &
sleep 2

RESP=$(curl -s -X POST http://localhost:8082/actions/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@demo.com","password":"demo"}')
TOKEN=$(echo "$RESP" | python3 -c "import json,sys;print(json.load(sys.stdin)['auth']['token'])")

echo "--- portfolio screen has include-closed form ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def walk(x, acc):
    if x.get('id') in ('include-closed-form', 'include-closed-checkbox'):
        acc.append(x['id'])
    for c in x.get('children', []):
        walk(c, acc)
a = []; walk(d, a); print(sorted(a))
"

echo "--- action POST /actions/portfolio/include_closed with true ---"
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"include_closed":true}' http://localhost:8082/actions/portfolio/include_closed \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
print('action:', d.get('action'), 'target_id:', d.get('target_id'), 'tree.id:', d.get('tree',{}).get('id'))
"

echo "--- malformed body returns 400 ---"
curl -s -o /dev/null -w '%{http_code}\n' -X POST -H "Authorization: Bearer $TOKEN" \
  -d 'not json' http://localhost:8082/actions/portfolio/include_closed

echo "--- missing auth returns 401 ---"
curl -s -w '\n%{http_code}\n' -X POST -H "Content-Type: application/json" \
  -d '{"include_closed":true}' http://localhost:8082/actions/portfolio/include_closed

lsof -ti:8082 | xargs kill -9 2>/dev/null; true
```

Expected:
- `sorted(['include-closed-form', 'include-closed-checkbox'])` prints `['include-closed-checkbox', 'include-closed-form']`.
- Action response: `action: replace target_id: positions-table-card tree.id: positions-table-card`.
- Malformed body: `400`.
- Missing auth: `{"error":"unauthorized","redirect":"/screens/login"}` + `401`.

Report the observed output verbatim.

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go
git commit -m "feat(server): wire protected POST /actions/portfolio/include_closed"
```

---

## Self-Review Results

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| `GET /screens/portfolio` includes `form#include-closed-form` with `checkbox#include-closed-checkbox`, initially unchecked | Task 3 `TestBuildScreen_IncludeClosedFormPresent` + `_EmptyHasNoIncludeClosedForm` |
| Checkbox has submit action to `/actions/portfolio/include_closed` POST targeting `include-closed-form` | Task 3 `TestBuildScreen_IncludeClosedFormPresent` action assertions |
| Checkbox lives outside `positions-table-card` | Task 3 `TestBuildScreen_CheckboxOutsidePositionsTableCard` |
| `POST /actions/portfolio/include_closed` with `true` issues `GET /v1/portfolio?include_closed=true`; forwards Authorization | Task 4 `TestIncludeClosedHandler_SuccessTrueReturnsReplaceActionResponse` + Task 1 `TestClient_GetPositions_ForwardsIncludeClosed` |
| With `false` issues `GET /v1/portfolio?include_closed=false` | Task 4 `TestIncludeClosedHandler_SuccessFalsePassesFalse` |
| Response is `ActionResponse{replace, target_id: positions-table-card, tree: <card with id positions-table-card>}` | Task 4 `TestIncludeClosedHandler_SuccessTrueReturnsReplaceActionResponse` |
| Missing/malformed body → 400 BAD_REQUEST | Task 4 `TestIncludeClosedHandler_MalformedJSONReturns400`, `_MissingFieldReturns400` |
| Backend 401 → 401 + redirect | Task 4 `TestIncludeClosedHandler_BackendUnauthorizedReturns401WithRedirect` |
| Backend 5xx → 502 BACKEND_ERROR | Task 4 `TestIncludeClosedHandler_BackendErrorReturns502` |
| `portfolio.include_closed` in both locales | Task 3 Steps 1–2 + `TestBuildScreen_IncludeClosedLocalizedEs` |
| `BuildPositionsTable` returns the same card shape from both paths | Task 2 `TestBuildPositionsTable_ReturnsCardWithExpectedID`; Task 4 asserts the replace `tree` has `type: card, id: positions-table-card` |

**Placeholder scan:** none.

**Type consistency:**
- `GetPositions(ctx, authorization string, includeClosed bool) ([]Position, error)` is the single signature used by `*Client`, `portfolioFetcher` (use case), `positionsFetcherWithInclude` (action handler), `fakeFetcher` (use case test), `stubFetcher` (main handler test), `stubIncludeClosedFetcher` (action handler test).
- `BuildPositionsTable(positions []Position, lang string, now time.Time) components.Component` — same signature in `builder.go`, `BuildScreen`, and the action handler.
- Component IDs consistent: `positions-table-card`, `include-closed-form`, `include-closed-checkbox`, `include-closed-row` — used identically across builder, tests, and action handler.
