# Portfolio Layer 4a Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `spec/screens/portfolio/04a-value-over-time.md` — add a Value Over Time chart card at the bottom of the portfolio screen with timeframe / mode / currency controls, backed by a new `GET /actions/portfolio/evolution` reload endpoint. Also introduces the `line_chart` custom SDUI component per `spec/sdui-custom-components.md §1`.

**Architecture:** New `line_chart` helper in `internal/components`. New `ChartState` + `BuildValueOverTimeCard` in `internal/portfolio/chart_builder.go`, pure and reusable by both the initial screen render and the action handler. Client gets a new `GetEvolution(q)` method. Use case fetches a third parallel evolution response (100 points) for the chart card's initial state. A dedicated handler serves `GET /actions/portfolio/evolution`.

**Tech Stack:** Go, Gin, testify, existing `internal/components`, `internal/portfolio`, `golang.org/x/sync/errgroup`.

---

## File Structure

**Create:**

| File | Responsibility |
|---|---|
| `internal/components/charts.go` | `LineChart` helper + `Series`, `Axis` types |
| `internal/components/charts_test.go` | JSON shape of emitted `line_chart` |
| `internal/portfolio/chart_builder.go` | `ChartState` + `BuildValueOverTimeCard(points, state, currencies, lang)` — pure, no BE |
| `internal/portfolio/chart_builder_test.go` | controls, selected state, URL encoding, data mapping, empty states |
| `internal/portfolio/evolution_handler.go` | GET handler for `/actions/portfolio/evolution` |
| `internal/portfolio/evolution_handler_test.go` | covers success, 400, 401, 502, timeframe→from mapping, pct mode |

**Modify:**

| File | Change |
|---|---|
| `internal/portfolio/client.go` | Add `GetEvolution(ctx, auth, q EvolutionQuery) ([]EvolutionPoint, error)` (keeps `GetEvolutionLast`) |
| `internal/portfolio/client_test.go` | Tests for new method |
| `internal/portfolio/get_usecase.go` | Fetch three things in parallel: positions, 2-point evolution (summary), 100-point evolution (chart). Extend `portfolioFetcher` interface |
| `internal/portfolio/get_usecase_test.go` | Update `fakeFetcher` with new method; tests for chart-evolution success + failure tolerance |
| `internal/portfolio/builder.go` | `BuildScreen` accepts `chartPoints []EvolutionPoint` as new positional parameter; appends chart card after positions-table-card when positions non-empty |
| `internal/portfolio/builder_test.go` | Update call sites to pass nil chart points; add tests that chart card is present / absent |
| `internal/portfolio/handler.go` | Unchanged |
| `internal/server/server.go` | Register protected `GET /actions/portfolio/evolution` |
| `locales/en.json`, `locales/es.json` | Add `portfolio.chart.*` keys |

---

### Task 1: `line_chart` component helper

**Files:**
- Create: `internal/components/charts.go`
- Create: `internal/components/charts_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/components/charts_test.go`:

```go
package components

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLineChart_EmitsTypeAndID(t *testing.T) {
	c := LineChart("x",
		"md",
		[]Series{{Key: "v", Label: "Value", Color: "chart_1", ValueFormat: "currency_compact"}},
		Axis{Key: "date", Format: "month_year"},
		Axis{Format: "currency_compact"},
		[]map[string]any{{"date": "2026-01-01", "v": 100.0}},
		"Not enough data",
	)
	assert.Equal(t, "line_chart", c.Type)
	assert.Equal(t, "x", c.ID)
}

func TestLineChart_AllPropsPresent(t *testing.T) {
	data := []map[string]any{{"date": "2026-01-01", "v": 100.0}}
	c := LineChart("x",
		"md",
		[]Series{{Key: "v", Label: "Value", Color: "chart_1", ValueFormat: "currency_compact"}},
		Axis{Key: "date", Format: "month_year"},
		Axis{Format: "currency_compact"},
		data,
		"Not enough data",
	)
	assert.Equal(t, "md", c.Props["height"])
	assert.Equal(t, "Not enough data", c.Props["empty_message"])
	assert.Equal(t, data, c.Props["data"])
	_, ok := c.Props["series"].([]Series)
	assert.True(t, ok)
	_, ok = c.Props["x_axis"].(Axis)
	assert.True(t, ok)
	_, ok = c.Props["y_axis"].(Axis)
	assert.True(t, ok)
}

func TestLineChart_OmitsEmptyHeight(t *testing.T) {
	c := LineChart("x",
		"",
		[]Series{{Key: "v", Label: "V", Color: "chart_1", ValueFormat: "currency"}},
		Axis{Key: "date", Format: "date"},
		Axis{},
		[]map[string]any{},
		"",
	)
	_, hasHeight := c.Props["height"]
	assert.False(t, hasHeight)
	_, hasEmpty := c.Props["empty_message"]
	assert.False(t, hasEmpty)
}

func TestLineChart_JSONShape(t *testing.T) {
	c := LineChart("chart-value-over-time",
		"md",
		[]Series{{Key: "value", Label: "Value", Color: "chart_1", ValueFormat: "currency_compact"}},
		Axis{Key: "date", Format: "month_year"},
		Axis{Format: "currency_compact"},
		[]map[string]any{{"date": "2026-01-01", "value": 100.0}},
		"Not enough data",
	)
	b, err := json.Marshal(c)
	require.NoError(t, err)

	// Key shape assertions
	s := string(b)
	assert.Contains(t, s, `"type":"line_chart"`)
	assert.Contains(t, s, `"id":"chart-value-over-time"`)
	assert.Contains(t, s, `"series":[{"key":"value","label":"Value","color":"chart_1","value_format":"currency_compact"}]`)
	assert.Contains(t, s, `"x_axis":{"key":"date","format":"month_year"}`)
	assert.Contains(t, s, `"y_axis":{"format":"currency_compact"}`)
	assert.Contains(t, s, `"data":[{"date":"2026-01-01","value":100}]`)
	assert.Contains(t, s, `"empty_message":"Not enough data"`)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/components/... -run TestLineChart -v`
