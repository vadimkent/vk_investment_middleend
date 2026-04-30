# SDUI Custom Components

Project-specific SDUI extensions: custom components, custom attributes, and custom actions that extend the base set in `sdui-base-components.md` and `sdui-actions.md`. The middleend emits them exactly like base primitives; the frontend maintains registries that map types, attributes, and action names to behavior.

This file is the single source of truth for project-specific SDUI extensions.

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
| `show_legend` | bool | no | Whether a legend renders below the chart. Default `false`. When `true`, the legend is interactive: clicking an entry toggles that line's visibility. Non-interactive-but-visible is not supported. |

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

### Frontend implementation notes

- Height tokens map to `13rem / 18rem / 24rem`. The frontend may adjust to its design system.
- CSS variable names `--chart-1`..`--chart-5` live in the frontend's global stylesheet.
- Tooltip: the frontend shows `<series.label>: <formatted value>` per point. `x_axis.format` also applies to tooltip date headers.
- Currency awareness: for `currency` / `currency_compact` formats, the currency is a screen-level concept — the selected currency control determines which data the middleend emits. The frontend does not need to resolve currency from the chart data itself.

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

---

## 3. `wizard`

Multi-step form container with local step state, Back/Next navigation without round-trips, and per-step include/skip logic. Used by the snapshots screen's create/edit flow; reusable for other multi-step flows (import, analysis).

### Presentation

