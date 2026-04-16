# Portfolio Layer 5 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `spec/screens/portfolio/05-live-mode.md` — add a Live toggle, live banner with refresh, per-position price-source dots, and warnings. Wrap the "data" portion of the screen in a `live-data-section` that is the reload target. Charts and allocation stay outside.

**Architecture:** (1) Extend the types+parser to produce `PortfolioResponse` (with `IsLive`, `PricesAsOf`, `Warnings`, and per-position `PriceSource`/`PriceAsOf`). (2) `Client.GetPositions` signature evolves to return `*PortfolioResponse` with `live`/`refresh` bool params — all callers updated atomically. (3) New `BuildLiveDataSection` builder composes the header+banner+summary+form+table, conditionally including live components. (4) `BuildScreen` restructured: `portfolio-root` becomes `[live-data-section, charts-section, allocation-section]`. (5) New `GET /actions/portfolio/live_data` handler.

**Tech Stack:** Go, Gin, testify, existing packages.

---

## File Structure

**Create:**

| File | Responsibility |
|---|---|
| `internal/portfolio/live_builder.go` | `LiveState`, `BuildLiveDataSection` — builds the header + banner + summary + form + table with optional dots |
| `internal/portfolio/live_builder_test.go` | standard mode (no banner/dots), live mode (banner+dots+warnings), toggle URL, refresh URL |
| `internal/portfolio/live_handler.go` | `GET /actions/portfolio/live_data` |
| `internal/portfolio/live_handler_test.go` | success live on/off, refresh, BE 401, BE 502 |

**Modify:**

| File | Change |
|---|---|
| `internal/portfolio/types.go` | Add `PortfolioResponse`, `LiveWarning`; add `PriceSource *string`, `PriceAsOf *time.Time` to `Position`; add `ParsePortfolioResponse`; make `ParsePositions` a thin wrapper |
| `internal/portfolio/types_test.go` | Tests for new fields + response wrapper |
| `internal/portfolio/client.go` | `GetPositions` returns `*PortfolioResponse`; gains `live, refresh bool` params |
| `internal/portfolio/client_test.go` | All tests updated |
| `internal/portfolio/get_usecase.go` | `portfolioFetcher` interface updated; `Execute` extracts `.Positions` from response |
| `internal/portfolio/get_usecase_test.go` | `fakeFetcher` updated |
| `internal/portfolio/handler_test.go` | `stubFetcher` updated |
| `internal/portfolio/include_closed_handler.go` | Interface + call updated |
| `internal/portfolio/include_closed_handler_test.go` | Stub updated |
| `internal/portfolio/allocation_handler.go` | Interface + call updated |
| `internal/portfolio/allocation_handler_test.go` | Stub updated |
| `internal/portfolio/builder.go` | `BuildScreen` restructured: wraps data in `live-data-section` via the live builder |
| `internal/portfolio/builder_test.go` | `live-data-section` assertions |
| `internal/server/server.go` | Register `GET /actions/portfolio/live_data` |
| `locales/{en,es}.json` | `portfolio.live.*` keys |

---

### Task 1: Types — `PortfolioResponse`, live fields on `Position`, `ParsePortfolioResponse`

**Files:**
- Modify: `internal/portfolio/types.go`
- Modify: `internal/portfolio/types_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/portfolio/types_test.go`:

