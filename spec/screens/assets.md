# Assets Screen

Management screen for the **asset catalog** — financial instruments the user has registered (tickers, stocks, ETFs, crypto, bonds, and "complex" real-world assets). Assets are reference data: every trade and snapshot in the system refers back to an asset. Creating or correcting an asset here is what makes it available elsewhere.

## Purpose

Let the user:

1. **Browse** the catalog with pagination and a type filter.
2. **Register** a new asset (ticker, name, type, currency, and optionally a price provider for auto-fetched prices).
3. **Correct** mutable fields on an existing asset (name, price provider, external ticker).
4. **Remove** an asset, with a safety flag for cascading through dependent trades and snapshots when needed.

## Endpoints

All endpoints are **protected** (JWT). Missing / invalid / expired JWT → `401 {"error":"unauthorized","redirect":"/login"}`. Backend 5xx / network / malformed → `502 BACKEND_ERROR`. Invalid query or body → `400 BAD_REQUEST`.

| Method | Path | Purpose |
|---|---|---|
| `GET`    | `/screens/assets` | Full screen render. Query: optional `asset_type` (enum), `offset` (non-negative integer). Reading the URL params means the screen can be deep-linked to a filtered / paginated state. |
| `GET`    | `/actions/assets/list` | Re-renders the filter+table+pagination subtree. Used by the filter select (on change) and by the Prev/Next buttons. Query: `asset_type`, `offset`. |
| `GET`    | `/actions/assets/create_modal` | Returns the Create Asset modal with an empty form. Query: current list context (`asset_type`, `offset`) so the submit URL can preserve it. |
| `GET`    | `/actions/assets/edit_modal?id=<id>` | Fetches the asset and returns the Edit modal with mutable fields pre-populated. |
| `GET`    | `/actions/assets/delete_modal?id=<id>` | Returns the Delete confirmation modal. |
| `POST`   | `/actions/assets/create?asset_type=<f>&offset=<n>` | Creates an asset. On success returns a full-list refresh; on validation error returns the same create modal with an inline error. |
| `PATCH`  | `/actions/assets/:id?asset_type=<f>&offset=<n>` | Updates mutable fields. Same response shape as create. |
| `DELETE` | `/actions/assets/:id?asset_type=<f>&offset=<n>` | Deletes. Reads `force` (bool) from the request body. Same response shape as create. |

Every non-screen endpoint returns an `ActionResponse`. Mutation endpoints that succeed emit a `replace` of the entire screen root plus a success snackbar; mutation endpoints that fail with a backend validation error emit a `replace` of the modal subtree only, re-populated with the user's input and an inline error message. The `asset_type` and `offset` query params on mutation endpoints carry the current list context so the refreshed list preserves filter and page after a mutation.

## Backend dependencies

- `GET /v1/assets` — paginated list. The middleend always sends `size=10`, `sort=ticker`, `order=desc` plus optional `asset_type` and `offset`. Response shape: `{ assets, total, size, offset }`.
- `GET /v1/assets/:id` — single asset. Used by the edit and delete modal endpoints (to fetch fresh data and to have the ticker for the delete confirmation message). `404 NOT_FOUND` when the asset doesn't exist.
- `POST /v1/assets` — creates. Body has `ticker`, `name`, `asset_type`, `currency`, `is_complex`, optional `price_provider`, optional `external_ticker`. Validation errors come back as `422` with a `code` (e.g. `ASSET_ALREADY_EXISTS`, `INVALID_TICKER`, `INVALID_PRICE_PROVIDER`, `COMPLEX_ASSET_PRICE_PROVIDER`).
- `PATCH /v1/assets/:id` — updates. Only `name`, `price_provider`, `external_ticker` are mutable; the backend rejects any immutable field.
- `DELETE /v1/assets/:id?force=<bool>` — deletes. Without `force`, returns `422 ASSET_HAS_DATA` when the asset has trades or snapshot entries. With `force=true`, cascades through all dependent rows in a single transaction.

## Layout

Three logical regions stacked vertically:

1. **Screen header** — title and room for future controls (toggles, bulk actions).
2. **List region** — the main interactive area that holds the filter + table + pagination. This is the subtree the list / filter / pagination actions replace.
3. **Modal slot** — initially empty; each of the three modal actions inserts its own modal tree here. The slot is a **sibling** of the list region so filter changes and pagination do not wipe an open modal, and mutation success replaces the root entirely (collapsing the modal back to empty).

## Data and business rules

### List

