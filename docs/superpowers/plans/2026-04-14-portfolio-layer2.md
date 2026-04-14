# Portfolio Layer 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `spec/screens/portfolio/02-summary.md` — replace the single Total-Value card with a row of five stat cards (Total Value, Total P&L, Performance, Snapshot Change, Open Positions), backed by a parallel second BE call to `/v1/portfolio/evolution?last=2`.

**Architecture:** Extend the `internal/portfolio/` package. New `evolution.go` (types + parser) and `summary.go` (pure `ComputeMetrics`). `client.go` gains `GetEvolutionLast`. `get_usecase.go` fans out the two backend calls with `errgroup`, tolerating evolution failure. `builder.go`'s `buildSummary` is rewritten to emit five cards from the pre-computed metrics.

**Tech Stack:** Go, `golang.org/x/sync/errgroup`, existing `internal/components`, `internal/i18n`, testify.

---

## File Structure

**Create:**

| File | Responsibility |
|---|---|
| `internal/portfolio/evolution.go` | `EvolutionPoint` type + `ParseEvolution(body []byte)` |
| `internal/portfolio/evolution_test.go` | parser edge cases |
| `internal/portfolio/summary.go` | `SummaryMetrics` struct + `ComputeMetrics(positions, evolution)` — pure |
| `internal/portfolio/summary_test.go` | ComputeMetrics edge cases |

**Modify:**

| File | Change |
|---|---|
| `internal/portfolio/client.go` | add `GetEvolutionLast(ctx, auth, n) ([]EvolutionPoint, error)` |
| `internal/portfolio/client_test.go` | new tests for the new method |
| `internal/portfolio/get_usecase.go` | parallel fetch via `errgroup`; positions critical, evolution best-effort |
| `internal/portfolio/get_usecase_test.go` | new tests; update `fakeClient` to include evolution |
| `internal/portfolio/builder.go` | rewrite `buildSummary` to produce the five-card row from `SummaryMetrics` |
| `internal/portfolio/builder_test.go` | replace old summary tests with five-card assertions |
| `locales/en.json`, `locales/es.json` | add `portfolio.total_pnl`, `portfolio.performance`, `portfolio.snapshot_change`, `portfolio.open_positions` |

---

### Task 1: Evolution type and parser

**Files:**
- Create: `internal/portfolio/evolution.go`
- Create: `internal/portfolio/evolution_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/evolution_test.go`:

```go
package portfolio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEvolution_Basic(t *testing.T) {
	raw := []byte(`{
	  "evolution":[
	    {"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"1000.00","currency":"USD"},
	    {"snapshot_id":"s2","recorded_at":"2026-04-13T10:00:00Z","is_full_snapshot":true,"total_value":"1200.00","currency":"USD"}
	  ],
	  "total": 2
	}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 2)
	assert.Equal(t, "s1", points[0].SnapshotID)
	assert.Equal(t, time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC), points[0].RecordedAt)
	assert.True(t, points[0].IsFullSnapshot)
	assert.InDelta(t, 1000.0, points[0].TotalValue, 1e-9)
	assert.Equal(t, "USD", points[0].Currency)
	assert.InDelta(t, 1200.0, points[1].TotalValue, 1e-9)
}