```go
func TestParsePortfolioResponse_LiveFields(t *testing.T) {
	raw := []byte(`{
	  "positions":[
	    {
	      "asset_id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD",
	      "quantity":"10","avg_cost":"150","total_cost":"1500",
	      "current_price":"180","current_value":"1800",
	      "unrealized_pnl":"300","realized_pnl":"0",
	      "last_snapshot_at":"2024-06-01T10:00:00Z",
	      "price_source":"live",
	      "price_as_of":"2026-04-14T12:00:00Z"
	    }
	  ],
	  "is_live": true,
	  "prices_as_of": "2026-04-14T12:00:00Z",
	  "warnings": [
	    {"asset_id":"w1","ticker":"DOGE","error":"provider timeout"}
	  ]
	}`)

	resp, err := ParsePortfolioResponse(raw)
	require.NoError(t, err)
	assert.True(t, resp.IsLive)
	require.NotNil(t, resp.PricesAsOf)
	assert.Equal(t, time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC), *resp.PricesAsOf)
	require.Len(t, resp.Warnings, 1)
	assert.Equal(t, "DOGE", resp.Warnings[0].Ticker)
	assert.Equal(t, "provider timeout", resp.Warnings[0].Error)

	require.Len(t, resp.Positions, 1)
	p := resp.Positions[0]
	require.NotNil(t, p.PriceSource)
	assert.Equal(t, "live", *p.PriceSource)
	require.NotNil(t, p.PriceAsOf)
	assert.Equal(t, time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC), *p.PriceAsOf)
}

func TestParsePortfolioResponse_StandardMode(t *testing.T) {
	raw := []byte(`{
	  "positions":[
	    {"asset_id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD",
	     "quantity":"10","avg_cost":"150","total_cost":"1500",
	     "current_price":"180","current_value":"1800",
	     "unrealized_pnl":"300","realized_pnl":"0"}
	  ]
	}`)

	resp, err := ParsePortfolioResponse(raw)
	require.NoError(t, err)
	assert.False(t, resp.IsLive)
	assert.Nil(t, resp.PricesAsOf)
	assert.Empty(t, resp.Warnings)
	assert.Nil(t, resp.Positions[0].PriceSource)
	assert.Nil(t, resp.Positions[0].PriceAsOf)
}

func TestParsePositions_StillWorks(t *testing.T) {
	raw := []byte(`{"positions":[{"asset_id":"a1","ticker":"X","name":"X","asset_type":"STOCK","currency":"USD","quantity":"1","avg_cost":"1","total_cost":"1","current_value":"1","unrealized_pnl":"0","realized_pnl":"0"}]}`)
	positions, err := ParsePositions(raw)
	require.NoError(t, err)
	require.Len(t, positions, 1)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/portfolio/... -run "TestParsePortfolioResponse|TestParsePositions_StillWorks" -v`
Expected: FAIL — `ParsePortfolioResponse`, `PortfolioResponse`, etc. undefined.

- [ ] **Step 3: Implement**

In `internal/portfolio/types.go`:

a) Add these new types after `Position`:

```go
// LiveWarning represents a warning for an asset whose live price could not be fetched.
type LiveWarning struct {
	AssetID string
	Ticker  string
	Error   string
}

// PortfolioResponse wraps the full backend response, including live metadata.
type PortfolioResponse struct {
	Positions  []Position
	IsLive     bool
	PricesAsOf *time.Time
	Warnings   []LiveWarning
}
```

b) Add `PriceSource` and `PriceAsOf` to `Position`:

```go
type Position struct {
	AssetID        string
	Ticker         string
	Name           string
	AssetType      string
	Currency       string
	Quantity       *float64
	AvgCost        *float64
	TotalCost      *float64
	CurrentPrice   *float64
	CurrentValue   *float64
	UnrealizedPnL  *float64
	RealizedPnL    float64
	LastSnapshotAt *time.Time
	PriceSource    *string    // "live", "snapshot", "none"; nil in standard mode
	PriceAsOf      *time.Time // nil in standard mode
}
```

c) Add corresponding fields to `rawPosition`:

```go
	PriceSource    *string `json:"price_source"`
	PriceAsOfRaw   *string `json:"price_as_of"`
```

d) Extend `rawResponse` with live metadata:

```go
type rawResponse struct {
	Positions  []rawPosition    `json:"positions"`
	IsLive     bool             `json:"is_live"`
	PricesAsOf *string          `json:"prices_as_of"`
	Warnings   []rawLiveWarning `json:"warnings"`
}

type rawLiveWarning struct {
	AssetID string `json:"asset_id"`
	Ticker  string `json:"ticker"`
	Error   string `json:"error"`
}
```

e) Add `ParsePortfolioResponse`:

```go
// ParsePortfolioResponse parses the full backend /v1/portfolio body including
// live metadata.
func ParsePortfolioResponse(body []byte) (*PortfolioResponse, error) {
	var r rawResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}

	resp := &PortfolioResponse{
		IsLive: r.IsLive,
	}

	if r.PricesAsOf != nil {
		if t, err := time.Parse(time.RFC3339, *r.PricesAsOf); err == nil {
			resp.PricesAsOf = &t
		}
	}

	for _, rw := range r.Warnings {
		resp.Warnings = append(resp.Warnings, LiveWarning{
			AssetID: rw.AssetID,
			Ticker:  rw.Ticker,
			Error:   rw.Error,
		})
	}

	for _, rp := range r.Positions {
		p := Position{
			AssetID:   rp.AssetID,
			Ticker:    rp.Ticker,
			Name:      rp.Name,
			AssetType: rp.AssetType,
			Currency:  rp.Currency,
		}
		p.Quantity = parseFloatPtr(rp.Quantity)
		p.AvgCost = parseFloatPtr(rp.AvgCost)
		p.TotalCost = parseFloatPtr(rp.TotalCost)
		p.CurrentPrice = parseFloatPtr(rp.CurrentPrice)
		p.CurrentValue = parseFloatPtr(rp.CurrentValue)
		p.UnrealizedPnL = parseFloatPtr(rp.UnrealizedPnL)
		if v := parseFloatPtr(rp.RealizedPnL); v != nil {
			p.RealizedPnL = *v
		}
		if rp.LastSnapshotAt != nil {
			if t, err := time.Parse(time.RFC3339, *rp.LastSnapshotAt); err == nil {
				p.LastSnapshotAt = &t
			}
		}
		p.PriceSource = rp.PriceSource
		if rp.PriceAsOfRaw != nil {
			if t, err := time.Parse(time.RFC3339, *rp.PriceAsOfRaw); err == nil {
				p.PriceAsOf = &t
			}
		}
		resp.Positions = append(resp.Positions, p)
	}

	if resp.Positions == nil {
		resp.Positions = []Position{}
	}

	return resp, nil
}
```