- **Fixed server-side sort**: `ticker DESC`. Not user-configurable in this screen.
- **Page size**: 10 (hardcoded here; a future extension could configure it).
- **Filter**: a single `asset_type` select with options `Any` / `STOCK` / `ETF` / `CRYPTO` / `BOND`. Selecting `Any` sends an empty value, which the handler treats as "no filter" and does not forward to the backend. Changing the filter resets `offset` to 0.
- **Pagination math**: `page = offset/size + 1`, `total_pages = ceil(total/size)`. Prev disabled at `offset == 0`; Next disabled when `offset + size >= total`. Pagination row is omitted entirely when `total <= size`. Each pagination button's action URL carries the full target state (current `asset_type` + new `offset`) so the frontend is stateless.
- **Table columns** (6): Ticker · Name · Type · Currency · Complex · Price Provider — plus a per-row **Actions** cell with edit and delete icon buttons.
- **Cell rendering**: Ticker upper-cased, bold. Currency upper-cased. Complex renders as `✓` when `is_complex = true`, `—` otherwise. Price Provider renders the value when present; `—` when null **or** when `is_complex = true` (complex assets can't have a provider).

### Empty states

- **No assets at all**: localized title + subtitle (`assets.empty_title` / `assets.empty_subtitle`). No CTA inside the empty state — the user still has the New Asset button in the header.
- **No assets match the filter**: same structure, different copy (`assets.empty_filtered_title` / `assets.empty_filtered_subtitle`). The filter control stays visible so the user can clear it. In both cases the pagination row is not emitted.

### Create

Modal opened by the header "New Asset" button. Fields:

| Field | Input | Rules |
|---|---|---|
| `ticker` | text | Required, max 20 chars, pattern `^[A-Z0-9.\-]+$`, auto-uppercase as the user types. |
| `name` | text | Required, max 100 chars. |
| `asset_type` | select | Required. One of `STOCK`, `ETF`, `CRYPTO`, `BOND`. |
| `currency` | select | Required. Hardcoded set today: `USD`, `EUR`, `ARS`, `MXN`, `GBP`. |
| `is_complex` | checkbox | Defaults to false. |
| `price_provider` | select | Optional. Options: `(none)`, `COINGECKO`, `TWELVE_DATA`, `ALPHA_VANTAGE`. **Hidden when `is_complex = true`** via `visible_when`. |
| `external_ticker` | text | Optional, max 100. **Hidden when `price_provider = ""`** via `visible_when`. |

Form validation runs in the frontend using the `required`, `max_length`, `pattern`, and `auto_uppercase` input props plus the `visible_when` prop on the two conditional fields. Hidden fields are not submitted.

Immutable-after-creation fields: `ticker`, `asset_type`, `currency`, `is_complex`. Per the backend contract, to change any of these the user would delete and re-create the asset — we do not expose that as a workflow in this screen.

### Edit

Modal opened by the pencil icon in a row. The handler fetches the asset first (returning `404` if gone) so the modal shows fresh data.

- Immutable fields (`ticker`, `asset_type`, `currency`, `is_complex`) are rendered as **static text** (labeled lines), not as disabled inputs. They cannot be changed here.
- Mutable fields: `name` (required), `price_provider` (optional select), `external_ticker` (optional text, shown only when `price_provider != ""`).
- When the asset is complex, `price_provider` and `external_ticker` are omitted entirely from the form — they are never valid for complex assets.
- The modal title interpolates the ticker (e.g. `Edit AAPL`).

### Delete

Modal opened by the trash icon in a row. Always a single confirmation modal with:

- A confirmation message that interpolates the ticker.
- A **force** checkbox (`Also delete associated trades and snapshots`). Default unchecked.
- Cancel (`dismiss`) + destructive Delete buttons.

The handler reads `force` from the submitted body and passes `force=true|false` through to the backend. If the user leaves it unchecked and the asset has data, the backend returns `ASSET_HAS_DATA` (422); we surface that as an inline error on the same modal — the user can then tick Force and retry.

### Post-mutation refresh

A successful create / update / delete replaces the entire screen root with a fresh tree:
- Re-fetches the list with the filter + offset that were active before the mutation (passed through via the mutation endpoint's query params).
- Collapses the modal slot back to empty.
- Attaches a success snackbar as the `feedback` of the `ActionResponse`.

Success copy keys:

| Operation | Key |
|---|---|
| Create | `assets.create.success` |
| Update | `assets.edit.success` |
| Delete (no force) | `assets.delete.success` |
| Delete (force) | `assets.delete.success_force` |

On backend **validation error**, the same modal is re-emitted (with the user's values still filled in where possible) and the localized `message` from the backend error is rendered as an inline error at the top of the form. Only `assets-modal-slot` is replaced; the list is not touched. On non-validation errors (5xx, unauthorized, etc.) the normal HTTP error response shape applies.

### Filter and page preservation across mutations

Every mutation endpoint carries `?asset_type=<f>&offset=<n>` so the handler can rebuild the list using that same context. This is what keeps the user on the same filter and page after creating, editing, or deleting an asset. The middleend is the sole owner of that state — the frontend has nothing to persist.

## i18n keys

Namespace `assets.*`, plus `common.cancel` shared across screens.

### Screen structure

`assets.title`, `assets.new`, `assets.empty_title`, `assets.empty_subtitle`, `assets.empty_filtered_title`, `assets.empty_filtered_subtitle`.

### Filter

`assets.filter.type`, `assets.filter.type_any`.

### Table headers

`assets.col.ticker`, `assets.col.name`, `assets.col.type`, `assets.col.currency`, `assets.col.complex`, `assets.col.price_provider`.

### Pagination

`assets.pagination.prev`, `assets.pagination.next`, `assets.pagination.page_of` (`Page {current} of {total}`).

### Create modal

`assets.create.title`, `assets.create.submit`, `assets.create.success`.

### Edit modal

`assets.edit.title` (`Edit {ticker}`), `assets.edit.submit`, `assets.edit.success`.

### Delete modal

`assets.delete.title`, `assets.delete.confirm` (`Delete {ticker}? This cannot be undone.`), `assets.delete.force_label`, `assets.delete.submit`, `assets.delete.success`, `assets.delete.success_force`.

### Form labels (shared across create and edit)

`assets.form.is_complex`, `assets.form.is_complex_description`, `assets.form.external_ticker`, `assets.form.external_ticker_placeholder`.

### Shared

`common.cancel`.

Concrete strings live in `locales/en.json` and `locales/es.json`. Missing-key fallback: `en`, then the key itself.

## Error handling

| Situation | HTTP | Body |
|---|---|---|
| Missing / invalid / expired JWT (any endpoint) | 401 | `{"error":"unauthorized","redirect":"/login"}` |
| Backend 401 downstream | 401 | same |
| Backend 5xx / network / malformed | 502 | `{"error":{"code":"BACKEND_ERROR","message":"..."}}` |
| Invalid query param (`asset_type` outside the enum, `offset` non-integer or negative, missing `id` on modal endpoints) | 400 | `{"error":{"code":"BAD_REQUEST","message":"..."}}` |
| Asset not found (edit / delete modal GET) | 404 | `{"error":{"code":"NOT_FOUND"}}` |
| Backend validation error on a mutation (4xx with a `code`) | 200 | `ActionResponse{replace, target_id: <modal>, tree: <same modal + inline error>}` |

Backend error codes that need to surface as inline form errors include (non-exhaustive): `INVALID_TICKER`, `INVALID_ASSET_TYPE`, `INVALID_CURRENCY`, `INVALID_PRICE_PROVIDER`, `ASSET_ALREADY_EXISTS`, `COMPLEX_ASSET_PRICE_PROVIDER`, `ASSET_HAS_DATA`. Use the localized message from the backend body; do not re-translate codes in the middleend.

## Acceptance criteria

- [ ] `GET /screens/assets` without a valid JWT returns `401` with the documented redirect.
- [ ] With a valid JWT the middleend issues `GET /v1/assets?size=10&sort=ticker&order=desc[&asset_type=…][&offset=…]`, forwarding `Authorization`.
- [ ] Screen renders with three logical regions: header (with title and a New Asset button), list region (filter + table + pagination), and a modal slot that starts empty.
- [ ] List region is replaceable independently of the modal slot — filter / pagination actions target the list region only.
- [ ] Mutation actions replace the screen root and collapse the modal slot back to empty on success; on backend validation error they replace only the modal slot with the form pre-populated and a localized inline error.
- [ ] Filter select exposes `Any` (empty value) plus four type options; changing it re-fetches the list and resets `offset` to 0.
- [ ] Table has 6 data columns in documented order plus a per-row Actions cell with edit and delete buttons.
- [ ] Ticker upper-cased, bold. Currency upper-cased. Complex `✓` / `—`. Price Provider as-is, `—` when null or when `is_complex`.
- [ ] Pagination omitted when `total <= size`; otherwise shows Prev (disabled at start), localized `Page X of Y`, Next (disabled at end); each button's URL carries the current filter and the target offset.
- [ ] Empty list distinguishes "no assets" from "no match for filter" via two different copy pairs.
- [ ] Create modal: full form with the seven fields above; `price_provider` hidden when `is_complex`, `external_ticker` hidden when `price_provider` empty. Ticker field has `pattern` and `auto_uppercase`. Cancel dismisses client-side.
- [ ] Edit modal: immutable fields rendered as text; for complex assets, `price_provider` and `external_ticker` are omitted entirely; title interpolates the ticker.
- [ ] Delete modal: confirmation message interpolates the ticker; force checkbox defaults to unchecked; submit passes `force=true|false` through to the backend DELETE.
- [ ] Filter and offset persist across successful mutations (the fresh list uses the same values that were active at mutation time).
- [ ] Success feedback uses the four success keys (including `success_force` for force-deletes).
- [ ] All user-facing strings resolve via i18n `en` / `es`. Backend error messages for validation errors surface localized per `Accept-Language` if the backend provides them; otherwise fall back to the backend `code`.
- [ ] 404 on edit / delete modal GET when the asset is gone; 400 on invalid query (`asset_type`, `offset`, missing `id`); 401 with redirect on auth issues; 502 on BE 5xx.
