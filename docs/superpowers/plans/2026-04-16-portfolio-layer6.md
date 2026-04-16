# Portfolio Layer 6 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `spec/screens/portfolio/06-polish.md` — migrate positions from `list`/`list_item`/`row` to `table`/`table_row` (CSS subgrid alignment), add a HideValues `icon_toggle` with client-side `toggle_sensitive` action, and mark monetary values with the `sensitive: true` custom attribute.

**Architecture:** New `Table`/`TableRow`/`TableColumn` helpers in `internal/components`. Rewrite `BuildPositionsTable` to emit `table`+`table_row` instead of `row`+`list`+`list_item`. Add `hide-values-toggle` to the header row. Mark monetary texts with `sensitive: true`. No new endpoints — HideValues is client-side only.

**Tech Stack:** Go, testify, existing packages.

---

## File Structure

**Create:**

| File | Responsibility |
|---|---|
| `internal/components/table.go` | `TableColumn` type, `Table` helper, `TableRow` helper |
| `internal/components/table_test.go` | JSON shape tests |

**Modify:**

| File | Change |
|---|---|
| `internal/portfolio/chart_builder.go` | Rewrite `BuildPositionsTable` + `buildPositionItem` to use `Table`/`TableRow`; add `sensitive: true` on monetary cells |
| `internal/portfolio/chart_builder_test.go` | Update all structure assertions (table instead of list) |
| `internal/portfolio/live_builder.go` | Add hide-values toggle to `BuildPortfolioHeaderRow`; add `sensitive: true` to summary value texts |
| `internal/portfolio/live_builder_test.go` | Assert hide-values toggle + sensitive prop |
| `internal/portfolio/builder.go` | Add `sensitive: true` to summary card value texts in `buildTotalValueCard` and `buildTotalPnLCard` |
| `internal/portfolio/builder_test.go` | Assert sensitive on summary texts |
| `locales/{en,es}.json` | Add `portfolio.hide_values.*` keys |

---

### Task 1: `table` / `table_row` component helpers

**Files:**
- Create: `internal/components/table.go`
- Create: `internal/components/table_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/components/table_test.go`:

```go
package components

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTable_EmitsTypeAndColumns(t *testing.T) {
	cols := []TableColumn{
		{ID: "ticker", Header: "Ticker", Width: "80px"},
		{ID: "name", Header: "Name", Width: "1fr"},
		{ID: "value", Header: "Value", Width: "120px", Align: "right"},
	}
	c := Table("t1", cols,
		TableRow("r1", Text("c1", "AAPL", "sm", "bold")),
	)
	assert.Equal(t, "table", c.Type)
	assert.Equal(t, "t1", c.ID)
	assert.Equal(t, cols, c.Props["columns"])
	require.Len(t, c.Children, 1)
	assert.Equal(t, "table_row", c.Children[0].Type)
}

func TestTableRow_EmitsTypeAndChildren(t *testing.T) {
	r := TableRow("r1",
		Text("c1", "AAPL", "sm", "bold"),
		Text("c2", "Apple", "sm", "normal"),
	)
	assert.Equal(t, "table_row", r.Type)
	assert.Equal(t, "r1", r.ID)
	require.Len(t, r.Children, 2)
}

func TestTableColumn_OmitsEmptyWidthAndAlign(t *testing.T) {
	col := TableColumn{ID: "x", Header: "X"}
	b, err := json.Marshal(col)
	require.NoError(t, err)
	s := string(b)
	assert.NotContains(t, s, "width")
	assert.NotContains(t, s, "align")
}

func TestTableColumn_IncludesWidthAndAlign(t *testing.T) {
	col := TableColumn{ID: "v", Header: "Value", Width: "120px", Align: "right"}
	b, err := json.Marshal(col)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"width":"120px"`)
	assert.Contains(t, s, `"align":"right"`)
}

