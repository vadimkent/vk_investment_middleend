# Snapshots Screen — Design

Design spec for the Snapshots screen in the vk-investment middleend. Follows the same screen-as-SDUI pattern as trades and assets, with two net-new SDUI additions: a custom `wizard` component and an `expandable` prop on the base `table_row`.

---

## 1. Overview

Management screen for portfolio **snapshots** — timestamped captures of asset prices/values. One snapshot has N entries (one per asset). The user either fills entries manually (via a multi-step wizard) or triggers an auto-snapshot that fetches prices from configured providers. Snapshots feed the portfolio evolution chart and valuation.

Screen capabilities:

1. Browse paginated list with a Full/Partial filter; expand rows inline to view per-snapshot entries and compare quickly across snapshots.
2. Register a snapshot manually via a multi-step wizard (one step per asset).
3. Trigger an **auto-snapshot** that calls `POST /v1/snapshots/auto`. The BE creates the snapshot from provider prices; the middleend then **opens the edit wizard pre-populated** on the newly-created snapshot so the user can refine notes / prices / add manual entries.
4. Edit an existing snapshot (pencil icon) — reuses the same wizard in edit mode.
5. Delete a snapshot (simple confirmation modal).

---

## 2. Endpoints

All endpoints are **protected** (JWT). Missing / invalid / expired JWT → `401 {"error":"unauthorized","redirect":"/login"}`. Backend 5xx / network / malformed → `502 BACKEND_ERROR`. Invalid query or body → `400 BAD_REQUEST`. Missing snapshot (edit / delete modal GET) → `404 NOT_FOUND`.

| Method | Path | Purpose |
|---|---|---|
| `GET`    | `/screens/snapshots` | Full screen render. Query: optional `is_full_snapshot` (`true` / `false`), `offset` (non-negative integer). Deep-linkable filtered / paginated state. |
| `GET`    | `/actions/snapshots/list` | Re-renders the filter+table+pagination subtree. Used by the filter select and the Prev/Next buttons. Query: `is_full_snapshot`, `offset`. |
| `GET`    | `/actions/snapshots/create_wizard` | Returns the create wizard with an empty form. Query: current list context (`is_full_snapshot`, `offset`) so the submit URL preserves it. |
| `GET`    | `/actions/snapshots/edit_wizard?id=<id>` | Fetches the snapshot and returns the edit wizard pre-populated. |
| `GET`    | `/actions/snapshots/delete_modal?id=<id>` | Returns the delete confirmation modal. |
| `POST`   | `/actions/snapshots/create?is_full_snapshot=<f>&offset=<n>` | Creates a snapshot from the wizard payload. On success returns a full-list refresh; on validation error returns the same wizard with an inline error. |
| `POST`   | `/actions/snapshots/auto?is_full_snapshot=<f>&offset=<n>` | Triggers `POST /v1/snapshots/auto`. On success returns a full-list refresh **plus** the edit wizard pre-populated on the newly-created snapshot, with an info banner and any provider warnings. On terminal failure returns a snackbar. |
| `PATCH`  | `/actions/snapshots/:id?is_full_snapshot=<f>&offset=<n>` | Updates a snapshot (notes + entry prices diff + new entries). Same response shape as create. |
| `DELETE` | `/actions/snapshots/:id?is_full_snapshot=<f>&offset=<n>` | Deletes. Same response shape as create. No `force` flag. |

Every non-screen endpoint returns an `ActionResponse`. Mutations that succeed emit a `replace` of the entire screen root plus a success snackbar; mutations that fail with a BE validation error emit a `replace` of the wizard / modal subtree only, re-populated with the user's input and an inline error banner. The `is_full_snapshot` and `offset` query params on mutation endpoints carry the current list context so the refreshed list preserves filter and page after a mutation.

### Backend dependencies

