# SDUI Custom Components

Project-specific SDUI components that extend the base set in `sdui-base-components.md`. The middleend emits them exactly like base components (type + id + props + children + actions); the frontend maintains a registry that maps type names to renderers.

This file documents the contract the middleend is expected to emit. It is the single source of truth for custom components used by any screen in this project.

---

## 1. `line_chart`

Single- or multi-series line chart with time on the x-axis. Used by the portfolio screen's evolution chart. Numeric data plus format tokens drive the rendering.

### Why numbers, not pre-formatted strings

Chart libraries (recharts, chart.js, highcharts) plot from numeric values and compute tick positions from the numeric range. A chart cannot derive ticks from strings like `"$10,500.00"`. For this reason — and only for this reason — the `line_chart` contract breaks the project's general "middleend emits formatted strings" convention: `data` rows are numeric, and tick/tooltip formatting is driven by format tokens the frontend resolves. The locale (`Accept-Language`) is still set by the frontend on its requests, so the frontend has full context to format per locale.

### Props

| Prop | Type | Required | Description |
|---|---|---|---|
| `title` | string | no | Optional chart title rendered above the plot area. Callers typically emit their own `text` header and omit this. |
| `height` | enum | no | `sm` (13rem) / `md` (18rem) / `lg` (24rem). Default `md`. |
| `series` | `Series[]` | yes | One entry per line. A single-series chart sends one entry. |
| `x_axis` | `{ key, format }` | yes | Which key of each data row drives the x-axis; and how to format tick labels. |
| `y_axis` | `{ format }` | no | How to format y-axis tick labels. If omitted, the frontend picks a sensible default based on series value_format. |
| `data` | `Row[]` | yes | Array of rows; may be empty to trigger empty state. Each row contains the x-axis key plus one key per series. |
| `empty_message` | string | no | Text to render in place of the plot when the dataset is empty or insufficient (<2 points). Localized by the middleend. |

### Sub-types

**`Series`**

| Field | Type | Required | Description |
|---|---|---|---|
| `key` | string | yes | Data row field that holds this series' y-values. |
| `label` | string | yes | Legend/tooltip label for this series. Localized by the middleend. |
| `color` | `ChartColorToken` | yes | Color token; frontend maps to a CSS var. |
| `value_format` | `ValueFormat` | yes | How tooltip values for this series are formatted. |

**`Row`**: `map<string, number | string | null>`. Contains the x-axis key (typically a date string) and one numeric value per series key. Any series key may be `null` to render a gap.

### Token enums

#### `ChartColorToken`

Semantic color slots. Frontend maps them to its CSS variables (`--chart-1`…`--chart-5`).

| Token | Typical use |
|---|---|
| `chart_1` | Primary series |
| `chart_2` | Secondary series |
| `chart_3` | Tertiary |
| `chart_4` | Fourth |
| `chart_5` | Fifth |

Beyond five series the frontend cycles through the palette; the middleend can reuse `chart_1..chart_5` for additional series.

#### `ValueFormat`

Applied to y-axis ticks and tooltip values.

| Token | Rendering |
|---|---|
| `currency` | `$1,234.56` (per locale) |
| `currency_compact` | `$1.5k`, `$2M` (compact notation) |
| `percent` | `12.34%` |
| `percent_signed` | `+12.34%` / `-5.68%` |
| `integer` | `1234` |
| `decimal_2` | `1234.56` |
| `raw` | Whatever the number renders as by default |

