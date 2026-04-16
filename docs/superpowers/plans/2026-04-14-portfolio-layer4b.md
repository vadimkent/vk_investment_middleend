# Portfolio Layer 4b Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `spec/screens/portfolio/04b-asset-value-over-time.md` — add a second chart card (Asset Value Over Time, multi-series) that shares the same controls as the value chart. Both cards live inside a new `charts-section` wrapper that is the reload target.

**Architecture:** (1) Parse per-asset values from evolution; (2) restructure the chart area: controls move out of the value card, both cards become siblings inside a `column#charts-section`, and reload targets the section. (3) Add multi-series asset chart builder. The evolution endpoint returns the full section tree on each reload.

**Tech Stack:** Go, existing `internal/components`, `internal/portfolio`, testify.

---

## File Structure

**Modify:**

| File | Change |
|---|---|
| `internal/portfolio/evolution.go` | Add `AssetValue{AssetID, Ticker, Value}` + `Assets []AssetValue` on `EvolutionPoint`. Parse `assets` in `ParseEvolution`. |
| `internal/portfolio/evolution_test.go` | Tests for asset parsing. |
| `locales/en.json`, `locales/es.json` | Add `portfolio.chart.asset_value_over_time.title`. |
| `internal/portfolio/chart_builder.go` | Rewrite: simplify `BuildValueOverTimeCard` (just title + line_chart, no controls), add `BuildAssetValueOverTimeCard`, add `BuildChartsSection` wrapper (controls + both cards), change `chartButton` target to `charts-section`. |
| `internal/portfolio/chart_builder_test.go` | Replace existing tests with new structural assertions (charts-section, both cards, controls siblings). |
| `internal/portfolio/evolution_handler.go` | Return `BuildChartsSection` tree instead of the single card; set response `target_id` to `charts-section`. |
| `internal/portfolio/evolution_handler_test.go` | Update assertions to the new `target_id` and tree shape. |
| `internal/portfolio/builder.go` | `buildInitialChartCard` → `buildInitialChartsSection`. `BuildScreen` uses the new helper. |
| `internal/portfolio/builder_test.go` | Update chart-card tests to check `charts-section` + both cards. |

**No change:** `internal/server/server.go` (same endpoint, same wiring).

---

### Task 1: Parse `assets` in evolution

**Files:**
- Modify: `internal/portfolio/evolution.go`
- Modify: `internal/portfolio/evolution_test.go`

- [ ] **Step 1: Append a failing test**

Append to `internal/portfolio/evolution_test.go`:

```go
func TestParseEvolution_ParsesAssets(t *testing.T) {
	raw := []byte(`{
	  "evolution":[
	    {
	      "snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,
	      "total_value":"15420.50","total_cost":"12000.00","currency":"USD",
	      "assets":[
	        {"asset_id":"u1","ticker":"AAPL","value":"5000.00"},
	        {"asset_id":"u2","ticker":"GOOG","value":"10420.50"}
	      ]
	    }
	  ]
	}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Len(t, points[0].Assets, 2)
	assert.Equal(t, "u1", points[0].Assets[0].AssetID)
	assert.Equal(t, "AAPL", points[0].Assets[0].Ticker)
	assert.InDelta(t, 5000.0, points[0].Assets[0].Value, 1e-9)
	assert.Equal(t, "GOOG", points[0].Assets[1].Ticker)
	assert.InDelta(t, 10420.50, points[0].Assets[1].Value, 1e-9)
}