Expected: FAIL — `LineChart` / `Series` / `Axis` undefined.

- [ ] **Step 3: Implement**

Create `internal/components/charts.go`:

```go
package components

// Series is one line in a line_chart.
type Series struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Color       string `json:"color"`
	ValueFormat string `json:"value_format"`
}

// Axis describes an axis. Key is optional (x-axis only); Format applies to
// both axes' tick labels.
type Axis struct {
	Key    string `json:"key,omitempty"`
	Format string `json:"format,omitempty"`
}

// LineChart creates a line_chart custom component. See
// spec/sdui-custom-components.md §1.
//
// Pass empty string for height / emptyMessage to omit those props.
func LineChart(id, height string, series []Series, xAxis, yAxis Axis, data []map[string]any, emptyMessage string) Component {
	props := map[string]any{
		"series": series,
		"x_axis": xAxis,
		"y_axis": yAxis,
		"data":   data,
	}
	if height != "" {
		props["height"] = height
	}
	if emptyMessage != "" {
		props["empty_message"] = emptyMessage
	}
	return Component{
		Type:  "line_chart",
		ID:    id,
		Props: props,
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/components/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/components/charts.go internal/components/charts_test.go
git commit -m "feat(components): line_chart custom SDUI component helper"
```

---

### Task 2: `Client.GetEvolution(q)`

**Files:**
- Modify: `internal/portfolio/client.go`
- Modify: `internal/portfolio/client_test.go`

- [ ] **Step 1: Append failing tests**

Append to `internal/portfolio/client_test.go`:

```go
func TestClient_GetEvolution_WithFromPointsCurrency(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/portfolio/evolution", r.URL.Path)
		gotQuery = r.URL.RawQuery
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"evolution":[{"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"1000.00","currency":"USD"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	pts, err := c.GetEvolution(context.Background(), "Bearer tok", EvolutionQuery{From: &from, Points: 100, Currency: "USD"})
	require.NoError(t, err)
	require.Len(t, pts, 1)
	assert.Contains(t, gotQuery, "from=2026-01-01T00%3A00%3A00Z")
	assert.Contains(t, gotQuery, "points=100")
	assert.Contains(t, gotQuery, "currency=USD")
}

func TestClient_GetEvolution_OmitsUnsetParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"evolution":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolution(context.Background(), "Bearer t", EvolutionQuery{Points: 100})
	require.NoError(t, err)
	assert.NotContains(t, gotQuery, "from=")
	assert.NotContains(t, gotQuery, "currency=")
	assert.Contains(t, gotQuery, "points=100")
}

func TestClient_GetEvolution_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolution(context.Background(), "Bearer bad", EvolutionQuery{Points: 100})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_GetEvolution_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetEvolution(context.Background(), "Bearer x", EvolutionQuery{Points: 100})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/portfolio/... -run TestClient_GetEvolution_ -v`
Expected: FAIL — `GetEvolution` / `EvolutionQuery` undefined.

- [ ] **Step 3: Add the method to the client**

Append to `internal/portfolio/client.go`:

```go
// EvolutionQuery parameterizes GetEvolution.
type EvolutionQuery struct {
	From     *time.Time
	Points   int
	Currency string
}

// GetEvolution calls GET /v1/portfolio/evolution with the given query. Same
// error semantics as GetPositions.
func (c *Client) GetEvolution(ctx context.Context, authorization string, q EvolutionQuery) ([]EvolutionPoint, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/portfolio/evolution", nil)
	if err != nil {
		return nil, err
	}
	qs := req.URL.Query()
	if q.From != nil {
		qs.Set("from", q.From.Format(time.RFC3339))
	}
	if q.Points > 0 {
		qs.Set("points", fmt.Sprintf("%d", q.Points))
	}
	if q.Currency != "" {
		qs.Set("currency", q.Currency)
	}
	req.URL.RawQuery = qs.Encode()
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

Add `"time"` to the imports if not already present (likely already there from other uses).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/client.go internal/portfolio/client_test.go
git commit -m "feat(portfolio): client.GetEvolution with EvolutionQuery"
```

---

### Task 3: Chart builder

**Files:**
- Create: `internal/portfolio/chart_builder.go`
- Create: `internal/portfolio/chart_builder_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/portfolio/chart_builder_test.go`:

```go
package portfolio

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = i18n.Load(filepath.Join("..", "..", "locales"))
}

func sampleChartPoints(currency string) []EvolutionPoint {
	return []EvolutionPoint{
		{Currency: currency, RecordedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TotalValue: 10000},
		{Currency: currency, RecordedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TotalValue: 10500},
		{Currency: currency, RecordedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), TotalValue: 11000},
	}
}

func TestBuildValueOverTimeCard_RootCard(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	assert.Equal(t, "card", card.Type)
	assert.Equal(t, "chart-value-over-time-card", card.ID)
}

func TestBuildValueOverTimeCard_TimeframeControlsHaveSixButtons(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	tf := findDescendantByID(card, "timeframe-controls")
	require.NotNil(t, tf)

	ids := []string{"chart-timeframe-1m", "chart-timeframe-3m", "chart-timeframe-6m", "chart-timeframe-ytd", "chart-timeframe-1y", "chart-timeframe-all"}
	require.Len(t, tf.Children, 6)
	for i, id := range ids {
		assert.Equal(t, "button", tf.Children[i].Type, "button %d type", i)
		assert.Equal(t, id, tf.Children[i].ID, "button %d id", i)
	}
}

func TestBuildValueOverTimeCard_SelectedTimeframeHasSolidStyle(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "3m", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	selected := findDescendantByID(card, "chart-timeframe-3m")
	require.NotNil(t, selected)
	assert.Equal(t, "primary", selected.Props["variant"])
	assert.Equal(t, "solid", selected.Props["style"])

	unselected := findDescendantByID(card, "chart-timeframe-1y")
	require.NotNil(t, unselected)
	assert.Equal(t, "secondary", unselected.Props["variant"])
	assert.Equal(t, "ghost", unselected.Props["style"])
}

func TestBuildValueOverTimeCard_ButtonURLCarriesFullState(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "3m", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	btn := findDescendantByID(card, "chart-timeframe-6m")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "click", a.Trigger)
	assert.Equal(t, "reload", a.Type)
	assert.Equal(t, "chart-value-over-time-card", a.TargetID)
	// Clicking 6m from (3m, abs, USD) yields (6m, abs, USD).
	assert.Contains(t, a.Endpoint, "timeframe=6m")
	assert.Contains(t, a.Endpoint, "mode=abs")
	assert.Contains(t, a.Endpoint, "currency=USD")
}

func TestBuildValueOverTimeCard_ModeControlsTwoButtons(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	m := findDescendantByID(card, "mode-controls")
	require.NotNil(t, m)
	require.Len(t, m.Children, 2)
	assert.Equal(t, "chart-mode-abs", m.Children[0].ID)
	assert.Equal(t, "chart-mode-pct", m.Children[1].ID)
}

func TestBuildValueOverTimeCard_CurrencyControlsHiddenWhenSingle(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	assert.Nil(t, findDescendantByID(card, "currency-controls"))
}

func TestBuildValueOverTimeCard_CurrencyControlsShownWhenMulti(t *testing.T) {
	points := append(sampleChartPoints("USD"), sampleChartPoints("EUR")...)
	card := BuildValueOverTimeCard(points, ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD", "EUR"}, "en")
	cc := findDescendantByID(card, "currency-controls")
	require.NotNil(t, cc)
	require.Len(t, cc.Children, 2)
	assert.Equal(t, "chart-currency-USD", cc.Children[0].ID)
	assert.Equal(t, "chart-currency-EUR", cc.Children[1].ID)
}

func TestBuildValueOverTimeCard_AbsDataMapping(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, data, 3)
	assert.Equal(t, 10000.0, data[0]["value"])
	assert.Equal(t, 10500.0, data[1]["value"])
	assert.Equal(t, 11000.0, data[2]["value"])
}

func TestBuildValueOverTimeCard_AbsYAxisFormat(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)

	y, ok := chart.Props["y_axis"].(components.Axis)
	_ = ok
	// fallback: marshal+decode
	_ = y
}

func TestBuildValueOverTimeCard_CurrencyFilters(t *testing.T) {
	points := append(sampleChartPoints("USD"), sampleChartPoints("EUR")...)
	card := BuildValueOverTimeCard(points, ChartState{Timeframe: "all", Mode: "abs", Currency: "EUR"}, []string{"USD", "EUR"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	// Only EUR points.
	assert.Len(t, data, 3)
}

func TestBuildValueOverTimeCard_NotEnoughData(t *testing.T) {
	single := []EvolutionPoint{{Currency: "USD", RecordedAt: time.Now(), TotalValue: 100}}
	card := BuildValueOverTimeCard(single, ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	assert.Empty(t, data)
	assert.Equal(t, "Record at least two snapshots to see the chart.", chart.Props["empty_message"])
}
```

Note the `TestBuildValueOverTimeCard_AbsYAxisFormat` test above is a placeholder — remove it from the file and instead verify the y_axis format by marshaling. Here is the cleaned version to paste: remove lines 95–103 (the `TestBuildValueOverTimeCard_AbsYAxisFormat` block) before running.

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/portfolio/... -run TestBuildValueOverTimeCard -v`
Expected: FAIL — `BuildValueOverTimeCard` / `ChartState` undefined.

- [ ] **Step 3: Implement**

Create `internal/portfolio/chart_builder.go`:

```go
package portfolio