The middleend emits a bare `wizard` component — it is **not** wrapped in a `modal`. The frontend is responsible for presenting it. When the wizard lives in a [modal slot](sdui-base-components.md#modal-slot-pattern) (e.g. `snapshots-modal-slot`), the slot's overlay container renders it as a dialog on desktop or a drawer/sheet on mobile. When placed inline as a child of a screen's content tree, the frontend renders it inline. The `dismiss_action` is the wizard's contract for "the user closed it" — typically `components.Dismiss()`, which the frontend interprets as "close the overlay" or "remove from the tree."

### Why custom

A wizard requires local state for `currentStep`, Back/Next navigation without a server round-trip, per-step input persistence while the user moves between steps, and per-step validation before advancing. Composing this from base primitives (`visible_when` + a counter) is fragile and can't cleanly encode include/skip semantics. Server-driven navigation (one round-trip per Next) is chatty and sluggish for flows with many steps. The wizard encapsulates the step-state machine in the frontend, following the same pattern as `line_chart` and `pie_chart` encapsulate their interactive state.

### Props

| Prop | Type | Required | Description |
|---|---|---|---|
| `mode` | enum | yes | `create` / `edit`. Used by the frontend to pick button copy and entry-step semantics (see `skippable`). |
| `title` | string | yes | Wizard title. Localized by the middleend. |
| `steps` | `Step[]` | yes | Ordered steps; at least 1. |
| `submit_action` | `Action` | yes | Action executed from the summary step's Submit button (typically `submit` targeting the create / PATCH endpoint). |
| `dismiss_action` | `Action` | yes | Action executed when the user closes the wizard (typically a client-side `replace` that empties the modal slot). |
| `banner` | `Banner` | no | Optional banner rendered above the step content. Used by the auto-snapshot flow (info / warning) and validation-error re-emission (error). |
| `initial_step_id` | string | no | Step id to open the wizard on. Defaults to the first step. The middleend sets this when re-emitting the wizard after a validation error, to focus the user on the relevant step. |

### Sub-types

**`Step`**

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | string | yes | Stable identifier (react key + hidden-input grouping). |
| `label` | string | yes | Short label for the step indicator (e.g. `Info`, `AAPL`, `Summary`). Localized by the middleend. |
| `kind` | enum | yes | `info` / `entry` / `summary`. Drives the button set: `info` → Next; `entry` → Back / Skip / Include (or Back / Update when editing an existing entry); `summary` → Back / Submit. |
| `skippable` | bool | yes | Only meaningful when `kind=entry`. `false` disables Skip and hides the "exclude" affordance — used in edit mode on entries that already exist in the snapshot. |
| `include_default` | bool | yes | Only meaningful when `kind=entry`. Initial state of the step's "included" flag — `true` for existing entries in edit mode, `false` for new entries in create mode. |
| `children` | `Component[]` | yes | The step's content (inputs, text). The wizard shows only the active step's children; other steps are hidden but their inputs persist client-side. |

**`Banner`**

| Field | Type | Required | Description |
|---|---|---|---|
| `variant` | enum | yes | `info` / `success` / `warning` / `error`. |
| `message` | string | yes | Localized. |
| `title` | string | no | Optional bold prefix (used for `warnings_title` in the auto-snapshot flow). |
| `dismissible` | bool | no | Default `false`. When `true`, the user can close the banner without closing the wizard. |

### Frontend behavior

1. **Step indicator**: renders `Step X of Y` + a chip row with each step's `label`. Chips are clickable — free jump between steps. Jumping does not validate.
2. **Buttons per kind**:
   - `info` step: Next only.
   - `entry` step: Back + Skip + Include (create mode) or Back + Update (edit mode, existing entry, `skippable: false`).
   - `summary` step: Back + Submit.
3. **Include map**: the wizard holds `{ stepId → included: bool }`, seeded from each step's `include_default`. Skip sets `false`; Include sets `true`; Update (edit mode) keeps `true` and advances.
4. **Navigation**: Back always works (no validation). Next / Include validates the required inputs of the current step before advancing (using the input's own `required`, `pattern`, `min`, `max` props). Skip bypasses validation and marks the step excluded.
5. **Summary step rendering**: the summary step's children come from the middleend as a short descriptive paragraph. The list of included entries is derived client-side from the include-map — not server-emitted — so it stays reactive to Skip/Include changes without re-emitting the wizard.
6. **Submit**: on the summary step, the wizard collects inputs from (a) all `kind=info` steps (always included) and (b) all `kind=entry` steps where `included=true`. It then executes `submit_action` with that body.
7. **Dismiss**: executes `dismiss_action`.

### Hidden input naming

For each `kind=entry` step representing an asset, inputs use bracket notation so the middleend can parse them into a structured entries array:

- `entries[<asset_id>].mode` — `price` or `override`.
- `entries[<asset_id>].current_price` — present when `mode=price`.
- `entries[<asset_id>].current_value_override` — present when `mode=override`.

`kind=info` steps use plain names: `recorded_at`, `notes`. Complex-asset entry steps omit `mode` (always `override`). The middleend handlers parse this flat shape into the backend's nested `entries` array.

### Validation and BE-error handling

Format validation runs through the `input` props (`required`, `max_length`, `pattern`, `min`, `max`). The wizard does not define its own validation primitives. Backend validation errors (422) arrive as an `ActionResponse` that replaces the modal subtree with the same wizard re-emitted — inputs preserved — plus an `error` banner. The wizard re-opens on the summary step by default; the middleend can override this via `initial_step_id` (e.g. to land on the `info` step for a `FUTURE_DATED_SNAPSHOT` error).

### Example

Minimal wizard with one info step, one entry step, and one summary step:

```json
{
  "type": "wizard",
  "id": "snapshot-create-wizard",
  "props": {
    "mode": "create",
    "title": "New Snapshot",
    "submit_action": { "trigger": "click", "type": "submit", "endpoint": "/actions/snapshots/create", "method": "POST", "target_id": "snapshots-root" },
    "dismiss_action": { "trigger": "click", "type": "replace", "target_id": "snapshots-modal-slot", "tree": null },
    "steps": [
      {
        "id": "info",
        "label": "Info",
        "kind": "info",
        "skippable": false,
        "include_default": true,
        "children": [
          { "type": "input", "id": "recorded-at", "props": { "name": "recorded_at", "input_type": "datetime-local", "label": "Date", "required": true } },
          { "type": "textarea", "id": "notes", "props": { "name": "notes", "label": "Notes", "max_length": 500 } }
        ]
      },
      {
        "id": "entry-aapl-uuid",
        "label": "AAPL",
        "kind": "entry",
        "skippable": true,
        "include_default": false,
        "children": [
          { "type": "text", "id": "entry-header", "props": { "content": "Apple Inc", "size": "md", "weight": "bold" } },
          { "type": "input", "id": "price-input", "props": { "name": "entries[aapl-uuid].current_price", "input_type": "number", "label": "Current Price", "required": true } }
        ]
      },
      {
        "id": "summary",
        "label": "Summary",
        "kind": "summary",
        "skippable": false,
        "include_default": true,
        "children": [
          { "type": "text", "id": "summary-desc", "props": { "content": "Review your entries and submit.", "size": "md", "weight": "normal" } }
        ]
      }
    ]
  }
}
```

---

## 4. `file_upload`

Drag-and-drop + click-to-browse file picker with local validation. Used by the Import & Export screen for the AI Import upload form and the Restore upload form. Generic by design — any future flow that needs a file as part of a multipart submit can reuse it.

### Why custom

The base SDUI catalog has no `input` variant for files. Browsers do not let JavaScript programmatically reattach a previously-picked File across re-renders, and SDUI re-renders are server-driven — so a custom component that owns local file state, drag-and-drop affordances, and pre-submit validation (size, format) is the cleanest way to model file inputs without leaking browser-specific quirks into every consumer.

### Props

| Prop | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Multipart field name on submit (e.g. `"file"`). |
| `label` | string | yes | Visible label rendered above the dropzone. Localized by the middleend. |
| `placeholder` | string | yes | Dropzone copy when no file is selected (e.g. *"Drop a file here or click to browse"*). Localized. |
| `hint` | string | no | Auxiliary copy beneath the dropzone (formats / size limit). Localized. |
| `accept` | string | no | Comma-separated extensions / MIME types (e.g. `".csv,.tsv,.xlsx"`). Drives the native `<input type="file" accept>` and the local format check. Absent → any file. |
| `max_size_bytes` | int | no | Local size limit in bytes. When the user picks a larger file, render `error_message_size` inline and clear the selection. Absent → no local limit. |
| `error_message_size` | string | no | Localized message when `max_size_bytes` is exceeded. May contain `{limit}` rendered as a human-readable size (e.g. "5 MB"). |
| `error_message_format` | string | no | Localized message when the file's extension / MIME type doesn't match `accept`. |
| `prefill_filename` | string | no | When set, render the dropzone in the "file selected" state with this filename **but no actual File object behind it** — purely informational. Used by the middleend when re-emitting a form after a server-side error. To re-submit, the user must re-pick the file (browsers do not let JS reattach a previously-picked File). The dropzone signals this state with the small caption from `reattach_hint`. |
| `reattach_hint` | string | no | Localized small caption shown alongside `prefill_filename` (e.g. "Re-select the file to retry"). |

### Frontend behavior

- Render: a dashed-bordered dropzone (~10rem tall) with an upload icon centered and the placeholder text below. When a file is selected, the placeholder is replaced by the filename (mono-friendly truncation if long). Hover, drag-over, and focus states match the design system's other interactive controls.
- Native `<input type="file">` is hidden; the dropzone forwards click to it. Drop events on the dropzone are captured (`preventDefault` on dragover, intercept the file from `dataTransfer.files[0]` on drop).
- On a new file selection: run the format check against `accept` (if set), then the size check against `max_size_bytes` (if set). On failure, show the corresponding error inline beneath the dropzone and do **not** retain the file.
- On `submit` of the enclosing form: contributes its file to the `multipart/form-data` body under `name`. If no file is present, the form-level submit button must be disabled by its consumer (the file_upload does not own form-level disabling).
- Reset: a fresh `replace` from the server (matching `id`) clears any local file and any local error. `prefill_filename` lets the server hint at the previously-uploaded filename for context.

### Example

```json
{
  "type": "file_upload",
  "id": "import-file",
  "props": {
    "name": "file",
    "label": "File",
    "placeholder": "Drop a file here or click to browse",
    "hint": "CSV, TSV, XLS, XLSX, TXT — max 5 MB",
    "accept": ".csv,.tsv,.xls,.xlsx,.txt",
    "max_size_bytes": 5242880,
    "error_message_size": "File exceeds the {limit} limit.",
    "error_message_format": "Unsupported file format."
  }
}
```

---

## 5. Custom Attributes

Project-specific props that may appear on any component. The frontend reads them alongside base shared props (`align_items`, `gap`, etc.) and applies project-specific behavior.

### `sensitive`

Available on any component. When `true`, the frontend masks the component's visible content with `"••••"` while the HideValues toggle is active. The middleend decides **what** is sensitive; the frontend decides **when** to mask.

Not all monetary values are sensitive. The rule:

| Sensitive (`true`) | Not sensitive |
|---|---|
| Absolute monetary values: Total Value, Total P&L, Avg Cost, Total Cost, Market Value, Unrealized P&L, Realized P&L | Percentages: Performance, Snapshot Change, % P&L |
| | Counts: Open Positions, Quantity |
| | Metadata: Ticker, Name, Type, Last Snapshot |

The frontend must not infer sensitivity from the value's format or color — only from the explicit `sensitive: true` prop.

```json
{
  "type": "text",
  "id": "summary-value-total-value-USD",
  "props": {
    "content": "$12,345.67",
    "size": "xl",
    "weight": "bold",
    "sensitive": true
  }
}
```

When HideValues is active, the frontend renders `"••••"` instead of `"$12,345.67"`. The original `content` stays in the tree — the frontend just visually replaces it.

---

## 6. Custom Actions

Project-specific action types that extend the base set in `sdui-actions.md`. The frontend maps these types to local behavior; no server round-trip is involved.

### `toggle_sensitive`

Toggles the visibility of all components marked with `sensitive: true`. Fired by the HideValues `icon_toggle`. No `endpoint` or `target_id` — purely client-side.

```json
{
  "trigger": "click",
  "type": "toggle_sensitive"
}
```

When the frontend receives this action:
1. Flip the local `hideValues` boolean state.
2. All components with `sensitive: true` in the current screen tree are masked (`"••••"`) or unmasked based on the new state.
3. No HTTP request is made.

The `icon_toggle` for HideValues carries this action in both slots (the action is the same regardless of direction):

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

### `download`

Triggers a file download from a middleend endpoint. The frontend hands the URL to the browser's native download mechanism so the response (bytes with `Content-Disposition: attachment`) is saved as a file rather than parsed as an `ActionResponse`.

| Param | Type | Description |
|---|---|---|
| `url` | string | Middleend endpoint (relative or absolute). Must respond with `Content-Disposition: attachment; filename="..."` and the body bytes. |

```json
{
  "trigger": "click",
  "type": "download",
  "url": "/actions/import/export"
}
```

When the frontend receives this action:
1. Create a transient hidden `<a href={url} download>` element in the DOM (or the equivalent platform primitive on native), click it, then remove it. The browser handles the rest: sends the GET with the user's auth (cookies / `Authorization` per the FE's HTTP layer), reads the `Content-Disposition`, and saves the file to the user's downloads folder.
2. No `ActionResponse` is parsed. No SDUI subtree is replaced. No loading indicator is rendered (the browser shows its own download UI).
3. **Auth handling.** If the endpoint returns `401`, it must respond with an HTTP `302 Location: /login` redirect (not the JSON `{error:"unauthorized", redirect:"/login"}` shape used by `submit` / `reload`). The browser follows the redirect natively. Endpoints serving `download` traffic must implement this.
4. **Errors.** `5xx` surfaces as whatever the browser does with a failed download (typically the error body shown as text or saved as a file with the error body — acceptable v1). The frontend has no opportunity to render an inline error.

**When to use:**
- Use `download` for any middleend endpoint that returns binary / CSV / file bytes meant to be saved by the user.
- Do **not** use `open_url` for this purpose — `open_url` is for navigating to external URLs (external docs, third-party sites).
- Do **not** use `submit` or `reload` — those parse JSON `ActionResponse` and replace SDUI subtrees, which is wrong for a file body.

```go
Download(url string) Action
```

```go
components.Button("export-btn", "Export all data",
    components.Download("/actions/import/export"),
)
```
