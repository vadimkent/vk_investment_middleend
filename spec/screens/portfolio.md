# Portfolio Screen

Main screen for a logged-in user. Shows the current state of their investment portfolio — summary metrics, positions, historical charts, asset allocation, and an optional real-time price mode.

## Purpose

Answer three questions at a glance:

1. **Where do I stand today?** — total value, P&L, performance, recent change, open-position count.
2. **What's in the portfolio?** — per-asset breakdown (quantity, cost basis, current value, realized and unrealized P&L).
3. **How has it evolved?** — portfolio value over time, per-asset value over time, and current allocation by asset or type.

A Live toggle lets the user pull real-time prices from the configured external providers; when off, values come from the user's own snapshots.

## Endpoints

All endpoints are **protected** (JWT required) and forward the caller's `Authorization` header to the backend verbatim. Missing, invalid, or expired JWT returns `401 {"error":"unauthorized","redirect":"/login"}`. Backend 5xx / network / malformed JSON returns `502 BACKEND_ERROR`.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/screens/portfolio` | Full screen render. Triggers parallel backend calls for positions and recent evolution. |
| `POST` | `/actions/portfolio/include_closed` | Toggles inclusion of fully-closed positions in the positions table. Returns a partial tree. |
| `GET` | `/actions/portfolio/evolution` | Re-renders the Value/Asset Value Over Time charts section for a new timeframe / mode / currency. |
| `GET` | `/actions/portfolio/allocation` | Re-renders the Allocation section for a new group-by / currency. |
| `GET` | `/actions/portfolio/live_data` | Re-renders the live-data section (summary + table + banner) for a new live/refresh state. |

Every non-screen endpoint returns a Server-Driven UI `ActionResponse` — either a `replace` with a tree for the affected subtree, or an error following the same 401/502 mapping.

## Backend dependencies

- `GET /v1/portfolio` — positions. Critical path. Supports query params `include_closed`, `live`, `refresh`. When `live=true` the response is extended with `is_live`, `prices_as_of`, `warnings[]` (failed provider fetches keyed by `asset_id` + `ticker`) and per-position `price_source` (`live` / `snapshot` / `none`) + `price_as_of`.
- `GET /v1/portfolio/evolution` — historical snapshots. Used in two modes:
  - `?last=2` for the Snapshot Change summary card. Best-effort — if it fails, Snapshot Change falls back to `—` per currency and the rest of the screen still renders.
  - `?from=<iso>&points=100[&currency=X]` for the Value Over Time / Asset Value Over Time charts. Each point carries `total_value`, optional `total_cost`, and an `assets[]` array keyed by ticker.

## Layout and sections

The screen composes six major sections from top to bottom. Sections are named so they can be addressed by partial-update actions — the middleend decides the exact tree, but the interaction contract below refers to these names.

1. **Live data section** — wraps everything above the charts; the target of the live-data reload action.
   - Header row: screen title, hide-values toggle, live toggle. The hide-values toggle masks monetary values client-side; the live toggle flips between standard and live mode.
   - Live banner (only when live is on): status text with relative timestamp of `prices_as_of` + refresh button. When any providers failed, a separate muted warnings line lists the failing tickers.
   - Summary row: five cards in a single horizontal row — Total Value, Total P&L, Total Performance, Snapshot Change, Open Positions.
   - Include-closed form: a single checkbox that toggles the table between "open positions only" and "all positions (incl. closed)". Toggling it replaces only the positions table card.
   - Positions table card: a `table` with 11 columns (see below) — one `table_row` per position.
2. **Charts section** — contains the timeframe / mode / currency controls shared by both time-series charts, a Value Over Time card (single-series line chart), and an Asset Value Over Time card (multi-series line chart). The whole section is the target of the evolution reload action.
3. **Allocation section** — its own group-by and currency controls plus a donut pie chart. Target of the allocation reload action.

When `positions` is empty, the screen shows a localized empty block (title + subtitle) and **none** of the summary row, include-closed form, positions table, charts section, or allocation section are emitted. No call-to-action buttons are added in the empty state.

## Data rules

### Summary cards

All monetary and ratio cards produce **one line per currency** present in the data. Currency order is computed once per screen build — by total value descending, derived from positions — and reused across Total Value, Total P&L, Performance, and Snapshot Change so lines align vertically.