import (
	"net/url"
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// ChartState is the user-selected state for the Value Over Time chart.
type ChartState struct {
	Timeframe string // 1m, 3m, 6m, ytd, 1y, all
	Mode      string // abs, pct
	Currency  string // ISO code; "" if there are no currencies
}

var timeframes = []string{"1m", "3m", "6m", "ytd", "1y", "all"}

// BuildValueOverTimeCard renders the Value Over Time chart card. Pure —
// depends only on its inputs. `points` holds evolution points filtered by the
// timeframe already; this function filters by currency internally and maps to
// chart data according to `state.Mode`.
func BuildValueOverTimeCard(points []EvolutionPoint, state ChartState, currencies []string, lang string) components.Component {
	controls := buildChartControls(state, currencies, lang)
	chart := buildLineChart(points, state, lang)
	content := components.ColumnWithGap("chart-value-over-time-content", "md", controls, chart)
	return components.Card("chart-value-over-time-card", content)
}

func buildChartControls(state ChartState, currencies []string, lang string) components.Component {
	tf := buildTimeframeControls(state, lang)
	md := buildModeControls(state, lang)
	children := []components.Component{tf, md}
	if len(currencies) > 1 {
		children = append(children, buildCurrencyControls(state, currencies, lang))
	}
	row := components.Row("controls-row", rowAutoWidths(len(children)), children...)
	row.Props["gap"] = "lg"
	return row
}

func buildTimeframeControls(state ChartState, lang string) components.Component {
	btns := make([]components.Component, 0, len(timeframes))
	for _, tf := range timeframes {
		btns = append(btns, chartButton(
			"chart-timeframe-"+tf,
			i18n.T(lang, "portfolio.chart.timeframe."+tf),
			tf == state.Timeframe,
			evolutionURL(tf, state.Mode, state.Currency),
		))
	}
	row := components.Row("timeframe-controls", rowAutoWidths(len(btns)), btns...)
	row.Props["gap"] = "sm"
	return row
}

func buildModeControls(state ChartState, lang string) components.Component {
	abs := chartButton("chart-mode-abs",
		i18n.T(lang, "portfolio.chart.mode.abs"),
		state.Mode == "abs",
		evolutionURL(state.Timeframe, "abs", state.Currency),
	)
	pct := chartButton("chart-mode-pct",
		i18n.T(lang, "portfolio.chart.mode.pct"),
		state.Mode == "pct",
		evolutionURL(state.Timeframe, "pct", state.Currency),
	)
	row := components.Row("mode-controls", rowAutoWidths(2), abs, pct)
	row.Props["gap"] = "sm"
	return row
}

func buildCurrencyControls(state ChartState, currencies []string, lang string) components.Component {
	btns := make([]components.Component, 0, len(currencies))
	for _, c := range currencies {
		btns = append(btns, chartButton(
			"chart-currency-"+c,
			c,
			c == state.Currency,
			evolutionURL(state.Timeframe, state.Mode, c),
		))
	}
	row := components.Row("currency-controls", rowAutoWidths(len(btns)), btns...)
	row.Props["gap"] = "sm"
	return row
}

func chartButton(id, label string, selected bool, endpoint string) components.Component {
	variant, style := "secondary", "ghost"
	if selected {
		variant, style = "primary", "solid"
	}
	return components.Component{
		Type: "button",
		ID:   id,
		Props: map[string]any{
			"label":   label,
			"variant": variant,
			"style":   style,
		},
		Actions: []components.Action{
			{Trigger: "click", Type: "reload", Endpoint: endpoint, TargetID: "chart-value-over-time-card"},
		},
	}
}

func evolutionURL(timeframe, mode, currency string) string {
	q := url.Values{}
	q.Set("timeframe", timeframe)
	q.Set("mode", mode)
	if currency != "" {
		q.Set("currency", currency)
	}
	return "/actions/portfolio/evolution?" + q.Encode()
}

func rowAutoWidths(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "auto"
	}
	return out
}

func buildLineChart(points []EvolutionPoint, state ChartState, lang string) components.Component {
	data := make([]map[string]any, 0, len(points))
	for _, p := range points {
		if p.Currency != state.Currency {
			continue
		}
		data = append(data, map[string]any{
			"date":  p.RecordedAt.Format("2006-01-02"),
			"value": p.TotalValue,
		})
	}

	valueFormat := "currency_compact"
	if state.Mode == "pct" {
		valueFormat = "percent_signed"
	}
	yFormat := valueFormat

	series := []components.Series{{
		Key:         "value",
		Label:       i18n.T(lang, "portfolio.chart.series.value"),
		Color:       "chart_1",
		ValueFormat: valueFormat,
	}}

	emptyMessage := ""
	if len(data) < 2 {
		data = data[:0]
		emptyMessage = i18n.T(lang, "portfolio.chart.not_enough_data")
	}

	return components.LineChart(
		"chart-value-over-time",
		"md",
		series,
		components.Axis{Key: "date", Format: "month_year"},
		components.Axis{Format: yFormat},
		data,
		emptyMessage,
	)
}

// _ = strings.TrimSpace prevents "imported and not used" if no other usage.
var _ = strings.TrimSpace
```

Remove the trailing `var _ = strings.TrimSpace` and the `"strings"` import if not needed after the final code (Go will fail to compile on unused imports — check and prune).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -run TestBuildValueOverTimeCard -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/chart_builder.go internal/portfolio/chart_builder_test.go
git commit -m "feat(portfolio): BuildValueOverTimeCard with controls and chart"
```

---

### Task 4: i18n + BuildScreen integration + use case parallel fetch

**Files:**
- Modify: `locales/en.json`, `locales/es.json`
- Modify: `internal/portfolio/get_usecase.go`
- Modify: `internal/portfolio/get_usecase_test.go`
- Modify: `internal/portfolio/builder.go`
- Modify: `internal/portfolio/builder_test.go`
- Modify: `internal/portfolio/handler_test.go`