func TestParseEvolution_Empty(t *testing.T) {
	raw := []byte(`{"evolution":[],"total":0}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	assert.Empty(t, points)
}

func TestParseEvolution_MultiCurrency(t *testing.T) {
	raw := []byte(`{
	  "evolution":[
	    {"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"1000.00","currency":"USD"},
	    {"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"800.00","currency":"EUR"}
	  ]
	}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 2)
	assert.Equal(t, "USD", points[0].Currency)
	assert.Equal(t, "EUR", points[1].Currency)
}

func TestParseEvolution_InvalidJSON(t *testing.T) {
	_, err := ParseEvolution([]byte(`not json`))
	require.Error(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/portfolio/... -run TestParseEvolution -v`
Expected: FAIL — `ParseEvolution` undefined.

- [ ] **Step 3: Implement**

Create `internal/portfolio/evolution.go`:

```go
package portfolio

import (
	"encoding/json"
	"strconv"
	"time"
)

// EvolutionPoint is one (snapshot, currency) value from the backend
// /v1/portfolio/evolution endpoint.
type EvolutionPoint struct {
	SnapshotID     string
	RecordedAt     time.Time
	IsFullSnapshot bool
	TotalValue     float64
	Currency       string
}

type rawEvolutionPoint struct {
	SnapshotID     string `json:"snapshot_id"`
	RecordedAt     string `json:"recorded_at"`
	IsFullSnapshot bool   `json:"is_full_snapshot"`
	TotalValue     string `json:"total_value"`
	Currency       string `json:"currency"`
}

type rawEvolutionResponse struct {
	Evolution []rawEvolutionPoint `json:"evolution"`
}

// ParseEvolution parses the backend /v1/portfolio/evolution body.
func ParseEvolution(body []byte) ([]EvolutionPoint, error) {
	var r rawEvolutionResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := make([]EvolutionPoint, 0, len(r.Evolution))
	for _, rp := range r.Evolution {
		p := EvolutionPoint{
			SnapshotID:     rp.SnapshotID,
			IsFullSnapshot: rp.IsFullSnapshot,
			Currency:       rp.Currency,
		}
		if v, err := strconv.ParseFloat(rp.TotalValue, 64); err == nil {
			p.TotalValue = v
		}
		if t, err := time.Parse(time.RFC3339, rp.RecordedAt); err == nil {
			p.RecordedAt = t
		}
		out = append(out, p)
	}
	return out, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS (4 new + all existing).

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/evolution.go internal/portfolio/evolution_test.go
git commit -m "feat(portfolio): EvolutionPoint type + parser"
```

---

### Task 2: `SummaryMetrics` + `ComputeMetrics` (pure)

**Files:**
- Create: `internal/portfolio/summary.go`
- Create: `internal/portfolio/summary_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/summary_test.go`:

```go
package portfolio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeMetrics_Empty(t *testing.T) {
	m := ComputeMetrics(nil, nil)
	assert.Equal(t, 0, m.OpenPositions)
	assert.Empty(t, m.TotalValue)
	assert.Empty(t, m.TotalPnL)
	assert.Empty(t, m.Performance)
	assert.Empty(t, m.SnapshotChange)
	assert.Empty(t, m.CurrencyOrder)
}

func TestComputeMetrics_SingleCurrency(t *testing.T) {
	tc := 1000.0
	cur := 1200.0
	pnl := 200.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", TotalCost: &tc, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 50.0},
	}
	m := ComputeMetrics(positions, nil)

	assert.Equal(t, 1, m.OpenPositions)
	assert.InDelta(t, 1200.0, m.TotalValue["USD"], 1e-9)
	assert.InDelta(t, 250.0, m.TotalPnL["USD"], 1e-9)
	require.NotNil(t, m.Performance["USD"])
	assert.InDelta(t, 20.0, *m.Performance["USD"], 1e-9)
	assert.Nil(t, m.SnapshotChange["USD"])
	assert.Equal(t, []string{"USD"}, m.CurrencyOrder)
}

func TestComputeMetrics_MultiCurrencyOrderByTotalValueDesc(t *testing.T) {
	u := 1000.0
	e := 1500.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", CurrentValue: &u},
		{Ticker: "B", Currency: "EUR", CurrentValue: &e},
	}
	m := ComputeMetrics(positions, nil)
	assert.Equal(t, []string{"EUR", "USD"}, m.CurrencyOrder)
}

func TestComputeMetrics_TotalPnLIncludesRealized(t *testing.T) {
	pnl := 100.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", UnrealizedPnL: &pnl, RealizedPnL: 25.0},
		{Ticker: "B", Currency: "USD", RealizedPnL: -10.0}, // only realized
	}
	m := ComputeMetrics(positions, nil)
	assert.InDelta(t, 115.0, m.TotalPnL["USD"], 1e-9)
}

func TestComputeMetrics_PerformanceNilWhenZeroCost(t *testing.T) {
	pnl := 100.0
	zero := 0.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", UnrealizedPnL: &pnl, TotalCost: &zero, RealizedPnL: 0},
	}
	m := ComputeMetrics(positions, nil)
	assert.Nil(t, m.Performance["USD"])
}

func TestComputeMetrics_PerformanceSkipsNullEntries(t *testing.T) {
	tc := 1000.0
	pnl := 200.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", UnrealizedPnL: &pnl, TotalCost: &tc},
		{Ticker: "B", Currency: "USD"}, // null cost and pnl — skipped
	}
	m := ComputeMetrics(positions, nil)
	require.NotNil(t, m.Performance["USD"])
	assert.InDelta(t, 20.0, *m.Performance["USD"], 1e-9)
}

func TestComputeMetrics_SnapshotChangeBasic(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
	}
	m := ComputeMetrics(positions, evo)
	require.NotNil(t, m.SnapshotChange["USD"])
	assert.InDelta(t, 10.0, *m.SnapshotChange["USD"], 1e-9)
}

func TestComputeMetrics_SnapshotChangeTakesLastTwoByDate(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC), TotalValue: 1050},
	}
	m := ComputeMetrics(positions, evo)
	require.NotNil(t, m.SnapshotChange["USD"])
	// last two by date: 1050 -> 1100 = ~4.76%
	assert.InDelta(t, 4.7619, *m.SnapshotChange["USD"], 0.01)
}

func TestComputeMetrics_SnapshotChangeNilWhenLessThanTwoPoints(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Now(), TotalValue: 1000},
	}
	m := ComputeMetrics(positions, evo)
	assert.Nil(t, m.SnapshotChange["USD"])
}

func TestComputeMetrics_SnapshotChangeNilWhenZeroBase(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 0},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 100},
	}
	m := ComputeMetrics(positions, evo)
	assert.Nil(t, m.SnapshotChange["USD"])
}

func TestComputeMetrics_SnapshotChangeOnlyForActiveCurrencies(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "EUR", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 800},
		{Currency: "EUR", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 900},
	}
	m := ComputeMetrics(positions, evo)
	// EUR is not in positions, so it is not in CurrencyOrder
	_, ok := m.SnapshotChange["EUR"]
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/portfolio/... -run TestComputeMetrics -v`
Expected: FAIL — `ComputeMetrics` undefined.

- [ ] **Step 3: Implement**

Create `internal/portfolio/summary.go`:

```go
package portfolio

import "sort"

// SummaryMetrics holds the aggregated values that feed the five summary cards.
// All per-currency maps are keyed by currency code (e.g. "USD").
type SummaryMetrics struct {
	TotalValue     map[string]float64  // sum of non-null current_value
	TotalPnL       map[string]float64  // sum of non-null unrealized_pnl + sum of realized_pnl
	Performance    map[string]*float64 // Σ unrealized / Σ total_cost × 100; nil when Σ cost == 0 or no data
	SnapshotChange map[string]*float64 // (last.total - prev.total) / prev.total × 100; nil when < 2 points or zero base
	OpenPositions  int
	CurrencyOrder  []string // currencies present in positions, sorted by TotalValue DESC
}