| Card | Computation per currency `c` | Value format | Color rule | Fallback |
|---|---|---|---|---|
| Total Value | `Σ current_value` over open positions with `currency == c and current_value != nil` | Money (unsigned) | none | single `—` if no currency has any non-null value |
| Total P&L | `Σ (unrealized_pnl∣0) + Σ realized_pnl` for positions with `currency == c` | Signed money | positive / negative / none (≈0) | currency omitted if no positions |
| Total Performance | `Σ unrealized_pnl / Σ total_cost × 100` (positions with both non-null in currency `c`) | Signed percent | positive / negative / none | `—` per currency when `Σ total_cost == 0` or no eligible rows |
| Snapshot Change | Sorted last-2 evolution points for `c`: `(latest − prev) / prev × 100` | Signed percent | positive / negative / none | `—` per currency when fewer than 2 points, division by zero, or the evolution call failed entirely |
| Open Positions | `len(positions)` | Integer | none | always rendered; no currency breakdown |

### Positions table

11 columns in order: Ticker · Name · Type · Quantity · Avg Cost · Total Cost · Market Value · Unrealized P&L · % P&L · Realized P&L · Last Snapshot.

- Emitted via the base `table` / `table_row` components so the header (driven by `columns[].header`) and body rows share the same grid and align perfectly.
- Ticker is uppercase. Numeric / monetary columns right-aligned; Ticker, Name, Type left-aligned.
- All values are **formatted server-side per `Accept-Language`**. Raw numbers never ship to the frontend. Currency, quantity, signed currency, signed percent, and relative date each have an `en`/`es` formatter. Null renders as `—`.
- P&L cells (Unrealized P&L, % P&L, Realized P&L) carry `color: positive` when `> 0`, `negative` when `< 0`, no color otherwise. The `error` color token is reserved for validation errors — do not use it for P&L.
- Complex assets (`is_complex = true`) and any field that depends on a missing snapshot render as `—`. Follow the backend's own null semantics.
- Default sort: `current_value DESC` (nulls last), then `ticker ASC`. No interactive sort.

**Price-source indicator (live mode only):** every position row prepends a `●` mark to the Market Value cell — `positive` color for `price_source=live`, `muted` for `snapshot`, `negative` for `none`. When live is off, no dot is emitted and the 11-column layout is unchanged.

**Sensitive marking for the hide-values toggle:** monetary cells — Avg Cost, Total Cost, Market Value, Unrealized P&L, Realized P&L — and the summary values for Total Value and Total P&L are marked with `sensitive: true` so the toggle can mask them client-side. Percentage, quantity, count, ticker/name/type/date cells are not marked.

### Include-closed toggle

A single checkbox between the summary row and the positions table. When checked, the table includes positions with `quantity == 0`. Toggling **does not** recompute the summary cards — Open Positions is a count of currently open positions by definition, and including closed is a view-only concern. Toggle state is **not persisted** across full screen reloads; every `GET /screens/portfolio` starts with the checkbox unchecked.

### Value Over Time chart

Single-series line chart. Controls sit at the `charts-section` level (shared with the asset chart below): timeframe `1m / 3m / 6m / ytd / 1y / all`, mode `$` / `%`, currency. Selected option renders as `primary/solid`, non-selected as `secondary/ghost`.

- `timeframe → from` server-side mapping: `1m` = now − 30d, `3m` = 90d, `6m` = 180d, `ytd` = start-of-year UTC, `1y` = 365d, `all` = omit `from`. Always sends `points=100` to let the backend downsample. Currency passed through.
- `mode=abs`: value = `point.total_value`; formats `currency_compact`.
- `mode=pct`: value = `(total_value − total_cost) / total_cost × 100`; formats `percent_signed`. Requires `total_cost` on the points. If no returned point has `total_cost`, the chart renders empty with `portfolio.chart.no_cost_data`.
- Empty state (`<2` points after filtering): `portfolio.chart.not_enough_data`.

### Asset Value Over Time chart

Multi-series line chart — one line per ticker that appears in the filtered data. Shares the same controls row as Value Over Time.

- Series determined from distinct `(asset_id, ticker)` pairs across the filtered points. Order: most-recent value descending, ticker ascending as tiebreaker. Colors cycle `chart_1 … chart_5`.
- Data rows: one per filtered point, sorted by `recorded_at` ascending. Row carries `date` + one key per ticker (null when the ticker is absent in that snapshot). Frontend draws gaps with `connectNulls: false`.
- The `mode` toggle does **not** affect this chart — it is always absolute currency with `currency_compact` formatting.
- Legend: frontend renders an interactive legend (per-line visibility toggles, client-side only) when `series.length > 1`.
- Empty state (`<2` filtered points, or no tickers appear in any `assets[]`): `portfolio.chart.not_enough_data`.

### Allocation

Pie chart (donut) with its own controls: group by `asset` / `type`, and a currency selector (only when multiple currencies exist). Snapshot of current positions — historical data is not used.