- `GET /v1/snapshots` — paginated list. The middleend always sends `size=10`, `sort=recorded_at`, `order=desc`, plus optional `is_full_snapshot` and `offset`. Response shape: `{ snapshots, total, size, offset }`.
- `GET /v1/snapshots/:id` — single snapshot. Used by edit and delete modal GETs (fresh data + 404).
- `POST /v1/snapshots` — create. Body: `{ recorded_at, notes?, entries:[{asset_id, current_price?, current_value_override?}] }`. `source`, `quantity`, `is_full_snapshot` are all system-assigned. Validation errors (422) include `ASSET_NOT_FOUND`, `FUTURE_DATED_SNAPSHOT`, `DUPLICATE_SNAPSHOT_ENTRY`, `MISSING_VALUE_OVERRIDE`, `CONFLICTING_SNAPSHOT_VALUE`.
- `POST /v1/snapshots/auto` — auto create from providers. Body optional (middleend sends `{}`). Response: `{ snapshot, warnings? }`. Terminal errors: `NO_PRICE_PROVIDERS_CONFIGURED` (422), `ALL_PROVIDERS_FAILED` (502), `PROVIDER_NOT_CONFIGURED` (500 — surfaced as `BACKEND_ERROR`).
- `PATCH /v1/snapshots/:id` — update. Body: `{ notes?, entries? }`. The middleend diffs against the original (fetched at wizard-open time) and sends only changed entries + `notes` if it changed. Existing entries cannot be removed; `recorded_at`, `quantity`, `source` are immutable. Validation errors: same set as create (minus `FUTURE_DATED_SNAPSHOT`, since `recorded_at` is immutable).
- `DELETE /v1/snapshots/:id` — no flags.
- **Assets catalog** (see [`../../../spec/shared/assets-catalog.md`](../../../spec/shared/assets-catalog.md)) — loaded on the screen render (for ticker resolution in the list + entries panel) and on every wizard GET (to iterate assets for per-asset steps).

---

## 3. Layout

Three logical regions stacked vertically (same shape as trades/assets):

1. **Screen header** — title `Snapshots` + two buttons: `New Snapshot` (opens the create wizard) and `Auto Snapshot` (triggers the auto flow).
2. **List region** — filter select + table + pagination. This is the subtree the list / filter / pagination actions replace.
3. **Modal slot** — initially empty; the wizard and delete modal insert here. Sibling of the list region so filter changes and pagination do not wipe an open wizard, and mutation success replaces the root entirely (collapsing the modal back to empty).

---

## 4. Data and business rules

### List

- **Fixed server-side sort**: `recorded_at DESC`. Not user-configurable.
- **Page size**: 10 (hardcoded here; mirrors trades/assets).
- **Filter**: a single `is_full_snapshot` select with options `Any` (empty value, not forwarded) / `Full` (`true`) / `Partial` (`false`). Changing resets `offset=0`.
- **Pagination math**: `page = offset/size + 1`, `total_pages = ceil(total/size)`. Prev disabled at `offset == 0`; Next disabled when `offset + size >= total`. Pagination row omitted entirely when `total <= size`. Each pagination button's action URL carries the full target state.

### Table columns

Six cells per row (Date, Type, Entries, Sources, Notes, Actions). The frontend auto-adds a chevron column to the left since rows are `expandable: true`.

| Column | Rendering |
|---|---|
| Date | `YYYY-MM-DD HH:mm` from `recorded_at` (datetime, not date-only — snapshots can be taken multiple times per day). |
| Type | Badge: `Full` (color `positive`) / `Partial` (color `neutral`). Uppercase. |
| Entries | Integer count (`len(snapshot.entries)`). |
| Sources | Compact list of unique source values across this snapshot's entries (e.g. `MANUAL`, `MANUAL · COINGECKO`). Capped at 3 visible, with `+N` suffix if more. Uppercase. |
| Notes | Plain text truncated to 40 chars with ellipsis when longer. Full notes visible in the edit wizard. |
| Actions | Edit (pencil) + Delete (trash) icon buttons. |

No cell carries `sensitive: true`. Snapshots are not masked by HideValues (consistent with trades).

### Expanded row (details)

Each table row is `expandable: true` and carries a `details` subtree with a nested `table` of entries:

| Column | Rendering |
|---|---|
| Asset | Ticker resolved via the assets catalog. If the asset no longer exists, render the raw UUID as a fallback. |
| Quantity | `FormatQuantity(quantity, lang)`; `—` when `quantity` is `null` (complex assets). |
| Price | `FormatMoney(current_price, asset.currency, lang)`; `—` when `null`. |
| Value Override | `FormatMoney(current_value_override, asset.currency, lang)`; `—` when `null`. |
| Source | Badge: `MANUAL` / `COINGECKO` / `TWELVE_DATA` / `ALPHA_VANTAGE`. |

All entries are pre-rendered in the tree (hidden by default). Expand is 100% client-side; multiple rows may be open simultaneously. Expansion state resets on any `replace` that rebuilds the list subtree.