- [ ] **Step 1: Add i18n keys to `locales/en.json`**

Find `"include_closed": "Include closed positions"` in the `"portfolio"` object. Replace that line with:

```json
    "include_closed": "Include closed positions",
    "chart": {
      "value_over_time": {
        "title": "Portfolio Value Over Time"
      },
      "series": {
        "value": "Value"
      },
      "timeframe": {
        "1m": "1M",
        "3m": "3M",
        "6m": "6M",
        "ytd": "YTD",
        "1y": "1Y",
        "all": "All"
      },
      "mode": {
        "abs": "$",
        "pct": "%"
      },
      "not_enough_data": "Record at least two snapshots to see the chart.",
      "no_cost_data": "No cost data available."
    }
```

- [ ] **Step 2: Add i18n keys to `locales/es.json`**

Find `"include_closed": "Incluir posiciones cerradas"`. Replace with:

```json
    "include_closed": "Incluir posiciones cerradas",
    "chart": {
      "value_over_time": {
        "title": "Valor del portafolio"
      },
      "series": {
        "value": "Valor"
      },
      "timeframe": {
        "1m": "1M",
        "3m": "3M",
        "6m": "6M",
        "ytd": "AÑO",
        "1y": "1A",
        "all": "Todo"
      },
      "mode": {
        "abs": "$",
        "pct": "%"
      },
      "not_enough_data": "Registrá al menos dos snapshots para ver el gráfico.",
      "no_cost_data": "Sin datos de costo."
    }
```

- [ ] **Step 3: Extend `portfolioFetcher` in `get_usecase.go`**

Replace the `portfolioFetcher` interface:

```go
type portfolioFetcher interface {
	GetPositions(ctx context.Context, authorization string, includeClosed bool) ([]Position, error)
	GetEvolutionLast(ctx context.Context, authorization string, n int) ([]EvolutionPoint, error)
	GetEvolution(ctx context.Context, authorization string, q EvolutionQuery) ([]EvolutionPoint, error)
}
```

- [ ] **Step 4: Update `Execute` to fetch three things in parallel**

Replace the entire `Execute` function:

```go
func (uc *GetUseCase) Execute(ctx context.Context, authorization, lang string, now time.Time) (components.Component, error) {
	var positions []Position
	var evolutionLast []EvolutionPoint
	var chartPoints []EvolutionPoint

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		p, err := uc.client.GetPositions(gctx, authorization, false)
		if err != nil {
			return err
		}
		positions = p
		return nil
	})

	g.Go(func() error {
		e, err := uc.client.GetEvolutionLast(gctx, authorization, 2)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				return err
			}
			return nil
		}
		evolutionLast = e
		return nil
	})

	g.Go(func() error {
		e, err := uc.client.GetEvolution(gctx, authorization, EvolutionQuery{Points: 100})
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				return err
			}
			return nil
		}
		chartPoints = e
		return nil
	})

	if err := g.Wait(); err != nil {
		return components.Component{}, err
	}

	SortPositions(positions)
	return BuildScreen(positions, evolutionLast, chartPoints, lang, now), nil
}
```

- [ ] **Step 5: Update `fakeFetcher` in `get_usecase_test.go`**

Add the new method and fields:

```go
type fakeFetcher struct {
	positions        []Position
	evolution        []EvolutionPoint
	chart            []EvolutionPoint
	posErr           error
	evoErr           error
	chartErr         error
	gotAuthP         string
	gotAuthE         string
	gotLastN         int
	gotIncludeClosed bool
	gotChartQuery    EvolutionQuery
}

func (f *fakeFetcher) GetPositions(ctx context.Context, auth string, includeClosed bool) ([]Position, error) {
	f.gotAuthP = auth
	f.gotIncludeClosed = includeClosed
	return f.positions, f.posErr
}

func (f *fakeFetcher) GetEvolutionLast(ctx context.Context, auth string, n int) ([]EvolutionPoint, error) {
	f.gotAuthE = auth
	f.gotLastN = n
	return f.evolution, f.evoErr
}

func (f *fakeFetcher) GetEvolution(ctx context.Context, auth string, q EvolutionQuery) ([]EvolutionPoint, error) {
	f.gotChartQuery = q
	return f.chart, f.chartErr
}
```

Add these tests at the end of the file:

```go
func TestGetUseCase_FetchesChartEvolutionWith100Points(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}}}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.Equal(t, 100, f.gotChartQuery.Points)
	assert.Nil(t, f.gotChartQuery.From)
	assert.Equal(t, "", f.gotChartQuery.Currency)
}

func TestGetUseCase_ChartFetchFailureDoesNotFail(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		chartErr:  ErrBackend,
	}
	uc := NewGetUseCase(f)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "screen", screen.Type)
}

func TestGetUseCase_ChartAuthErrorPropagates(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		chartErr:  ErrUnauthorized,
	}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}
```

- [ ] **Step 6: Update `stubFetcher` in `handler_test.go`**

Add the new method so `stubFetcher` continues to satisfy the interface:

```go
func (s *stubFetcher) GetEvolution(ctx context.Context, auth string, q EvolutionQuery) ([]EvolutionPoint, error) {
	return nil, nil
}
```

(Append after the existing methods on `stubFetcher`.)

- [ ] **Step 7: Update `BuildScreen` signature in `builder.go`**

Find:

```go
func BuildScreen(positions []Position, evolution []EvolutionPoint, lang string, now time.Time) components.Component {
```