f) Make `ParsePositions` a thin wrapper:

```go
// ParsePositions is a convenience wrapper over ParsePortfolioResponse that
// returns only the positions slice. Existing callers that don't need live
// metadata can continue using this.
func ParsePositions(body []byte) ([]Position, error) {
	resp, err := ParsePortfolioResponse(body)
	if err != nil {
		return nil, err
	}
	return resp.Positions, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS (all old + new tests).

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/types.go internal/portfolio/types_test.go
git commit -m "feat(portfolio): PortfolioResponse + live fields on Position"
```

---

### Task 2: Client signature cascade — `GetPositions` returns `*PortfolioResponse` with `live`/`refresh` params

This is the breaking-change task. All interfaces + callers + test fakes are updated in one atomic commit.

**Files:**
- Modify: `internal/portfolio/client.go`
- Modify: `internal/portfolio/client_test.go`
- Modify: `internal/portfolio/get_usecase.go`
- Modify: `internal/portfolio/get_usecase_test.go`
- Modify: `internal/portfolio/handler_test.go`
- Modify: `internal/portfolio/include_closed_handler.go`
- Modify: `internal/portfolio/include_closed_handler_test.go`
- Modify: `internal/portfolio/allocation_handler.go`
- Modify: `internal/portfolio/allocation_handler_test.go`

- [ ] **Step 1: Update `Client.GetPositions` in `client.go`**

Replace the `GetPositions` function. New signature:

```go
func (c *Client) GetPositions(ctx context.Context, authorization string, includeClosed, live, refresh bool) (*PortfolioResponse, error)
```

Changes inside the function body:
- Add `live` and `refresh` query params:
  ```go
  if live {
      q.Set("live", "true")
  }
  if refresh {
      q.Set("refresh", "true")
  }
  ```
- Change the success branch to use `ParsePortfolioResponse` instead of `ParsePositions`:
  ```go
  case http.StatusOK:
      resp, err := ParsePortfolioResponse(body)
      if err != nil {
          return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
      }
      return resp, nil
  ```
- Return type for errors: `return nil, ErrUnauthorized` and `return nil, fmt.Errorf(...)`.

- [ ] **Step 2: Update `client_test.go`**

Every existing `GetPositions` call has 3 args `(ctx, auth, includeClosed)`. Add `false, false` for `live, refresh` to each call. Also update the return type assertion — existing tests expect `([]Position, error)`, now it's `(*PortfolioResponse, error)`. Extract `.Positions` where the test checks positions.

For `TestClient_GetPositions_ForwardsAuthorization`:
```go
resp, err := c.GetPositions(context.Background(), "Bearer abc", false, false, false)
require.NoError(t, err)
require.Len(t, resp.Positions, 1)
assert.Equal(t, "AAPL", resp.Positions[0].Ticker)
```

For `TestClient_GetPositions_ForwardsIncludeClosed`:
```go
resp, err := c.GetPositions(context.Background(), "Bearer t", true, false, false)
// ...
resp, err = c.GetPositions(context.Background(), "Bearer t", false, false, false)
```

Add a new test:

```go
func TestClient_GetPositions_ForwardsLiveAndRefresh(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"positions":[],"is_live":true,"prices_as_of":"2026-04-14T12:00:00Z"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	resp, err := c.GetPositions(context.Background(), "Bearer t", false, true, true)
	require.NoError(t, err)
	assert.Contains(t, gotQuery, "live=true")
	assert.Contains(t, gotQuery, "refresh=true")
	assert.True(t, resp.IsLive)
	require.NotNil(t, resp.PricesAsOf)
}
```