// ComputeMetrics computes the five stat cards' inputs from positions and
// (best-effort) evolution points. Currencies that appear only in evolution
// (not in positions) are ignored.
func ComputeMetrics(positions []Position, evolution []EvolutionPoint) SummaryMetrics {
	m := SummaryMetrics{
		TotalValue:     map[string]float64{},
		TotalPnL:       map[string]float64{},
		Performance:    map[string]*float64{},
		SnapshotChange: map[string]*float64{},
		OpenPositions:  len(positions),
	}

	sumUnrealized := map[string]float64{}
	sumTotalCost := map[string]float64{}
	hasUnrealized := map[string]bool{}
	hasTotalCost := map[string]bool{}
	currencySet := map[string]struct{}{}

	for _, p := range positions {
		if p.CurrentValue != nil {
			m.TotalValue[p.Currency] += *p.CurrentValue
			currencySet[p.Currency] = struct{}{}
		}
		if p.UnrealizedPnL != nil {
			m.TotalPnL[p.Currency] += *p.UnrealizedPnL
			sumUnrealized[p.Currency] += *p.UnrealizedPnL
			hasUnrealized[p.Currency] = true
		}
		m.TotalPnL[p.Currency] += p.RealizedPnL
		if p.TotalCost != nil {
			sumTotalCost[p.Currency] += *p.TotalCost
			hasTotalCost[p.Currency] = true
		}
		// Ensure every currency seen in positions participates in ordering,
		// even if its current_value is nil (e.g. complex assets with only cost).
		currencySet[p.Currency] = struct{}{}
	}

	for c := range currencySet {
		if hasUnrealized[c] && hasTotalCost[c] && sumTotalCost[c] != 0 {
			pct := sumUnrealized[c] / sumTotalCost[c] * 100
			m.Performance[c] = &pct
		} else {
			m.Performance[c] = nil
		}
		m.SnapshotChange[c] = snapshotChangeFor(c, evolution)
	}

	m.CurrencyOrder = currencyOrderByValueDesc(currencySet, m.TotalValue)
	return m
}

func snapshotChangeFor(currency string, evo []EvolutionPoint) *float64 {
	pts := make([]EvolutionPoint, 0, len(evo))
	for _, p := range evo {
		if p.Currency == currency {
			pts = append(pts, p)
		}
	}
	if len(pts) < 2 {
		return nil
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].RecordedAt.Before(pts[j].RecordedAt) })
	prev := pts[len(pts)-2]
	last := pts[len(pts)-1]
	if prev.TotalValue == 0 {
		return nil
	}
	pct := (last.TotalValue - prev.TotalValue) / prev.TotalValue * 100
	return &pct
}

func currencyOrderByValueDesc(set map[string]struct{}, totals map[string]float64) []string {
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		vi, vj := totals[out[i]], totals[out[j]]
		if vi == vj {
			return out[i] < out[j] // deterministic secondary order by code
		}
		return vi > vj
	})
	return out
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS (all summary tests + existing).

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/summary.go internal/portfolio/summary_test.go
git commit -m "feat(portfolio): ComputeMetrics aggregates positions + evolution"
```

---

### Task 3: Client `GetEvolutionLast`

**Files:**
- Modify: `internal/portfolio/client.go`
- Modify: `internal/portfolio/client_test.go`

- [ ] **Step 1: Append the failing tests**

Append to `internal/portfolio/client_test.go` (at the end of the file, before the closing of the package):

```go
func TestClient_GetEvolutionLast_ForwardsAuthAndQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/portfolio/evolution", r.URL.Path)
		assert.Equal(t, "2", r.URL.Query().Get("last"))
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"evolution":[{"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"1000.00","currency":"USD"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	pts, err := c.GetEvolutionLast(context.Background(), "Bearer tok", 2)
	require.NoError(t, err)
	require.Len(t, pts, 1)
	assert.Equal(t, "s1", pts[0].SnapshotID)
}

func TestClient_GetEvolutionLast_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolutionLast(context.Background(), "Bearer x", 2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_GetEvolutionLast_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolutionLast(context.Background(), "Bearer x", 2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_GetEvolutionLast_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolutionLast(context.Background(), "Bearer x", 2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/portfolio/... -run TestClient_GetEvolutionLast -v`
Expected: FAIL — `GetEvolutionLast` undefined.

- [ ] **Step 3: Add the method to the client**

Open `internal/portfolio/client.go`. Append at the end of the file:

```go
// GetEvolutionLast calls GET /v1/portfolio/evolution?last=N with the caller's
// Authorization header forwarded verbatim. Same error semantics as GetPositions.
func (c *Client) GetEvolutionLast(ctx context.Context, authorization string, n int) ([]EvolutionPoint, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/portfolio/evolution", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("last", fmt.Sprintf("%d", n))
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
		points, err := ParseEvolution(body)
		if err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return points, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/client.go internal/portfolio/client_test.go
git commit -m "feat(portfolio): client.GetEvolutionLast for /v1/portfolio/evolution?last=N"
```

---

### Task 4: Parallel use case

**Files:**
- Modify: `internal/portfolio/get_usecase.go`
- Modify: `internal/portfolio/get_usecase_test.go`

- [ ] **Step 1: Replace `internal/portfolio/get_usecase_test.go` entirely with**

```go
package portfolio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFetcher struct {
	positions []Position
	evolution []EvolutionPoint
	posErr    error
	evoErr    error
	gotAuthP  string
	gotAuthE  string
	gotLastN  int
}

func (f *fakeFetcher) GetPositions(ctx context.Context, auth string) ([]Position, error) {
	f.gotAuthP = auth
	return f.positions, f.posErr
}

func (f *fakeFetcher) GetEvolutionLast(ctx context.Context, auth string, n int) ([]EvolutionPoint, error) {
	f.gotAuthE = auth
	f.gotLastN = n
	return f.evolution, f.evoErr
}

func TestGetUseCase_FetchesBothInParallel(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		evolution: []EvolutionPoint{
			{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 100},
			{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 110},
		},
	}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "Bearer t", f.gotAuthP)
	assert.Equal(t, "Bearer t", f.gotAuthE)
	assert.Equal(t, 2, f.gotLastN)
}

func TestGetUseCase_EvolutionFailureDoesNotFail(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		evoErr:    ErrBackend,
	}
	uc := NewGetUseCase(f)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "screen", screen.Type)
}

func TestGetUseCase_PositionsFailurePropagates(t *testing.T) {
	f := &fakeFetcher{posErr: ErrUnauthorized}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestGetUseCase_EmptyPositionsReturnsEmptyScreen(t *testing.T) {
	f := &fakeFetcher{positions: []Position{}}
	uc := NewGetUseCase(f)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.NotNil(t, findDescendantByID(screen, "portfolio-empty"))
}

func TestGetUseCase_EvolutionAuthErrorTreatedAsPositionsAuthError(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		evoErr:    ErrUnauthorized,
	}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}
```

- [ ] **Step 2: Run tests to verify the expected failure**

Run: `go test ./internal/portfolio/... -run TestGetUseCase -v`
Expected: FAIL — interface mismatch (`portfolioFetcher` shape must expand).

- [ ] **Step 3: Replace `internal/portfolio/get_usecase.go` with**

```go
package portfolio

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/project/vk-investment-middleend/internal/components"
)