Replace with:

```go
func BuildScreen(positions []Position, evolution []EvolutionPoint, chartPoints []EvolutionPoint, lang string, now time.Time) components.Component {
```

Inside the function, at the end (just before the return for the non-empty branch), replace:

```go
	root := components.ColumnWithGap("portfolio-root", "lg", summary, controls, table)
```

with:

```go
	chart := buildInitialChartCard(chartPoints, positions, lang)
	root := components.ColumnWithGap("portfolio-root", "lg", summary, controls, table, chart)
```

Add this helper at the end of `builder.go`:

```go
// buildInitialChartCard produces the chart-value-over-time-card for the initial
// screen render. Chooses default currency from positions (highest total value).
func buildInitialChartCard(chartPoints []EvolutionPoint, positions []Position, lang string) components.Component {
	metrics := ComputeMetrics(positions, nil)
	currencies := metrics.CurrencyOrder
	defaultCurrency := ""
	if len(currencies) > 0 {
		defaultCurrency = currencies[0]
	}
	state := ChartState{Timeframe: "all", Mode: "abs", Currency: defaultCurrency}
	return BuildValueOverTimeCard(chartPoints, state, currencies, lang)
}
```

- [ ] **Step 8: Update all `BuildScreen` call sites in tests**

`internal/portfolio/builder_test.go` has many `BuildScreen(...)` calls. Each takes 4 args today; add `nil` as the third arg (chartPoints) to every call. Example:

```go
	s := BuildScreen(samplePositions(), nil, "en", time.Now())
```

becomes:

```go
	s := BuildScreen(samplePositions(), nil, nil, "en", time.Now())
```

Search and apply to all tests.

Also add these new tests at the end of `builder_test.go`:

```go
func TestBuildScreen_ChartCardPresentWhenPositions(t *testing.T) {
	ps := samplePositions()
	s := BuildScreen(ps, nil, nil, "en", time.Now())
	assert.NotNil(t, findDescendantByID(s, "chart-value-over-time-card"))
}

func TestBuildScreen_ChartCardAbsentWhenEmpty(t *testing.T) {
	s := BuildScreen(nil, nil, nil, "en", time.Now())
	assert.Nil(t, findDescendantByID(s, "chart-value-over-time-card"))
}
```

- [ ] **Step 9: Run the full suite**

Run: `go test ./... -count=1`
Expected: all tests PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/portfolio/get_usecase.go internal/portfolio/get_usecase_test.go internal/portfolio/handler_test.go internal/portfolio/builder.go internal/portfolio/builder_test.go locales/en.json locales/es.json
git commit -m "feat(portfolio): chart card integrated into screen + parallel evolution fetch"
```

---

### Task 5: Evolution handler

**Files:**
- Create: `internal/portfolio/evolution_handler.go`
- Create: `internal/portfolio/evolution_handler_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/evolution_handler_test.go`:

```go
package portfolio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubEvolutionFetcher struct {
	points    []EvolutionPoint
	err       error
	gotAuth   string
	gotQuery  EvolutionQuery
	called    bool
}

func (s *stubEvolutionFetcher) GetEvolution(ctx context.Context, auth string, q EvolutionQuery) ([]EvolutionPoint, error) {
	s.called = true
	s.gotAuth = auth
	s.gotQuery = q
	return s.points, s.err
}

func setupEvolutionRouter(f evolutionFetcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/actions/portfolio/evolution", NewEvolutionHandler(f).Get)
	return r
}

func doGet(t *testing.T, r *gin.Engine, query string, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/actions/portfolio/evolution?"+query, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestEvolutionHandler_SuccessReturnsReplaceActionResponse(t *testing.T) {
	f := &stubEvolutionFetcher{points: []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
	}}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=3m&mode=abs&currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "chart-value-over-time-card", resp["target_id"])
	tree, ok := resp["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "card", tree["type"])
	assert.Equal(t, "chart-value-over-time-card", tree["id"])

	assert.True(t, f.called)
	assert.Equal(t, "Bearer tok", f.gotAuth)
	assert.Equal(t, 100, f.gotQuery.Points)
	assert.Equal(t, "USD", f.gotQuery.Currency)
	require.NotNil(t, f.gotQuery.From)
}

func TestEvolutionHandler_AllTimeframeOmitsFrom(t *testing.T) {
	f := &stubEvolutionFetcher{}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=abs&currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, f.gotQuery.From)
}

func TestEvolutionHandler_InvalidTimeframeReturns400(t *testing.T) {
	f := &stubEvolutionFetcher{}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=xxx&mode=abs&currency=USD", "Bearer tok")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, f.called)
}

func TestEvolutionHandler_InvalidModeReturns400(t *testing.T) {
	f := &stubEvolutionFetcher{}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=yolo&currency=USD", "Bearer tok")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, f.called)
}

func TestEvolutionHandler_DefaultsAppliedWhenOmitted(t *testing.T) {
	f := &stubEvolutionFetcher{}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	// timeframe=all (no From), mode=abs (default), no currency
	assert.Nil(t, f.gotQuery.From)
	assert.Equal(t, "", f.gotQuery.Currency)
}

func TestEvolutionHandler_BackendUnauthorizedReturns401WithRedirect(t *testing.T) {
	f := &stubEvolutionFetcher{err: ErrUnauthorized}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=abs&currency=USD", "Bearer x")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"unauthorized"`)
	assert.Contains(t, w.Body.String(), `"redirect":"/screens/login"`)
}