- [ ] **Step 3: Update `portfolioFetcher` in `get_usecase.go`**

```go
type portfolioFetcher interface {
	GetPositions(ctx context.Context, authorization string, includeClosed, live, refresh bool) (*PortfolioResponse, error)
	GetEvolutionLast(ctx context.Context, authorization string, n int) ([]EvolutionPoint, error)
	GetEvolution(ctx context.Context, authorization string, q EvolutionQuery) ([]EvolutionPoint, error)
}
```

In `Execute`, update the positions fetch goroutine:

```go
g.Go(func() error {
    resp, err := uc.client.GetPositions(gctx, authorization, false, false, false)
    if err != nil {
        return err
    }
    positions = resp.Positions
    return nil
})
```

- [ ] **Step 4: Update `fakeFetcher` in `get_usecase_test.go`**

```go
func (f *fakeFetcher) GetPositions(ctx context.Context, auth string, includeClosed, live, refresh bool) (*PortfolioResponse, error) {
	f.gotAuthP = auth
	f.gotIncludeClosed = includeClosed
	return &PortfolioResponse{Positions: f.positions}, f.posErr
}
```

- [ ] **Step 5: Update `stubFetcher` in `handler_test.go`**

```go
func (s *stubFetcher) GetPositions(ctx context.Context, auth string, includeClosed, live, refresh bool) (*PortfolioResponse, error) {
	s.gotAuth = auth
	return &PortfolioResponse{Positions: s.positions}, s.err
}
```

- [ ] **Step 6: Update `positionsFetcherWithInclude` in `include_closed_handler.go`**

```go
type positionsFetcherWithInclude interface {
	GetPositions(ctx context.Context, authorization string, includeClosed, live, refresh bool) (*PortfolioResponse, error)
}
```

In the `Post` handler body, update the call:

```go
resp, err := h.client.GetPositions(c.Request.Context(), auth, *req.IncludeClosed, false, false)
// ...
positions := resp.Positions
SortPositions(positions)
```

- [ ] **Step 7: Update `stubIncludeClosedFetcher` in `include_closed_handler_test.go`**

```go
func (s *stubIncludeClosedFetcher) GetPositions(ctx context.Context, auth string, includeClosed, live, refresh bool) (*PortfolioResponse, error) {
	s.called = true
	s.gotAuth = auth
	s.gotIncludeClosed = includeClosed
	return &PortfolioResponse{Positions: s.positions}, s.err
}
```

- [ ] **Step 8: Update `allocationFetcher` in `allocation_handler.go`**

```go
type allocationFetcher interface {
	GetPositions(ctx context.Context, authorization string, includeClosed, live, refresh bool) (*PortfolioResponse, error)
}
```

In the `Get` handler body:

```go
resp, err := h.client.GetPositions(c.Request.Context(), auth, false, false, false)
// ...
positions := resp.Positions
```

- [ ] **Step 9: Update `stubAllocationFetcher` in `allocation_handler_test.go`**

```go
func (s *stubAllocationFetcher) GetPositions(ctx context.Context, auth string, includeClosed, live, refresh bool) (*PortfolioResponse, error) {
	s.called = true
	s.gotAuth = auth
	return &PortfolioResponse{Positions: s.positions}, s.err
}
```

- [ ] **Step 10: Run full suite**

Run: `go test ./... -count=1`
Expected: all tests pass.

- [ ] **Step 11: Commit**

```bash
git add internal/portfolio/client.go internal/portfolio/client_test.go internal/portfolio/get_usecase.go internal/portfolio/get_usecase_test.go internal/portfolio/handler_test.go internal/portfolio/include_closed_handler.go internal/portfolio/include_closed_handler_test.go internal/portfolio/allocation_handler.go internal/portfolio/allocation_handler_test.go
git commit -m "refactor(portfolio): GetPositions returns PortfolioResponse with live/refresh params"
```

---

### Task 3: i18n keys

**Files:**
- Modify: `locales/en.json`, `locales/es.json`

- [ ] **Step 1: Add `portfolio.live.*` keys to `locales/en.json`**

Inside the `"portfolio"` object, after the `"allocation"` block, add:

```json
    "live": {
      "toggle": "Live",
      "status": "● Live prices · Updated {time}",
      "refresh": "Refresh",
      "warning_prefix": "⚠ Could not fetch:"
    }
```

- [ ] **Step 2: Add to `locales/es.json`**