### Empty states

- **No snapshots at all** (no filter, `total == 0`): `snapshots.empty_title` / `snapshots.empty_subtitle`. No CTA in the empty state — the user still has the header buttons.
- **No snapshots match the filter**: `snapshots.empty_filtered_title` / `snapshots.empty_filtered_subtitle`. Filter stays visible so the user can clear it.

Pagination omitted in both.

---

## 5. Create flow (manual wizard)

Trigger: header `New Snapshot` button.

1. `GET /actions/snapshots/create_wizard?is_full_snapshot=<f>&offset=<n>` — the middleend loads the assets catalog and emits a `wizard` with `mode: "create"`:
   - **Step `info`** — inputs `recorded_at` (datetime-local, required, `max` = now) and `notes` (textarea, optional, max 500 chars).
   - **Steps `entry`** — one per asset in the catalog. Step header displays ticker + name + asset_type badge. Per-step content:
     - If `is_complex = true`: single input `current_value_override` (text, required-when-included, `> 0`). No toggle.
     - Otherwise: a segmented toggle **Price / Value Override** (default: Price) + a single input bound to the selected mode. Switching toggles clears the other field. Required-when-included, `> 0`.
     - `skippable: true`, `include_default: false`.
   - **Step `summary`** — descriptive text. Submit action: `POST /actions/snapshots/create?is_full_snapshot=<f>&offset=<n>`.
2. The create `POST` handler parses hidden form inputs per the wizard naming convention (`entries[<asset_id>].mode`, `entries[<asset_id>].current_price`, `entries[<asset_id>].current_value_override`), plus `recorded_at` and `notes`. It builds the BE body using only entries the wizard included (excluded steps are not submitted by the frontend). `notes` is omitted from the body when empty.
3. BE validation errors (422) surface as inline errors on the wizard (banner `variant: error`): `ASSET_NOT_FOUND`, `FUTURE_DATED_SNAPSHOT`, `DUPLICATE_SNAPSHOT_ENTRY`, `MISSING_VALUE_OVERRIDE`, `CONFLICTING_SNAPSHOT_VALUE`. The wizard is re-emitted with the user's inputs preserved; the list is not touched.
4. Success → `ActionResponse` replaces the screen root with a fresh list + empty modal slot + success snackbar `snapshots.create.success`.

---

## 6. Auto-snapshot flow

Trigger: header `Auto Snapshot` button. No pre-confirmation — submit is direct.

1. Middleend calls `POST /v1/snapshots/auto` with body `{}` (notes are entered later via the edit wizard if desired).
2. **Success (`201`)** — the middleend captures the full `snapshot` object and `warnings` from the response body (no extra GET needed), then builds an `ActionResponse` that performs two replaces:
   - Replace the screen root with a fresh list (the newly-created snapshot appears on page 1 by virtue of `recorded_at DESC`).
   - Replace `snapshots-modal-slot` with the **edit wizard** pre-populated on the newly-created snapshot, carrying:
     - `banner`: `{ variant: "info", message: snapshots.auto.banner, dismissible: true }` — *"Snapshot creado automáticamente. Ajustá entries o cerrá para dejarlo así."*
     - If `warnings` is present, a second inline banner `variant: warning` with title `snapshots.auto.warnings_title` and the list of failed tickers.
   - `feedback`: success snackbar `snapshots.auto.success`.
3. **Terminal failures** (no snapshot created):
   - `NO_PRICE_PROVIDERS_CONFIGURED` (422) → `ActionResponse{feedback: snackbar error}` using `snapshots.auto.no_providers` (includes a hint to configure a price provider on an asset).
   - `ALL_PROVIDERS_FAILED` (502) → snackbar error using `snapshots.auto.all_failed`.
   - `PROVIDER_NOT_CONFIGURED` (500) → surfaced as `502 BACKEND_ERROR` (operator misconfiguration, not user error).

"Cancel" on the post-auto wizard **does not delete** the snapshot — the banner copy makes this explicit. To remove an unwanted auto-snapshot, the user uses the list's Delete action.

---

## 7. Edit flow

Trigger: pencil icon in a row, or opened automatically after an auto-snapshot (same endpoint).