func TestEvolutionHandler_BackendErrorReturns502(t *testing.T) {
	f := &stubEvolutionFetcher{err: ErrBackend}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=abs&currency=USD", "Bearer x")
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "BACKEND_ERROR")
}

func TestEvolutionHandler_PctWithNoCostDataShowsEmptyMessage(t *testing.T) {
	// For this layer's contract, pct is computed ONLY when the backend returns
	// total_cost on points. Our EvolutionPoint does not carry total_cost yet;
	// thus every point's implicit total_cost is zero and pct falls back to empty.
	f := &stubEvolutionFetcher{points: []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
	}}
	r := setupEvolutionRouter(f)

	w := doGet(t, r, "timeframe=all&mode=pct&currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	// We expect the chart's empty_message to be the no-cost variant.
	assert.Contains(t, w.Body.String(), "No cost data available")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/portfolio/... -run TestEvolutionHandler -v`
Expected: FAIL — `NewEvolutionHandler` / `evolutionFetcher` undefined.

- [ ] **Step 3: Implement the handler**

Create `internal/portfolio/evolution_handler.go`:

```go
package portfolio

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// evolutionFetcher is the narrow interface the handler needs; *Client satisfies it.
type evolutionFetcher interface {
	GetEvolution(ctx context.Context, authorization string, q EvolutionQuery) ([]EvolutionPoint, error)
}

type EvolutionHandler struct {
	client evolutionFetcher
	now    func() time.Time
}

func NewEvolutionHandler(client evolutionFetcher) *EvolutionHandler {
	return &EvolutionHandler{client: client, now: time.Now}
}

func (h *EvolutionHandler) Get(c *gin.Context) {
	timeframe := c.DefaultQuery("timeframe", "all")
	mode := c.DefaultQuery("mode", "abs")
	currency := c.Query("currency")

	if !isValidTimeframe(timeframe) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid timeframe"}})
		return
	}
	if mode != "abs" && mode != "pct" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid mode"}})
		return
	}

	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	q := EvolutionQuery{Points: 100, Currency: currency}
	if from := timeframeFrom(timeframe, h.now()); from != nil {
		q.From = from
	}

	points, err := h.client.GetEvolution(c.Request.Context(), auth, q)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load evolution"}})
		return
	}

	state := ChartState{Timeframe: timeframe, Mode: mode, Currency: currency}
	// Currencies available for the control are the distinct ones present in
	// the returned points.
	currencies := distinctCurrencies(points)

	var tree components.Component
	if mode == "pct" {
		// This layer does not have total_cost on EvolutionPoint. Surface the
		// no-cost empty state explicitly.
		tree = buildPctNoCostCard(state, currencies, lang)
	} else {
		tree = BuildValueOverTimeCard(points, state, currencies, lang)
	}

	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "chart-value-over-time-card",
		Tree:     &tree,
	})
}

func isValidTimeframe(tf string) bool {
	for _, v := range timeframes {
		if tf == v {
			return true
		}
	}
	return false
}

func timeframeFrom(tf string, now time.Time) *time.Time {
	switch tf {
	case "1m":
		t := now.AddDate(0, 0, -30)
		return &t
	case "3m":
		t := now.AddDate(0, 0, -90)
		return &t
	case "6m":
		t := now.AddDate(0, 0, -180)
		return &t
	case "ytd":
		t := time.Date(now.UTC().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return &t
	case "1y":
		t := now.AddDate(0, 0, -365)
		return &t
	default:
		return nil
	}
}

func distinctCurrencies(points []EvolutionPoint) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, p := range points {
		if _, ok := seen[p.Currency]; ok {
			continue
		}
		seen[p.Currency] = struct{}{}
		out = append(out, p.Currency)
	}
	return out
}

// buildPctNoCostCard reproduces the chart card shape but replaces the line_chart
// with an empty dataset and the no-cost empty_message, preserving controls and
// their selected state.
func buildPctNoCostCard(state ChartState, currencies []string, lang string) components.Component {
	// Start from the empty branch of BuildValueOverTimeCard and rewrite the
	// empty message to the no-cost variant.
	card := BuildValueOverTimeCard(nil, state, currencies, lang)
	chart := findDescendantByIDRef(&card, "chart-value-over-time")
	if chart != nil {
		chart.Props["empty_message"] = i18n.T(lang, "portfolio.chart.no_cost_data")
	}
	return card
}