```json
    "live": {
      "toggle": "En vivo",
      "status": "● Precios en vivo · Actualizado {time}",
      "refresh": "Actualizar",
      "warning_prefix": "⚠ No se pudo obtener:"
    }
```

- [ ] **Step 3: Run full suite**

Run: `go test ./... -count=1`
Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "feat(i18n): portfolio.live.* keys"
```

---

### Task 4: Live builder — `BuildLiveDataSection`

**Files:**
- Create: `internal/portfolio/live_builder.go`
- Create: `internal/portfolio/live_builder_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/live_builder_test.go` with tests covering:

1. `TestBuildLiveDataSection_StandardMode_HasHeaderSummaryFormTable` — verify `live-data-section` contains `live-header-row`, `portfolio-summary-row`, `include-closed-form`, `positions-table-card`. No `live-banner`. No `live-warnings`.
2. `TestBuildLiveDataSection_StandardMode_ToggleButtonIsGhost` — toggle button has `variant: secondary, style: ghost`, URL `?live=true`.
3. `TestBuildLiveDataSection_LiveMode_HasBannerAndDots` — verify `live-banner` present, `live-status` text present, `live-refresh` button present.
4. `TestBuildLiveDataSection_LiveMode_ToggleButtonIsSolid` — toggle `variant: primary, style: solid`, URL `?live=false`.
5. `TestBuildLiveDataSection_LiveMode_RefreshButtonURL` — URL `?live=true&refresh=true`.
6. `TestBuildLiveDataSection_LiveMode_WarningsPresent` — `live-warnings` text with tickers.
7. `TestBuildLiveDataSection_LiveMode_WarningsAbsentWhenEmpty` — no `live-warnings` when warnings empty.
8. `TestBuildLiveDataSection_LiveMode_PriceSourceDots` — position rows contain a `"●"` text with appropriate color.
9. `TestBuildLiveDataSection_StandardMode_NoDots` — no dots in standard mode.

Each test constructs a `PortfolioResponse` with appropriate fields, a `SummaryMetrics`, and calls `BuildLiveDataSection`. Tests use `findDescendantByID` (available from `builder_test.go` in the same package).

- [ ] **Step 2: Run to verify failure**

Expected: FAIL — `BuildLiveDataSection` undefined.

- [ ] **Step 3: Implement `internal/portfolio/live_builder.go`**

```go
package portfolio

