# Portfolio Layer 4c Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `spec/screens/portfolio/04c-allocation.md` — add an allocation (pie chart) section at the end of `portfolio-root` with its own controls (group by asset/type + currency) backed by a new `GET /actions/portfolio/allocation` reload endpoint. Introduces the `pie_chart` custom component per `spec/sdui-custom-components.md §2`.

**Architecture:** New `PieChart` helper + `Slice` type in `internal/components`. New `AllocationState`, `BuildAllocationSection`, `BuildAllocationCard` in `internal/portfolio/allocation_builder.go` — pure, reusable by both the initial screen render and the action handler. New handler in `allocation_handler.go` that fetches positions via the existing client. `BuildScreen` appends `allocation-section` after `charts-section` when positions are non-empty.

**Tech Stack:** Go, Gin, testify, existing `internal/components`, `internal/portfolio`.

---

## File Structure

**Create:**

| File | Responsibility |
|---|---|
| `internal/components/pie.go` | `PieChart` helper + `Slice` type |
| `internal/components/pie_test.go` | JSON shape of emitted `pie_chart` |
| `internal/portfolio/allocation_builder.go` | `AllocationState`, `BuildAllocationSection`, pure |
| `internal/portfolio/allocation_builder_test.go` | controls order, selected state, URL encoding, slice computation per group_by, empty state |
| `internal/portfolio/allocation_handler.go` | `GET /actions/portfolio/allocation` |
| `internal/portfolio/allocation_handler_test.go` | success, 400, 401, 502 |

**Modify:**

| File | Change |
|---|---|
| `internal/portfolio/builder.go` | `BuildScreen` appends `allocation-section` after `charts-section` when positions non-empty |
| `internal/portfolio/builder_test.go` | assert `allocation-section` present/absent |
| `internal/server/server.go` | register protected `GET /actions/portfolio/allocation` |
| `locales/en.json`, `locales/es.json` | add `portfolio.allocation.*` keys |

---

### Task 1: `pie_chart` component helper

**Files:**
- Create: `internal/components/pie.go`
- Create: `internal/components/pie_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/components/pie_test.go`:

```go
package components

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPieChart_EmitsTypeAndID(t *testing.T) {
	c := PieChart("x", "md", "donut", "currency_compact",
		[]Slice{{Key: "s1", Label: "S1", Value: 100, Color: "chart_1"}},
		true,
		"",
	)
	assert.Equal(t, "pie_chart", c.Type)
	assert.Equal(t, "x", c.ID)
}

func TestPieChart_AllPropsPresent(t *testing.T) {
	slices := []Slice{{Key: "s1", Label: "S1", Value: 100, Color: "chart_1"}}
	c := PieChart("x", "md", "donut", "currency_compact", slices, true, "No data")
	assert.Equal(t, "md", c.Props["height"])
	assert.Equal(t, "donut", c.Props["shape"])
	assert.Equal(t, "currency_compact", c.Props["value_format"])
	assert.Equal(t, true, c.Props["show_legend"])
	assert.Equal(t, "No data", c.Props["empty_message"])
	_, ok := c.Props["slices"].([]Slice)
	assert.True(t, ok)
}

func TestPieChart_OmitsEmptyHeightAndMessage(t *testing.T) {
	c := PieChart("x", "", "pie", "integer",
		[]Slice{{Key: "s", Label: "S", Value: 1, Color: "chart_1"}},
		false,
		"",
	)
	_, hasHeight := c.Props["height"]
	assert.False(t, hasHeight)
	_, hasEmpty := c.Props["empty_message"]
	assert.False(t, hasEmpty)
	assert.Equal(t, false, c.Props["show_legend"])
}

func TestPieChart_JSONShape(t *testing.T) {
	c := PieChart("chart-allocation", "md", "donut", "currency_compact",
		[]Slice{
			{Key: "AAPL", Label: "AAPL", Value: 12500, Color: "chart_1"},
			{Key: "MSFT", Label: "MSFT", Value: 8200, Color: "chart_2"},
		},
		true,
		"No positions with known value",
	)
	b, err := json.Marshal(c)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"type":"pie_chart"`)
	assert.Contains(t, s, `"id":"chart-allocation"`)
	assert.Contains(t, s, `"shape":"donut"`)
	assert.Contains(t, s, `"value_format":"currency_compact"`)
	assert.Contains(t, s, `"show_legend":true`)
	assert.Contains(t, s, `"slices":[{"key":"AAPL","label":"AAPL","value":12500,"color":"chart_1"},{"key":"MSFT","label":"MSFT","value":8200,"color":"chart_2"}]`)
	assert.Contains(t, s, `"empty_message":"No positions with known value"`)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/components/... -run TestPieChart -v`
Expected: FAIL — `PieChart` / `Slice` undefined.

- [ ] **Step 3: Implement**

Create `internal/components/pie.go`:

```go
package components

