# Snapshots Screen

Management screen for portfolio **snapshots** — timestamped captures of asset prices and values. Each snapshot has one entry per included asset; quantity is auto-computed from trade history at `recorded_at`. Snapshots feed the portfolio evolution chart and valuation. Entries are built manually via a multi-step wizard or auto-populated by triggering a price-provider fetch.

## Purpose

Let the user:

1. **Browse** all snapshots with pagination and a Full/Partial filter; expand rows inline to view per-snapshot entries.
2. **Register** a snapshot manually via a multi-step wizard (one step per asset in the catalog).
3. **Trigger an auto-snapshot** (`POST /v1/snapshots/auto`) that creates a snapshot from provider prices; on success the edit wizard opens pre-populated so the user can refine entries or notes.
4. **Correct** a snapshot's notes and mutable entry price fields, or add new entries to an existing snapshot, via the same wizard in edit mode.
5. **Remove** a snapshot with a simple confirmation modal.

## Endpoints

All endpoints are **protected** (JWT). Missing / invalid / expired JWT → `401 {"error":"unauthorized","redirect":"/login"}`. Backend 5xx / network / malformed → `502 BACKEND_ERROR`. Invalid query or body → `400 BAD_REQUEST`.

| Method | Path | Purpose |
|---|---|---|
| `GET`    | `/screens/snapshots` | Full screen render. Query: optional `is_full_snapshot` (`true` / `false`), `offset` (non-negative integer). Deep-linkable filtered / paginated state. |
| `GET`    | `/actions/snapshots/list` | Re-renders the filter+table+pagination subtree. Used by the filter select and Prev/Next buttons. Query: `is_full_snapshot`, `offset`. |
| `GET`    | `/actions/snapshots/create_wizard` | Returns the create wizard with an empty form. Query: current list context (`is_full_snapshot`, `offset`) so the submit URL preserves it. |
| `GET`    | `/actions/snapshots/edit_wizard?id=<id>` | Fetches the snapshot and returns the edit wizard pre-populated. |
| `GET`    | `/actions/snapshots/delete_modal?id=<id>` | Returns the delete confirmation modal. |
| `POST`   | `/actions/snapshots/create?is_full_snapshot=<f>&offset=<n>` | Creates a snapshot from the wizard payload. On success returns a full-list refresh; on validation error returns the same wizard with an inline error. |
| `POST`   | `/actions/snapshots/auto?is_full_snapshot=<f>&offset=<n>` | Triggers `POST /v1/snapshots/auto`. On success returns a full-list refresh plus the edit wizard pre-populated on the new snapshot. On terminal failure returns a snackbar error. |
| `PATCH`  | `/actions/snapshots/:id?is_full_snapshot=<f>&offset=<n>` | Updates a snapshot. Same response shape as create. |
| `DELETE` | `/actions/snapshots/:id?is_full_snapshot=<f>&offset=<n>` | Deletes. Same response shape as create. No `force` flag. |

Every non-screen endpoint returns an `ActionResponse`. Mutation endpoints that succeed emit a `replace` of the entire screen root plus a success snackbar; mutation endpoints that fail with a backend validation error emit a `replace` of the wizard / modal subtree only, re-populated with the user's input and an inline error banner. The `is_full_snapshot` and `offset` query params on mutation endpoints carry the current list context so the refreshed list preserves filter and page after a mutation.

## Backend dependencies