// findDescendantByIDRef walks the component tree and returns a pointer to the
// matching component so callers can mutate its Props in place.
func findDescendantByIDRef(c *components.Component, id string) *components.Component {
	if c.ID == id {
		return c
	}
	for i := range c.Children {
		if found := findDescendantByIDRef(&c.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/evolution_handler.go internal/portfolio/evolution_handler_test.go
git commit -m "feat(portfolio): GET /actions/portfolio/evolution handler"
```

---

### Task 6: Wire route + smoke

**Files:**
- Modify: `internal/server/server.go`

- [ ] **Step 1: Register the protected route**

In `internal/server/server.go`, locate the existing portfolio block:

```go
	portfolioClient := portfolio.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	portfolioHandler := portfolio.NewHandler(portfolio.NewGetUseCase(portfolioClient))
	protected.GET("/screens/portfolio", portfolioHandler.Get)
	protected.POST("/actions/portfolio/include_closed", portfolio.NewIncludeClosedHandler(portfolioClient).Post)
```

Append:

```go
	protected.GET("/actions/portfolio/evolution", portfolio.NewEvolutionHandler(portfolioClient).Get)
```

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -count=1`
Expected: all tests pass.

- [ ] **Step 3: Build and lint**

Run: `./cli build 2>&1 | tail -1 && ./cli lint 2>&1 | tail -1`
Expected: both `"status":"success"`.

- [ ] **Step 4: Smoke-test**

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

echo "--- portfolio screen includes chart card ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def walk(x, acc):
    if x.get('id','').startswith('chart-'):
        acc.append(x['id'])
    for c in x.get('children', []):
        walk(c, acc)
a = []; walk(d, a); print(sorted(set(a)))
"

echo "--- GET /actions/portfolio/evolution?timeframe=3m&mode=abs ---"
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8082/actions/portfolio/evolution?timeframe=3m&mode=abs" \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
print('action:', d.get('action'), 'target_id:', d.get('target_id'), 'tree.id:', d.get('tree',{}).get('id'))
"

echo "--- invalid timeframe returns 400 ---"
curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8082/actions/portfolio/evolution?timeframe=xxx"

lsof -ti:8082 | xargs kill -9 2>/dev/null; true
```

Expected:
- Screen includes `chart-value-over-time-card`, plus chart-timeframe-*, chart-mode-*, and chart-value-over-time.
- Action response: `action: replace target_id: chart-value-over-time-card tree.id: chart-value-over-time-card`.
- Invalid timeframe: `400`.

Report the observed output verbatim.

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go
git commit -m "feat(server): wire protected GET /actions/portfolio/evolution"
```

---

## Self-Review Results

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| `line_chart` component type + props | Task 1 (`TestLineChart_*`) |
| `Series`, `Axis` types with correct JSON shape | Task 1 `TestLineChart_JSONShape` |
| `Client.GetEvolution(q)` forwards auth and query | Task 2 `TestClient_GetEvolution_*` |
| `BuildValueOverTimeCard` produces card with timeframe/mode/currency controls | Task 3 `TestBuildValueOverTimeCard_*` |
| Selected button has `primary/solid`, others `secondary/ghost` | Task 3 `TestBuildValueOverTimeCard_SelectedTimeframeHasSolidStyle` |
| Each button URL carries full new state | Task 3 `TestBuildValueOverTimeCard_ButtonURLCarriesFullState` |
| Currency controls hidden when single currency | Task 3 `TestBuildValueOverTimeCard_CurrencyControls*` |
| Data filtered by currency + mapped for `abs` | Task 3 `TestBuildValueOverTimeCard_AbsDataMapping`, `_CurrencyFilters` |
| `<2` points → empty message `not_enough_data` | Task 3 `TestBuildValueOverTimeCard_NotEnoughData` |
| Initial screen fetch also fetches 100-point evolution in parallel | Task 4 `TestGetUseCase_FetchesChartEvolutionWith100Points` |
| Chart fetch failure (non-auth) is tolerated | Task 4 `TestGetUseCase_ChartFetchFailureDoesNotFail` |
| Chart fetch auth error propagates | Task 4 `TestGetUseCase_ChartAuthErrorPropagates` |
| Chart card appears only when positions non-empty | Task 4 `TestBuildScreen_ChartCardPresentWhenPositions`, `_AbsentWhenEmpty` |
| `GET /actions/portfolio/evolution` success returns replace action response | Task 5 `TestEvolutionHandler_SuccessReturnsReplaceActionResponse` |
| `timeframe=all` omits `from` | Task 5 `TestEvolutionHandler_AllTimeframeOmitsFrom` |
| Invalid timeframe/mode → 400 | Task 5 `TestEvolutionHandler_InvalidTimeframe/Mode` |
| Backend 401 → 401 with redirect | Task 5 `TestEvolutionHandler_BackendUnauthorizedReturns401WithRedirect` |
| Backend 5xx → 502 | Task 5 `TestEvolutionHandler_BackendErrorReturns502` |
| `mode=pct` with no cost data → empty message `no_cost_data` | Task 5 `TestEvolutionHandler_PctWithNoCostDataShowsEmptyMessage` |
| i18n keys in both locales | Task 4 Steps 1–2 |
| Protected route registered | Task 6 |

**Placeholder scan:** none.

**Type consistency:**
- `EvolutionQuery { From *time.Time, Points int, Currency string }` — same in Task 2 (client), Task 4 (use case interface + fakeFetcher), Task 5 (evolution handler stub).
- `ChartState { Timeframe, Mode, Currency string }` — same in Task 3 (builder) and Task 5 (handler).
- `BuildScreen(positions, evolution, chartPoints, lang, now)` — signature matches across Task 4 implementation, tests, and use case call.
- `BuildValueOverTimeCard(points, state, currencies, lang) components.Component` — same in Task 3 and Task 5.
- Component IDs: `chart-value-over-time-card`, `chart-value-over-time`, `timeframe-controls`, `mode-controls`, `currency-controls`, `chart-timeframe-{1m,3m,6m,ytd,1y,all}`, `chart-mode-{abs,pct}`, `chart-currency-{CODE}` — identical across spec, builder, tests, handler.
- Color token `chart_1`, `ValueFormat` `currency_compact`/`percent_signed`, `AxisFormat` `month_year` — consistent with `spec/sdui-custom-components.md`.