// Slice is one slice in a pie_chart.
type Slice struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Color string  `json:"color"`
}

// PieChart creates a pie_chart custom component. See
// spec/sdui-custom-components.md §2.
//
// Pass empty string for height / emptyMessage to omit those props.
// show_legend is always included in the payload.
func PieChart(id, height, shape, valueFormat string, slices []Slice, showLegend bool, emptyMessage string) Component {
	props := map[string]any{
		"shape":        shape,
		"value_format": valueFormat,
		"slices":       slices,
		"show_legend":  showLegend,
	}
	if height != "" {
		props["height"] = height
	}
	if emptyMessage != "" {
		props["empty_message"] = emptyMessage
	}
	return Component{
		Type:  "pie_chart",
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
git add internal/components/pie.go internal/components/pie_test.go
git commit -m "feat(components): pie_chart custom SDUI component helper"
```

---

### Task 2: i18n keys

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

- [ ] **Step 1: Add keys to `locales/en.json`**

Find the `"portfolio"` object. Inside it, after the last existing key (likely the closing of `"chart"`), and before the closing `}` of `"portfolio"`, add:

```json
    "allocation": {
      "title": "Allocation",
      "group_by": {
        "asset": "By asset",
        "type": "By type"
      },
      "empty": "No positions with known value"
    }
```

Make sure the trailing commas are correct (the previous key should have a trailing comma).

- [ ] **Step 2: Add keys to `locales/es.json`**

Same position, Spanish values:

```json
    "allocation": {
      "title": "Distribución",
      "group_by": {
        "asset": "Por activo",
        "type": "Por tipo"
      },
      "empty": "Sin posiciones con valor conocido"
    }
```

- [ ] **Step 3: Run full suite**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./... -count=1`
Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "feat(i18n): portfolio.allocation.* keys"
```

---

### Task 3: Allocation builder (pure)

**Files:**
- Create: `internal/portfolio/allocation_builder.go`
- Create: `internal/portfolio/allocation_builder_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/allocation_builder_test.go`:

```go
package portfolio

import (
	"testing"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func samplePositionsForAllocation() []Position {
	v1, v2, v3 := 1200.0, 800.0, 400.0
	return []Position{
		{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v1},
		{AssetID: "a2", Ticker: "BND",  AssetType: "BOND",  Currency: "USD", CurrentValue: &v2},
		{AssetID: "a3", Ticker: "TSLA", AssetType: "STOCK", Currency: "USD", CurrentValue: &v3},
	}
}

func TestBuildAllocationSection_RootIsColumn(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	assert.Equal(t, "column", s.Type)
	assert.Equal(t, "allocation-section", s.ID)
}

func TestBuildAllocationSection_HasControlsThenCard(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	require.Len(t, s.Children, 2)
	assert.Equal(t, "allocation-controls-row", s.Children[0].ID)
	assert.Equal(t, "chart-allocation-card", s.Children[1].ID)
}

func TestBuildAllocationSection_GroupByControlsHaveTwoButtons(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	gb := findDescendantByID(s, "allocation-group-by-controls")
	require.NotNil(t, gb)
	require.Len(t, gb.Children, 2)
	assert.Equal(t, "allocation-group-by-asset", gb.Children[0].ID)
	assert.Equal(t, "allocation-group-by-type", gb.Children[1].ID)
}

func TestBuildAllocationSection_SelectedGroupByHasSolidStyle(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "type", Currency: "USD"}, []string{"USD"}, "en")
	selected := findDescendantByID(s, "allocation-group-by-type")
	require.NotNil(t, selected)
	assert.Equal(t, "primary", selected.Props["variant"])
	assert.Equal(t, "solid", selected.Props["style"])
	unselected := findDescendantByID(s, "allocation-group-by-asset")
	require.NotNil(t, unselected)
	assert.Equal(t, "secondary", unselected.Props["variant"])
	assert.Equal(t, "ghost", unselected.Props["style"])
}

