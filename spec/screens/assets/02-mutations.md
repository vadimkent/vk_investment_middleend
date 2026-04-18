# Assets — Layer 2: Mutations

Second and final layer of the assets screen. Adds create, edit, and delete operations via modal dialogs. Read-only listing (Layer 1) ships in `01-list.md`.

Scope: one ship cycle covers all three mutations. Web-only tree (responsive comes later).

## SDUI spec additions

Layer 2 required three additions to `spec/sdui-*`. These are shared primitives, documented in `../../sdui-actions.md` and `../../sdui-base-components.md`.

### `VisibleWhen` (reactive visibility)

Optional prop `visible_when` on form components (`input`, `select`, `checkbox`, `textarea`, `radio_group`):

```go
type VisibleWhen struct {
    Field string // name of another form control in the same form
    Op    string // "eq" | "ne"
    Value any    // string, bool, or number
}
```

When the expression evaluates `true`, the component is visible; when `false`, the frontend hides it. Hidden components do not contribute to form data on submit.

### `input.pattern`

Optional ECMAScript regex validated client-side on change/blur.

### `input.auto_uppercase`

Optional bool. When `true`, the FE transforms the entered value to uppercase as the user types.

## Screen tree changes (vs L1)

The `assets` screen gains two things: a "New Asset" button in the filter row, and an `assets-modal-slot` sibling of `assets-section` inside `assets-root`.

```
screen id=assets
  column assets-root (gap=lg)
    row assets-header-row ...                                (from L1: title + 1fr spacer)
    column assets-section (gap=sm)
      row assets-filter-row widths=["240px","1fr","auto"]
        select asset-type-select ...                          (same as L1)
        spacer filter-spacer size="none"
        button assets-new-btn
          actions: [{ trigger:"click", type:"reload",
                      endpoint:"/actions/assets/create_modal",
                      target_id:"assets-modal-slot", loading:"section" }]
      table assets-table
        columns: [...6 L1 columns..., { id:"actions", header:"", width:"120px", align:"right" }]
        children: table_row asset-<id>
          ...6 L1 cells...
          row actions-<id> widths=["auto","auto"] gap="sm"
            button edit-<id>  icon:"pencil" → reload /actions/assets/edit_modal?id=<id>
            button delete-<id> icon:"trash" → reload /actions/assets/delete_modal?id=<id>
      row assets-pagination ...                                (same as L1)
    column assets-modal-slot (gap=none)
      (initially empty — modal inserted here on demand)
```

The `assets-modal-slot` sits outside `assets-section` so that filter changes and pagination (which replace `assets-section`) do not wipe an open modal.

## Endpoints

Six new endpoints, all protected (JWT).

| Method | Path | Response |
|---|---|---|
| GET | `/actions/assets/create_modal` | `ActionResponse{replace, target_id:"assets-modal-slot", tree: <modal>}` |
| GET | `/actions/assets/edit_modal?id=<id>` | Same shape, modal pre-populated with asset data |
| GET | `/actions/assets/delete_modal?id=<id>` | Same shape, confirmation modal |
| POST | `/actions/assets/create?asset_type=<f>&offset=<n>` | Success: `ActionResponse{replace, target_id:"assets-root", tree:<fresh root>, feedback:<snackbar>}`. Error: `ActionResponse{replace, target_id:"assets-modal-slot", tree:<same modal with error>}` |
| PATCH | `/actions/assets/:id?asset_type=<f>&offset=<n>` | Same shape as create |
| DELETE | `/actions/assets/:id?asset_type=<f>&offset=<n>` | Same shape as create. `force` read from JSON body. |

Mutation endpoints accept `asset_type` and `offset` query params so the handler can rebuild `assets-section` with the list context the user was viewing. Filter and page survive mutations.

Downstream calls: `POST /v1/assets`, `GET /v1/assets/:id`, `PATCH /v1/assets/:id`, `DELETE /v1/assets/:id?force=<bool>`.

## Response pattern

