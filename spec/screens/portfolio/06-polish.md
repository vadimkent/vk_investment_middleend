# Portfolio — Layer 6: Polish

Final layer for the portfolio screen. Migrates the positions list from `list`/`list_item`/`row` to the new `table`/`table_row` base components (fixing column alignment), and adds a HideValues toggle that masks monetary values client-side via the `sensitive` custom attribute and `toggle_sensitive` custom action.

## 1. Positions table migration: `table` / `table_row`

### Current (layers 1–5)

```
card positions-table-card
  column positions-table (gap sm)
    row positions-header widths=[11 widths]
      text "Ticker" | text "Name" | ... | text "Last Snapshot"
    list positions-body
      list_item position-<id>
        row position-<id>-row widths=[same 11 widths]
          text ticker | text name | ... | text last_snapshot
```

Problems: header and body rows are independent CSS grids — columns don't align when content varies. The `1fr` name column computes differently in the header vs each row.

### New

```
card positions-table-card
  table positions-table columns=[11 TableColumn definitions]
    table_row position-<id>
      text ticker | text name | ... | text last_snapshot
    table_row position-<id2>
      ...
```

The `table` component owns the `columns` prop which defines all 11 column IDs, headers, widths, and alignments. The frontend renders the header automatically from `columns[].header`. Each `table_row`'s children map to columns in order via CSS subgrid — guaranteed alignment.

### Column definitions

| # | ID | Header (i18n key) | Width | Align |
|---|---|---|---|---|
| 1 | `ticker` | `portfolio.col.ticker` | `80px` | left |
| 2 | `name` | `portfolio.col.name` | `1fr` | left |
| 3 | `type` | `portfolio.col.type` | `80px` | left |
| 4 | `quantity` | `portfolio.col.quantity` | `80px` | right |
| 5 | `avg_cost` | `portfolio.col.avg_cost` | `110px` | right |
| 6 | `total_cost` | `portfolio.col.total_cost` | `110px` | right |
| 7 | `market_value` | `portfolio.col.market_value` | `120px` | right |
| 8 | `unrealized_pnl` | `portfolio.col.unrealized_pnl` | `130px` | right |
| 9 | `pnl_pct` | `portfolio.col.pnl_pct` | `80px` | right |
| 10 | `realized_pnl` | `portfolio.col.realized_pnl` | `120px` | right |
| 11 | `last_snapshot` | `portfolio.col.last_snapshot` | `120px` | right |

Numeric/monetary columns are right-aligned. Ticker, name, type are left-aligned.

### What changes in the builder

`BuildPositionsTable` (in `chart_builder.go`) is rewritten:
- Replaces `Row` header + `List` + `ListItem` + `Row` per position with `Table` + `TableRow` per position.
- Removes the manual header `Row` — the table's `columns[].header` drives it.
- Each `table_row` receives 11 `text` children directly (no wrapping `Row`).
- The live-mode price-source dot (prepended `"● "` to market value) still works inside a `table_row` cell.

### Go helpers (new)

```go
// internal/components/table.go

type TableColumn struct {
    ID     string `json:"id"`
    Header string `json:"header"`
    Width  string `json:"width,omitempty"`
    Align  string `json:"align,omitempty"`
}

func Table(id string, columns []TableColumn, children ...Component) Component
func TableRow(id string, children ...Component) Component
```

## 2. HideValues toggle

### Placement

A new `icon_toggle#hide-values-toggle` in the `live-header-row`, between the spacer and the live toggle:

```
row live-header-row widths=["auto", "1fr", "auto", "auto"]
  text portfolio-title
  column spacer
  icon_toggle hide-values-toggle
  icon_toggle live-toggle
```

### Component shape

```json
{
  "type": "icon_toggle",
  "id": "hide-values-toggle",
  "props": {
    "active": false,
    "icon_inactive": "eye",
    "icon_active": "eye-off",
    "tooltip_inactive": "Hide values",
    "tooltip_active": "Show values"
  },
  "actions": [
    { "trigger": "click", "type": "toggle_sensitive" },
    { "trigger": "click", "type": "toggle_sensitive" }
  ]
}
```