1. `GET /actions/snapshots/edit_wizard?id=<id>` — middleend calls `GET /v1/snapshots/:id` (404 if gone), loads the assets catalog, and emits a `wizard` with `mode: "edit"`:
   - **Step `info`** — `recorded_at` rendered as static `text` (immutable per BE contract). `notes` textarea editable, pre-filled.
   - **Steps `entry`** — one per asset in the catalog:
     - **Asset already in the snapshot**: `include_default: true`, `skippable: false`, toggle initialized per which field has a value, inputs pre-filled. A subtle indicator reads *"Ya en snapshot, no se puede quitar"* (`snapshots.wizard.already_included`) — BE does not allow removing entries via PATCH.
     - **Asset not in the snapshot**: `include_default: false`, `skippable: true`, inputs empty — same UX as create.
   - **Step `summary`** — submit action: `PATCH /actions/snapshots/:id?is_full_snapshot=<f>&offset=<n>`.
2. The PATCH handler diffs the submitted form against the original snapshot (fetched at modal-open time — the handler re-fetches inside the PATCH handler too, to avoid stale diffs on concurrent edits):
   - `notes` included only if it changed.
   - `entries` contains only: (a) new entries (asset not in the original, marked `included=true` in the submit), and (b) existing entries whose `current_price`, `current_value_override`, or mode changed.
   - Entries that are unchanged from the original are omitted entirely (no-op prevention).
3. BE validation errors surface inline on the wizard as in create. Success → replace root + `snapshots.edit.success` snackbar.

---

## 8. Delete flow

Trigger: trash icon in a row.

1. `GET /actions/snapshots/delete_modal?id=<id>` — middleend calls `GET /v1/snapshots/:id` to have the date for the confirmation message; emits a simple modal:
   - Title: `snapshots.delete.title`.
   - Message interpolated: `snapshots.delete.confirm` → *"Delete snapshot from {date}? This will affect portfolio calculations."*
   - Cancel (`dismiss`) + destructive Delete buttons.
2. `DELETE /actions/snapshots/:id` handler calls `DELETE /v1/snapshots/:id`. No `force` flag — deletion is always unconditional.
3. Success → replace root + `snapshots.delete.success` snackbar.

---

## 9. New custom component: `wizard`

Net-new custom SDUI component, added to `spec/sdui-custom-components.md`. Generic — not snapshot-specific — so it can be reused later (import, analysis).

### Why custom

A wizard needs local state for `currentStep`, Back/Next navigation without a round-trip, per-step input persistence while the user moves around, and per-step validation before advancing. Composing this from base primitives (`visible_when` + counter) is fragile; server-driven (one round-trip per Next) is chatty and feels wrong for a flow with many steps. The wizard encapsulates the state machine in the frontend, same pattern as `line_chart` and `pie_chart` encapsulate their interactive state.

### Props

| Prop | Type | Required | Description |
|---|---|---|---|
| `mode` | enum | yes | `create` / `edit`. Used by the frontend to pick button copy and entry-step semantics (see `skippable`). |
| `title` | string | yes | Wizard title. Localized by the middleend. |
| `steps` | `Step[]` | yes | Ordered steps; at least 1. |
| `submit_action` | `Action` | yes | Action executed from the summary step's Submit button (typically `submit` targeting the create/PATCH endpoint). |
| `dismiss_action` | `Action` | yes | Action executed when the user closes the wizard (typically a client-side `replace` that empties the modal slot). |
| `banner` | `Banner` | no | Optional banner rendered above the step content. Used by auto-snapshot flow and validation-error re-emission. |
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
| `dismissible` | bool | no | Default `false`. |

### Frontend behavior

1. **Step indicator** at the top: `Step X of Y` + a chip row with each step's `label`. Chips are clickable — free jump between steps. Jumping does not validate.
2. **Navigation**: Back always works (no validation). Next / Include validates required inputs of the current step before advancing (validation uses the existing input `required`, `pattern`, `min`, `max` props). Skip marks the step as excluded and advances.
3. **Include map**: the wizard holds `{stepId → included:bool}`, seeded from each step's `include_default`. Skip sets `false`; Include sets `true`; Update (edit mode, existing entry) keeps `true` and advances. Excluded steps don't contribute inputs on Submit.
4. **Submit**: on the summary step, the wizard collects inputs from (a) all `kind=info` steps (always included) and (b) all `kind=entry` steps where `included=true`. The form body uses the naming convention below. The wizard then executes `submit_action` with that body.
5. **Dismiss**: executes `dismiss_action`.