func TestBuildAllocationSection_ButtonActionTargetsAllocationSection(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	btn := findDescendantByID(s, "allocation-group-by-type")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "click", a.Trigger)
	assert.Equal(t, "reload", a.Type)
	assert.Equal(t, "allocation-section", a.TargetID)
	assert.Contains(t, a.Endpoint, "group_by=type")
	assert.Contains(t, a.Endpoint, "currency=USD")
}

func TestBuildAllocationSection_CurrencyControlsHiddenWhenSingle(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	assert.Nil(t, findDescendantByID(s, "currency-controls"))
}

func TestBuildAllocationSection_CurrencyControlsShownWhenMulti(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD", "EUR"}, "en")
	cc := findDescendantByID(s, "currency-controls")
	require.NotNil(t, cc)
	require.Len(t, cc.Children, 2)
}

func TestBuildAllocationSection_GroupByAssetSlices(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(s, "chart-allocation")
	require.NotNil(t, chart)
	slices, ok := chart.Props["slices"].([]components.Slice)
	require.True(t, ok)
	// Expect 3 slices sorted DESC by value: AAPL(1200), BND(800), TSLA(400).
	require.Len(t, slices, 3)
	assert.Equal(t, "a1", slices[0].Key)
	assert.Equal(t, "AAPL", slices[0].Label)
	assert.InDelta(t, 1200.0, slices[0].Value, 1e-9)
	assert.Equal(t, "chart_1", slices[0].Color)

	assert.Equal(t, "a2", slices[1].Key)
	assert.Equal(t, "BND", slices[1].Label)
	assert.InDelta(t, 800.0, slices[1].Value, 1e-9)
	assert.Equal(t, "chart_2", slices[1].Color)

	assert.Equal(t, "a3", slices[2].Key)
	assert.Equal(t, "TSLA", slices[2].Label)
	assert.InDelta(t, 400.0, slices[2].Value, 1e-9)
	assert.Equal(t, "chart_3", slices[2].Color)
}

func TestBuildAllocationSection_GroupByTypeSlices(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "type", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(s, "chart-allocation")
	require.NotNil(t, chart)
	slices, ok := chart.Props["slices"].([]components.Slice)
	require.True(t, ok)
	// STOCK = AAPL+TSLA = 1600, BOND = BND = 800. DESC by value.
	require.Len(t, slices, 2)
	assert.Equal(t, "STOCK", slices[0].Key)
	assert.Equal(t, "STOCK", slices[0].Label)
	assert.InDelta(t, 1600.0, slices[0].Value, 1e-9)
	assert.Equal(t, "BOND", slices[1].Key)
	assert.InDelta(t, 800.0, slices[1].Value, 1e-9)
}

func TestBuildAllocationSection_FiltersByCurrency(t *testing.T) {
	v := 2000.0
	positions := append(samplePositionsForAllocation(),
		Position{AssetID: "e1", Ticker: "SAP", AssetType: "STOCK", Currency: "EUR", CurrentValue: &v},
	)
	s := BuildAllocationSection(positions, AllocationState{GroupBy: "asset", Currency: "EUR"}, []string{"USD", "EUR"}, "en")
	chart := findDescendantByID(s, "chart-allocation")
	require.NotNil(t, chart)
	slices, ok := chart.Props["slices"].([]components.Slice)
	require.True(t, ok)
	require.Len(t, slices, 1)
	assert.Equal(t, "e1", slices[0].Key)
	assert.Equal(t, "SAP", slices[0].Label)
	assert.InDelta(t, 2000.0, slices[0].Value, 1e-9)
}