- `GET /v1/snapshots` — paginated list. The middleend always sends `size=10`, `sort=recorded_at`, `order=desc`, plus optional `is_full_snapshot` and `offset`. Response shape: `{ snapshots, total, size, offset }`. The backend default order is `asc`; the middleend always sends `desc` explicitly.
- `GET /v1/snapshots/:id` — single snapshot. Used by the edit and delete modal endpoints (fresh data + 404).
- `POST /v1/snapshots` — create. Body: `{ recorded_at, notes?, entries:[{asset_id, current_price?, current_value_override?}] }`. `source`, `quantity`, and `is_full_snapshot` are system-assigned; the middleend never sends them. Validation errors (`422`): `ASSET_NOT_FOUND`, `FUTURE_DATED_SNAPSHOT`, `DUPLICATE_SNAPSHOT_ENTRY`, `MISSING_VALUE_OVERRIDE`, `CONFLICTING_SNAPSHOT_VALUE`.
- `POST /v1/snapshots/auto` — auto-create from provider prices. Middleend sends `{}`. Response: `{ snapshot, warnings? }`. Terminal errors: `NO_PRICE_PROVIDERS_CONFIGURED` (422), `ALL_PROVIDERS_FAILED` (502), `PROVIDER_NOT_CONFIGURED` (500, surfaced as `BACKEND_ERROR`).
- `PATCH /v1/snapshots/:id` — update. Body: `{ notes?, entries? }`. The middleend diffs against the original and sends only changed entries plus `notes` if it changed. `recorded_at`, `quantity`, and `source` are immutable; existing entries cannot be removed. Validation errors: same set as create, minus `FUTURE_DATED_SNAPSHOT` (since `recorded_at` is immutable).
- `DELETE /v1/snapshots/:id` — no flags. Returns `204`.
- **Assets catalog** (see [`../shared/assets-catalog.md`](../shared/assets-catalog.md)) — loaded on every screen render and every wizard GET, for ticker resolution in the list and per-asset steps in the wizard.

## Layout

Three logical regions stacked vertically:

1. **Screen header** — title (`Snapshots`) and two buttons: `New Snapshot` (opens the create wizard) and `Auto Snapshot` (triggers the auto flow directly).
2. **List region** — the main interactive area holding the filter select on one row, then the table, then pagination. This is the subtree the list / filter / pagination actions replace.
3. **Modal slot** — initially empty; the wizard and delete modal insert here. The slot is a **sibling** of the list region so filter changes and pagination do not wipe an open wizard, and mutation success replaces the root entirely (collapsing the modal back to empty).

## Data and business rules

### List

- **Fixed server-side sort**: `recorded_at DESC`. Not user-configurable.
- **Page size**: 10 (hardcoded; mirrors trades/assets).
- **Filter**: a single `is_full_snapshot` select with options `Any` (empty value, not forwarded to the backend) / `Full` (`true`) / `Partial` (`false`). Changing the filter resets `offset` to 0.
- **Pagination math**: `page = offset/size + 1`, `total_pages = ceil(total/size)`. Prev disabled at `offset == 0`; Next disabled when `offset + size >= total`. Pagination row omitted entirely when `total <= size`. Each pagination button's action URL carries the full target state (current filter + new `offset`).
- **Table columns**: Date · Type · Entries · Sources · Notes — plus a per-row **Actions** cell with edit and delete icon buttons. The frontend auto-adds a chevron column to the left of the header whenever any row in the table is `expandable: true`.
- **Cell rendering**:
  - `Date` — `YYYY-MM-DD HH:mm` from `recorded_at` (datetime, not date-only — snapshots can be taken multiple times per day).
  - `Type` — badge: `Full` (color `positive`) / `Partial` (color `neutral`). Text uppercase.
  - `Entries` — integer count of `len(snapshot.entries)`.
  - `Sources` — compact list of unique `source` values across the snapshot's entries (e.g. `MANUAL`, `MANUAL · COINGECKO`). Capped at 3 visible, with `+N` suffix if more. Uppercase.
  - `Notes` — plain text truncated to 40 chars with an ellipsis when longer. Full notes visible in the edit wizard.
- No cell carries `sensitive: true`. Snapshots are not masked by the HideValues toggle (consistent with trades).

### Expanded row (entries detail)

Every table row is `expandable: true` and carries a `details` subtree containing a nested `table` of the snapshot's entries:

| Column | Rendering |
|---|---|
| Asset | Ticker resolved via the assets catalog. If the asset no longer exists, render the raw UUID as a fallback. |
| Quantity | `FormatQuantity(quantity, lang)`; `—` when `quantity` is `null` (complex assets). |
| Price | `FormatMoney(current_price, asset.currency, lang)`; `—` when `null`. |
| Value Override | `FormatMoney(current_value_override, asset.currency, lang)`; `—` when `null`. |
| Source | Badge: `MANUAL` / `COINGECKO` / `TWELVE_DATA` / `ALPHA_VANTAGE`. |

All entries are pre-rendered in the tree (hidden by default). Expanding is 100% client-side; multiple rows may be expanded simultaneously. Expansion state resets on any `replace` that rebuilds the list subtree.

### Empty states

- **No snapshots at all** (no filter active and `total == 0`): `snapshots.empty_title` / `snapshots.empty_subtitle`. No CTA inside the empty state — the user still has the header buttons.
- **No snapshots match the filter** (filter active and `total == 0`): `snapshots.empty_filtered_title` / `snapshots.empty_filtered_subtitle`. The filter control stays visible so the user can clear it.
- Pagination omitted in both cases.

### Create (manual wizard)

Triggered by the header `New Snapshot` button.

1. `GET /actions/snapshots/create_wizard` — the middleend loads the assets catalog and emits a `wizard` with `mode: "create"`:
   - **Step `info`** — inputs: `recorded_at` (datetime-local, required, `max` = now) and `notes` (textarea, optional, max 500 chars).
   - **Steps `entry`** — one per asset in the catalog. Step header shows ticker + name + asset_type badge. Per-step content:
     - If `is_complex = true`: single input `current_value_override` (text, required-when-included, `> 0`). No toggle.
     - Otherwise: a `radio_group` **Price / Value Override** (`default_value: "price"`) plus two conditional inputs — `current_price` (visible when mode is `"price"`) and `current_value_override` (visible when mode is `"override"`). Required-when-included, `> 0`.
     - `skippable: true`, `include_default: false`.
   - **Step `summary`** — descriptive text; submit action targets `POST /actions/snapshots/create?is_full_snapshot=<f>&offset=<n>`.
2. The create `POST` handler parses hidden form inputs per the wizard naming convention (`entries[<asset_id>].mode`, `entries[<asset_id>].current_price`, `entries[<asset_id>].current_value_override`), plus `recorded_at` and `notes`. Only included steps contribute entries to the BE body. `notes` is omitted when empty.
3. Backend validation errors (`422`) surface as inline `error` banners on the wizard; the wizard is re-emitted with inputs preserved; the list is not touched.
4. Success → replace screen root + `snapshots.create.success` snackbar.

### Auto-snapshot flow

Triggered by the header `Auto Snapshot` button. No pre-confirmation — the action fires directly.

1. The middleend calls `POST /v1/snapshots/auto` with body `{}`.
2. **Success (`201`)** — the middleend builds an `ActionResponse` (`action: "replace"`) that replaces the screen root with a fresh list AND injects the **edit wizard** pre-populated on the newly-created snapshot into `snapshots-modal-slot`:
   - The wizard carries `banner: { variant: "info", message: <combined_message>, dismissible: true }`.
   - `<combined_message>` is `snapshots.auto.banner`. If `warnings` is non-empty, the failed tickers are appended to the same message: `\n\n<snapshots.auto.warnings_title>: <ticker list>` (no separate warning banner).
   - `feedback`: `snapshots.auto.success` snackbar.
3. **Terminal failures** (no snapshot created):
   - `NO_PRICE_PROVIDERS_CONFIGURED` (422) → `ActionResponse{action: "none", feedback: snackbar warning}` using `snapshots.auto.no_providers`.
   - `ALL_PROVIDERS_FAILED` (502) → `ActionResponse{action: "none", feedback: snackbar error}` using `snapshots.auto.all_failed`.
   - `PROVIDER_NOT_CONFIGURED` (500) → surfaced as `502 BACKEND_ERROR`.