func TestParseEvolution_AssetsAbsentIsEmpty(t *testing.T) {
	raw := []byte(`{"evolution":[{"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"100","currency":"USD"}]}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 1)
	assert.Empty(t, points[0].Assets)
}

func TestParseEvolution_AssetWithMalformedValueSkipped(t *testing.T) {
	raw := []byte(`{
	  "evolution":[
	    {
	      "snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,
	      "total_value":"100","currency":"USD",
	      "assets":[
	        {"asset_id":"u1","ticker":"AAPL","value":"abc"},
	        {"asset_id":"u2","ticker":"GOOG","value":"200"}
	      ]
	    }
	  ]
	}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Len(t, points[0].Assets, 1)
	assert.Equal(t, "GOOG", points[0].Assets[0].Ticker)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/portfolio/... -run TestParseEvolution -v`
Expected: FAIL — `Assets` field and `AssetValue` type undefined.

- [ ] **Step 3: Extend `EvolutionPoint` and the parser**

In `internal/portfolio/evolution.go`:

a) Add this type below `EvolutionPoint`:

```go
// AssetValue is one asset's contribution inside an evolution point.
type AssetValue struct {
	AssetID string
	Ticker  string
	Value   float64
}
```

b) Add the `Assets []AssetValue` field to `EvolutionPoint`:

```go
type EvolutionPoint struct {
	SnapshotID     string
	RecordedAt     time.Time
	IsFullSnapshot bool
	TotalValue     float64
	TotalCost      *float64
	Currency       string
	Assets         []AssetValue
}
```

c) Extend `rawEvolutionPoint` with the raw `assets` field:

```go
type rawAssetValue struct {
	AssetID string `json:"asset_id"`
	Ticker  string `json:"ticker"`
	Value   string `json:"value"`
}

type rawEvolutionPoint struct {
	SnapshotID     string          `json:"snapshot_id"`
	RecordedAt     string          `json:"recorded_at"`
	IsFullSnapshot bool            `json:"is_full_snapshot"`
	TotalValue     string          `json:"total_value"`
	TotalCost      *string         `json:"total_cost"`
	Currency       string          `json:"currency"`
	Assets         []rawAssetValue `json:"assets"`
}
```

d) In `ParseEvolution`, inside the per-point loop, after the `LastSnapshotAt` parsing, add:

```go
for _, ra := range rp.Assets {
	v, err := strconv.ParseFloat(ra.Value, 64)
	if err != nil {
		continue
	}
	p.Assets = append(p.Assets, AssetValue{AssetID: ra.AssetID, Ticker: ra.Ticker, Value: v})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/evolution.go internal/portfolio/evolution_test.go
git commit -m "feat(portfolio): parse per-asset values on EvolutionPoint"
```

---

### Task 2: i18n key for the asset chart title

**Files:**
- Modify: `locales/en.json`, `locales/es.json`

- [ ] **Step 1: Add the key to `locales/en.json`**

Find the existing `"value_over_time": {` object inside `portfolio.chart`. Replace the entire `"value_over_time"` block with:

```json
      "value_over_time": {
        "title": "Portfolio Value Over Time"
      },
      "asset_value_over_time": {
        "title": "Asset Value Over Time"
      },
```

- [ ] **Step 2: Add the key to `locales/es.json`**

Find the existing `"value_over_time": {` in `portfolio.chart`. Replace with:

```json
      "value_over_time": {
        "title": "Valor del portafolio en el tiempo"
      },
      "asset_value_over_time": {
        "title": "Valor por activo en el tiempo"
      },
```

- [ ] **Step 3: Run full suite**

Run: `go test ./... -count=1`
Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "feat(i18n): asset_value_over_time.title + rename value_over_time.title"
```

---

### Task 3: Restructure charts area — extract controls, add `charts-section`

This is the architectural change. Controls move out of the value card; both cards become siblings inside a new wrapper column `charts-section`. For this task the `charts-section` contains only the simplified value card — the asset card is added in Task 4.

**Files:**
- Modify: `internal/portfolio/chart_builder.go`
- Modify: `internal/portfolio/chart_builder_test.go`
- Modify: `internal/portfolio/evolution_handler.go`
- Modify: `internal/portfolio/evolution_handler_test.go`
- Modify: `internal/portfolio/builder.go`
- Modify: `internal/portfolio/builder_test.go`

- [ ] **Step 1: Replace `internal/portfolio/chart_builder.go` entirely**

Open the file and replace its contents with:

```go
package portfolio

import (
	"net/url"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// ChartState is the user-selected state shared by all charts in the section.
type ChartState struct {
	Timeframe string // 1m, 3m, 6m, ytd, 1y, all
	Mode      string // abs, pct
	Currency  string // ISO code; "" when no currencies
}

var timeframes = []string{"1m", "3m", "6m", "ytd", "1y", "all"}

// BuildChartsSection wraps controls and the chart cards in a column that
// serves as the reload target. Pure — no BE dependency.
func BuildChartsSection(points []EvolutionPoint, state ChartState, currencies []string, lang string) components.Component {
	controls := buildChartControls(state, currencies, lang)
	value := BuildValueOverTimeCard(points, state, lang)
	col := components.ColumnWithGap("charts-section", "lg", controls, value)
	return col
}

// BuildValueOverTimeCard builds the Value Over Time card: title + line_chart.
// Controls live outside (at the charts-section level).
func BuildValueOverTimeCard(points []EvolutionPoint, state ChartState, lang string) components.Component {
	title := components.Text("chart-value-over-time-title", i18n.T(lang, "portfolio.chart.value_over_time.title"), "md", "bold")
	chart := buildValueLineChart(points, state, lang)
	content := components.ColumnWithGap("chart-value-over-time-content", "md", title, chart)
	return components.Card("chart-value-over-time-card", content)
}

func buildChartControls(state ChartState, currencies []string, lang string) components.Component {
	tf := buildTimeframeControls(state, lang)
	md := buildModeControls(state, lang)
	spacer := components.Column("controls-spacer")

	var children []components.Component
	var widths []string
	if len(currencies) > 1 {
		children = []components.Component{buildCurrencyControls(state, currencies, lang), spacer, md, tf}
		widths = []string{"auto", "1fr", "auto", "auto"}
	} else {
		children = []components.Component{spacer, md, tf}
		widths = []string{"1fr", "auto", "auto"}
	}
	row := components.Row("controls-row", widths, children...)
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
			"size":    "sm",
		},
		Actions: []components.Action{
			{Trigger: "click", Type: "reload", Endpoint: endpoint, TargetID: "charts-section"},
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

func buildValueLineChart(points []EvolutionPoint, state ChartState, lang string) components.Component {
	data := make([]map[string]any, 0, len(points))
	anyCost := false
	for _, p := range points {
		if p.Currency != state.Currency {
			continue
		}
		row := map[string]any{"date": p.RecordedAt.Format("2006-01-02")}
		if state.Mode == "pct" {
			if p.TotalCost != nil && *p.TotalCost != 0 {
				anyCost = true
				row["value"] = (p.TotalValue - *p.TotalCost) / *p.TotalCost * 100
			} else {
				row["value"] = nil
			}
		} else {
			row["value"] = p.TotalValue
		}
		data = append(data, row)
	}

	valueFormat := "currency_compact"
	if state.Mode == "pct" {
		valueFormat = "percent_signed"
	}

	series := []components.Series{{
		Key:         "value",
		Label:       i18n.T(lang, "portfolio.chart.series.value"),
		Color:       "chart_1",
		ValueFormat: valueFormat,
	}}

	emptyMessage := ""
	switch {
	case state.Mode == "pct" && len(data) > 0 && !anyCost:
		data = data[:0]
		emptyMessage = i18n.T(lang, "portfolio.chart.no_cost_data")
	case len(data) < 2:
		data = data[:0]
		emptyMessage = i18n.T(lang, "portfolio.chart.not_enough_data")
	}

	return components.LineChart(
		"chart-value-over-time",
		"md",
		series,
		components.Axis{Key: "date", Format: "month_year"},
		components.Axis{Format: valueFormat},
		data,
		emptyMessage,
	)
}
```

- [ ] **Step 2: Replace `internal/portfolio/chart_builder_test.go` entirely**

Replace its contents with:

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

func TestBuildChartsSection_RootIsColumn(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	assert.Equal(t, "column", s.Type)
	assert.Equal(t, "charts-section", s.ID)
}

func TestBuildChartsSection_ContainsControlsThenValueCard(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	require.GreaterOrEqual(t, len(s.Children), 2)
	assert.Equal(t, "controls-row", s.Children[0].ID)
	assert.Equal(t, "chart-value-over-time-card", s.Children[1].ID)
}

func TestBuildValueOverTimeCard_HasTitleAndChart(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, "en")
	assert.Equal(t, "card", card.Type)
	assert.Equal(t, "chart-value-over-time-card", card.ID)

	title := findDescendantByID(card, "chart-value-over-time-title")
	require.NotNil(t, title)
	assert.Equal(t, "Portfolio Value Over Time", title.Props["content"])

	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	assert.Equal(t, "line_chart", chart.Type)
}

func TestBuildValueOverTimeCard_DoesNotContainControls(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, "en")
	assert.Nil(t, findDescendantByID(card, "controls-row"))
	assert.Nil(t, findDescendantByID(card, "timeframe-controls"))
	assert.Nil(t, findDescendantByID(card, "mode-controls"))
	assert.Nil(t, findDescendantByID(card, "currency-controls"))
}

func TestBuildChartsSection_TimeframeControlsHaveSixButtons(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	tf := findDescendantByID(s, "timeframe-controls")
	require.NotNil(t, tf)
	require.Len(t, tf.Children, 6)
	ids := []string{"chart-timeframe-1m", "chart-timeframe-3m", "chart-timeframe-6m", "chart-timeframe-ytd", "chart-timeframe-1y", "chart-timeframe-all"}
	for i, id := range ids {
		assert.Equal(t, id, tf.Children[i].ID)
	}
}

func TestBuildChartsSection_SelectedTimeframeHasSolidStyle(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "3m", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	selected := findDescendantByID(s, "chart-timeframe-3m")
	require.NotNil(t, selected)
	assert.Equal(t, "primary", selected.Props["variant"])
	assert.Equal(t, "solid", selected.Props["style"])
}

func TestBuildChartsSection_ButtonActionTargetsChartsSection(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "3m", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	btn := findDescendantByID(s, "chart-timeframe-6m")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "reload", a.Type)
	assert.Equal(t, "charts-section", a.TargetID)
	assert.Contains(t, a.Endpoint, "timeframe=6m")
	assert.Contains(t, a.Endpoint, "mode=abs")
	assert.Contains(t, a.Endpoint, "currency=USD")
}

func TestBuildChartsSection_CurrencyControlsHiddenWhenSingle(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	assert.Nil(t, findDescendantByID(s, "currency-controls"))
}

func TestBuildChartsSection_CurrencyControlsShownWhenMulti(t *testing.T) {
	points := append(sampleChartPoints("USD"), sampleChartPoints("EUR")...)
	s := BuildChartsSection(points, ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD", "EUR"}, "en")
	cc := findDescendantByID(s, "currency-controls")
	require.NotNil(t, cc)
	require.Len(t, cc.Children, 2)
	assert.Equal(t, "chart-currency-USD", cc.Children[0].ID)
	assert.Equal(t, "chart-currency-EUR", cc.Children[1].ID)
}

func TestBuildValueOverTimeCard_AbsDataMapping(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, data, 3)
	assert.Equal(t, 10000.0, data[0]["value"])
	assert.Equal(t, 10500.0, data[1]["value"])
	assert.Equal(t, 11000.0, data[2]["value"])
}

func TestBuildValueOverTimeCard_NotEnoughData(t *testing.T) {
	single := []EvolutionPoint{{Currency: "USD", RecordedAt: time.Now(), TotalValue: 100}}
	card := BuildValueOverTimeCard(single, ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	assert.Empty(t, data)
	assert.Equal(t, "Record at least two snapshots to see the chart.", chart.Props["empty_message"])
}
```

- [ ] **Step 3: Update `internal/portfolio/evolution_handler.go` response**

Find the block inside `Get`:

```go
	state := ChartState{Timeframe: timeframe, Mode: mode, Currency: currency}
	currencies := distinctCurrencies(points)
	tree := BuildValueOverTimeCard(points, state, currencies, lang)

	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "chart-value-over-time-card",
		Tree:     &tree,
	})
```

Replace with:

```go
	state := ChartState{Timeframe: timeframe, Mode: mode, Currency: currency}
	currencies := distinctCurrencies(points)
	tree := BuildChartsSection(points, state, currencies, lang)

	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "charts-section",
		Tree:     &tree,
	})