func TestBuildAllocationSection_EmptyWhenNoPositionsWithValue(t *testing.T) {
	positions := []Position{
		{AssetID: "n1", Ticker: "NULL1", AssetType: "COMPLEX", Currency: "USD"}, // no current_value
	}
	s := BuildAllocationSection(positions, AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(s, "chart-allocation")
	require.NotNil(t, chart)
	slices, ok := chart.Props["slices"].([]components.Slice)
	require.True(t, ok)
	assert.Empty(t, slices)
	assert.Equal(t, "No positions with known value", chart.Props["empty_message"])
}

func TestBuildAllocationSection_CardContainsTitleAndPie(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	card := findDescendantByID(s, "chart-allocation-card")
	require.NotNil(t, card)
	title := findDescendantByID(*card, "chart-allocation-title")
	require.NotNil(t, title)
	assert.Equal(t, "Allocation", title.Props["content"])
	chart := findDescendantByID(*card, "chart-allocation")
	require.NotNil(t, chart)
	assert.Equal(t, "pie_chart", chart.Type)
	assert.Equal(t, "donut", chart.Props["shape"])
	assert.Equal(t, "currency_compact", chart.Props["value_format"])
	assert.Equal(t, true, chart.Props["show_legend"])
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/portfolio/... -run TestBuildAllocationSection -v`
Expected: FAIL — `BuildAllocationSection` / `AllocationState` undefined.

- [ ] **Step 3: Implement**

Create `internal/portfolio/allocation_builder.go`:

```go
package portfolio

import (
	"net/url"
	"sort"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// AllocationState holds the user's selections for the allocation section.
type AllocationState struct {
	GroupBy  string // "asset" | "type"
	Currency string
}

var allocationColors = []string{"chart_1", "chart_2", "chart_3", "chart_4", "chart_5"}

// BuildAllocationSection wraps controls + the pie chart card in a column
// serving as the reload target. Pure.
func BuildAllocationSection(positions []Position, state AllocationState, currencies []string, lang string) components.Component {
	controls := buildAllocationControls(state, currencies, lang)
	card := buildAllocationCard(positions, state, lang)
	return components.ColumnWithGap("allocation-section", "lg", controls, card)
}

func buildAllocationControls(state AllocationState, currencies []string, lang string) components.Component {
	gb := buildGroupByControls(state, lang)
	spacer := components.Column("allocation-controls-spacer")

	var children []components.Component
	var widths []string
	if len(currencies) > 1 {
		children = []components.Component{buildAllocationCurrencyControls(state, currencies), spacer, gb}
		widths = []string{"auto", "1fr", "auto"}
	} else {
		children = []components.Component{spacer, gb}
		widths = []string{"1fr", "auto"}
	}
	row := components.Row("allocation-controls-row", widths, children...)
	row.Props["gap"] = "lg"
	return row
}

func buildGroupByControls(state AllocationState, lang string) components.Component {
	asset := allocationButton("allocation-group-by-asset",
		i18n.T(lang, "portfolio.allocation.group_by.asset"),
		state.GroupBy == "asset",
		allocationURL("asset", state.Currency),
	)
	typ := allocationButton("allocation-group-by-type",
		i18n.T(lang, "portfolio.allocation.group_by.type"),
		state.GroupBy == "type",
		allocationURL("type", state.Currency),
	)
	row := components.Row("allocation-group-by-controls", []string{"auto", "auto"}, asset, typ)
	row.Props["gap"] = "sm"
	return row
}

func buildAllocationCurrencyControls(state AllocationState, currencies []string) components.Component {
	btns := make([]components.Component, 0, len(currencies))
	for _, c := range currencies {
		btns = append(btns, allocationButton(
			"chart-currency-"+c,
			c,
			c == state.Currency,
			allocationURL(state.GroupBy, c),
		))
	}
	widths := make([]string, len(btns))
	for i := range widths {
		widths[i] = "auto"
	}
	row := components.Row("currency-controls", widths, btns...)
	row.Props["gap"] = "sm"
	return row
}

func allocationButton(id, label string, selected bool, endpoint string) components.Component {
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
			{Trigger: "click", Type: "reload", Endpoint: endpoint, TargetID: "allocation-section"},
		},
	}
}

func allocationURL(groupBy, currency string) string {
	q := url.Values{}
	q.Set("group_by", groupBy)
	if currency != "" {
		q.Set("currency", currency)
	}
	return "/actions/portfolio/allocation?" + q.Encode()
}

func buildAllocationCard(positions []Position, state AllocationState, lang string) components.Component {
	title := components.Text("chart-allocation-title", i18n.T(lang, "portfolio.allocation.title"), "md", "bold")
	slices := computeAllocationSlices(positions, state)
	emptyMessage := ""
	if len(slices) == 0 {
		emptyMessage = i18n.T(lang, "portfolio.allocation.empty")
	}
	chart := components.PieChart(
		"chart-allocation",
		"md",
		"donut",
		"currency_compact",
		slices,
		true,
		emptyMessage,
	)
	content := components.ColumnWithGap("chart-allocation-content", "md", title, chart)
	return components.Card("chart-allocation-card", content)
}

// computeAllocationSlices filters positions by currency + non-null current_value,
// groups by (asset|type), sums values, returns sorted DESC by value (ties by label ASC).
func computeAllocationSlices(positions []Position, state AllocationState) []components.Slice {
	type agg struct {
		key   string
		label string
		value float64
	}
	aggMap := map[string]*agg{}
	order := []string{}

	for _, p := range positions {
		if p.Currency != state.Currency || p.CurrentValue == nil {
			continue
		}
		var key, label string
		if state.GroupBy == "type" {
			key = p.AssetType
			label = p.AssetType
		} else {
			key = p.AssetID
			label = p.Ticker
		}
		if existing, ok := aggMap[key]; ok {
			existing.value += *p.CurrentValue
		} else {
			aggMap[key] = &agg{key: key, label: label, value: *p.CurrentValue}
			order = append(order, key)
		}
	}

	aggs := make([]*agg, 0, len(order))
	for _, k := range order {
		aggs = append(aggs, aggMap[k])
	}
	sort.SliceStable(aggs, func(i, j int) bool {
		if aggs[i].value == aggs[j].value {
			return aggs[i].label < aggs[j].label
		}
		return aggs[i].value > aggs[j].value
	})

	out := make([]components.Slice, 0, len(aggs))
	for i, a := range aggs {
		out = append(out, components.Slice{
			Key:   a.key,
			Label: a.label,
			Value: a.value,
			Color: allocationColors[i%len(allocationColors)],
		})
	}
	return out
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/allocation_builder.go internal/portfolio/allocation_builder_test.go
git commit -m "feat(portfolio): BuildAllocationSection with pie chart and controls"
```

---

### Task 4: Allocation handler

**Files:**
- Create: `internal/portfolio/allocation_handler.go`
- Create: `internal/portfolio/allocation_handler_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/portfolio/allocation_handler_test.go`:

```go
package portfolio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAllocationFetcher struct {
	positions []Position
	err       error
	gotAuth   string
	called    bool
}

func (s *stubAllocationFetcher) GetPositions(ctx context.Context, auth string, includeClosed bool) ([]Position, error) {
	s.called = true
	s.gotAuth = auth
	return s.positions, s.err
}

func setupAllocationRouter(f allocationFetcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/actions/portfolio/allocation", NewAllocationHandler(f).Get)
	return r
}

func allocationGet(t *testing.T, r *gin.Engine, query, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/actions/portfolio/allocation?"+query, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAllocationHandler_SuccessReturnsReplaceActionResponse(t *testing.T) {
	v := 1000.0
	f := &stubAllocationFetcher{positions: []Position{
		{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
	}}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "group_by=asset&currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "allocation-section", resp["target_id"])
	tree, ok := resp["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "column", tree["type"])
	assert.Equal(t, "allocation-section", tree["id"])

	assert.True(t, f.called)
	assert.Equal(t, "Bearer tok", f.gotAuth)
}

func TestAllocationHandler_DefaultsGroupByAsset(t *testing.T) {
	v := 1000.0
	f := &stubAllocationFetcher{positions: []Position{
		{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v},
	}}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "currency=USD", "Bearer tok")
	require.Equal(t, http.StatusOK, w.Code)
	// The tree should contain the asset group-by button selected solid.
	assert.Contains(t, w.Body.String(), `"allocation-group-by-asset"`)
}

func TestAllocationHandler_InvalidGroupByReturns400(t *testing.T) {
	f := &stubAllocationFetcher{}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "group_by=xxx&currency=USD", "Bearer tok")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, f.called)
}

func TestAllocationHandler_BackendUnauthorizedReturns401WithRedirect(t *testing.T) {
	f := &stubAllocationFetcher{err: ErrUnauthorized}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "group_by=asset&currency=USD", "Bearer x")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"unauthorized"`)
	assert.Contains(t, w.Body.String(), `"redirect":"/screens/login"`)
}

func TestAllocationHandler_BackendErrorReturns502(t *testing.T) {
	f := &stubAllocationFetcher{err: ErrBackend}
	r := setupAllocationRouter(f)

	w := allocationGet(t, r, "group_by=asset&currency=USD", "Bearer x")
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "BACKEND_ERROR")
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/portfolio/... -run TestAllocationHandler -v`
Expected: FAIL — `NewAllocationHandler` / `allocationFetcher` undefined.

- [ ] **Step 3: Implement**

Create `internal/portfolio/allocation_handler.go`:

```go
package portfolio

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// allocationFetcher is the narrow interface the handler needs; *Client satisfies it.
type allocationFetcher interface {
	GetPositions(ctx context.Context, authorization string, includeClosed bool) ([]Position, error)
}

type AllocationHandler struct {
	client allocationFetcher
}

func NewAllocationHandler(client allocationFetcher) *AllocationHandler {
	return &AllocationHandler{client: client}
}

// Get handles GET /actions/portfolio/allocation.
func (h *AllocationHandler) Get(c *gin.Context) {
	groupBy := c.DefaultQuery("group_by", "asset")
	currency := c.Query("currency")

	if groupBy != "asset" && groupBy != "type" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid group_by"}})
		return
	}

	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	positions, err := h.client.GetPositions(c.Request.Context(), auth, false)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load allocation"}})
		return
	}

	state := AllocationState{GroupBy: groupBy, Currency: currency}
	currencies := distinctPositionCurrencies(positions)
	tree := BuildAllocationSection(positions, state, currencies, lang)

	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "allocation-section",
		Tree:     &tree,
	})
}

// distinctPositionCurrencies returns currencies present in positions (with
// non-null current_value), ordered by each currency's total value DESC. Matches
// the ordering used on the initial screen render so reloads preserve it.
func distinctPositionCurrencies(positions []Position) []string {
	totals := map[string]float64{}
	for _, p := range positions {
		if p.CurrentValue == nil {
			continue
		}
		totals[p.Currency] += *p.CurrentValue
	}
	out := make([]string, 0, len(totals))
	for c := range totals {
		out = append(out, c)
	}
	sortStringsByDescAndTieAsc(out, totals)
	return out
}

func sortStringsByDescAndTieAsc(s []string, values map[string]float64) {
	// Stable-sort by values DESC, tie by string ASC.
	// Using sort.SliceStable from the allocation_builder's import via package scope
	// is not guaranteed; use a local variant.
	n := len(s)
	for i := 1; i < n; i++ {
		for j := i; j > 0; j-- {
			a, b := s[j-1], s[j]
			va, vb := values[a], values[b]
			if vb > va || (vb == va && b < a) {
				s[j-1], s[j] = b, a
			} else {
				break
			}
		}
	}
}
```

Note: the final `sortStringsByDescAndTieAsc` is an insertion sort to avoid importing `sort` in this file (insertion is fine for small N). Alternatively import `sort` and use `sort.SliceStable`. Either is acceptable; prefer `sort.SliceStable` if the file already imports it:

If you choose `sort.SliceStable`, replace the last function with:

```go
import "sort"  // add to imports if needed

func sortStringsByDescAndTieAsc(s []string, values map[string]float64) {
	sort.SliceStable(s, func(i, j int) bool {
		vi, vj := values[s[i]], values[s[j]]
		if vi == vj {
			return s[i] < s[j]
		}
		return vi > vj
	})
}
```

Use whichever you prefer; the tests only care about the result.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/allocation_handler.go internal/portfolio/allocation_handler_test.go
git commit -m "feat(portfolio): GET /actions/portfolio/allocation handler"
```

---

### Task 5: Integrate `allocation-section` into `BuildScreen`

**Files:**
- Modify: `internal/portfolio/builder.go`
- Modify: `internal/portfolio/builder_test.go`

- [ ] **Step 1: Add failing tests**

Append to `internal/portfolio/builder_test.go`:

```go
func TestBuildScreen_AllocationSectionPresentWhenPositions(t *testing.T) {
	ps := samplePositions()
	s := BuildScreen(ps, nil, nil, "en", time.Now())
	assert.NotNil(t, findDescendantByID(s, "allocation-section"))
}

func TestBuildScreen_AllocationSectionAbsentWhenEmpty(t *testing.T) {
	s := BuildScreen(nil, nil, nil, "en", time.Now())
	assert.Nil(t, findDescendantByID(s, "allocation-section"))
}

func TestBuildScreen_AllocationSectionIsLastInRoot(t *testing.T) {
	s := BuildScreen(samplePositions(), nil, nil, "en", time.Now())
	root := findDescendantByID(s, "portfolio-root")
	require.NotNil(t, root)
	require.GreaterOrEqual(t, len(root.Children), 1)
	last := root.Children[len(root.Children)-1]
	assert.Equal(t, "allocation-section", last.ID)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/portfolio/... -run TestBuildScreen_Allocation -v`
Expected: FAIL — `allocation-section` not present in tree.

- [ ] **Step 3: Update `BuildScreen` in `builder.go`**

In `internal/portfolio/builder.go`, find the non-empty branch of `BuildScreen`:

```go
	chartsSection := buildInitialChartsSection(chartPoints, positions, lang)
	root := components.ColumnWithGap("portfolio-root", "lg", summary, controls, table, chartsSection)
```

Replace with:

```go
	chartsSection := buildInitialChartsSection(chartPoints, positions, lang)
	allocation := buildInitialAllocationSection(positions, lang)
	root := components.ColumnWithGap("portfolio-root", "lg", summary, controls, table, chartsSection, allocation)
```

Append this helper to the end of `builder.go`:

```go
// buildInitialAllocationSection produces the allocation-section for the initial
// screen render. Defaults group_by=asset and currency=<first by total value DESC>.
func buildInitialAllocationSection(positions []Position, lang string) components.Component {
	metrics := ComputeMetrics(positions, nil)
	currencies := metrics.CurrencyOrder
	defaultCurrency := ""
	if len(currencies) > 0 {
		defaultCurrency = currencies[0]
	}
	state := AllocationState{GroupBy: "asset", Currency: defaultCurrency}
	return BuildAllocationSection(positions, state, currencies, lang)
}
```

- [ ] **Step 4: Run full suite**

Run: `go test ./... -count=1`
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/builder.go internal/portfolio/builder_test.go
git commit -m "feat(portfolio): allocation-section in initial screen render"
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
	protected.GET("/actions/portfolio/evolution", portfolio.NewEvolutionHandler(portfolioClient).Get)
```

Append:

```go
	protected.GET("/actions/portfolio/allocation", portfolio.NewAllocationHandler(portfolioClient).Get)
```

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -count=1`
Expected: all pass.

- [ ] **Step 3: Build and lint**

Run: `./cli build 2>&1 | tail -1 && ./cli lint 2>&1 | tail -1`
Expected: both `"status":"success"`.

- [ ] **Step 4: Smoke-test**

Run and report output verbatim:

```bash
cd /Users/vadimkent/repos/vk_investment_middleend_v2
lsof -ti:8082 | xargs kill -9 2>/dev/null; sleep 1
./cli run >/tmp/srv.log 2>&1 &
sleep 2

RESP=$(curl -s -X POST http://localhost:8082/actions/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@demo.com","password":"demo"}')
TOKEN=$(echo "$RESP" | python3 -c "import json,sys;print(json.load(sys.stdin)['auth']['token'])")

echo "--- portfolio screen has allocation-section as last ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
root = None
def find(x, id):
    if x.get('id') == id: return x
    for c in x.get('children', []):
        r = find(c, id)
        if r: return r
root = find(d, 'portfolio-root')
print('last child id:', root['children'][-1]['id'])
ids = [c['id'] for c in root['children']]
print('portfolio-root children:', ids)
"

echo "--- GET /actions/portfolio/allocation?group_by=asset&currency=USD ---"
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8082/actions/portfolio/allocation?group_by=asset&currency=USD" \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
print('action:', d.get('action'), 'target_id:', d.get('target_id'), 'tree.id:', d.get('tree',{}).get('id'), 'tree.type:', d.get('tree',{}).get('type'))
"

echo "--- allocation slices count (group_by=asset, currency=UYU) ---"
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8082/actions/portfolio/allocation?group_by=asset&currency=UYU" \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def walk(x):
    if x.get('id') == 'chart-allocation':
        slices = x['props'].get('slices', [])
        print('slices count:', len(slices), 'keys:', [s.get('label') for s in slices])
        return True
    for c in x.get('children', []):
        if walk(c): return True
walk(d.get('tree', {}))
"

echo "--- invalid group_by returns 400 ---"
curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8082/actions/portfolio/allocation?group_by=xxx&currency=USD"

lsof -ti:8082 | xargs kill -9 2>/dev/null; true
```

Expected:
- Last child of `portfolio-root` is `allocation-section`.
- Action response: `action: replace target_id: allocation-section tree.id: allocation-section tree.type: column`.
- Allocation slices count matches the number of distinct assets in the user's UYU portfolio.
- Invalid `group_by`: `400`.

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go
git commit -m "feat(server): wire protected GET /actions/portfolio/allocation"
```

---

## Self-Review Results

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| `pie_chart` custom component with documented props | Task 1 |
| `GET /screens/portfolio` appends `allocation-section` after `charts-section` | Task 5 `TestBuildScreen_AllocationSectionIsLastInRoot` |
| Empty positions → no `allocation-section` | Task 5 `TestBuildScreen_AllocationSectionAbsentWhenEmpty` |
| `allocation-section` has `allocation-controls-row` + `chart-allocation-card` | Task 3 `TestBuildAllocationSection_HasControlsThenCard` |
| Card has title + pie chart | Task 3 `TestBuildAllocationSection_CardContainsTitleAndPie` |
| `allocation-group-by-controls` has 2 buttons (asset + type) | Task 3 `TestBuildAllocationSection_GroupByControlsHaveTwoButtons` |
| Selected button styled `primary/solid`; others `secondary/ghost` | Task 3 `TestBuildAllocationSection_SelectedGroupByHasSolidStyle` |
| Button action `reload` with `target_id: allocation-section` and encoded state | Task 3 `TestBuildAllocationSection_ButtonActionTargetsAllocationSection` |
| Currency controls hidden when single, shown when multi | Task 3 `TestBuildAllocationSection_CurrencyControls*` |
| Initial state: `group_by=asset, currency=<first DESC>` | Task 5 helper `buildInitialAllocationSection`; covered by the presence test |
| `group_by=asset` → slice per `asset_id`, label=ticker, value=sum current_value | Task 3 `TestBuildAllocationSection_GroupByAssetSlices` |
| `group_by=type` → slice per `asset_type`, value=sum | Task 3 `TestBuildAllocationSection_GroupByTypeSlices` |
| Filter by currency + current_value != nil | Task 3 `TestBuildAllocationSection_FiltersByCurrency` |
| Sort slices DESC by value, tie by label ASC | Task 3 `TestBuildAllocationSection_GroupByAssetSlices`, `_GroupByTypeSlices` |
| Colors cycle `chart_1..chart_5` | Task 3 `TestBuildAllocationSection_GroupByAssetSlices` |
| Pie config: `shape: donut`, `value_format: currency_compact`, `show_legend: true` | Task 3 `TestBuildAllocationSection_CardContainsTitleAndPie` |
| Empty slices → `empty_message` | Task 3 `TestBuildAllocationSection_EmptyWhenNoPositionsWithValue` |
| `GET /actions/portfolio/allocation` returns `ActionResponse{replace}` | Task 4 `TestAllocationHandler_SuccessReturnsReplaceActionResponse` |
| Default `group_by=asset` when omitted | Task 4 `TestAllocationHandler_DefaultsGroupByAsset` |
| Invalid `group_by` → 400 | Task 4 `TestAllocationHandler_InvalidGroupByReturns400` |
| BE 401 → 401 + redirect | Task 4 `TestAllocationHandler_BackendUnauthorizedReturns401WithRedirect` |
| BE 5xx → 502 | Task 4 `TestAllocationHandler_BackendErrorReturns502` |
| `Authorization` forwarded | Task 4 success test (asserts `gotAuth`) |
| i18n keys in both locales | Task 2 |
| Route registered | Task 6 |

**Placeholder scan:** none. (Task 4's alternative `sort.SliceStable` section is a choice, not a placeholder — the default insertion sort works and there is a clearly described alternative.)

**Type consistency:**
- `AllocationState{GroupBy, Currency}` — consistent across Task 3 builder, tests, Task 4 handler.
- `Slice{Key, Label, Value, Color}` — consistent across Task 1 components and Task 3 builder.
- `PieChart(id, height, shape, valueFormat string, slices []Slice, showLegend bool, emptyMessage string) Component` — same signature in Task 1 impl, tests, Task 3 builder use.
- `BuildAllocationSection(positions, state, currencies, lang)` — consistent in Task 3 impl + tests, Task 4 handler, Task 5 integration.
- Component IDs: `allocation-section`, `allocation-controls-row`, `allocation-controls-spacer`, `allocation-group-by-controls`, `allocation-group-by-asset`, `allocation-group-by-type`, `chart-allocation-card`, `chart-allocation-content`, `chart-allocation-title`, `chart-allocation`, `currency-controls`, `chart-currency-<CODE>` — consistent across tasks.