- Source: the same `positions` array the screen loads; no extra backend call on initial render.
- Filter: keep positions where `currency == state.currency` and `current_value != nil`.
- Group by `asset`: one slice per distinct `asset_id` (`key: asset_id`, `label: ticker`), value = sum of `current_value`.
- Group by `type`: one slice per distinct `asset_type` (`key: asset_type`, `label: asset_type` raw BE string — no i18n translation of type values today), value = sum of `current_value`.
- Sort: value DESC, tiebreaker label ASC. Colors cycle `chart_1 … chart_5`.
- Empty: `slices: []` + `portfolio.allocation.empty` when no filtered positions remain.

### Live mode

A single toggle in the header flips the screen between standard and live mode. Live mode adds a banner (status + refresh) and per-position price-source dots; when off, the tree looks exactly like the historical mode.

- Live state is encoded in the toggle button's action URL — there is no client-side state machine. Clicking the button fetches `/actions/portfolio/live_data?live=<opposite>` which replaces the `live-data-section`.
- `refresh=true` is only meaningful together with `live=true` — it forces the backend to re-fetch prices from providers (cache bust). Supplied by the Refresh button in the banner. If someone sends `refresh=true` with `live=false`, treat as `live=false` and ignore the refresh flag.
- State is not persisted: every `GET /screens/portfolio` starts in standard mode.
- Live mode never affects `charts-section` or `allocation-section` — those remain outside the `live-data-section` and are not rebuilt on the live toggle. Similarly the include-closed form keeps working: its partial update targets the positions-table subtree wherever it lives in the tree.

### Hide-values toggle

Client-side only. No server round-trip. The middleend marks every `text` that displays a monetary value with `sensitive: true`. The toggle (`icon_toggle` with `eye` / `eye-off` icons) fires a `toggle_sensitive` action that the frontend uses to mask/unmask those texts locally. Sits in the header row between the title and the live toggle.

## Interactions summary

| User action | Effect | Underlying call |
|---|---|---|
| Load screen | Full render | `GET /screens/portfolio` → parallel `GET /v1/portfolio` + `GET /v1/portfolio/evolution?last=2` |
| Toggle include-closed checkbox | Replace positions-table subtree only | `POST /actions/portfolio/include_closed` with `{include_closed}` body |
| Change chart timeframe / mode / currency | Replace `charts-section` | `GET /actions/portfolio/evolution?timeframe=…&mode=…&currency=…` |
| Change allocation group-by or currency | Replace allocation section | `GET /actions/portfolio/allocation?group_by=…&currency=…` |
| Toggle Live | Replace `live-data-section` | `GET /actions/portfolio/live_data?live=<bool>` |
| Click Refresh (live only) | Replace `live-data-section` with fresh prices | `GET /actions/portfolio/live_data?live=true&refresh=true` |
| Toggle Hide-values | Mask monetary texts locally | `toggle_sensitive` (client-side; no request) |

Control buttons for timeframe / mode / currency / group-by use a stateless pattern: each button's action URL encodes the **full new state** that clicking it would produce (current value for the fields it doesn't change, its own new value for the field it does). The frontend is a pure renderer — no state machine required.

## Formatting

All user-facing strings are produced in the middleend per `Accept-Language` (default `en`; `en` / `es` supported today).

| Field type | `en` | `es` | Null |
|---|---|---|---|
| Currency | `$1,234.56` | `$1.234,56` | `—` |
| Signed currency | `+$321.67` / `−$85.00` | `+$321,67` / `−$85,00` | `—` |
| Signed percent | `+12.34%` | `+12,34%` | `—` |
| Quantity | `10` / `10.5` (trailing zeros stripped) | `10` / `10,5` | `—` |
| Relative date | `2 days ago`, `just now` | `hace 2 días`, `hace instantes` | `—` |

Currency symbols are derived from the position's `currency` field (`USD` → `$`, `EUR` → `€`, etc.).

## i18n keys

All user-facing strings resolve via the `portfolio.*` namespace plus `time.relative.*` for date formatting. Below is the canonical set of keys this screen owns.

### Screen structure

`portfolio.title`, `portfolio.empty_title`, `portfolio.empty_subtitle`, `portfolio.include_closed`.

### Summary cards

`portfolio.total_value`, `portfolio.total_pnl`, `portfolio.performance`, `portfolio.snapshot_change`, `portfolio.open_positions`.

### Table headers

`portfolio.col.ticker`, `portfolio.col.name`, `portfolio.col.type`, `portfolio.col.quantity`, `portfolio.col.avg_cost`, `portfolio.col.total_cost`, `portfolio.col.market_value`, `portfolio.col.unrealized_pnl`, `portfolio.col.pnl_pct`, `portfolio.col.realized_pnl`, `portfolio.col.last_snapshot`.