### Hidden input naming

For each `kind=entry` step representing an asset, inputs are named with a flat bracket convention:

- `entries[<asset_id>].mode` — `price` or `override`.
- `entries[<asset_id>].current_price` — present when `mode=price`.
- `entries[<asset_id>].current_value_override` — present when `mode=override`.

The `info` step uses plain names: `recorded_at`, `notes`. Complex-asset entry steps omit `mode` (always `override`).

The middleend handlers parse this flat shape into the BE's nested `entries` array.

### Validation

Format validation runs through the `input` props (`required`, `max_length`, `pattern`, `min`, `max`). The wizard does not invent its own validation primitives. BE validation errors (422) arrive as an `ActionResponse` that replaces the modal subtree with the same wizard (inputs preserved) plus an `error` banner. The wizard re-opens on the **summary** step by default (entry-level errors usually aggregate there); the middleend can set a `wizard.initial_step_id` prop to override (used for `FUTURE_DATED_SNAPSHOT` which belongs to the `info` step).

### Summary step content

The summary step's children come from the middleend as a short descriptive paragraph; the **list of included entries is derived client-side** from the include-map rather than server-emitted. This keeps the summary reactive to Skip/Include changes without re-emitting the wizard.

---

## 10. Base component change: `table_row` gains `expandable` + `details`

Additive change to the base SDUI set. Spec updated in `spec/sdui-base-components.md`; helper updated in `internal/components/table.go`.

### New prop and slot

| Prop | Type | Required | Description |
|---|---|---|---|
| `expandable` | bool | no | Default `false`. When `true`, the row is toggleable (click anywhere on the main row to expand / collapse) and the frontend renders a chevron indicator. |

| Slot | Type | Description |
|---|---|---|
| `details` | `Component[]` | Subtree rendered as a full-width panel directly beneath the row when expanded. Pre-emitted in the tree (not fetched on expand). |

### Go helper

```go
TableRowExpandable(id string, cells []Component, details ...Component) Component
```

The existing `TableRow` signature is unchanged (non-expandable rows remain the default). The frontend auto-adds a chevron column (24px fixed) to the left of the table header iff any row in the table is `expandable: true`, so the header alignment is preserved.

### State semantics

- Expand state is **local per row id**. Multiple rows in the same table may be expanded simultaneously.
- State resets on any `replace` that rebuilds the table subtree (e.g. filter change, pagination, mutation refresh). Not persisted across page loads.
- `details` children subtree is already in the DOM — expanding never fires a network request.

### Why change the base vs a new custom component

Expandable rows are a generic table pattern. Reusing `table` / `table_row` means one subgrid implementation, one header-alignment path, one styling contract. A `custom_expandable_table` would duplicate the base `table`'s layout logic without adding value.

Cost: one new prop + one new slot on `table_row`. Additive, non-breaking.

---

## 11. i18n keys

Namespace `snapshots.*`. Shared: `common.cancel`.

### Screen structure

`snapshots.title`, `snapshots.new`, `snapshots.auto`,
`snapshots.empty_title`, `snapshots.empty_subtitle`,
`snapshots.empty_filtered_title`, `snapshots.empty_filtered_subtitle`.

### Filter

`snapshots.filter.type`, `snapshots.filter.type_any`, `snapshots.filter.type_full`, `snapshots.filter.type_partial`.

### Table headers

`snapshots.col.date`, `snapshots.col.type`, `snapshots.col.entries`, `snapshots.col.sources`, `snapshots.col.notes`.

### Entries (nested table)

`snapshots.entries.col.asset`, `snapshots.entries.col.quantity`, `snapshots.entries.col.price`, `snapshots.entries.col.value_override`, `snapshots.entries.col.source`.

### Badges

`snapshots.type.full`, `snapshots.type.partial`,
`snapshots.source.manual`, `snapshots.source.coingecko`, `snapshots.source.twelve_data`, `snapshots.source.alpha_vantage`.

### Pagination

`snapshots.pagination.prev`, `snapshots.pagination.next`, `snapshots.pagination.page_of` (`Page {current} of {total}`).

### Wizard

`snapshots.wizard.info`, `snapshots.wizard.summary`,
`snapshots.wizard.step_of` (`Step {current} of {total}`),
`snapshots.wizard.back`, `snapshots.wizard.next`,
`snapshots.wizard.skip`, `snapshots.wizard.include`, `snapshots.wizard.update`,
`snapshots.wizard.already_included`.