Cancelling the post-auto wizard does not delete the snapshot — the banner copy makes this explicit.

### Edit

Triggered by the pencil icon in a row, or opened automatically after an auto-snapshot.

1. `GET /actions/snapshots/edit_wizard?id=<id>` — the middleend calls `GET /v1/snapshots/:id` (returning `404` if gone), loads the assets catalog, and emits a `wizard` with `mode: "edit"`:
   - **Step `info`** — `recorded_at` rendered as static `text` (immutable per BE contract). `notes` textarea editable, pre-filled.
   - **Steps `entry`** — one per asset in the catalog:
     - **Asset already in the snapshot**: `include_default: true`, `skippable: false`, toggle initialized to the field that has a value, inputs pre-filled. An indicator reads `snapshots.wizard.already_included` (*"Already in snapshot, cannot be removed"*) — the BE does not allow removing existing entries via PATCH.
     - **Asset not in the snapshot**: `include_default: false`, `skippable: true`, inputs empty — same UX as create.
   - **Step `summary`** — submit action targets `PATCH /actions/snapshots/:id?is_full_snapshot=<f>&offset=<n>`.
2. The PATCH handler diffs the submitted form against the original snapshot (the handler re-fetches `GET /v1/snapshots/:id` to avoid stale diffs on concurrent edits):
   - `notes` included in the body only if it changed.
   - `entries` contains only: (a) new entries (asset not in the original, marked `included=true`), and (b) existing entries whose `current_price`, `current_value_override`, or mode changed.
   - Unchanged entries are omitted entirely.
3. Backend validation errors surface inline on the wizard as in create. Success → replace root + `snapshots.edit.success` snackbar.

### Delete

Triggered by the trash icon in a row.

1. `GET /actions/snapshots/delete_modal?id=<id>` — the middleend calls `GET /v1/snapshots/:id` to obtain the date for the confirmation message; emits a simple modal:
   - Title: `snapshots.delete.title`.
   - Message: `snapshots.delete.confirm` interpolating the `recorded_at` date: *"Delete snapshot from {date}? This will affect portfolio calculations."*
   - Cancel (`dismiss`) + destructive Delete buttons.
2. `DELETE /actions/snapshots/:id` calls `DELETE /v1/snapshots/:id`. No `force` flag — deletion is always unconditional.
3. Success → replace root + `snapshots.delete.success` snackbar.

### Post-mutation refresh

A successful create / auto / update / delete replaces the entire screen root with a fresh tree, re-fetching the snapshot list at the same filter + offset, collapsing the modal slot to empty, and attaching a success snackbar (`snapshots.create.success` / `snapshots.auto.success` / `snapshots.edit.success` / `snapshots.delete.success`) as `ActionResponse.feedback`.