// portfolioFetcher is the interface the use case depends on; *Client satisfies it.
type portfolioFetcher interface {
	GetPositions(ctx context.Context, authorization string) ([]Position, error)
	GetEvolutionLast(ctx context.Context, authorization string, n int) ([]EvolutionPoint, error)
}

type GetUseCase struct {
	client portfolioFetcher
}

func NewGetUseCase(client portfolioFetcher) *GetUseCase {
	return &GetUseCase{client: client}
}

// Execute fetches positions and evolution in parallel, sorts positions, computes
// summary metrics, and builds the SDUI tree. Positions is the critical path —
// its failure aborts. Evolution failure (unless it is an auth error, which
// indicates the token is bad and must be surfaced) is tolerated and results in
// an empty evolution list.
func (uc *GetUseCase) Execute(ctx context.Context, authorization, lang string, now time.Time) (components.Component, error) {
	var positions []Position
	var evolution []EvolutionPoint

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		p, err := uc.client.GetPositions(gctx, authorization)
		if err != nil {
			return err
		}
		positions = p
		return nil
	})

	// Evolution error channel — we must distinguish auth failures (propagate) from
	// other failures (swallow).
	var evoErr error
	g.Go(func() error {
		e, err := uc.client.GetEvolutionLast(gctx, authorization, 2)
		if err != nil {
			evoErr = err
			if errors.Is(err, ErrUnauthorized) {
				return err
			}
			return nil
		}
		evolution = e
		return nil
	})

	if err := g.Wait(); err != nil {
		return components.Component{}, err
	}
	_ = evoErr // best-effort: non-auth evolution errors are silently dropped

	SortPositions(positions)
	return BuildScreen(positions, evolution, lang, now), nil
}
```

Note: the builder signature changes in Task 5 to accept `evolution`. The tests here call `BuildScreen(positions, evolution, lang, now)` indirectly via `Execute`, so they will pass once the builder accepts the new arg.

- [ ] **Step 4: Update `go.mod` with the errgroup dependency**

Run: `go get golang.org/x/sync/errgroup`
Then: `go mod tidy`

- [ ] **Step 5: Skip direct test run**

The tests still fail at this point because `BuildScreen` has the old signature — that is fixed in Task 5. We commit this task's use-case changes together with Task 5's builder changes to keep each commit green. **Do not commit yet.** Proceed to Task 5.

---

### Task 5: i18n keys + builder rewrite (five cards)

**Files:**
- Modify: `locales/en.json`, `locales/es.json`
- Modify: `internal/portfolio/builder.go`
- Modify: `internal/portfolio/builder_test.go`

- [ ] **Step 1: Add the four new keys to `locales/en.json`**

Open `locales/en.json`. Find the `"portfolio"` object. Replace the `"total_value"` line with the six-line block below:

```json
    "total_value": "Total Value",
    "total_pnl": "Total P&L",
    "performance": "Total Performance",
    "snapshot_change": "Snapshot Change",
    "open_positions": "Open Positions",
```

- [ ] **Step 2: Add the four new keys to `locales/es.json`**

Open `locales/es.json`. Find the `"portfolio"` object. Replace the `"total_value"` line with:

```json
    "total_value": "Valor total",
    "total_pnl": "G/P total",
    "performance": "Rendimiento total",
    "snapshot_change": "Cambio último snapshot",
    "open_positions": "Posiciones abiertas",