### Charts

`portfolio.chart.value_over_time.title`, `portfolio.chart.asset_value_over_time.title`, `portfolio.chart.series.value`, `portfolio.chart.timeframe.1m / 3m / 6m / ytd / 1y / all`, `portfolio.chart.mode.abs` (`$`), `portfolio.chart.mode.pct` (`%`), `portfolio.chart.not_enough_data`, `portfolio.chart.no_cost_data`.

### Allocation

`portfolio.allocation.title`, `portfolio.allocation.group_by.asset`, `portfolio.allocation.group_by.type`, `portfolio.allocation.empty`.

### Live mode

`portfolio.live.toggle`, `portfolio.live.status` (`● Live prices · Updated {time}` — `{time}` is the relative timestamp), `portfolio.live.refresh`, `portfolio.live.warning_prefix` (`⚠ Could not fetch:`).

### Hide values

`portfolio.hide_values.tooltip_inactive` (`Hide values`), `portfolio.hide_values.tooltip_active` (`Show values`).

### Shared date formatting

`time.relative.just_now`, `time.relative.seconds_ago`, `time.relative.minutes_ago`, `time.relative.hours_ago`, `time.relative.days_ago` — each with `{n}` interpolation where applicable.

Concrete strings live in `locales/en.json` and `locales/es.json`. Missing-key fallback: `en`, then the key itself.

## Error handling

| Situation | HTTP | Response |
|---|---|---|
| Missing / invalid / expired JWT | 401 | `{"error":"unauthorized","redirect":"/login"}` |
| Backend returns 401 | 401 | same |
| Backend 5xx / network / malformed | 502 | `{"error":{"code":"BACKEND_ERROR","message":"..."}}` |
| Invalid query param on a partial-update endpoint (enum, bool, integer) | 400 | `{"error":{"code":"BAD_REQUEST","message":"..."}}` |
| Evolution call for the Snapshot Change card fails | 200 on `/screens/portfolio` | Screen renders; `Snapshot Change` falls back to `—` per currency. |

Positions is always the critical path; everything else is best-effort where noted.

## Acceptance criteria

- [ ] `GET /screens/portfolio` without a valid JWT returns `401` with the documented redirect.
- [ ] With a valid JWT the middleend issues `GET /v1/portfolio` and `GET /v1/portfolio/evolution?last=2` in parallel, forwarding `Authorization`.
- [ ] The response is a `screen` with `id: portfolio` and `props.title` resolved per `Accept-Language`.
- [ ] Non-empty positions produce, in order: a live-data section (header + summary row of 5 cards + include-closed form + positions table), a charts section (controls + Value Over Time + Asset Value Over Time), and an allocation section.
- [ ] Empty positions render only the localized empty block — none of the data sections.
- [ ] Summary cards emit one line per currency, shared currency order across Total Value / Total P&L / Performance / Snapshot Change.
- [ ] Positions table uses the `table` / `table_row` base components with 11 columns, ids and alignments per the table above.
- [ ] Monetary cells and Total Value / Total P&L summary values carry `sensitive: true`. Percentages, counts, and non-monetary cells do not.
- [ ] P&L values render with `color: positive` when `> 0`, `negative` when `< 0`, no color otherwise.
- [ ] All user-facing strings resolve via i18n `en` / `es`; no hardcoded literals in the response.
- [ ] Include-closed toggle: the checkbox lives outside the positions-table subtree; `POST /actions/portfolio/include_closed` replaces only that subtree; toggle state resets on every full screen load.
- [ ] Chart controls: every button's reload URL carries the full new state; selected option styled `primary/solid`, others `secondary/ghost`.
- [ ] `mode=pct` with no `total_cost` returns an empty chart with `portfolio.chart.no_cost_data`. Under 2 points in either chart returns `portfolio.chart.not_enough_data`.
- [ ] Allocation initial state: `group_by=asset`, currency = first by total-value DESC. Slices sorted by value DESC, colors cycle `chart_1..chart_5`.
- [ ] Live toggle: default off on every full load; toggling replaces only `live-data-section`; `refresh=true` requires `live=true`; live mode emits the banner and price-source dots, standard mode does not.
- [ ] Hide-values toggle emits `toggle_sensitive` actions on click — no server request. The middleend marks the correct texts with `sensitive: true`.
- [ ] Charts and allocation sections are outside `live-data-section` and do not rebuild when the live toggle fires.
- [ ] Backend 5xx on the positions call returns `502 BACKEND_ERROR`. Evolution (last=2) failure does not break the screen; Snapshot Change falls back to `—`.
- [ ] Every partial-update endpoint follows the same 400 / 401 / 502 error mapping documented above.