func TestTable_JSONShape(t *testing.T) {
	cols := []TableColumn{
		{ID: "ticker", Header: "Ticker", Width: "80px"},
	}
	c := Table("tbl", cols,
		TableRow("r1", Text("c1", "AAPL", "sm", "bold")),
	)
	b, err := json.Marshal(c)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"type":"table"`)
	assert.Contains(t, s, `"columns":[{"id":"ticker","header":"Ticker","width":"80px"}]`)
	assert.Contains(t, s, `"type":"table_row"`)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/components/... -run TestTable -v`
Expected: FAIL — `Table`/`TableRow`/`TableColumn` undefined.

- [ ] **Step 3: Implement**

Create `internal/components/table.go`:

```go
package components

// TableColumn defines a single column in a table.
type TableColumn struct {
	ID     string `json:"id"`
	Header string `json:"header"`
	Width  string `json:"width,omitempty"`
	Align  string `json:"align,omitempty"`
}

// Table creates a table component. The frontend renders the header
// automatically from columns[].Header. Children must be table_row components.
func Table(id string, columns []TableColumn, children ...Component) Component {
	return Component{
		Type:     "table",
		ID:       id,
		Props:    map[string]any{"columns": columns},
		Children: children,
	}
}

// TableRow creates a table_row component. Each child maps to a column
// in order via CSS subgrid.
func TableRow(id string, children ...Component) Component {
	return Component{
		Type:     "table_row",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/components/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/components/table.go internal/components/table_test.go
git commit -m "feat(components): table + table_row base SDUI components"
```

---

### Task 2: i18n keys for HideValues

**Files:**
- Modify: `locales/en.json`, `locales/es.json`

- [ ] **Step 1: Add to en.json**

Inside `"portfolio"`, after the `"live"` block, add:

```json
    "hide_values": {
      "tooltip_inactive": "Hide values",
      "tooltip_active": "Show values"
    }
```

- [ ] **Step 2: Add to es.json**

```json
    "hide_values": {
      "tooltip_inactive": "Ocultar valores",
      "tooltip_active": "Mostrar valores"
    }
```

- [ ] **Step 3: Run full suite**

`go test ./... -count=1`

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "feat(i18n): portfolio.hide_values.* keys"
```

---

### Task 3: Rewrite `BuildPositionsTable` to use `table`/`table_row`

**Files:**
- Modify: `internal/portfolio/chart_builder.go`
- Modify: `internal/portfolio/chart_builder_test.go`

- [ ] **Step 1: Define the column config**

In `chart_builder.go`, replace the existing `columnWidths` and `columnKeys` vars with a single `positionColumns` function:

```go
func positionColumns(lang string) []components.TableColumn {
	return []components.TableColumn{
		{ID: "ticker", Header: i18n.T(lang, "portfolio.col.ticker"), Width: "80px"},
		{ID: "name", Header: i18n.T(lang, "portfolio.col.name"), Width: "1fr"},
		{ID: "type", Header: i18n.T(lang, "portfolio.col.type"), Width: "80px"},
		{ID: "quantity", Header: i18n.T(lang, "portfolio.col.quantity"), Width: "80px", Align: "right"},
		{ID: "avg_cost", Header: i18n.T(lang, "portfolio.col.avg_cost"), Width: "110px", Align: "right"},
		{ID: "total_cost", Header: i18n.T(lang, "portfolio.col.total_cost"), Width: "110px", Align: "right"},
		{ID: "market_value", Header: i18n.T(lang, "portfolio.col.market_value"), Width: "120px", Align: "right"},
		{ID: "unrealized_pnl", Header: i18n.T(lang, "portfolio.col.unrealized_pnl"), Width: "130px", Align: "right"},
		{ID: "pnl_pct", Header: i18n.T(lang, "portfolio.col.pnl_pct"), Width: "80px", Align: "right"},
		{ID: "realized_pnl", Header: i18n.T(lang, "portfolio.col.realized_pnl"), Width: "120px", Align: "right"},
		{ID: "last_snapshot", Header: i18n.T(lang, "portfolio.col.last_snapshot"), Width: "120px", Align: "right"},
	}
}
```

- [ ] **Step 2: Rewrite `BuildPositionsTable`**

Replace the function body:

```go
func BuildPositionsTable(ps []Position, lang string, now time.Time, isLive bool) components.Component {
	cols := positionColumns(lang)
	rows := make([]components.Component, 0, len(ps))
	for _, p := range ps {
		rows = append(rows, buildPositionRow(p, lang, now, isLive))
	}
	table := components.Table("positions-table", cols, rows...)
	return components.Card("positions-table-card", table)
}
```

- [ ] **Step 3: Rewrite `buildPositionItem` → `buildPositionRow`**

Rename and simplify — no wrapping `Row`, direct `TableRow` with 11 text children. Add `sensitive: true` on monetary cells:

```go
func buildPositionRow(p Position, lang string, now time.Time, isLive bool) components.Component {
	realized := p.RealizedPnL
	pct := PnLPct(p.UnrealizedPnL, p.TotalCost)

	marketValueContent := FormatMoney(p.CurrentValue, p.Currency, lang)
	if isLive && p.PriceSource != nil {
		marketValueContent = priceSourceDot(*p.PriceSource) + " " + marketValueContent
	}

	cells := []components.Component{
		components.Text("cell-ticker", p.Ticker, "sm", "bold"),
		components.Text("cell-name", p.Name, "sm", "normal"),
		components.Text("cell-type", p.AssetType, "sm", "normal"),
		components.Text("cell-quantity", FormatQuantity(p.Quantity, lang), "sm", "normal"),
		sensitiveText("cell-avg-cost", FormatMoney(p.AvgCost, p.Currency, lang), ""),
		sensitiveText("cell-total-cost", FormatMoney(p.TotalCost, p.Currency, lang), ""),
		sensitiveText("cell-market-value", marketValueContent, pnlColor(nil)),
		sensitiveColoredText("cell-unrealized-pnl", FormatSignedMoney(p.UnrealizedPnL, p.Currency, lang), pnlColor(p.UnrealizedPnL)),
		coloredCell("cell-pnl-pct", FormatSignedPercent(pct, lang), pnlColor(pct)),
		sensitiveColoredText("cell-realized-pnl", FormatSignedMoney(&realized, p.Currency, lang), pnlColor(&realized)),
		components.Text("cell-last-snapshot", FormatRelativeTime(p.LastSnapshotAt, now, lang), "sm", "normal"),
	}
	return components.TableRow("position-"+p.AssetID, cells...)
}

func sensitiveText(id, content, color string) components.Component {
	c := components.Text(id, content, "sm", "normal")
	c.Props["sensitive"] = true
	if color != "" {
		c.Props["color"] = color
	}
	return c
}

func sensitiveColoredText(id, content, color string) components.Component {
	var c components.Component
	if color == "" {
		c = components.Text(id, content, "sm", "normal")
	} else {
		c = components.TextStyled(id, content, "sm", "normal", "", color, "", "")
	}
	c.Props["sensitive"] = true
	return c
}

func priceSourceDot(source string) string {
	switch source {
	case "live":
		return "●"
	case "snapshot":
		return "●"
	case "none":
		return "●"
	default:
		return ""
	}
}
```

- [ ] **Step 4: Remove old vars and helpers**

Delete `columnWidths`, `columnKeys`, `columnShortID` variables/functions. Delete the old `buildPositionItem` function if it still exists. Remove `"sort"` import if no longer needed (check).

- [ ] **Step 5: Update tests in `chart_builder_test.go`**

Key changes:
- Tests that checked `findDescendantByID(card, "positions-header")` — remove (no explicit header row; table generates it).
- Tests that checked `findDescendantByID(card, "positions-body")` with `list` type — replace with `findDescendantByID(card, "positions-table")` with `table` type.
- Tests that checked `row.Children[i]` inside a list_item row — replace with `tableRow.Children[i]` directly (table_row's children are the cells).
- Add assertions for `sensitive: true` on monetary cells.

Update `TestBuildPositionsTable_ReturnsCardWithExpectedID`:

```go
func TestBuildPositionsTable_ReturnsCardWithExpectedID(t *testing.T) {
	ps := samplePositions()
	card := BuildPositionsTable(ps, "en", time.Now(), false)

	assert.Equal(t, "card", card.Type)
	assert.Equal(t, "positions-table-card", card.ID)

	table := findDescendantByID(card, "positions-table")
	require.NotNil(t, table)
	assert.Equal(t, "table", table.Type)

	cols, ok := table.Props["columns"].([]components.TableColumn)
	require.True(t, ok)
	assert.Len(t, cols, 11)

	require.Len(t, table.Children, len(ps))
	for _, child := range table.Children {
		assert.Equal(t, "table_row", child.Type)
	}
}
```

Update the existing builder_test.go tests that reference `positions-header` or `positions-body`:
- `TestBuildScreen_HeaderHas11Columns` — replace with `TestBuildScreen_TableHas11Columns` checking `positions-table`'s `columns` prop.
- `TestBuildScreen_BodyUsesListWithOneItemPerPosition` — replace with `TestBuildScreen_TableHasOneRowPerPosition`.
- `TestBuildScreen_PositionRowValuesInOrder` — update to find `table_row` children directly.

Add sensitive test:

```go
func TestBuildPositionsTable_MonetaryCellsAreSensitive(t *testing.T) {
	ps := samplePositions()
	card := BuildPositionsTable(ps, "en", time.Now(), false)
	table := findDescendantByID(card, "positions-table")
	require.NotNil(t, table)
	require.Len(t, table.Children, 1)
	row := table.Children[0]

	sensitiveIDs := []string{"cell-avg-cost", "cell-total-cost", "cell-market-value", "cell-unrealized-pnl", "cell-realized-pnl"}
	for _, id := range sensitiveIDs {
		cell := findDescendantByID(row, id)
		require.NotNil(t, cell, "missing %s", id)
		assert.Equal(t, true, cell.Props["sensitive"], "%s should be sensitive", id)
	}

	notSensitiveIDs := []string{"cell-ticker", "cell-name", "cell-type", "cell-quantity", "cell-pnl-pct", "cell-last-snapshot"}
	for _, id := range notSensitiveIDs {
		cell := findDescendantByID(row, id)
		require.NotNil(t, cell, "missing %s", id)
		_, has := cell.Props["sensitive"]
		assert.False(t, has, "%s should not be sensitive", id)
	}
}
```

- [ ] **Step 6: Run full suite**

`go test ./... -count=1`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/portfolio/chart_builder.go internal/portfolio/chart_builder_test.go internal/portfolio/builder_test.go
git commit -m "refactor(portfolio): positions table uses table/table_row + sensitive cells"
```

---

### Task 4: HideValues toggle + sensitive on summary

**Files:**
- Modify: `internal/portfolio/live_builder.go`
- Modify: `internal/portfolio/live_builder_test.go`
- Modify: `internal/portfolio/builder.go`
- Modify: `internal/portfolio/builder_test.go`

- [ ] **Step 1: Add hide-values toggle to header row**

In `live_builder.go`, update `BuildPortfolioHeaderRow`. Add the hide-values toggle between the spacer and the live toggle. Update the row widths to `["auto", "1fr", "auto", "auto"]`:

```go
func BuildPortfolioHeaderRow(state LiveState, lang string) components.Component {
	title := components.Text("portfolio-title", i18n.T(lang, "portfolio.title"), "lg", "bold")
	spacer := components.Column("live-header-spacer")

	hideValues := components.IconToggle("hide-values-toggle", false,
		"eye", "eye-off",
		i18n.T(lang, "portfolio.hide_values.tooltip_inactive"),
		i18n.T(lang, "portfolio.hide_values.tooltip_active"),
		components.Action{Trigger: "click", Type: "toggle_sensitive"},
		components.Action{Trigger: "click", Type: "toggle_sensitive"},
	)

	toggle := components.IconToggle("live-toggle", state.Live,
		"radio", "radio",
		i18n.T(lang, "portfolio.live.toggle"),
		i18n.T(lang, "portfolio.live.toggle"),
		components.Reload("/actions/portfolio/live_data?live=true", "live-data-section"),
		components.Reload("/actions/portfolio/live_data?live=false", "live-data-section"),
	)

	return components.Row("live-header-row", []string{"auto", "1fr", "auto", "auto"}, title, spacer, hideValues, toggle)
}
```

- [ ] **Step 2: Add `sensitive: true` to summary value texts**

In `builder.go`, update `buildTotalValueCard` — after creating each value text, set `sensitive: true`:

In the loop where value texts are appended:
```go
txt := components.Text("summary-value-total-value-"+c, FormatMoney(&v, c, lang), "xl", "bold")
txt.Props["sensitive"] = true
values.Children = append(values.Children, txt)
```

Same for `buildTotalPnLCard`:
```go
txt := coloredValue("summary-value-total-pnl-"+c, FormatSignedMoney(&v, c, lang), pnlColor(&v))
txt.Props["sensitive"] = true
values.Children = append(values.Children, txt)
```

Do NOT add `sensitive` to: Performance (percentage), Snapshot Change (percentage), Open Positions (count), or the "—" empty texts.

- [ ] **Step 3: Add tests**

In `live_builder_test.go`, add:

```go
func TestBuildPortfolioHeaderRow_HasHideValuesToggle(t *testing.T) {
	row := BuildPortfolioHeaderRow(LiveState{Live: false}, "en")
	hv := findDescendantByID(row, "hide-values-toggle")
	require.NotNil(t, hv)
	assert.Equal(t, "icon_toggle", hv.Type)
	assert.Equal(t, false, hv.Props["active"])
	assert.Equal(t, "eye", hv.Props["icon_inactive"])
	assert.Equal(t, "eye-off", hv.Props["icon_active"])

	require.Len(t, hv.Actions, 2)
	assert.Equal(t, "toggle_sensitive", hv.Actions[0].Type)
	assert.Equal(t, "toggle_sensitive", hv.Actions[1].Type)
	assert.Equal(t, "", hv.Actions[0].Endpoint)
}

func TestBuildPortfolioHeaderRow_HideValuesBeforeLiveToggle(t *testing.T) {
	row := BuildPortfolioHeaderRow(LiveState{Live: false}, "en")
	// Children: [title, spacer, hide-values, live-toggle]
	require.Len(t, row.Children, 4)
	assert.Equal(t, "hide-values-toggle", row.Children[2].ID)
	assert.Equal(t, "live-toggle", row.Children[3].ID)
}
```

In `builder_test.go`, add:

```go
func TestBuildScreen_SummaryTotalValueIsSensitive(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	tv := findDescendantByID(s, "summary-value-total-value-USD")
	require.NotNil(t, tv)
	assert.Equal(t, true, tv.Props["sensitive"])
}

func TestBuildScreen_SummaryTotalPnLIsSensitive(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	pnl := findDescendantByID(s, "summary-value-total-pnl-USD")
	require.NotNil(t, pnl)
	assert.Equal(t, true, pnl.Props["sensitive"])
}

func TestBuildScreen_SummaryPerformanceNotSensitive(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	perf := findDescendantByID(s, "summary-value-performance-USD")
	require.NotNil(t, perf)
	_, has := perf.Props["sensitive"]
	assert.False(t, has)
}

func TestBuildScreen_SummaryOpenPositionsNotSensitive(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	op := findDescendantByID(s, "summary-value-open-positions")
	require.NotNil(t, op)
	_, has := op.Props["sensitive"]
	assert.False(t, has)
}
```

- [ ] **Step 4: Run full suite**

`go test ./... -count=1`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/live_builder.go internal/portfolio/live_builder_test.go internal/portfolio/builder.go internal/portfolio/builder_test.go locales/en.json locales/es.json
git commit -m "feat(portfolio): HideValues icon_toggle + sensitive on summary values"
```

---

### Task 5: Smoke test

**Files:** none (verification only).

- [ ] **Step 1: Restart and smoke**

```bash
cd /Users/vadimkent/repos/vk_investment_middleend_v2
lsof -ti:8082 | xargs kill -9 2>/dev/null; sleep 1
./cli run >/tmp/srv.log 2>&1 &
sleep 2

RESP=$(curl -s -X POST http://localhost:8082/actions/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@demo.com","password":"demo"}')
TOKEN=$(echo "$RESP" | python3 -c "import json,sys;print(json.load(sys.stdin)['auth']['token'])")

echo "--- table type in positions ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def find(x, id):
    if x.get('id') == id: return x
    for c in x.get('children', []):
        r = find(c, id)
        if r: return r
tbl = find(d, 'positions-table')
print('type:', tbl['type'])
cols = tbl['props']['columns']
print('columns:', len(cols), [c['id'] for c in cols])
print('rows:', len(tbl['children']), [c['type'] for c in tbl['children'][:3]])
"

echo "--- hide-values-toggle present ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def find(x, id):
    if x.get('id') == id: return x
    for c in x.get('children', []):
        r = find(c, id)
        if r: return r
hv = find(d, 'hide-values-toggle')
print('type:', hv['type'], 'icon_inactive:', hv['props']['icon_inactive'], 'action:', hv['actions'][0]['type'])
"

echo "--- sensitive on monetary cells ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio \
  | python3 -c "
import json,sys
d = json.load(sys.stdin)
def find(x, id):
    if x.get('id') == id: return x
    for c in x.get('children', []):
        r = find(c, id)
        if r: return r
for cid in ['cell-avg-cost','cell-market-value','summary-value-total-value-USD','summary-value-total-value-UYU']:
    node = find(d, cid)
    if node:
        print(cid, 'sensitive:', node['props'].get('sensitive'))
    else:
        print(cid, 'NOT FOUND')
"

lsof -ti:8082 | xargs kill -9 2>/dev/null; true
```

Expected:
- `positions-table` type is `table` with 11 columns.
- `hide-values-toggle` type `icon_toggle`, icon_inactive `eye`, action `toggle_sensitive`.
- Monetary cells and summary values carry `sensitive: true`.

---

## Self-Review Results

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| `table` + `table_row` helpers | Task 1 |
| 11 columns with correct widths + align | Task 3 `positionColumns` |
| Header rendered by FE from columns (no explicit row) | Task 3 — no header row emitted |
| Each position as `table_row` with 11 direct children | Task 3 `buildPositionRow` |
| `list`, `list_item`, per-row `row` removed | Task 3 |
| Monetary cells `sensitive: true` | Task 3 `sensitiveText`/`sensitiveColoredText` |
| Non-monetary cells not sensitive | Task 3 test `TestBuildPositionsTable_MonetaryCellsAreSensitive` |
| Summary Total Value + Total P&L sensitive | Task 4 |
| Performance, Snapshot Change, Open Positions not sensitive | Task 4 test |
| `hide-values-toggle` in header row with `eye`/`eye-off` | Task 4 |
| Toggle between spacer and live toggle | Task 4 test `_HideValuesBeforeLiveToggle` |
| `toggle_sensitive` action (no endpoint) | Task 4 |
| Include-closed still works (uses BuildPositionsTable) | Task 3 — same function, now emits table |
| Live mode dots still work | Task 3 — `priceSourceDot` still prepends to market value content |
| i18n keys | Task 2 |

**Placeholder scan:** none.

**Type consistency:**
- `TableColumn{ID, Header, Width, Align}` — consistent in Task 1 and Task 3.
- `Table(id, columns, children...)` and `TableRow(id, children...)` — same in Task 1 and Task 3.
- `BuildPositionsTable(ps, lang, now, isLive)` — signature unchanged from layer 5; callers don't need updates.
- `sensitive: true` prop — set identically on text components in both chart_builder (cells) and builder (summary values).