```

- [ ] **Step 4: Update `internal/portfolio/evolution_handler_test.go`**

Find the assertion block inside `TestEvolutionHandler_SuccessReturnsReplaceActionResponse`:

```go
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "chart-value-over-time-card", resp["target_id"])
	tree, ok := resp["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "card", tree["type"])
	assert.Equal(t, "chart-value-over-time-card", tree["id"])
```

Replace with:

```go
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "charts-section", resp["target_id"])
	tree, ok := resp["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "column", tree["type"])
	assert.Equal(t, "charts-section", tree["id"])
```

- [ ] **Step 5: Update `internal/portfolio/builder.go`**

Find `buildInitialChartCard`:

```go
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

Replace with:

```go
func buildInitialChartsSection(chartPoints []EvolutionPoint, positions []Position, lang string) components.Component {
	metrics := ComputeMetrics(positions, nil)
	currencies := metrics.CurrencyOrder
	defaultCurrency := ""
	if len(currencies) > 0 {
		defaultCurrency = currencies[0]
	}
	state := ChartState{Timeframe: "all", Mode: "abs", Currency: defaultCurrency}
	return BuildChartsSection(chartPoints, state, currencies, lang)
}
```

Then in `BuildScreen` find:

```go
	chart := buildInitialChartCard(chartPoints, positions, lang)
	root := components.ColumnWithGap("portfolio-root", "lg", summary, controls, table, chart)
```

Replace with:

```go
	chartsSection := buildInitialChartsSection(chartPoints, positions, lang)
	root := components.ColumnWithGap("portfolio-root", "lg", summary, controls, table, chartsSection)
```

- [ ] **Step 6: Update `internal/portfolio/builder_test.go`**

Find the existing tests that reference `chart-value-over-time-card` at the screen level:

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

Replace with:

```go
func TestBuildScreen_ChartsSectionPresentWhenPositions(t *testing.T) {
	ps := samplePositions()
	s := BuildScreen(ps, nil, nil, "en", time.Now())
	assert.NotNil(t, findDescendantByID(s, "charts-section"))
	assert.NotNil(t, findDescendantByID(s, "chart-value-over-time-card"))
}

func TestBuildScreen_ChartsSectionAbsentWhenEmpty(t *testing.T) {
	s := BuildScreen(nil, nil, nil, "en", time.Now())
	assert.Nil(t, findDescendantByID(s, "charts-section"))
}
```

- [ ] **Step 7: Run full suite**

Run: `go test ./... -count=1`
Expected: all tests pass.

- [ ] **Step 8: Build and lint**

Run: `./cli build 2>&1 | tail -1 && ./cli lint 2>&1 | tail -1`
Expected: both `"status":"success"`.

- [ ] **Step 9: Commit**

```bash
git add internal/portfolio/chart_builder.go internal/portfolio/chart_builder_test.go internal/portfolio/evolution_handler.go internal/portfolio/evolution_handler_test.go internal/portfolio/builder.go internal/portfolio/builder_test.go
git commit -m "refactor(portfolio): wrap charts in charts-section, extract controls"
```

---

### Task 4: Asset Value Over Time card

**Files:**
- Modify: `internal/portfolio/chart_builder.go`
- Modify: `internal/portfolio/chart_builder_test.go`

- [ ] **Step 1: Append failing tests**

Append to `internal/portfolio/chart_builder_test.go`:

```go
func sampleAssetPoints(currency string) []EvolutionPoint {
	return []EvolutionPoint{
		{Currency: currency, RecordedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1500, Assets: []AssetValue{
			{AssetID: "u1", Ticker: "AAPL", Value: 500},
			{AssetID: "u2", Ticker: "GOOG", Value: 1000},
		}},
		{Currency: currency, RecordedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TotalValue: 2100, Assets: []AssetValue{
			{AssetID: "u1", Ticker: "AAPL", Value: 600},
			{AssetID: "u2", Ticker: "GOOG", Value: 1100},
			{AssetID: "u3", Ticker: "TSLA", Value: 400},
		}},
	}
}

func TestBuildAssetValueOverTimeCard_IsCard(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD"}, "en")
	assert.Equal(t, "card", card.Type)
	assert.Equal(t, "chart-asset-value-over-time-card", card.ID)
}

func TestBuildAssetValueOverTimeCard_HasTitle(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD"}, "en")
	title := findDescendantByID(card, "chart-asset-value-over-time-title")
	require.NotNil(t, title)
	assert.Equal(t, "Asset Value Over Time", title.Props["content"])
}

func TestBuildAssetValueOverTimeCard_SeriesPerTicker(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	series, ok := chart.Props["series"].([]components.Series)
	require.True(t, ok)
	// 3 distinct tickers, ordered by latest-point value desc: GOOG(1100), AAPL(600), TSLA(400).
	require.Len(t, series, 3)
	assert.Equal(t, "GOOG", series[0].Key)
	assert.Equal(t, "AAPL", series[1].Key)
	assert.Equal(t, "TSLA", series[2].Key)
	// Colors cycle through chart_1..chart_5.
	assert.Equal(t, "chart_1", series[0].Color)
	assert.Equal(t, "chart_2", series[1].Color)
	assert.Equal(t, "chart_3", series[2].Color)
	// All series use currency_compact format.
	for _, s := range series {
		assert.Equal(t, "currency_compact", s.ValueFormat)
	}
}

func TestBuildAssetValueOverTimeCard_DataRowsHaveAllTickerKeys(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, data, 2)

	// First snapshot: AAPL 500, GOOG 1000, TSLA nil (absent).
	assert.Equal(t, 500.0, data[0]["AAPL"])
	assert.Equal(t, 1000.0, data[0]["GOOG"])
	assert.Nil(t, data[0]["TSLA"])

	// Second snapshot: all three present.
	assert.Equal(t, 600.0, data[1]["AAPL"])
	assert.Equal(t, 1100.0, data[1]["GOOG"])
	assert.Equal(t, 400.0, data[1]["TSLA"])
}

func TestBuildAssetValueOverTimeCard_IgnoresMode(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD", Mode: "pct"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	series := chart.Props["series"].([]components.Series)
	for _, s := range series {
		assert.Equal(t, "currency_compact", s.ValueFormat, "pct mode should not change asset chart")
	}
}

func TestBuildAssetValueOverTimeCard_FiltersByCurrency(t *testing.T) {
	points := append(sampleAssetPoints("USD"), EvolutionPoint{
		Currency: "EUR", RecordedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		TotalValue: 500, Assets: []AssetValue{{AssetID: "u9", Ticker: "SAP", Value: 500}},
	})
	card := BuildAssetValueOverTimeCard(points, ChartState{Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, data, 2) // only USD points

	series := chart.Props["series"].([]components.Series)
	for _, s := range series {
		assert.NotEqual(t, "SAP", s.Key, "SAP is EUR and should not appear")
	}
}

func TestBuildAssetValueOverTimeCard_EmptyWhenLessThanTwoPoints(t *testing.T) {
	single := []EvolutionPoint{{Currency: "USD", RecordedAt: time.Now(), TotalValue: 100, Assets: []AssetValue{{AssetID: "u1", Ticker: "AAPL", Value: 100}}}}
	card := BuildAssetValueOverTimeCard(single, ChartState{Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	assert.Empty(t, data)
	assert.Equal(t, "Record at least two snapshots to see the chart.", chart.Props["empty_message"])
}

func TestBuildChartsSection_ContainsAssetCard(t *testing.T) {
	s := BuildChartsSection(sampleAssetPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	require.GreaterOrEqual(t, len(s.Children), 3)
	assert.Equal(t, "controls-row", s.Children[0].ID)
	assert.Equal(t, "chart-value-over-time-card", s.Children[1].ID)
	assert.Equal(t, "chart-asset-value-over-time-card", s.Children[2].ID)
}
```

Add this import at the top of `chart_builder_test.go` if not already present:

```go
"github.com/project/vk-investment-middleend/internal/components"
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/portfolio/... -run TestBuildAssetValueOverTimeCard -v`
Expected: FAIL — `BuildAssetValueOverTimeCard` undefined.

- [ ] **Step 3: Implement `BuildAssetValueOverTimeCard`**

Append to `internal/portfolio/chart_builder.go`:

```go
var assetColors = []string{"chart_1", "chart_2", "chart_3", "chart_4", "chart_5"}

// BuildAssetValueOverTimeCard builds the per-asset multi-series card. Always
// absolute currency, regardless of state.Mode.
func BuildAssetValueOverTimeCard(points []EvolutionPoint, state ChartState, lang string) components.Component {
	title := components.Text("chart-asset-value-over-time-title", i18n.T(lang, "portfolio.chart.asset_value_over_time.title"), "md", "bold")
	chart := buildAssetLineChart(points, state, lang)
	content := components.ColumnWithGap("chart-asset-value-over-time-content", "md", title, chart)
	return components.Card("chart-asset-value-over-time-card", content)
}

func buildAssetLineChart(points []EvolutionPoint, state ChartState, lang string) components.Component {
	// Filter to selected currency.
	filtered := make([]EvolutionPoint, 0, len(points))
	for _, p := range points {
		if p.Currency == state.Currency {
			filtered = append(filtered, p)
		}
	}

	// Determine ticker order: by latest-point value DESC, tie by ticker ASC.
	// "Latest" is the last point in filtered (which arrives in order).
	latestValues := map[string]float64{}
	seen := map[string]struct{}{}
	var tickers []string
	for _, p := range filtered {
		for _, a := range p.Assets {
			if _, ok := seen[a.Ticker]; !ok {
				seen[a.Ticker] = struct{}{}
				tickers = append(tickers, a.Ticker)
			}
		}
	}
	// For each ticker find its value in the latest point it appears in.
	for i := len(filtered) - 1; i >= 0; i-- {
		for _, a := range filtered[i].Assets {
			if _, have := latestValues[a.Ticker]; !have {
				latestValues[a.Ticker] = a.Value
			}
		}
	}
	sortTickersByLatestValueDesc(tickers, latestValues)

	// Build series with color cycling.
	series := make([]components.Series, 0, len(tickers))
	for i, t := range tickers {
		series = append(series, components.Series{
			Key:         t,
			Label:       t,
			Color:       assetColors[i%len(assetColors)],
			ValueFormat: "currency_compact",
		})
	}

	// Build data rows: one per filtered point, each row has the tickers' values or nil.
	data := make([]map[string]any, 0, len(filtered))
	for _, p := range filtered {
		row := map[string]any{"date": p.RecordedAt.Format("2006-01-02")}
		present := map[string]float64{}
		for _, a := range p.Assets {
			present[a.Ticker] = a.Value
		}
		for _, t := range tickers {
			if v, ok := present[t]; ok {
				row[t] = v
			} else {
				row[t] = nil
			}
		}
		data = append(data, row)
	}

	emptyMessage := ""
	if len(data) < 2 || len(tickers) == 0 {
		data = data[:0]
		emptyMessage = i18n.T(lang, "portfolio.chart.not_enough_data")
	}

	return components.LineChart(
		"chart-asset-value-over-time",
		"md",
		series,
		components.Axis{Key: "date", Format: "month_year"},
		components.Axis{Format: "currency_compact"},
		data,
		emptyMessage,
	)
}

func sortTickersByLatestValueDesc(tickers []string, latest map[string]float64) {
	// Simple insertion-ish sort via sort.Slice from the stdlib.
	// (Kept inline to avoid an extra helper file; sort is already imported.)
	// Stable ordering: value DESC, ticker ASC as tiebreaker.
	// Using sort.SliceStable to preserve input order as a secondary tiebreaker.
	sortSliceStable(tickers, func(i, j int) bool {
		vi := latest[tickers[i]]
		vj := latest[tickers[j]]
		if vi == vj {
			return tickers[i] < tickers[j]
		}
		return vi > vj
	})
}

// Local alias so the function body does not pull sort into the function's
// scope while keeping the code readable. Remove and inline if preferred.
func sortSliceStable(slice any, less func(i, j int) bool) {
	// Delegate to sort.SliceStable via a tiny shim. Keeping this indirection
	// because chart_builder.go already imports no sort helpers directly.
	sortStable(slice, less)
}

func sortStable(slice any, less func(i, j int) bool) {
	// Using the standard library sort.SliceStable.
	// Import is added at the top of the file.
	sortSliceStableImpl(slice, less)
}

func sortSliceStableImpl(slice any, less func(i, j int) bool) {
	// Final indirection to sort.SliceStable. Adding the sort import in the
	// file is required.
	sortReal(slice, less)
}
```

Note: the code above has three indirections that are **not** meaningful. Simplify step 3 by using `sort.SliceStable` directly. Replace the last four function definitions (`sortTickersByLatestValueDesc`, `sortSliceStable`, `sortStable`, `sortSliceStableImpl`, `sortReal`) with a single clean version:

```go
func sortTickersByLatestValueDesc(tickers []string, latest map[string]float64) {
	sort.SliceStable(tickers, func(i, j int) bool {
		vi := latest[tickers[i]]
		vj := latest[tickers[j]]
		if vi == vj {
			return tickers[i] < tickers[j]
		}
		return vi > vj
	})
}
```

Also add `"sort"` to the import block of `chart_builder.go`:

```go
import (
	"net/url"
	"sort"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)
```

- [ ] **Step 4: Wire `BuildAssetValueOverTimeCard` into `BuildChartsSection`**

Find in `chart_builder.go`:

```go
func BuildChartsSection(points []EvolutionPoint, state ChartState, currencies []string, lang string) components.Component {
	controls := buildChartControls(state, currencies, lang)
	value := BuildValueOverTimeCard(points, state, lang)
	col := components.ColumnWithGap("charts-section", "lg", controls, value)
	return col
}
```

Replace with:

```go
func BuildChartsSection(points []EvolutionPoint, state ChartState, currencies []string, lang string) components.Component {
	controls := buildChartControls(state, currencies, lang)
	value := BuildValueOverTimeCard(points, state, lang)
	asset := BuildAssetValueOverTimeCard(points, state, lang)
	col := components.ColumnWithGap("charts-section", "lg", controls, value, asset)
	return col
}
```

- [ ] **Step 5: Run full suite**

Run: `go test ./... -count=1`
Expected: all tests pass.

- [ ] **Step 6: Build and lint**

Run: `./cli build 2>&1 | tail -1 && ./cli lint 2>&1 | tail -1`
Expected: both success.

- [ ] **Step 7: Commit**

```bash
git add internal/portfolio/chart_builder.go internal/portfolio/chart_builder_test.go
git commit -m "feat(portfolio): BuildAssetValueOverTimeCard (multi-series per ticker)"
```

---

### Task 5: Smoke test end-to-end

**Files:** none (verification only).

- [ ] **Step 1: Restart the server**

Run:

```bash
cd /Users/vadimkent/repos/vk_investment_middleend_v2
lsof -ti:8082 | xargs kill -9 2>/dev/null; sleep 1
./cli run >/tmp/srv.log 2>&1 &
sleep 2
```

- [ ] **Step 2: Login and call the endpoint**

Run and report the output verbatim:

```bash
RESP=$(curl -s -X POST http://localhost:8082/actions/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@demo.com","password":"demo"}')
TOKEN=$(echo "$RESP" | python3 -c "import json,sys;print(json.load(sys.stdin)['auth']['token'])")

echo "--- /screens/portfolio has charts-section with both cards ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
ids=[]
def walk(x):
    if x.get('id','') in ('charts-section','chart-value-over-time-card','chart-asset-value-over-time-card','controls-row'):
        ids.append(x['id'])
    for c in x.get('children', []):
        walk(c)
walk(d); print(ids)
"

echo "--- /actions/portfolio/evolution?timeframe=3m&mode=abs returns charts-section ---"
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8082/actions/portfolio/evolution?timeframe=3m&mode=abs" \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
print('action:', d.get('action'), 'target_id:', d.get('target_id'), 'tree.id:', d.get('tree',{}).get('id'), 'tree.type:', d.get('tree',{}).get('type'))
"

echo "--- asset chart series count ---"
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8082/actions/portfolio/evolution?timeframe=all&mode=abs" \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def walk(x):
    if x.get('id') == 'chart-asset-value-over-time':
        print('series count:', len(x['props'].get('series', [])))
        return
    for c in x.get('children', []):
        walk(c)
walk(d.get('tree',{}))
"

lsof -ti:8082 | xargs kill -9 2>/dev/null; true
```

Expected:
- IDs list includes `charts-section`, `controls-row`, `chart-value-over-time-card`, `chart-asset-value-over-time-card` in that order.
- Action response: `action: replace target_id: charts-section tree.id: charts-section tree.type: column`.
- Asset chart has N series where N is the number of unique tickers in the user's portfolio.

- [ ] **Step 2: No commit for this task** (verification only)

---

## Self-Review Results

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| Parse `assets` from BE response on `EvolutionPoint` | Task 1 |
| `charts-section` wraps controls + both cards, is a column | Task 3 `TestBuildChartsSection_RootIsColumn`, `_ContainsControlsThenValueCard` |
| Controls move out of the value card | Task 3 `TestBuildValueOverTimeCard_DoesNotContainControls` |
| Each card has a localized title text + line_chart | Task 3 `TestBuildValueOverTimeCard_HasTitleAndChart`, Task 4 `TestBuildAssetValueOverTimeCard_HasTitle`, `_IsCard` |
| Reload `target_id` is `charts-section` | Task 3 `TestBuildChartsSection_ButtonActionTargetsChartsSection`, Task 3 Step 4 handler test update |
| Asset series one per distinct ticker, ordered by latest value DESC, ticker ASC tiebreak | Task 4 `TestBuildAssetValueOverTimeCard_SeriesPerTicker` |
| Colors cycle through `chart_1..chart_5` | Task 4 `TestBuildAssetValueOverTimeCard_SeriesPerTicker` |
| All asset series use `currency_compact` | Task 4 `TestBuildAssetValueOverTimeCard_SeriesPerTicker`, `_IgnoresMode` |
| Data rows carry key per ticker, null for absent in snapshot | Task 4 `TestBuildAssetValueOverTimeCard_DataRowsHaveAllTickerKeys` |
| `mode=pct` does not affect asset chart | Task 4 `TestBuildAssetValueOverTimeCard_IgnoresMode` |
| Asset chart filters by currency | Task 4 `TestBuildAssetValueOverTimeCard_FiltersByCurrency` |
| Empty state when <2 points or no tickers | Task 4 `TestBuildAssetValueOverTimeCard_EmptyWhenLessThanTwoPoints` |
| Empty portfolio hides `charts-section` | Task 3 `TestBuildScreen_ChartsSectionAbsentWhenEmpty` |
| i18n key for asset chart title | Task 2 |

**Placeholder scan:** none. (Step 3 of Task 4 had layered indirection on `sortSliceStable` but the final replacement simplifies to a single `sort.SliceStable` call.)

**Type consistency:**
- `AssetValue{AssetID, Ticker, Value}` — used in Task 1 and Task 4.
- `BuildAssetValueOverTimeCard(points, state, lang)` — same signature in Task 4 implementation, tests, and Task 4 Step 4 wiring.
- `BuildValueOverTimeCard(points, state, lang)` — signature changed vs 4a (dropped `currencies`); consistent in Task 3 impl + tests, Task 3 handler update, and Task 3 screen integration.
- `BuildChartsSection(points, state, currencies, lang)` — consistent in Task 3 impl + tests, Task 3 handler update, Task 4 wiring.
- Component IDs: `charts-section`, `chart-value-over-time-card`, `chart-value-over-time-title`, `chart-asset-value-over-time-card`, `chart-asset-value-over-time-title`, `chart-asset-value-over-time` — consistent across all tasks.