The frontend picks a currency symbol based on data context (the series' `label` or a separate currency field — see §1.4). For this layer's single-series value-over-time, currency is implicit in the screen state.

#### `AxisFormat`

Applied to x-axis tick labels.

| Token | Rendering |
|---|---|
| `date` | Short date per locale (`Apr 14`, `14 abr`) |
| `month_year` | `Apr 2026` / `abr 2026` |
| `integer` | Plain integer |
| `raw` | As-is |

### Empty / insufficient data

When `data` has fewer than 2 points the frontend does not attempt to plot. It renders `empty_message` centered in the plot area; legend (if any) is hidden.

Emitting `data: []` + a localized `empty_message` is the standard pattern. Do not swap the whole subtree for a different component — keep the `line_chart` with empty data so the action/reload cycle continues to work.

### Example

Single-series portfolio value over time in absolute mode:

```json
{
  "type": "line_chart",
  "id": "chart-value-over-time",
  "props": {
    "height": "md",
    "series": [
      {
        "key": "value",
        "label": "Value",
        "color": "chart_1",
        "value_format": "currency_compact"
      }
    ],
    "x_axis": { "key": "date", "format": "month_year" },
    "y_axis": { "format": "currency_compact" },
    "data": [
      { "date": "2026-01-15", "value": 10500.50 },
      { "date": "2026-02-15", "value": 10800.00 },
      { "date": "2026-03-15", "value": 11250.75 }
    ],
    "empty_message": "Record at least two snapshots to see the chart."
  }
}
```

### Open points (confirm with frontend)

- Height tokens map to `13rem / 18rem / 24rem`. Frontend can adjust to its design system.
- CSS variable names for `chart_1..chart_5` live in the frontend's global stylesheet.
- Tooltip format: the frontend shows `<series.label>: <formatted value>` per point. The `x_axis.format` also applies to tooltip headers.
- Currency awareness: for `currency` / `currency_compact`, the frontend needs a currency code. For this layer the currency is a screen-level concept (selected by control); future charts with per-series currencies may carry it per series.

---

## 2. `pie_chart`

Pie / donut chart for categorical allocation. Used by the portfolio allocation view. Numeric slice values plus format tokens drive the rendering; percentages are derived client-side from the currently visible slices.

### Why numbers, not pre-formatted strings

Same rationale as `line_chart` §1. Pie libraries compute arc angles from numeric slice values, and the legend toggle recomputes visible-slice percentages on the fly. Strings like `"$10,500.00"` would prevent both.

### Props

| Prop | Type | Required | Description |
|---|---|---|---|
| `title` | string | no | Optional chart title. Metadata — callers typically emit their own `text` header and omit this. |
| `height` | enum | no | `sm` (13rem) / `md` (18rem) / `lg` (24rem). Default `md`. Same tokens as `line_chart.height`. |
| `shape` | enum | no | `pie` / `donut`. Default `donut`. `donut` renders with a central hole; `pie` is a full pie. |
| `value_format` | `ValueFormat` | yes | Applied to slice values in tooltip and legend. Percentages are always rendered as `"xx.x%"` separately. |
| `slices` | `Slice[]` | yes | Array of slices. Ordered by the middleend. May be empty to trigger the empty state. |
| `show_legend` | bool | no | Whether a legend renders. Default `true`. When `true`, the frontend renders it as interactive (clicking an entry toggles that slice's visibility; percentages recompute across remaining visible slices). Non-interactive-but-visible is not supported. |
| `empty_message` | string | no | Text rendered in place of the chart when `slices` is empty. Localized by the middleend. |

### Sub-types

**`Slice`**

| Field | Type | Required | Description |
|---|---|---|---|
| `key` | string | yes | Stable identifier, unique within this chart. Used as the react key and the legend/tooltip join. |
| `label` | string | yes | Legend / tooltip label. Localized by the middleend. |
| `value` | number | yes | Slice magnitude (in `value_format` units). Slices with `value <= 0` are filtered out by the frontend before rendering. |
| `color` | `ChartColorToken` | yes | Color token (`chart_1`..`chart_5`, cycling beyond 5). |

### Middleend responsibilities (out of the component contract)

These belong to whatever handler composes the pie chart, not to the component props. They are listed here so every handler converges on the same conventions:

- **Ordering**: emit `slices` sorted by `value` descending; ties broken by `key` ascending.
- **"Other" bucket**: if the long tail of small slices is visually noisy, the handler may pool slices below a chosen threshold into a single `{ key: "other", label: <localized "Other">, value: <sum>, color: "chart_5" }` entry. Threshold is per-handler, not a component prop.
- **Max slices**: same — handler caps. The component does not truncate.

### Percentage calculation (frontend)

The frontend sums the `value` of currently visible slices (those not hidden via the legend toggle) and computes each slice's share as `slice.value / visible_total * 100`. Hiding a slice recomputes all remaining percentages so they sum to 100%. No middleend round-trip.

### Empty state

When `slices` is empty (after the frontend's `value > 0` filter), render `empty_message` in the plot area; the legend is hidden regardless of `show_legend`. Keep the `pie_chart` in the tree — do not swap it for a different component — so later `replace` / `refresh` flows can repopulate it.

### Out of v1 scope

- **Drill-down via slice click** — a per-slice `action` can be added as an optional `Slice.action` later. Not emitted in v1.

### Example

Allocation donut by asset:

```json
{
  "type": "pie_chart",
  "id": "chart-allocation",
  "props": {
    "height": "md",
    "shape": "donut",
    "value_format": "currency_compact",
    "show_legend": true,
    "slices": [
      { "key": "AAPL", "label": "AAPL", "value": 12500, "color": "chart_1" },
      { "key": "MSFT", "label": "MSFT", "value": 8200, "color": "chart_2" },
      { "key": "BTC",  "label": "BTC",  "value": 4300, "color": "chart_3" },
      { "key": "CASH", "label": "Cash", "value": 2000, "color": "chart_4" }
    ],
    "empty_message": "No positions with known value."
  }
}
```