```

- [ ] **Step 3: Replace `internal/portfolio/builder_test.go` entirely with**

```go
package portfolio

import (
	"testing"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEmpty_HasEmptyBlock(t *testing.T) {
	s := BuildEmpty("en")
	assert.Equal(t, "screen", s.Type)
	assert.Equal(t, "portfolio", s.ID)

	empty := findDescendantByID(s, "portfolio-empty")
	require.NotNil(t, empty)
	title := findDescendantByID(*empty, "empty-title")
	require.NotNil(t, title)
	assert.Equal(t, "No positions yet", title.Props["content"])
}

func TestBuildEmpty_NoSummaryCards(t *testing.T) {
	s := BuildEmpty("en")
	for _, id := range []string{"summary-card-total-value", "summary-card-total-pnl", "summary-card-performance", "summary-card-snapshot-change", "summary-card-open-positions"} {
		assert.Nil(t, findDescendantByID(s, id), "unexpected %s in empty tree", id)
	}
}

func TestBuildScreen_SummaryRowHasFiveCardsInOrder(t *testing.T) {
	s := BuildScreen(samplePositions(), nil, "en", time.Now())
	row := findDescendantByID(s, "portfolio-summary-row")
	require.NotNil(t, row)

	widths, ok := row.Props["widths"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"1fr", "1fr", "1fr", "1fr", "1fr"}, widths)

	want := []string{
		"summary-card-total-value",
		"summary-card-total-pnl",
		"summary-card-performance",
		"summary-card-snapshot-change",
		"summary-card-open-positions",
	}
	require.Len(t, row.Children, 5)
	for i, id := range want {
		assert.Equal(t, "card", row.Children[i].Type, "child %d type", i)
		assert.Equal(t, id, row.Children[i].ID, "child %d id", i)
	}
}

func TestBuildScreen_SummaryCardLabelsLocalized(t *testing.T) {
	s := BuildScreen(samplePositions(), nil, "es", time.Now())
	cases := map[string]string{
		"summary-label-total-value":      "Valor total",
		"summary-label-total-pnl":        "G/P total",
		"summary-label-performance":      "Rendimiento total",
		"summary-label-snapshot-change":  "Cambio último snapshot",
		"summary-label-open-positions":   "Posiciones abiertas",
	}
	for id, want := range cases {
		node := findDescendantByID(s, id)
		require.NotNil(t, node, "missing %s", id)
		assert.Equal(t, want, node.Props["content"])
		assert.Equal(t, "muted", node.Props["color"])
	}
}

func TestBuildScreen_TotalValueOneLinePerCurrencyDesc(t *testing.T) {
	u := 500.0
	e := 1500.0
	ps := []Position{
		{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &u},
		{AssetID: "b1", Ticker: "B", Currency: "EUR", CurrentValue: &e},
	}
	s := BuildScreen(ps, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-total-value")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 2)
	// EUR has higher total value → appears first
	assert.Equal(t, "summary-value-total-value-EUR", vals.Children[0].ID)
	assert.Equal(t, "€1,500.00", vals.Children[0].Props["content"])
	assert.Equal(t, "summary-value-total-value-USD", vals.Children[1].ID)
	assert.Equal(t, "$500.00", vals.Children[1].Props["content"])
}

func TestBuildScreen_TotalValueEmptyShowsDash(t *testing.T) {
	// Positions with no current_value at all
	ps := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD"}}
	s := BuildScreen(ps, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-total-value")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "summary-value-total-value-empty", vals.Children[0].ID)
	assert.Equal(t, "—", vals.Children[0].Props["content"])
}

func TestBuildScreen_TotalPnLSignedAndColored(t *testing.T) {
	tc := 1000.0
	cur := 1200.0
	pnl := 200.0
	positives := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD",
		TotalCost: &tc, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 50.0}}
	s := BuildScreen(positives, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-total-pnl")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "+$250.00", vals.Children[0].Props["content"])
	assert.Equal(t, "positive", vals.Children[0].Props["color"])
}

func TestBuildScreen_PerformanceFallsBackToDash(t *testing.T) {
	u := 100.0
	ps := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &u}}
	s := BuildScreen(ps, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-performance")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "—", vals.Children[0].Props["content"])
	_, hasColor := vals.Children[0].Props["color"]
	assert.False(t, hasColor)
}

func TestBuildScreen_SnapshotChangeBasic(t *testing.T) {
	u := 1000.0
	ps := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
	}
	s := BuildScreen(ps, evo, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-snapshot-change")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "+10.00%", vals.Children[0].Props["content"])
	assert.Equal(t, "positive", vals.Children[0].Props["color"])
}

func TestBuildScreen_SnapshotChangeDashWhenNoEvolution(t *testing.T) {
	u := 1000.0
	ps := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &u}}
	s := BuildScreen(ps, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-snapshot-change")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "—", vals.Children[0].Props["content"])
}