**Mutation success:** `ActionResponse{action:"replace", target_id:"assets-root", tree:<fresh root>, feedback:<snackbar>}`. The fresh root rebuilds `assets-section` with the filter+offset from the query params and re-initializes `assets-modal-slot` empty. This updates the list and closes the modal in one response.

**Mutation BE error:** `ActionResponse{action:"replace", target_id:"assets-modal-slot", tree:<same modal pre-populated with user input + error text at top>}`. The user corrects and re-submits.

**Feedback strings (i18n keys):**

| Operation | Key |
|---|---|
| create success | `assets.create.success` |
| edit success | `assets.edit.success` |
| delete success (no force) | `assets.delete.success` |
| delete success (force) | `assets.delete.success_force` |

## Create modal fields

- `ticker`: required, max 20, pattern `^[A-Z0-9.\-]+$`, auto_uppercase.
- `name`: required, max 100.
- `asset_type`: required select from `STOCK / ETF / CRYPTO / BOND`.
- `currency`: required select from `USD / EUR / ARS / MXN / GBP` (hardcoded in L2; config-driven later).
- `is_complex`: checkbox, default false.
- `price_provider`: select from `(none) / COINGECKO / TWELVE_DATA / ALPHA_VANTAGE`. `visible_when: is_complex == false`.
- `external_ticker`: optional, max 100. `visible_when: price_provider != ""`.

## Edit modal

- Title: `Edit {TICKER}` interpolated.
- Immutable fields (rendered as static `text`): `ticker`, `asset_type`, `currency`, `is_complex`.
- Mutable inputs: `name` (required), `price_provider`, `external_ticker`. `external_ticker` `visible_when: price_provider != ""`. If the asset is complex, the provider and external_ticker fields are omitted entirely from the tree.

## Delete modal

- Title: `Delete Asset`.
- Message: `Delete {TICKER}? This cannot be undone.` interpolated.
- `force` checkbox.
- Cancel button with `dismiss` action.
- Submit button with `DELETE /actions/assets/:id?asset_type=<f>&offset=<n>`.

## Error handling

| Situation | HTTP | Body |
|---|---|---|
| Missing / invalid / expired JWT | 401 | `{"error":"unauthorized","redirect":"/login"}` |
| Backend 401 downstream | 401 | same |
| Backend 5xx or network error | 502 | `{"error":{"code":"BACKEND_ERROR","message":"..."}}` |
| Backend 4xx validation (on mutation endpoints) | 200 | `ActionResponse{replace, assets-modal-slot, <modal + error>}` |
| Invalid query / body | 400 | `{"error":{"code":"BAD_REQUEST","message":"..."}}` |
| Asset not found (edit/delete modal GET) | 404 | `{"error":{"code":"NOT_FOUND"}}` |

## Acceptance criteria

- [x] `VisibleWhen` type + `input.pattern` + `input.auto_uppercase` documented in `spec/sdui-base-components.md`.
- [x] `/actions/assets/create_modal` returns an `ActionResponse{replace, assets-modal-slot, <modal>}`.
- [x] `/actions/assets/edit_modal?id=<id>` fetches the asset from the backend and returns a modal with mutable fields pre-populated and immutables as text.
- [x] `/actions/assets/delete_modal?id=<id>` returns a confirmation modal with a `force` checkbox.
- [x] `POST /actions/assets/create?...` on success returns `ActionResponse{replace, assets-root, <fresh root>, feedback}`.
- [x] `PATCH /actions/assets/<id>?...` same.
- [x] `DELETE /actions/assets/<id>?...` reads `force` from body; sends `?force=true|false` to the backend.
- [x] Mutation BE error → `ActionResponse{replace, assets-modal-slot, <modal repopulated + inline error>}`.
- [x] Filter value and offset persist across successful mutations.
- [x] Actions column renders edit + delete icon buttons per row.
- [x] `assets-new-btn` rendered in the filter row.
- [x] Missing JWT on any new endpoint → 401 with redirect.
- [x] Invalid id / unknown asset → 404 NOT_FOUND.
- [x] All new user-visible strings resolve via i18n en/es.