### Create / edit / delete / auto

`snapshots.create.title`, `snapshots.create.submit`, `snapshots.create.success`.

`snapshots.edit.title` (`Edit snapshot from {date}`), `snapshots.edit.submit`, `snapshots.edit.success`.

`snapshots.delete.title`, `snapshots.delete.confirm` (`Delete snapshot from {date}? This will affect portfolio calculations.`), `snapshots.delete.submit`, `snapshots.delete.success`.

`snapshots.auto.success`, `snapshots.auto.banner`,
`snapshots.auto.warnings_title` (`Algunos assets no pudieron actualizarse`),
`snapshots.auto.no_providers`, `snapshots.auto.all_failed`.

### Form labels (shared across create and edit)

`snapshots.form.recorded_at`, `snapshots.form.recorded_at_readonly`,
`snapshots.form.notes`, `snapshots.form.notes_placeholder`,
`snapshots.form.toggle_price`, `snapshots.form.toggle_override`,
`snapshots.form.current_price`, `snapshots.form.current_value_override`.

### Validation

The middleend does not translate BE error codes. It surfaces the localized `message` from the BE body (backend localizes by `Accept-Language`). Fallback: the raw code.

### Shared

`common.cancel`.

Concrete strings live in `locales/en.json` and `locales/es.json`. Missing-key fallback: `en`, then the key itself.

---

## 12. Error handling

| Situation | HTTP | Body |
|---|---|---|
| Missing / invalid / expired JWT (any endpoint) | 401 | `{"error":"unauthorized","redirect":"/login"}` |
| Backend 401 downstream | 401 | same |
| Backend 5xx / network / malformed | 502 | `{"error":{"code":"BACKEND_ERROR","message":"..."}}` |
| Invalid query param (`is_full_snapshot` outside `true`/`false`, `offset` non-integer or negative, missing `id` on modal endpoints) | 400 | `{"error":{"code":"BAD_REQUEST","message":"..."}}` |
| Snapshot not found (edit / delete modal GET) | 404 | `{"error":{"code":"NOT_FOUND"}}` |
| BE validation error on a mutation (4xx with a `code`) | 200 | `ActionResponse{replace, target_id: <modal>, tree: <same wizard + inline error>}` |
| `NO_PRICE_PROVIDERS_CONFIGURED` / `ALL_PROVIDERS_FAILED` (auto) | 200 | `ActionResponse{feedback: snackbar error}` (no modal open) |

BE error codes to surface as inline errors: `ASSET_NOT_FOUND`, `FUTURE_DATED_SNAPSHOT`, `DUPLICATE_SNAPSHOT_ENTRY`, `MISSING_VALUE_OVERRIDE`, `CONFLICTING_SNAPSHOT_VALUE`, `SNAPSHOT_NOT_FOUND`. Use the localized `message` from the BE body; do not re-translate codes in the middleend.

---

## 13. Acceptance criteria