import (
	"strings"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// LiveState holds the live toggle state.
type LiveState struct {
	Live    bool
	Refresh bool // only meaningful when Live=true
}

// BuildLiveDataSection builds the top portion of the portfolio screen that
// reacts to the live toggle: header row + optional banner + summary + form + table.
func BuildLiveDataSection(resp *PortfolioResponse, metrics SummaryMetrics, liveState LiveState, currencies []string, lang string, now time.Time) components.Component {
	children := []components.Component{}

	// Header: title + live toggle
	children = append(children, buildLiveHeaderRow(liveState, lang))

	// Banner (only in live mode)
	if resp.IsLive {
		children = append(children, buildLiveBanner(resp, lang, now))
		if len(resp.Warnings) > 0 {
			children = append(children, buildLiveWarnings(resp.Warnings, lang))
		}
	}

	// Summary
	children = append(children, buildSummaryRow(metrics, lang))

	// Include-closed form
	children = append(children, buildIncludeClosedForm(lang))

	// Positions table (with dots when live)
	children = append(children, BuildPositionsTable(resp.Positions, lang, now, resp.IsLive))

	return components.ColumnWithGap("live-data-section", "lg", children...)
}

func buildLiveHeaderRow(state LiveState, lang string) components.Component {
	title := components.Text("portfolio-title", i18n.T(lang, "portfolio.title"), "lg", "bold")
	spacer := components.Column("live-header-spacer")

	toggleVariant, toggleStyle := "secondary", "ghost"
	toggleURL := "/actions/portfolio/live_data?live=true"
	if state.Live {
		toggleVariant, toggleStyle = "primary", "solid"
		toggleURL = "/actions/portfolio/live_data?live=false"
	}

	toggle := components.Component{
		Type: "button",
		ID:   "live-toggle",
		Props: map[string]any{
			"label":   i18n.T(lang, "portfolio.live.toggle"),
			"variant": toggleVariant,
			"style":   toggleStyle,
			"size":    "sm",
		},
		Actions: []components.Action{
			{Trigger: "click", Type: "reload", Endpoint: toggleURL, TargetID: "live-data-section"},
		},
	}

	return components.Row("live-header-row", []string{"auto", "1fr", "auto"}, title, spacer, toggle)
}

func buildLiveBanner(resp *PortfolioResponse, lang string, now time.Time) components.Component {
	statusText := i18n.T(lang, "portfolio.live.status")
	if resp.PricesAsOf != nil {
		statusText = strings.Replace(statusText, "{time}", FormatRelativeTime(resp.PricesAsOf, now, lang), 1)
	}
	status := components.TextStyled("live-status", statusText, "sm", "normal", "", "primary", "", "")
	spacer := components.Column("live-banner-spacer")
	refresh := components.Component{
		Type: "button",
		ID:   "live-refresh",
		Props: map[string]any{
			"label":   i18n.T(lang, "portfolio.live.refresh"),
			"variant": "secondary",
			"style":   "outline",
			"size":    "sm",
		},
		Actions: []components.Action{
			{Trigger: "click", Type: "reload", Endpoint: "/actions/portfolio/live_data?live=true&refresh=true", TargetID: "live-data-section"},
		},
	}
	return components.Row("live-banner", []string{"auto", "1fr", "auto"}, status, spacer, refresh)
}

func buildLiveWarnings(warnings []LiveWarning, lang string) components.Component {
	tickers := make([]string, 0, len(warnings))
	for _, w := range warnings {
		tickers = append(tickers, w.Ticker)
	}
	content := i18n.T(lang, "portfolio.live.warning_prefix") + " " + strings.Join(tickers, ", ")
	return components.TextStyled("live-warnings", content, "sm", "normal", "", "muted", "", "")
}
```

Note: `BuildPositionsTable` needs a new `isLive bool` parameter to conditionally emit dots. This signature change is applied in this task. Update the existing function signature in `chart_builder.go` (which also calls it) — but actually `BuildPositionsTable` is in `builder.go`. Let me check.

`BuildPositionsTable` is in `chart_builder.go` (exported, renamed from `buildTable` in layer 4b). It currently takes `(ps []Position, lang string, now time.Time)`. We need to add `isLive bool`.

Add the parameter and update the two callers:
- Inside `BuildLiveDataSection` above: `BuildPositionsTable(resp.Positions, lang, now, resp.IsLive)`
- Inside `include_closed_handler.go`: `BuildPositionsTable(positions, lang, time.Now(), false)` (include_closed is always standard mode)

In `BuildPositionsTable`, when `isLive && p.PriceSource != nil`, prepend a dot text to the Market Value cell.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/live_builder.go internal/portfolio/live_builder_test.go internal/portfolio/chart_builder.go internal/portfolio/include_closed_handler.go
git commit -m "feat(portfolio): BuildLiveDataSection with banner, toggle, dots"
```

---

### Task 5: `BuildScreen` restructured — `live-data-section` wraps data, charts/allocation outside

**Files:**
- Modify: `internal/portfolio/builder.go`
- Modify: `internal/portfolio/builder_test.go`
- Modify: `internal/portfolio/get_usecase.go`

- [ ] **Step 1: Update `BuildScreen` signature and body**

`BuildScreen` needs the full `PortfolioResponse` (for `IsLive`, `Warnings`, etc.) instead of just `[]Position`. The signature becomes:

```go
func BuildScreen(resp *PortfolioResponse, evolution []EvolutionPoint, chartPoints []EvolutionPoint, lang string, now time.Time) components.Component
```

Body: if `resp.Positions` is empty → `BuildEmpty(lang)`. Otherwise:

```go
positions := resp.Positions
SortPositions(positions)

metrics := ComputeMetrics(positions, evolution)
currencies := metrics.CurrencyOrder

liveState := LiveState{Live: resp.IsLive}
liveDataSection := BuildLiveDataSection(resp, metrics, liveState, currencies, lang, now)
chartsSection := buildInitialChartsSection(chartPoints, positions, lang)
allocationSection := buildInitialAllocationSection(positions, lang)

root := components.ColumnWithGap("portfolio-root", "lg", liveDataSection, chartsSection, allocationSection)
return components.Screen("portfolio", i18n.T(lang, "portfolio.title"), root)
```

Remove the old `buildSummaryRow`, `buildIncludeClosedForm` calls from `BuildScreen` — they're now inside `BuildLiveDataSection`.

- [ ] **Step 2: Update `Execute` in `get_usecase.go`**

Pass the full `PortfolioResponse` to `BuildScreen`:

```go
return BuildScreen(portfolioResp, evolutionLast, chartPoints, lang, now), nil
```

Where `portfolioResp` is stored from the positions fetch goroutine (now `*PortfolioResponse` instead of `[]Position`).

- [ ] **Step 3: Update all `BuildScreen` call sites in tests**

In `builder_test.go`, every `BuildScreen(samplePositions(), nil, nil, "en", time.Now())` becomes:

```go
BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
```

Update assertions: `live-data-section` is now the wrapper. `portfolio-summary-row`, `include-closed-form`, `positions-table-card` are inside `live-data-section`, not direct children of `portfolio-root`.

Add tests:

```go
func TestBuildScreen_LiveDataSectionPresentWhenPositions(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	assert.NotNil(t, findDescendantByID(s, "live-data-section"))
}

func TestBuildScreen_PortfolioRootHasThreeTopChildren(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	root := findDescendantByID(s, "portfolio-root")
	require.NotNil(t, root)
	ids := []string{}
	for _, c := range root.Children {
		ids = append(ids, c.ID)
	}
	assert.Equal(t, []string{"live-data-section", "charts-section", "allocation-section"}, ids)
}
```

- [ ] **Step 4: Run full suite**

Run: `go test ./... -count=1`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/builder.go internal/portfolio/builder_test.go internal/portfolio/get_usecase.go internal/portfolio/get_usecase_test.go
git commit -m "refactor(portfolio): BuildScreen wraps data in live-data-section"
```

---

### Task 6: Live handler — `GET /actions/portfolio/live_data`

**Files:**
- Create: `internal/portfolio/live_handler.go`
- Create: `internal/portfolio/live_handler_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/portfolio/live_handler_test.go` with tests:

1. `TestLiveHandler_ToggleOnReturnsLiveDataSection` — `?live=true` → `ActionResponse{replace, target_id: live-data-section}`. Tree type=column, id=live-data-section. Banner present.
2. `TestLiveHandler_ToggleOffReturnsStandardDataSection` — `?live=false` → same response shape but no banner.
3. `TestLiveHandler_RefreshParam` — `?live=true&refresh=true` → client receives `refresh=true`. Banner present.
4. `TestLiveHandler_DefaultsToStandard` — no params → `live=false, refresh=false`.
5. `TestLiveHandler_BackendUnauthorized401` — error → 401 redirect.
6. `TestLiveHandler_BackendError502` — error → 502.

The handler:
- Reads `live` and `refresh` query params.
- Calls `GetPositions(ctx, auth, false, live, refresh)`.
- Also calls `GetEvolutionLast(ctx, auth, 2)` for summary metrics (Snapshot Change).
- Builds `BuildLiveDataSection(resp, metrics, liveState, currencies, lang, now)`.
- Returns `ActionResponse{replace, live-data-section, tree}`.

- [ ] **Step 2: Run to verify failure**

Expected: FAIL — `NewLiveHandler` undefined.

- [ ] **Step 3: Implement `internal/portfolio/live_handler.go`**

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

// liveFetcher is the narrow interface the live handler needs.
type liveFetcher interface {
	GetPositions(ctx context.Context, authorization string, includeClosed, live, refresh bool) (*PortfolioResponse, error)
	GetEvolutionLast(ctx context.Context, authorization string, n int) ([]EvolutionPoint, error)
}

type LiveHandler struct {
	client liveFetcher
}

func NewLiveHandler(client liveFetcher) *LiveHandler {
	return &LiveHandler{client: client}
}

func (h *LiveHandler) Get(c *gin.Context) {
	live := c.Query("live") == "true"
	refresh := live && c.Query("refresh") == "true"
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)
	now := time.Now()

	resp, err := h.client.GetPositions(c.Request.Context(), auth, false, live, refresh)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load portfolio"}})
		return
	}

	// Best-effort evolution for summary (Snapshot Change).
	evo, evoErr := h.client.GetEvolutionLast(c.Request.Context(), auth, 2)
	if evoErr != nil {
		if errors.Is(evoErr, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		evo = nil // tolerate
	}

	SortPositions(resp.Positions)
	metrics := ComputeMetrics(resp.Positions, evo)
	currencies := metrics.CurrencyOrder

	liveState := LiveState{Live: live, Refresh: refresh}
	tree := BuildLiveDataSection(resp, metrics, liveState, currencies, lang, now)

	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "live-data-section",
		Tree:     &tree,
	})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/live_handler.go internal/portfolio/live_handler_test.go
git commit -m "feat(portfolio): GET /actions/portfolio/live_data handler"
```

---

### Task 7: Wire route + smoke

**Files:**
- Modify: `internal/server/server.go`

- [ ] **Step 1: Register the protected route**

Append to the portfolio block:

```go
	protected.GET("/actions/portfolio/live_data", portfolio.NewLiveHandler(portfolioClient).Get)
```

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -count=1`
Expected: all pass.

- [ ] **Step 3: Build and lint**

Run: `./cli build 2>&1 | tail -1 && ./cli lint 2>&1 | tail -1`

- [ ] **Step 4: Smoke-test**

Run and report verbatim:

```bash
cd /Users/vadimkent/repos/vk_investment_middleend_v2
lsof -ti:8082 | xargs kill -9 2>/dev/null; sleep 1
./cli run >/tmp/srv.log 2>&1 &
sleep 2

RESP=$(curl -s -X POST http://localhost:8082/actions/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@demo.com","password":"demo"}')
TOKEN=$(echo "$RESP" | python3 -c "import json,sys;print(json.load(sys.stdin)['auth']['token'])")

echo "--- portfolio-root children ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def find(x, id):
    if x.get('id') == id: return x
    for c in x.get('children', []):
        r = find(c, id)
        if r: return r
root = find(d, 'portfolio-root')
print('root children:', [c['id'] for c in root['children']])
lds = find(d, 'live-data-section')
print('live-data-section children:', [c['id'] for c in lds['children']])
"

echo "--- toggle to live ---"
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8082/actions/portfolio/live_data?live=true" \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
print('action:', d['action'], 'target_id:', d['target_id'], 'tree.id:', d['tree']['id'])
def find(x, id):
    if x.get('id') == id: return x
    for c in x.get('children', []):
        r = find(c, id)
        if r: return r
banner = find(d['tree'], 'live-banner')
print('banner present:', banner is not None)
"

echo "--- toggle back to standard ---"
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8082/actions/portfolio/live_data?live=false" \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def find(x, id):
    if x.get('id') == id: return x
    for c in x.get('children', []):
        r = find(c, id)
        if r: return r
banner = find(d['tree'], 'live-banner')
print('banner present:', banner is not None)
"

lsof -ti:8082 | xargs kill -9 2>/dev/null; true
```

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go
git commit -m "feat(server): wire protected GET /actions/portfolio/live_data"
```

---

## Self-Review Results

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| `PortfolioResponse` with `IsLive`, `PricesAsOf`, `Warnings[]` | Task 1 |
| `Position.PriceSource`, `Position.PriceAsOf` parsed | Task 1 |
| `GetPositions` returns `*PortfolioResponse` with `live`/`refresh` params | Task 2 |
| All callers updated | Task 2 |
| i18n `portfolio.live.*` keys | Task 3 |
| `live-data-section` wraps header+banner+summary+form+table | Task 4 + Task 5 |
| Live toggle button alternates URL (on→off, off→on) | Task 4 |
| Toggle standard: `secondary/ghost`; live: `primary/solid` | Task 4 |
| Banner with status text + relative time + refresh button | Task 4 |
| Refresh button URL: `?live=true&refresh=true` | Task 4 |
| Warnings text when non-empty | Task 4 |
| Price source dots: `"●"` with `positive`/`muted`/`negative` color | Task 4 |
| No dots in standard mode | Task 4 |
| `portfolio-root` = `[live-data-section, charts-section, allocation-section]` | Task 5 |
| Charts/allocation outside live-data-section | Task 5 |
| Include-closed still targets `positions-table-card` (nested) | Preserved from layer 3 |
| `GET /actions/portfolio/live_data` handler | Task 6 |
| Standard mode on fresh screen load | Task 5 (BuildScreen passes `LiveState{Live: false}`) |
| BE 401 → 401 redirect; BE 5xx → 502 | Task 6 |
| Route registered | Task 7 |

**Placeholder scan:** none. Task 4's live_builder.go test list describes exact test names and behaviors but delegates code to the subagent — this is acceptable for tests whose shapes follow the established patterns.

**Type consistency:**
- `PortfolioResponse{Positions, IsLive, PricesAsOf, Warnings}` — consistent across Task 1, 2, 4, 5, 6.
- `GetPositions(ctx, auth, includeClosed, live, refresh bool) (*PortfolioResponse, error)` — same in Task 2 (client, interfaces, all stubs/fakes).
- `LiveState{Live, Refresh bool}` — consistent in Task 4 builder and Task 6 handler.
- `BuildLiveDataSection(resp, metrics, liveState, currencies, lang, now)` — same in Task 4 and Task 5/6.
- `BuildPositionsTable(ps, lang, now, isLive)` — signature updated in Task 4, callers updated.
- `BuildScreen(resp, evolution, chartPoints, lang, now)` — signature updated in Task 5, caller (use case) updated.