On backend **validation error** (`422` with a `code`), the same wizard is re-emitted (user's values preserved) with the localized `message` from the backend error rendered as an `error` banner. Only the modal slot is replaced; the list is not touched. On non-validation errors the normal HTTP error response shape applies.

### Filter and page preservation across mutations

Every mutation endpoint carries `?is_full_snapshot=<f>&offset=<n>` so the handler can rebuild the list using the same context. The middleend is the sole owner of that state.

## i18n keys

Namespace `snapshots.*`, plus `common.cancel` shared across screens.

### Screen structure

`snapshots.title`, `snapshots.new`, `snapshots.auto_btn`, `snapshots.empty_title`, `snapshots.empty_subtitle`, `snapshots.empty_filtered_title`, `snapshots.empty_filtered_subtitle`.

### Filter

`snapshots.filter.type`, `snapshots.filter.type_any`, `snapshots.filter.type_full`, `snapshots.filter.type_partial`.

### Table headers

`snapshots.col.date`, `snapshots.col.type`, `snapshots.col.entries`, `snapshots.col.sources`, `snapshots.col.notes`.

### Entries (nested table)

`snapshots.entries.col.asset`, `snapshots.entries.col.quantity`, `snapshots.entries.col.price`, `snapshots.entries.col.value_override`, `snapshots.entries.col.source`.

### Badges

`snapshots.type.full`, `snapshots.type.partial`, `snapshots.source.manual`, `snapshots.source.coingecko`, `snapshots.source.twelve_data`, `snapshots.source.alpha_vantage`.

### Pagination

`snapshots.pagination.prev`, `snapshots.pagination.next`, `snapshots.pagination.page_of` (`Page {current} of {total}`).

### Wizard

`snapshots.wizard.info`, `snapshots.wizard.info_label` (step label text), `snapshots.wizard.summary`, `snapshots.wizard.summary_instructions` (instructional text inside the summary step), `snapshots.wizard.step_of` (`Step {current} of {total}`), `snapshots.wizard.back`, `snapshots.wizard.next`, `snapshots.wizard.skip`, `snapshots.wizard.include`, `snapshots.wizard.update`, `snapshots.wizard.already_included`.

### Create modal

`snapshots.create.title`, `snapshots.create.submit`, `snapshots.create.success`.

### Edit modal

`snapshots.edit.title` (`Edit snapshot from {date}`), `snapshots.edit.submit`, `snapshots.edit.success`.

### Delete modal

`snapshots.delete.title`, `snapshots.delete.confirm` (`Delete snapshot from {date}? This will affect portfolio calculations.`), `snapshots.delete.submit`, `snapshots.delete.success`.

### Auto flow

`snapshots.auto.success`, `snapshots.auto.banner`, `snapshots.auto.warnings_title`, `snapshots.auto.no_providers`, `snapshots.auto.all_failed`.

### Form labels (shared across create and edit)

`snapshots.form.recorded_at`, `snapshots.form.recorded_at_readonly`, `snapshots.form.notes`, `snapshots.form.notes_placeholder`, `snapshots.form.toggle_price`, `snapshots.form.toggle_override`, `snapshots.form.current_price`, `snapshots.form.current_value_override`.

### Shared

`common.cancel`.

Concrete strings live in `locales/en.json` and `locales/es.json`. Missing-key fallback: `en`, then the key itself.

## Error handling

| Situation | HTTP | Body |
|---|---|---|
| Missing / invalid / expired JWT (any endpoint) | 401 | `{"error":"unauthorized","redirect":"/login"}` |
| Backend 401 downstream | 401 | same |
| Backend 5xx / network / malformed | 502 | `{"error":{"code":"BACKEND_ERROR","message":"..."}}` |
| Invalid query param (`is_full_snapshot` outside `true`/`false`, `offset` non-integer or negative, missing `id` on modal endpoints) | 400 | `{"error":{"code":"BAD_REQUEST","message":"..."}}` |
| Snapshot not found (edit / delete modal GET) | 404 | `{"error":{"code":"NOT_FOUND"}}` |
| Backend validation error on a mutation (4xx with a `code`) | 200 | `ActionResponse{replace, target_id: <modal>, tree: <same wizard + inline error banner>}` |
| `NO_PRICE_PROVIDERS_CONFIGURED` / `ALL_PROVIDERS_FAILED` (auto) | 200 | `ActionResponse{action: "none", feedback: snackbar}` (no modal opened) |

Backend error codes that surface as inline wizard errors: `ASSET_NOT_FOUND`, `FUTURE_DATED_SNAPSHOT`, `DUPLICATE_SNAPSHOT_ENTRY`, `MISSING_VALUE_OVERRIDE`, `CONFLICTING_SNAPSHOT_VALUE`, `SNAPSHOT_NOT_FOUND`. Use the localized `message` from the backend body; do not re-translate codes in the middleend.

## Acceptance criteria

- [ ] `GET /screens/snapshots` without a valid JWT returns `401` with the documented redirect.
- [ ] With a valid JWT the middleend issues `GET /v1/snapshots?size=10&sort=recorded_at&order=desc[&is_full_snapshot=…][&offset=…]`, forwarding `Authorization`.
- [ ] The middleend loads the full assets catalog on every screen render and every wizard GET (for ticker resolution and per-asset steps).
- [ ] Screen renders with three logical regions: header (title + `New Snapshot` + `Auto Snapshot`), list region (filter + table + pagination), and a modal slot that starts empty.
- [ ] List region is replaceable independently of the modal slot — filter / pagination actions target the list region only.
- [ ] Mutation actions replace the screen root and collapse the modal slot back to empty on success; on backend validation error they replace only the modal slot with the wizard pre-populated and a localized inline error banner.
- [ ] Filter select exposes `Any` (empty value) / `Full` (`true`) / `Partial` (`false`). Changing re-fetches the list and resets `offset` to 0.
- [ ] Table has 5 data columns + Actions + auto-added chevron column; every row is `expandable: true` and carries a `details` subtree with the snapshot's entries.
- [ ] Cell rendering matches the spec: `recorded_at` as `YYYY-MM-DD HH:mm`, type badge (Full `positive` / Partial `neutral`), entries count, sources compacted to 3 + `+N`, notes truncated at 40 chars.
- [ ] Expanded row renders a nested table of entries: asset ticker (UUID fallback if gone), quantity via `FormatQuantity` (`—` for null), price and value override via `FormatMoney` using the asset's currency (`—` for null), source badge.
- [ ] Multiple rows can be expanded simultaneously; expand state is client-side only and resets on any list replace.
- [ ] Pagination omitted when `total <= size`; otherwise shows Prev (disabled at start), localized `Page X of Y`, Next (disabled at end); each button's URL carries the current filter and target offset.
- [ ] Empty list distinguishes "no snapshots" from "no match for filter" via two different copy pairs.
- [ ] Create wizard: step `info` (required `recorded_at`, optional `notes`) + one `entry` step per asset (toggle Price/Override for non-complex; forced Override for complex assets; `skippable: true`, `include_default: false`) + `summary`. Skipped steps contribute no entries to the submit body. `notes` omitted from the body when empty.
- [ ] Edit wizard: `recorded_at` rendered as static text; existing-entry steps have `skippable: false, include_default: true` with pre-filled values and an `already_included` indicator; new-entry steps behave like create. PATCH body contains only the diff (changed entries + `notes` if changed).
- [ ] Auto-snapshot: triggers `POST /v1/snapshots/auto`, on `201` replaces root and opens the edit wizard on the new snapshot with a single `info` banner; if `warnings` is non-empty, the failed tickers are appended to that same banner message (no separate warning banner). On `NO_PRICE_PROVIDERS_CONFIGURED` or `ALL_PROVIDERS_FAILED`, emits `ActionResponse{action:"none", feedback: snackbar}` without opening a modal. `PROVIDER_NOT_CONFIGURED` surfaces as `502`.
- [ ] Delete modal: confirmation message interpolates the snapshot date; submit unconditionally deletes (no `force` flag).
- [ ] Filter and offset persist across successful mutations (the fresh list uses the same values active at mutation time).
- [ ] Success feedback uses the four success keys (`create.success`, `auto.success`, `edit.success`, `delete.success`).
- [ ] All user-facing strings resolve via i18n `en` / `es`. Backend validation messages surface localized per `Accept-Language`; otherwise fall back to the backend `code`.
- [ ] 404 on edit / delete modal GET when the snapshot is gone; 400 on invalid query params; 401 with redirect on auth issues; 502 on BE 5xx or `PROVIDER_NOT_CONFIGURED`.