func TestBuildScreen_OpenPositionsCount(t *testing.T) {
	u1, u2, u3 := 1.0, 2.0, 3.0
	ps := []Position{
		{AssetID: "a", Ticker: "A", Currency: "USD", CurrentValue: &u1},
		{AssetID: "b", Ticker: "B", Currency: "USD", CurrentValue: &u2},
		{AssetID: "c", Ticker: "C", Currency: "USD", CurrentValue: &u3},
	}
	s := BuildScreen(ps, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-open-positions")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "summary-value-open-positions", vals.Children[0].ID)
	assert.Equal(t, "3", vals.Children[0].Props["content"])
}

func TestBuildScreen_PositionsTablePreservedFromLayer1(t *testing.T) {
	s := BuildScreen(samplePositions(), nil, "en", time.Now())
	assert.NotNil(t, findDescendantByID(s, "positions-table-card"))
	assert.NotNil(t, findDescendantByID(s, "positions-header"))
	assert.NotNil(t, findDescendantByID(s, "positions-body"))
}

// ---- layer 1 row / column tests still needed ----

func TestBuildScreen_HeaderHas11Columns(t *testing.T) {
	s := BuildScreen(samplePositions(), nil, "en", time.Now())
	header := findDescendantByID(s, "positions-header")
	require.NotNil(t, header)
	widths, ok := header.Props["widths"].([]string)
	require.True(t, ok)
	assert.Len(t, widths, 11)
}

func TestBuildScreen_BodyUsesListWithOneItemPerPosition(t *testing.T) {
	ps := samplePositions()
	s := BuildScreen(ps, nil, "en", time.Now())
	body := findDescendantByID(s, "positions-body")
	require.NotNil(t, body)
	assert.Equal(t, "list", body.Type)
	assert.Len(t, body.Children, len(ps))
}

func TestBuildScreen_PositionRowValuesInOrder(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	qty, avg, total, cur, pnl, realized := 10.0, 153.33, 1533.33, 1855.0, 321.67, 175.0
	snap := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	ps := []Position{{
		AssetID: "a1", Ticker: "AAPL", Name: "Apple Inc", AssetType: "STOCK", Currency: "USD",
		Quantity: &qty, AvgCost: &avg, TotalCost: &total, CurrentValue: &cur,
		UnrealizedPnL: &pnl, RealizedPnL: realized, LastSnapshotAt: &snap,
	}}

	s := BuildScreen(ps, nil, "en", now)
	item := findDescendantByID(s, "position-a1")
	require.NotNil(t, item)
	row := findDescendantByType(*item, "row")
	require.NotNil(t, row)
	require.Len(t, row.Children, 11)

	want := []string{"AAPL", "Apple Inc", "STOCK", "10", "$153.33", "$1,533.33", "$1,855.00", "+$321.67", "+20.98%", "+$175.00", "2 days ago"}
	for i, w := range want {
		assert.Equal(t, w, row.Children[i].Props["content"], "col %d", i)
	}
}

// helpers

func samplePositions() []Position {
	qty, avg, total, cur, pnl := 10.0, 100.0, 1000.0, 1200.0, 200.0
	return []Position{
		{AssetID: "s1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD",
			Quantity: &qty, AvgCost: &avg, TotalCost: &total, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 0},
	}
}

func findDescendantByType(c components.Component, typ string) *components.Component {
	if c.Type == typ {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByType(c.Children[i], typ); found != nil {
			return found
		}
	}
	return nil
}

func findDescendantByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByID(c.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify failure**

Run: `go test ./internal/portfolio/... -v`
Expected: FAIL — `BuildScreen` still has old signature `(positions, lang, now)`; the test calls `BuildScreen(positions, evolution, lang, now)`.

- [ ] **Step 5: Rewrite `internal/portfolio/builder.go`**

Replace the full contents of `internal/portfolio/builder.go` with:

```go
package portfolio

import (
	"strconv"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

var columnWidths = []string{
	"80px",  // ticker
	"1fr",   // name
	"80px",  // type
	"80px",  // quantity
	"110px", // avg cost
	"110px", // total cost
	"120px", // market value
	"130px", // unrealized pnl
	"80px",  // % pnl
	"120px", // realized pnl
	"120px", // last snapshot
}

var columnKeys = []string{
	"portfolio.col.ticker",
	"portfolio.col.name",
	"portfolio.col.type",
	"portfolio.col.quantity",
	"portfolio.col.avg_cost",
	"portfolio.col.total_cost",
	"portfolio.col.market_value",
	"portfolio.col.unrealized_pnl",
	"portfolio.col.pnl_pct",
	"portfolio.col.realized_pnl",
	"portfolio.col.last_snapshot",
}

// BuildScreen builds the portfolio tree for the given positions and evolution
// points. now is used to format relative times.
func BuildScreen(positions []Position, evolution []EvolutionPoint, lang string, now time.Time) components.Component {
	if len(positions) == 0 {
		return BuildEmpty(lang)
	}

	metrics := ComputeMetrics(positions, evolution)
	summary := buildSummaryRow(metrics, lang)
	table := buildTable(positions, lang, now)

	root := components.ColumnWithGap("portfolio-root", "lg", summary, table)
	return components.Screen("portfolio", i18n.T(lang, "portfolio.title"), root)
}

// BuildEmpty builds the screen for an empty portfolio.
func BuildEmpty(lang string) components.Component {
	title := components.Text("empty-title", i18n.T(lang, "portfolio.empty_title"), "lg", "bold")
	subtitle := components.TextStyled("empty-subtitle", i18n.T(lang, "portfolio.empty_subtitle"), "md", "normal", "", "muted", "", "")
	empty := components.ColumnWithGap("portfolio-empty", "sm", title, subtitle)
	root := components.ColumnWithGap("portfolio-root", "lg", empty)
	return components.Screen("portfolio", i18n.T(lang, "portfolio.title"), root)
}

func buildSummaryRow(m SummaryMetrics, lang string) components.Component {
	cards := []components.Component{
		buildTotalValueCard(m, lang),
		buildTotalPnLCard(m, lang),
		buildPerformanceCard(m, lang),
		buildSnapshotChangeCard(m, lang),
		buildOpenPositionsCard(m, lang),
	}
	return components.Row("portfolio-summary-row", []string{"1fr", "1fr", "1fr", "1fr", "1fr"}, cards...)
}

func buildTotalValueCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-total-value")
	if len(m.CurrencyOrder) == 0 || !anyHasValue(m.TotalValue, m.CurrencyOrder) {
		values.Children = append(values.Children, components.Text("summary-value-total-value-empty", "—", "xl", "bold"))
	} else {
		for _, c := range m.CurrencyOrder {
			v, ok := m.TotalValue[c]
			if !ok {
				continue
			}
			values.Children = append(values.Children,
				components.Text("summary-value-total-value-"+c, FormatMoney(&v, c, lang), "xl", "bold"))
		}
	}
	return wrapCard("total-value", "portfolio.total_value", lang, values)
}

func buildTotalPnLCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-total-pnl")
	if len(m.CurrencyOrder) == 0 {
		values.Children = append(values.Children, components.Text("summary-value-total-pnl-empty", "—", "xl", "bold"))
	} else {
		for _, c := range m.CurrencyOrder {
			v := m.TotalPnL[c]
			values.Children = append(values.Children,
				coloredValue("summary-value-total-pnl-"+c, FormatSignedMoney(&v, c, lang), pnlColor(&v)))
		}
	}
	return wrapCard("total-pnl", "portfolio.total_pnl", lang, values)
}

func buildPerformanceCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-performance")
	if len(m.CurrencyOrder) == 0 {
		values.Children = append(values.Children, components.Text("summary-value-performance-empty", "—", "xl", "bold"))
	} else {
		for _, c := range m.CurrencyOrder {
			pct := m.Performance[c]
			values.Children = append(values.Children,
				coloredValue("summary-value-performance-"+c, FormatSignedPercent(pct, lang), pnlColor(pct)))
		}
	}
	return wrapCard("performance", "portfolio.performance", lang, values)
}

func buildSnapshotChangeCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-snapshot-change")
	if len(m.CurrencyOrder) == 0 {
		values.Children = append(values.Children, components.Text("summary-value-snapshot-change-empty", "—", "xl", "bold"))
	} else {
		for _, c := range m.CurrencyOrder {
			pct := m.SnapshotChange[c]
			values.Children = append(values.Children,
				coloredValue("summary-value-snapshot-change-"+c, FormatSignedPercent(pct, lang), pnlColor(pct)))
		}
	}
	return wrapCard("snapshot-change", "portfolio.snapshot_change", lang, values)
}

func buildOpenPositionsCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-open-positions",
		components.Text("summary-value-open-positions", strconv.Itoa(m.OpenPositions), "xl", "bold"),
	)
	return wrapCard("open-positions", "portfolio.open_positions", lang, values)
}

func wrapCard(id, labelKey, lang string, valuesCol components.Component) components.Component {
	label := components.TextStyled("summary-label-"+id, i18n.T(lang, labelKey), "sm", "normal", "", "muted", "", "")
	content := components.ColumnWithGap("summary-card-content-"+id, "sm", label, valuesCol)
	return components.Card("summary-card-"+id, content)
}

func anyHasValue(byCurrency map[string]float64, order []string) bool {
	for _, c := range order {
		if _, ok := byCurrency[c]; ok {
			return true
		}
	}
	return false
}

func coloredValue(id, content, color string) components.Component {
	if color == "" {
		return components.Text(id, content, "xl", "bold")
	}
	return components.TextStyled(id, content, "xl", "bold", "", color, "", "")
}

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

func columnShortID(i int) string {
	names := []string{"ticker", "name", "type", "quantity", "avg-cost", "total-cost", "market-value", "unrealized-pnl", "pnl-pct", "realized-pnl", "last-snapshot"}
	return names[i]
}

func buildPositionItem(p Position, lang string, now time.Time) components.Component {
	realized := p.RealizedPnL
	pct := PnLPct(p.UnrealizedPnL, p.TotalCost)

	cells := []components.Component{
		components.Text("cell-ticker", p.Ticker, "sm", "bold"),
		components.Text("cell-name", p.Name, "sm", "normal"),
		components.Text("cell-type", p.AssetType, "sm", "normal"),
		components.Text("cell-quantity", FormatQuantity(p.Quantity, lang), "sm", "normal"),
		components.Text("cell-avg-cost", FormatMoney(p.AvgCost, p.Currency, lang), "sm", "normal"),
		components.Text("cell-total-cost", FormatMoney(p.TotalCost, p.Currency, lang), "sm", "normal"),
		components.Text("cell-market-value", FormatMoney(p.CurrentValue, p.Currency, lang), "sm", "normal"),
		coloredCell("cell-unrealized-pnl", FormatSignedMoney(p.UnrealizedPnL, p.Currency, lang), pnlColor(p.UnrealizedPnL)),
		coloredCell("cell-pnl-pct", FormatSignedPercent(pct, lang), pnlColor(pct)),
		coloredCell("cell-realized-pnl", FormatSignedMoney(&realized, p.Currency, lang), pnlColor(&realized)),
		components.Text("cell-last-snapshot", FormatRelativeTime(p.LastSnapshotAt, now, lang), "sm", "normal"),
	}
	row := components.Row("position-"+p.AssetID+"-row", columnWidths, cells...)
	return components.ListItem("position-"+p.AssetID, row)
}

// pnlColor returns "positive", "negative" or "" (no color) based on v.
func pnlColor(v *float64) string {
	if v == nil || *v == 0 {
		return ""
	}
	if *v > 0 {
		return "positive"
	}
	return "negative"
}

func coloredCell(id, content, color string) components.Component {
	if color == "" {
		return components.Text(id, content, "sm", "normal")
	}
	return components.TextStyled(id, content, "sm", "normal", "", color, "", "")
}
```

Note: `sort` import is no longer needed (removed from builder.go — sort moved into `summary.go`).

- [ ] **Step 6: Run full suite**

Run: `go test ./... -count=1`
Expected: all tests PASS (summary + evolution + client + usecase + builder + existing).

- [ ] **Step 7: Build and lint**

Run: `./cli build 2>&1 | tail -1 && ./cli lint 2>&1 | tail -1`
Expected: both `"status":"success"`.

- [ ] **Step 8: Smoke-test end-to-end**

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

echo "--- /screens/portfolio with token — summary-row+5 cards expected ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def walk(n, depth=0):
    ids = []
    def rec(x):
        if x.get('id','').startswith('summary-card-'):
            ids.append(x['id'])
        for c in x.get('children', []):
            rec(c)
    rec(n)
    return ids
print(walk(d))
"

lsof -ti:8082 | xargs kill -9 2>/dev/null; true
```

Expected output: `['summary-card-total-value', 'summary-card-total-pnl', 'summary-card-performance', 'summary-card-snapshot-change', 'summary-card-open-positions']`.

- [ ] **Step 9: Commit Task 4 + Task 5 together**

```bash
git add internal/portfolio/get_usecase.go internal/portfolio/get_usecase_test.go internal/portfolio/builder.go internal/portfolio/builder_test.go locales/en.json locales/es.json go.mod go.sum
git commit -m "feat(portfolio): five-card summary with parallel evolution fetch"
```

The use case and builder land in a single commit because the builder signature change and the use case rewrite are mutually dependent (green at HEAD only when both land).

---

## Self-Review Results

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| `row#portfolio-summary-row` with 5 direct `card` children in order | Task 5 `TestBuildScreen_SummaryRowHasFiveCardsInOrder` |
| Card internal structure (label muted + values column) | Task 5 `TestBuildScreen_SummaryCardLabelsLocalized` + per-card structural tests |
| Total Value per-currency DESC ordering + "—" fallback | Task 5 `TestBuildScreen_TotalValueOneLinePerCurrencyDesc`, `TestBuildScreen_TotalValueEmptyShowsDash` |
| Total P&L signed money + positive/negative color | Task 5 `TestBuildScreen_TotalPnLSignedAndColored` + Task 2 `TestComputeMetrics_TotalPnLIncludesRealized` |
| Performance "—" when Σ cost == 0 or no data | Task 5 `TestBuildScreen_PerformanceFallsBackToDash`; Task 2 `TestComputeMetrics_PerformanceNilWhenZeroCost` / `_PerformanceSkipsNullEntries` |
| Snapshot Change — last two points by date, nil on < 2 or zero base | Task 2 `TestComputeMetrics_SnapshotChange*`; Task 5 `TestBuildScreen_SnapshotChangeBasic` / `_DashWhenNoEvolution` |
| Open Positions = `len(positions)` | Task 5 `TestBuildScreen_OpenPositionsCount`; Task 2 `TestComputeMetrics_SingleCurrency` |
| Parallel fetch of positions + evolution | Task 4 `TestGetUseCase_FetchesBothInParallel` |
| Evolution failure tolerated (non-auth) | Task 4 `TestGetUseCase_EvolutionFailureDoesNotFail` |
| Evolution 401 propagates | Task 4 `TestGetUseCase_EvolutionAuthErrorTreatedAsPositionsAuthError` |
| Positions failure propagates | Task 4 `TestGetUseCase_PositionsFailurePropagates` |
| Empty positions → no summary cards; empty block preserved | Task 4 `TestGetUseCase_EmptyPositionsReturnsEmptyScreen`, Task 5 `TestBuildEmpty_NoSummaryCards` |
| Evolution-only currencies skipped | Task 2 `TestComputeMetrics_SnapshotChangeOnlyForActiveCurrencies` |
| Currency order shared across cards | Task 2 `TestComputeMetrics_MultiCurrencyOrderByTotalValueDesc`; Task 5 card-order tests read the same `CurrencyOrder` |
| Positions table preserved from layer 1 | Task 5 `TestBuildScreen_PositionsTablePreservedFromLayer1` + retained header/body/row tests |
| i18n keys present in both locales | Task 5 Steps 1–2 |
| Backend 401 → middleend 401 with redirect; backend 5xx → 502 | Inherited from layer 1's existing handler tests (unchanged) |

**Placeholder scan:** none.

**Type consistency:**
- `SummaryMetrics.CurrencyOrder` is `[]string`; `TotalValue`, `TotalPnL` are `map[string]float64`; `Performance` and `SnapshotChange` are `map[string]*float64`. Consistent across Task 2 and Task 5.
- `ComputeMetrics(positions []Position, evolution []EvolutionPoint) SummaryMetrics` same signature in Task 2 and Task 5 use.
- `BuildScreen(positions, evolution, lang, now)` — new signature matches across Task 4 use case, Task 5 tests, and Task 5 builder.
- `portfolioFetcher` interface with two methods matches `*Client` (existing `GetPositions` + new `GetEvolutionLast`).
- Card IDs (`summary-card-total-value`, etc.) consistent between spec, tests, and builder.