- [ ] `GET /screens/snapshots` without a valid JWT returns `401` with the documented redirect.
- [ ] With a valid JWT the middleend issues `GET /v1/snapshots?size=10&sort=recorded_at&order=desc[&is_full_snapshot=…][&offset=…]`, forwarding `Authorization`.
- [ ] The middleend loads the full assets catalog on every screen render and every wizard GET (for ticker resolution and per-asset steps).
- [ ] Screen renders with three logical regions: header (title + `New Snapshot` + `Auto Snapshot`), list region (filter + table + pagination), modal slot starting empty.
- [ ] List region is replaceable independently of the modal slot — filter / pagination actions target the list region only.
- [ ] Mutation actions replace the screen root and collapse the modal slot back to empty on success; on BE validation error they replace only the modal slot with the wizard pre-populated and a localized inline error banner.
- [ ] Filter select exposes `Any` (empty value) plus `Full` (`true`) / `Partial` (`false`). Changing re-fetches the list and resets `offset` to 0.
- [ ] Table has 5 data columns + Actions + auto-added chevron column; every row is `expandable: true` and carries a `details` subtree.
- [ ] Cell rendering matches the spec: `recorded_at` as `YYYY-MM-DD HH:mm`, type badge (Full/Partial), entries count, sources compacted to 3 + `+N`, notes truncated at 40 chars.
- [ ] Expanded row renders a nested `table` of entries: asset (ticker / UUID fallback), quantity via `FormatQuantity`, price/override via `FormatMoney` using the asset's currency (`—` for nulls), source badge.
- [ ] Multiple rows can be expanded simultaneously; expand state is client-side only and resets on list replace.
- [ ] Pagination omitted when `total <= size`; otherwise Prev (disabled at start), localized `Page X of Y`, Next (disabled at end); button URLs carry the current filter + target offset.
- [ ] Empty list distinguishes "no snapshots" from "no match for filter" via two different copy pairs.
- [ ] Create wizard: step `info` (required `recorded_at`, optional `notes`) + one `entry` step per asset (toggle Price/Override or forced Override for complex assets) + `summary`. Step chip navigation jumps freely. Skip excludes a step from submit; Include validates + marks included.
- [ ] Edit wizard: `recorded_at` rendered static; existing-entry steps are `skippable: false, include_default: true` with pre-filled values; new-entry steps behave like create. PATCH body contains only the diff (changed entries + `notes` if changed).
- [ ] Auto-snapshot: triggers `POST /v1/snapshots/auto`, on 201 replaces root + opens edit wizard on the new snapshot with info banner and optional warning banner listing failed tickers. On `NO_PRICE_PROVIDERS_CONFIGURED` / `ALL_PROVIDERS_FAILED` emits a snackbar error without opening a modal.
- [ ] Delete modal: confirmation interpolates the snapshot date; submit unconditionally deletes (no `force`).
- [ ] Filter and offset persist across successful mutations (the fresh list uses the same values that were active at mutation time).
- [ ] Success feedback uses the four success keys (`create`, `edit`, `delete`, `auto`).
- [ ] All user-facing strings resolve via i18n `en` / `es`. BE validation messages surface localized per `Accept-Language`; otherwise fall back to the BE `code`.
- [ ] 404 on edit / delete modal GET when the snapshot is gone; 400 on invalid query; 401 with redirect on auth issues; 502 on BE 5xx / `PROVIDER_NOT_CONFIGURED`.

---

## 14. New SDUI additions summary

This design introduces two SDUI changes that must land before or alongside the snapshots screen:

1. **`wizard` custom component** — defined in §9. Update `spec/sdui-custom-components.md`, implement on the frontend, add a Go constructor in `internal/components/`.
2. **`table_row.expandable` + `details` slot** — defined in §10. Update `spec/sdui-base-components.md`, add `TableRowExpandable` helper in `internal/components/table.go`, implement on the frontend.

Both are genuinely cross-cutting: they extend the SDUI contract and will be reused by other screens (import flow, analysis). Treat them as prerequisite layers of the snapshots spec rather than snapshot-internal code.

---

## 15. Canonical spec updates

The canonical specs in `spec/` are the project source of truth and must be updated alongside the implementation. This design document (in `docs/superpowers/specs/`) is a brainstorm artifact, not a substitute.

The implementation plan must include these spec edits:

1. **`spec/screens/snapshots.md`** — new canonical screen spec. Mirrors shipped behavior at the level of detail trades.md / assets.md have today (purpose, endpoints, BE dependencies, layout, data and business rules per flow, i18n keys, error handling, acceptance criteria). It takes this design as input but is written for the shipped system, not the brainstorm.
2. **`spec/sdui-custom-components.md`** — add a new §3 (or appropriate position) documenting the `wizard` component: props, sub-types, frontend behavior, hidden-input naming convention, validation handling, example. Same level of detail as `line_chart` / `pie_chart`.
3. **`spec/sdui-base-components.md`** — update the `table_row` section to add the `expandable` prop and the `details` slot, including the auto-chevron-column behavior, state semantics, and the `TableRowExpandable` Go helper.
4. **`spec/spec.md`** — flip the `Snapshots | screens/snapshots.md — TBD` row to a real link.

The canonical specs must match what ships — no forward-looking text. If implementation reveals deviations from this design, the canonical spec reflects the shipped reality and this design doc stays as historical context.

---

## 16. Out of v1 scope

- Date-range filter (`from` / `to`) on the list — BE supports it, but v1 matches the legacy FE with only the type filter. Additive later.
- Bulk delete.
- Inline-edit a single entry from the expanded row (requires jumping into the wizard today).
- "Undo" for auto-snapshot — the created snapshot stays in the BE; user deletes explicitly.