Icons: `eye` (inactive — values visible) → `eye-off` (active — values hidden). Both actions fire `toggle_sensitive` (client-side, no server round-trip).

### `sensitive: true` marking

The middleend emits `sensitive: true` on the following text components:

**Summary cards:**
- `summary-value-total-value-<currency>` (all currencies)
- `summary-value-total-pnl-<currency>` (all currencies)

**Positions table cells (per row):**
- `cell-avg-cost`
- `cell-total-cost`
- `cell-market-value`
- `cell-unrealized-pnl`
- `cell-realized-pnl`

**Not marked:**
- `summary-value-performance-*` (percentage)
- `summary-value-snapshot-change-*` (percentage)
- `summary-value-open-positions` (count)
- `cell-ticker`, `cell-name`, `cell-type`, `cell-quantity`, `cell-pnl-pct`, `cell-last-snapshot`

### i18n keys introduced

| Key | en | es |
|---|---|---|
| `portfolio.hide_values.tooltip_inactive` | Hide values | Ocultar valores |
| `portfolio.hide_values.tooltip_active` | Show values | Mostrar valores |

## Package layout (incremental on layer 5)

| File | Change |
|---|---|
| `internal/components/table.go` | **new** — `TableColumn`, `Table`, `TableRow` helpers |
| `internal/components/table_test.go` | **new** — JSON shape tests |
| `internal/portfolio/chart_builder.go` | Rewrite `BuildPositionsTable` + `buildPositionItem`: `table` + `table_row` instead of `row` + `list` + `list_item`. Add `sensitive: true` on monetary cells. |
| `internal/portfolio/chart_builder_test.go` | Update structure assertions. |
| `internal/portfolio/live_builder.go` | `BuildPortfolioHeaderRow` adds the hide-values toggle. |
| `internal/portfolio/live_builder_test.go` | Assert hide-values toggle present with correct actions. |
| `internal/portfolio/builder.go` | Emit `sensitive: true` on summary card value texts. |
| `internal/portfolio/builder_test.go` | Assert `sensitive` prop on monetary summary texts. |
| `locales/{en,es}.json` | Add `portfolio.hide_values.*` keys. |

## Scope explicitly out

- **Responsive mobile** (cards per position on small screens, summary grid 2-col). Deferred to a future layer or handled entirely by the FE's responsive CSS.
- **Price source dots visual refinement** beyond the `"● "` prefix. The dot mechanism works; visual polish is FE CSS.

## Acceptance criteria

- [ ] `GET /screens/portfolio` emits `table#positions-table` with 11 `TableColumn` entries (correct ids, headers, widths, aligns).
- [ ] Table header is NOT a separate row in the tree — it comes from `columns[].header` (FE renders it).
- [ ] Each position is a `table_row` with 11 direct children (no wrapping `row`).
- [ ] The `list`, `list_item`, and per-row `row` components are gone from the positions tree.
- [ ] Monetary cells (`cell-avg-cost`, `cell-total-cost`, `cell-market-value`, `cell-unrealized-pnl`, `cell-realized-pnl`) carry `sensitive: true` in their props.
- [ ] Non-monetary cells do NOT carry `sensitive`.
- [ ] Summary value texts for Total Value and Total P&L carry `sensitive: true`.
- [ ] Summary texts for Performance, Snapshot Change, and Open Positions do NOT carry `sensitive`.
- [ ] `live-header-row` contains `hide-values-toggle` (icon_toggle) with `icon_inactive: "eye"`, `icon_active: "eye-off"`, both actions `toggle_sensitive`.
- [ ] `hide-values-toggle` sits between the spacer and the live toggle.
- [ ] `include_closed` action handler's `BuildPositionsTable` call still works (returns table-based card).
- [ ] Live mode dots still work inside `table_row` cells.
- [ ] All existing tests updated; no regressions.
