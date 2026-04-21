# Trades Screen

Management screen for **trade events** — the BUY/SELL transactions the user records against an asset. Trades are the primary input for AVCO cost basis and P&L, so correcting or deleting a trade affects the portfolio view.

## Purpose

Let the user:

1. **Browse** all trades with pagination and two filters (asset, trade type).
2. **Register** a new trade (asset, type, quantity, price per unit, fees, date, notes).
3. **Correct** mutable financial fields on an existing trade.
4. **Remove** a trade, acknowledging that AVCO will be recalculated.

## Endpoints

All endpoints are **protected** (JWT). Missing / invalid / expired JWT → `401 {"error":"unauthorized","redirect":"/login"}`. Backend 5xx / network / malformed → `502 BACKEND_ERROR`. Invalid query or body → `400 BAD_REQUEST`.

| Method | Path | Purpose |
|---|---|---|
| `GET`    | `/screens/trades` | Full screen render. Query: optional `asset_id` (UUID), `trade_type` (`BUY` / `SELL`), `offset` (non-negative integer). Deep-linkable filtered / paginated state. |
| `GET`    | `/actions/trades/list` | Re-renders the filter+table+pagination subtree. Used by both filter controls (on change) and the Prev/Next buttons. Query: `asset_id`, `trade_type`, `offset`. |
| `GET`    | `/actions/trades/create_modal` | Returns the Create Trade modal with an empty form. Query: current list context (`asset_id`, `trade_type`, `offset`) so the submit URL can preserve it. |
| `GET`    | `/actions/trades/edit_modal?id=<id>` | Fetches the trade and returns the Edit modal with fields pre-populated. |
| `GET`    | `/actions/trades/delete_modal?id=<id>` | Returns the Delete confirmation modal. |
| `POST`   | `/actions/trades/create?asset_id=<f>&trade_type=<f>&offset=<n>` | Creates a trade. On success returns a full-list refresh; on validation error returns the same create modal with an inline error. |
| `PATCH`  | `/actions/trades/:id?asset_id=<f>&trade_type=<f>&offset=<n>` | Updates mutable fields. Same response shape as create. |
| `DELETE` | `/actions/trades/:id?asset_id=<f>&trade_type=<f>&offset=<n>` | Deletes. Same response shape as create. No `force` flag — deletion is always unconditional. |

Every non-screen endpoint returns an `ActionResponse`. Mutation endpoints that succeed emit a `replace` of the entire screen root plus a success snackbar; mutation endpoints that fail with a backend validation error emit a `replace` of the modal subtree only, re-populated with the user's input and an inline error message. The `asset_id`, `trade_type`, and `offset` query params on mutation endpoints carry the current list context so the refreshed list preserves filter and page after a mutation.

## Backend dependencies

- `GET /v1/trades` — paginated list. The middleend always sends `size=10`, `sort=date`, `order=desc` plus optional `asset_id`, `trade_type`, and `offset`. Response shape: `{ trades, total, size, offset }`. The backend default order is `asc`; the middleend always sends `desc` explicitly.
- `GET /v1/trades/:id` — single trade. Used by the edit and delete modal endpoints (to fetch fresh data and to build the delete confirmation message). `404 NOT_FOUND` when the trade doesn't exist.
- `POST /v1/trades` — creates. Body has `asset_id`, `trade_type`, `quantity`, `price_per_unit`, `fees`, `date`, optional `notes`. `source` is set to `"MANUAL"` by the middleend (not user-chosen). Validation errors come back as `422` with a `code` (e.g. `INSUFFICIENT_QUANTITY`, `COMPLEX_ASSET_TRADE`, `FUTURE_DATED_TRADE`, `INVALID_TRADE_TYPE`, `INVALID_QUANTITY`, `INVALID_PRICE`, `INVALID_FEE`).
- `PATCH /v1/trades/:id` — updates. Mutable fields: `asset_id`, `trade_type`, `quantity`, `price_per_unit`, `fees`, `notes`. `date` and `source` are immutable; the middleend rejects them client-side (they are rendered as static text in the Edit modal, not as inputs).
- `DELETE /v1/trades/:id` — deletes. No conditional flag.
- **Assets catalog** (see [`../shared/assets-catalog.md`](../shared/assets-catalog.md)) — loaded on every screen render and every modal GET, for the asset filter dropdown and the create/edit form `asset_id` select.

## Layout

Three logical regions stacked vertically:

1. **Screen header** — title (`Trades`) and a `New Trade` button.
2. **List region** — the main interactive area that holds the two filters (asset select + trade-type toggle) on one row, then the table, then pagination. This is the subtree the list / filter / pagination actions replace.
3. **Modal slot** — initially empty; each modal action inserts its own modal tree here. The slot is a **sibling** of the list region so filter changes and pagination do not wipe an open modal, and mutation success replaces the root entirely (collapsing the modal back to empty).

## Data and business rules

### List

- **Fixed server-side sort**: `date DESC`. Ties broken by `created_at ASC` (backend behavior). Not user-configurable.
- **Page size**: 10 (hardcoded here; mirrors assets).
- **Filters**:
  - `asset_id` — single-select of the assets catalog (`Any` + one option per asset, labeled by ticker). Selecting `Any` sends an empty value; the handler treats it as "no filter" and does not forward to the backend. Complex assets appear in this filter (they can't have trades, but historical data may still reference them if any slipped through before the constraint was added).
  - `trade_type` — single-select dropdown with three options: `All` / `BUY` / `SELL`. `All` sends an empty value; the handler treats it as "no filter". Rendered as a `Select` to stay consistent with the Assets screen filter idiom and the available SDUI component set.
- Changing either filter resets `offset` to 0.
- **Pagination math**: `page = offset/size + 1`, `total_pages = ceil(total/size)`. Prev disabled at `offset == 0`; Next disabled when `offset + size >= total`. Pagination row is omitted entirely when `total <= size`. Each pagination button's action URL carries the full target state (current filters + new `offset`).
- **Table columns** (9): Date · Asset · Type · Quantity · Price/Unit · Total · Fees · Source · Notes — plus a per-row **Actions** cell with edit and delete icon buttons.
- **Cell rendering**:
  - `Date` — `YYYY-MM-DD` from the trade's `date` (the BE returns a full timestamp; we render the date portion only).
  - `Asset` — ticker resolved from `asset_id` via the asset catalog. If the asset is no longer in the catalog (e.g. deleted), render the raw UUID as a fallback (edge case; shouldn't normally happen).
  - `Type` — badge. `BUY` green, `SELL` red. Text is uppercase.
  - `Quantity` — `FormatQuantity` (locale-aware, strips trailing zeros, max 8 decimals).
  - `Price/Unit` — `FormatMoney(price_per_unit, asset.currency, lang)`.
  - `Total` — computed as `quantity × price_per_unit` and rendered via `FormatMoney(total, asset.currency, lang)`. Not sent by the backend.
  - `Fees` — `FormatMoney(fees, asset.currency, lang)`. Rendered as `—` when `fees == "0"`.
  - `Source` — badge. `MANUAL` or `IMPORT`, uppercase.
  - `Notes` — plain text truncated to 40 chars with an ellipsis when longer. Full notes are visible in the Edit modal.
- **No `sensitive` attribute** on any cell. Trades are not hidden by the portfolio HideValues toggle.

### Empty states

- **No trades at all** (no filter active and `total == 0`): localized title + subtitle (`trades.empty_title` / `trades.empty_subtitle`). No CTA inside the empty state — the user still has the New Trade button in the header.
- **No trades match the filter** (any filter active and `total == 0`): same structure, different copy (`trades.empty_filtered_title` / `trades.empty_filtered_subtitle`). Both filter controls stay visible so the user can clear them.
- In both cases the pagination row is not emitted.

### Create

Modal opened by the header `New Trade` button. Fields:

| Field | Input | Rules |
|---|---|---|
| `asset_id` | select | Required. Options come from the assets catalog **filtered to `is_complex = false`** (complex assets can't have trades). Labeled by ticker; disabled (greyed) if the catalog is empty (with a hint copy pointing at the Assets screen). |
| `trade_type` | select | Required. `BUY` / `SELL`. |
| `quantity` | text | Required, `> 0`, up to 8 decimals. Sent as string. |
| `price_per_unit` | text | Required, `> 0`. Sent as string. |
| `fees` | text | Optional, `≥ 0`. Defaults to `"0"` server-side if omitted. Sent as string. |
| `date` | date | Required. Must not be in the future (frontend enforces via `max` attribute = today; backend re-validates). `YYYY-MM-DD`. |
| `notes` | textarea | Optional. Max 500 chars. |

Frontend validation runs via the `required`, `min`, `max`, `max_length` input props. `asset_id`, `trade_type`, `quantity`, `price_per_unit`, and `date` are required; the form cannot submit without them.

The middleend sets `source = "MANUAL"` on the outgoing backend body; it is not part of the form. `notes` is omitted from the body when empty.

### Edit

Modal opened by the pencil icon in a row. The handler fetches the trade first (returning `404` if gone) so the modal shows fresh data.

- **Immutable fields (rendered as static text, not inputs):** `date`, `source`. Per the backend contract.
- **Mutable fields:**
  - `asset_id` — select, same options as the create modal (excludes complex assets). Default value is the trade's current `asset_id`. Changing it moves the trade to a different asset; the backend replays both histories for validation.
  - `trade_type` — select, `BUY` / `SELL`.
  - `quantity` — text, same rules as create.
  - `price_per_unit` — text, same rules as create.
  - `fees` — text, same rules as create.
  - `notes` — textarea, max 500 chars.
- The modal title interpolates the date and asset ticker (e.g. `Edit 2024-03-15 · AAPL`).
- On save: `PATCH /v1/trades/:id` with **only the changed fields** (the middleend diffs the submitted form against the original trade fetched at modal open time).

### Delete

Modal opened by the trash icon in a row. Single confirmation modal with:

- A confirmation message interpolating the trade's date and asset ticker: e.g. `Delete this BUY of 10 AAPL on 2024-03-15? This will affect AVCO calculations.`
- Cancel (`dismiss`) + destructive Delete buttons.
- No force flag — trades delete unconditionally.

### Post-mutation refresh

A successful create / update / delete replaces the entire screen root with a fresh tree:
- Re-fetches the trade list with the filters + offset that were active before the mutation (passed through via the mutation endpoint's query params).
- Collapses the modal slot back to empty.
- Attaches a success snackbar as the `feedback` of the `ActionResponse`.

Success copy keys:

| Operation | Key |
|---|---|
| Create | `trades.create.success` |
| Update | `trades.edit.success` |
| Delete | `trades.delete.success` |

On backend **validation error** (`422` with a `code`), the same modal is re-emitted (with the user's values still filled in) and the localized `message` from the backend error is rendered as an inline error at the top of the form. Only `trades-modal-slot` is replaced; the list is not touched. On non-validation errors (5xx, unauthorized, etc.) the normal HTTP error response shape applies.

### Filter and page preservation across mutations

Every mutation endpoint carries `?asset_id=<f>&trade_type=<f>&offset=<n>` so the handler can rebuild the list using that same context. The middleend is the sole owner of that state — the frontend has nothing to persist.

## i18n keys

Namespace `trades.*`, plus `common.cancel` shared across screens.

### Screen structure

`trades.title`, `trades.new`, `trades.empty_title`, `trades.empty_subtitle`, `trades.empty_filtered_title`, `trades.empty_filtered_subtitle`.

### Filters

`trades.filter.asset`, `trades.filter.asset_any`, `trades.filter.type`, `trades.filter.type_all`, `trades.filter.type_buy`, `trades.filter.type_sell`.

### Table headers

`trades.col.date`, `trades.col.asset`, `trades.col.type`, `trades.col.quantity`, `trades.col.price`, `trades.col.total`, `trades.col.fees`, `trades.col.source`, `trades.col.notes`.

### Badges

`trades.type.buy`, `trades.type.sell`, `trades.source.manual`, `trades.source.import`.

### Pagination

`trades.pagination.prev`, `trades.pagination.next`, `trades.pagination.page_of` (`Page {current} of {total}`).

### Create modal

`trades.create.title`, `trades.create.submit`, `trades.create.success`.

### Edit modal

`trades.edit.title` (`Edit {date} · {ticker}`), `trades.edit.submit`, `trades.edit.success`.

### Delete modal

`trades.delete.title`, `trades.delete.confirm` (`Delete this {type} of {quantity} {ticker} on {date}? This will affect AVCO calculations.`), `trades.delete.submit`, `trades.delete.success`.

### Form labels (shared across create and edit)

`trades.form.asset`, `trades.form.trade_type`, `trades.form.quantity`, `trades.form.price_per_unit`, `trades.form.fees`, `trades.form.date`, `trades.form.notes`, `trades.form.notes_placeholder`, `trades.form.no_assets_hint` (shown when the catalog is empty: `Register an asset first to record trades.`).

### Immutable-in-edit labels

`trades.form.date_readonly`, `trades.form.source_readonly`.

### Shared

`common.cancel`.

Concrete strings live in `locales/en.json` and `locales/es.json`. Missing-key fallback: `en`, then the key itself.

## Error handling

| Situation | HTTP | Body |
|---|---|---|
| Missing / invalid / expired JWT (any endpoint) | 401 | `{"error":"unauthorized","redirect":"/login"}` |
| Backend 401 downstream | 401 | same |
| Backend 5xx / network / malformed | 502 | `{"error":{"code":"BACKEND_ERROR","message":"..."}}` |
| Invalid query param (`asset_id` not a UUID, `trade_type` outside `BUY`/`SELL`, `offset` non-integer or negative, missing `id` on modal endpoints) | 400 | `{"error":{"code":"BAD_REQUEST","message":"..."}}` |
| Trade not found (edit / delete modal GET) | 404 | `{"error":{"code":"NOT_FOUND"}}` |
| Backend validation error on a mutation (4xx with a `code`) | 200 | `ActionResponse{replace, target_id: <modal>, tree: <same modal + inline error>}` |

Backend error codes that need to surface as inline form errors include (non-exhaustive): `INSUFFICIENT_QUANTITY`, `COMPLEX_ASSET_TRADE`, `FUTURE_DATED_TRADE`, `INVALID_TRADE_TYPE`, `INVALID_QUANTITY`, `INVALID_PRICE`, `INVALID_FEE`, `ASSET_NOT_FOUND`. Use the localized message from the backend body; do not re-translate codes in the middleend.

## Acceptance criteria

- [ ] `GET /screens/trades` without a valid JWT returns `401` with the documented redirect.
- [ ] With a valid JWT the middleend issues `GET /v1/trades?size=10&sort=date&order=desc[&asset_id=…][&trade_type=…][&offset=…]`, forwarding `Authorization`.
- [ ] The middleend loads the full assets catalog on every screen render and every modal GET (for selector options and asset lookup).
- [ ] Screen renders with three logical regions: header (with title and a New Trade button), list region (two filters on one row + table + pagination), and a modal slot that starts empty.
- [ ] List region is replaceable independently of the modal slot — filter / pagination actions target the list region only.
- [ ] Mutation actions replace the screen root and collapse the modal slot back to empty on success; on backend validation error they replace only the modal slot with the form pre-populated and a localized inline error.
- [ ] Asset filter exposes `Any` (empty value) plus one option per asset in the catalog; type filter is a three-option select: `All` (empty value) / `BUY` / `SELL`. Changing either re-fetches the list and resets `offset` to 0.
- [ ] Table has 9 data columns in documented order plus a per-row Actions cell with edit and delete buttons.
- [ ] Cell rendering matches the spec: date as `YYYY-MM-DD`, asset as ticker (UUID fallback), type badge (BUY green / SELL red), quantity via `FormatQuantity`, price/total/fees via `FormatMoney` using the asset's `currency`, fees `—` when `"0"`, source badge (MANUAL / IMPORT), notes truncated at 40 chars.
- [ ] Total column value equals `quantity × price_per_unit`, formatted with the asset's currency.
- [ ] Pagination omitted when `total <= size`; otherwise shows Prev (disabled at start), localized `Page X of Y`, Next (disabled at end); each button's URL carries the current filters and the target offset.
- [ ] Empty list distinguishes "no trades" from "no match for filters" via two different copy pairs.
- [ ] Create modal: full form with the seven fields above; `asset_id` options exclude complex assets; `fees` defaults to `"0"` if left blank; `date` cannot be in the future; middleend adds `source = "MANUAL"` to the outgoing body.
- [ ] Edit modal: `date` and `source` rendered as static text; all other financial fields (`asset_id`, `trade_type`, `quantity`, `price_per_unit`, `fees`, `notes`) editable; title interpolates date and ticker; PATCH body contains only fields whose value differs from the fetched original.
- [ ] Delete modal: confirmation message interpolates type, quantity, ticker, and date; submit unconditionally deletes (no `force` flag).
- [ ] Filters and offset persist across successful mutations (the fresh list uses the same values that were active at mutation time).
- [ ] Success feedback uses the three success keys.
- [ ] All user-facing strings resolve via i18n `en` / `es`. Backend error messages for validation errors surface localized per `Accept-Language` if the backend provides them; otherwise fall back to the backend `code`.
- [ ] 404 on edit / delete modal GET when the trade is gone; 400 on invalid query (`asset_id`, `trade_type`, `offset`, missing `id`); 401 with redirect on auth issues; 502 on BE 5xx.
