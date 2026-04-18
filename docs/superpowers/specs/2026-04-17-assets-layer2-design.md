# Assets — Layer 2: Mutations (design)

Second and final layer of the assets screen. Adds create, edit, and delete operations via modal dialogs. Read-only listing (Layer 1) already ships in `spec/screens/assets/01-list.md`.

Scope: one ship cycle covers all three mutations. Web-only tree (responsive comes later).

## SDUI spec additions

Layer 2 requires three additions to `spec/sdui-*`. These are shared primitives, not assets-specific.

### `VisibleWhen` (reactive visibility)

New optional prop `visible_when` on form components (`input`, `select`, `checkbox`, `textarea`, `radio_group`). Structure:

```go
type VisibleWhen struct {
    Field string      `json:"field"` // name of another form control in the same form
    Op    string      `json:"op"`    // "eq" | "ne"
    Value interface{} `json:"value"` // string, bool, or number
}
```

When the expression evaluates `false` against current form state, the component is hidden. Only `eq` and `ne` are defined; compound expressions (`and`/`or`) are out of scope.

Documented in `spec/sdui-base-components.md` per applicable component.

### `input.pattern`

New optional prop on `input`. String, ECMAScript regex. The FE validates the current value against the pattern on change/blur; non-matching values are flagged as invalid and block form submission.

### `input.auto_uppercase`

New optional prop on `input` (bool). When `true`, the FE transforms the entered value to uppercase automatically (e.g., `aapl` → `AAPL`).

## Screen tree changes (vs L1)

The `assets` screen gains two things: a "New Asset" button in the filter row, and an `assets-modal-slot` sibling of `assets-section` inside `assets-root`.

```
screen id=assets
  column assets-root (gap=lg)
    column assets-section (gap=sm)
      row assets-filter-row widths=["240px","1fr","auto"]      ← now 3 columns
        select asset-type-select ...                           (same as L1)
        spacer filter-spacer size="none"
        button assets-new-btn
          props: label=i18n "assets.new", variant="primary", style="solid"
          actions: [{ trigger:"click", type:"reload",
                      endpoint:"/actions/assets/create_modal",
                      target_id:"assets-modal-slot", loading:"section" }]
      table assets-table
        columns: [...6 L1 columns..., { id:"actions", header:"", width:"100px", align:"right" }]
        children: table_row asset-<id>
          ...6 L1 cells...
          row actions-<id> widths=["auto","auto"] gap="sm"
            button edit-<id>
              props: icon:"pencil", variant:"secondary", style:"ghost"
              actions: [{ trigger:"click", type:"reload",
                          endpoint:"/actions/assets/edit_modal?id=<asset_id>",
                          target_id:"assets-modal-slot", loading:"section" }]
            button delete-<id>
              props: icon:"trash", variant:"secondary", style:"ghost"
              actions: [{ trigger:"click", type:"reload",
                          endpoint:"/actions/assets/delete_modal?id=<asset_id>",
                          target_id:"assets-modal-slot", loading:"section" }]
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
| POST | `/actions/assets/create?asset_type=<f>&offset=<n>` | On success: `ActionResponse{replace, target_id:"assets-root", tree:<fresh root>, feedback:<snackbar>}`. On error: `ActionResponse{replace, target_id:"assets-modal-slot", tree:<same modal with error>}` |
| PATCH | `/actions/assets/:id?asset_type=<f>&offset=<n>` | Same shape as create |
| DELETE | `/actions/assets/:id?force=<bool>&asset_type=<f>&offset=<n>` | Same shape as create |

Mutation endpoints accept `asset_type` and `offset` query params so the handler can rebuild `assets-section` with the list context the user was viewing. Filter and page survive mutations.

Mutation endpoints read their payload from the JSON body that the FE serializes from the form referenced by the `submit` action's `target_id`.

Downstream calls:
- Create: `POST /v1/assets` with the form body.
- Edit: `GET /v1/assets/:id` (for the pre-populated edit modal) and `PATCH /v1/assets/:id` (for the submit).
- Delete: `DELETE /v1/assets/:id?force=<bool>` (no GET; the modal uses only ticker which we have from the list).

## Create modal tree

```
modal assets-create-modal
  props: visible=true, title=i18n "assets.create.title", presentation="dialog"
  form assets-create-form
    column assets-create-fields (gap="md")
      input create-ticker
        props: name="ticker", input_type="text", label=i18n "assets.col.ticker",
               required=true, max_length=20, pattern="^[A-Z0-9.\\-]+$",
               auto_uppercase=true
      input create-name
        props: name="name", input_type="text", label=i18n "assets.col.name",
               required=true, max_length=100
      select create-asset-type
        props: name="asset_type", label=i18n "assets.col.type", required=true,
               options=[{STOCK,STOCK},{ETF,ETF},{CRYPTO,CRYPTO},{BOND,BOND}]
      select create-currency
        props: name="currency", label=i18n "assets.col.currency", required=true,
               options=[USD,EUR,ARS,MXN,GBP]         ← hardcoded set for L2
      checkbox create-is-complex
        props: name="is_complex", label=i18n "assets.form.is_complex", checked=false
      select create-price-provider
        props: name="price_provider", label=i18n "assets.col.price_provider",
               options=[{"",(none)},{COINGECKO,COINGECKO},{TWELVE_DATA,TWELVE_DATA},{ALPHA_VANTAGE,ALPHA_VANTAGE}],
               visible_when={ field:"is_complex", op:"eq", value:false }
      input create-external-ticker
        props: name="external_ticker", input_type="text",
               label=i18n "assets.form.external_ticker",
               placeholder=i18n "assets.form.external_ticker_placeholder",
               max_length=100,
               visible_when={ field:"price_provider", op:"ne", value:"" }
    row create-actions widths=["1fr","auto","auto"] gap="sm"
      spacer create-actions-spacer size="none"
      button create-cancel
        props: label=i18n "common.cancel", variant="secondary", style="ghost"
        actions: [{ trigger:"click", type:"dismiss" }]
      button create-submit
        props: label=i18n "assets.create.submit", variant="primary", style="solid"
        actions: [{ trigger:"click", type:"submit", method:"POST",
                    endpoint:"/actions/assets/create?asset_type=<f>&offset=<n>",
                    target_id:"assets-create-form", loading:"section" }]
```

`<f>` and `<n>` are the current filter value and offset, substituted server-side when emitting the modal.

## Edit modal tree

Same skeleton as create, with these differences:
- Title: i18n "assets.edit.title" with the asset's ticker interpolated (e.g. "Edit AAPL").
- **Immutable fields rendered as static `text`** (ticker, asset_type, currency, is_complex). Each is a labeled read-only line with the current value.
- **Mutable fields** (inputs/selects): only `name`, `price_provider`, `external_ticker`. Same `visible_when` rules as create.
- Submit: `{ type:"submit", method:"PATCH", endpoint:"/actions/assets/<id>?asset_type=<f>&offset=<n>", target_id:"assets-edit-form" }`.

## Delete modal tree

```
modal assets-delete-modal
  props: visible=true, title=i18n "assets.delete.title", presentation="dialog"
  form assets-delete-form
    column (gap="md")
      text delete-message
        props: content=i18n "assets.delete.confirm" interpolated with ticker,
               size="md", weight="normal"
      checkbox delete-force
        props: name="force", label=i18n "assets.delete.force_label", checked=false
      row delete-actions widths=["1fr","auto","auto"] gap="sm"
        spacer delete-actions-spacer size="none"
        button delete-cancel
          props: label=i18n "common.cancel", variant="secondary", style="ghost"
          actions: [{ trigger:"click", type:"dismiss" }]
        button delete-submit
          props: label=i18n "assets.delete.submit", variant="primary", style="solid"   ← styled destructive by FE convention
          actions: [{ trigger:"click", type:"submit", method:"DELETE",
                      endpoint:"/actions/assets/<id>?asset_type=<f>&offset=<n>",
                      target_id:"assets-delete-form", loading:"section" }]
```

The handler reads the `force` checkbox from the submitted body and passes `?force=true|false` through to the backend.

## Response pattern

**Mutation success (any of the three):** `ActionResponse{ action:"replace", target_id:"assets-root", tree:<fresh root>, feedback:<snackbar> }`. The fresh root rebuilds `assets-section` with the filter+offset from the query params and re-initializes `assets-modal-slot` empty. This both updates the list and closes the modal in one response.

**Mutation BE error:** `ActionResponse{ action:"replace", target_id:"assets-modal-slot", tree:<same modal pre-populated with user input + error text at top> }`. The user corrects and re-submits. No snackbar.

**Feedback strings (i18n keys):**

| Operation | Key | en | es |
|---|---|---|---|
| create success | `assets.create.success` | Asset created | Activo creado |
| edit success | `assets.edit.success` | Asset updated | Activo actualizado |
| delete success (no force) | `assets.delete.success` | Asset deleted | Activo eliminado |
| delete success (force) | `assets.delete.success_force` | Asset and associated data deleted | Activo y datos asociados eliminados |

## i18n keys introduced (L2)

Added to `locales/en.json` and `locales/es.json`, extending the existing `assets` block:

```
assets.new                              → "New Asset" / "Nuevo Activo"
assets.create.title                     → "Create Asset" / "Crear Activo"
assets.create.submit                    → "Create" / "Crear"
assets.create.success                   → (above)
assets.edit.title                       → "Edit {ticker}" / "Editar {ticker}"
assets.edit.submit                      → "Save" / "Guardar"
assets.edit.success                     → (above)
assets.delete.title                     → "Delete Asset" / "Eliminar Activo"
assets.delete.confirm                   → "Delete {ticker}? This cannot be undone." /
                                          "¿Eliminar {ticker}? Esta acción no se puede deshacer."
assets.delete.force_label               → "Also delete associated trades and snapshots" /
                                          "Eliminar también trades y snapshots asociados"
assets.delete.submit                    → "Delete" / "Eliminar"
assets.delete.success                   → (above)
assets.delete.success_force             → (above)
assets.form.is_complex                  → "Complex asset" / "Activo complejo"
assets.form.external_ticker             → "External ticker" / "Ticker externo"
assets.form.external_ticker_placeholder → "Defaults to ticker if empty" /
                                          "Usa el ticker si queda vacío"
common.cancel                           → "Cancel" / "Cancelar"
```

## Error handling

Applies to all six new endpoints.

| Situation | HTTP | Body |
|---|---|---|
| Missing / invalid / expired JWT | 401 | `{"error":"unauthorized","redirect":"/login"}` |
| Backend 401 downstream | 401 | same |
| Backend 5xx or network error | 502 | `{"error":{"code":"BACKEND_ERROR","message":"..."}}` |
| Backend 4xx validation (on mutation endpoints) | 200 | `ActionResponse{replace, assets-modal-slot, <modal + error>}` |
| Invalid query / body | 400 | `{"error":{"code":"BAD_REQUEST","message":"..."}}` |
| Asset not found (edit/delete modal GET) | 404 | `{"error":{"code":"NOT_FOUND"}}` |

BE validation errors to render inline (mapped from BE error codes):
- `INVALID_TICKER`, `INVALID_ASSET_TYPE`, `INVALID_CURRENCY`, `INVALID_PRICE_PROVIDER`, `ASSET_ALREADY_EXISTS`, `COMPLEX_ASSET_PRICE_PROVIDER`, `ASSET_NOT_FOUND`, `ASSET_HAS_DATA` (when `force=false`).

The error text component lives at the top of the form inside the modal: `text modal-error { content: <localized message>, color:"negative", size:"sm", weight:"normal" }`.

## Package layout

```
internal/assets/
  ...existing L1 files...
  mutate_client.go                 - Client.CreateAsset, UpdateAsset, DeleteAsset, GetAsset
  mutate_client_test.go
  modal_builder.go                 - BuildCreateModal, BuildEditModal, BuildDeleteModal + helpers
  modal_builder_test.go
  create_modal_handler.go          - GET /actions/assets/create_modal
  create_modal_handler_test.go
  edit_modal_handler.go            - GET /actions/assets/edit_modal
  edit_modal_handler_test.go
  delete_modal_handler.go          - GET /actions/assets/delete_modal
  delete_modal_handler_test.go
  create_handler.go                - POST /actions/assets/create
  create_handler_test.go
  update_handler.go                - PATCH /actions/assets/:id
  update_handler_test.go
  delete_handler.go                - DELETE /actions/assets/:id
  delete_handler_test.go
```

Separation:
- `mutate_client.go`: backend calls for `POST/PATCH/DELETE/GET` on `/v1/assets`. No SDUI imports.
- `modal_builder.go`: constructs the three modal trees. No HTTP.
- `create_modal_handler.go` / `edit_modal_handler.go` / `delete_modal_handler.go`: thin HTTP adapters that build modal responses. Edit fetches the asset via `mutate_client.GetAsset`; delete takes only the id/ticker (ticker fetched from `GetAsset` too, or cached in the list — simplest is to fetch).
- `create_handler.go` / `update_handler.go` / `delete_handler.go`: HTTP adapters that call the mutate client, then rebuild `assets-root` via the use case on success, or return the modal with an inline error on failure.

## Rules and constraints

- Mutation endpoints always carry the current filter (`asset_type`) and offset through their query string so the post-mutation `assets-root` rebuild preserves list context.
- `delete_force` checkbox: sent as `force=true` or `force=false` in the DELETE query string. No two-stage UX.
- The modal slot is outside `assets-section`; filter changes and pagination do not affect an open modal.
- After success, the modal slot is reset to empty in the returned tree (visually closing the modal).
- Client-side validation (via `required`, `pattern`, `max_length`, `auto_uppercase`) is the first defense; the backend still enforces the same rules.
- Immutable fields (`ticker`, `asset_type`, `currency`, `is_complex`) are rendered as static text in the edit modal, not as disabled inputs. No client-side submission of these fields on edit.

## Acceptance criteria

- [ ] `spec/sdui-base-components.md` documents `visible_when` on each applicable form component.
- [ ] `spec/sdui-base-components.md` documents `input.pattern` and `input.auto_uppercase`.
- [ ] `GET /actions/assets/create_modal` (with valid JWT) returns an `ActionResponse{replace, target_id:"assets-modal-slot", tree:<modal>}` with an empty create form that validates client-side.
- [ ] `GET /actions/assets/edit_modal?id=<id>` fetches the asset from the backend and returns a modal with the mutable fields pre-populated and immutable fields shown as text.
- [ ] `GET /actions/assets/delete_modal?id=<id>` returns a confirmation modal with a `force` checkbox.
- [ ] `POST /actions/assets/create?asset_type=<f>&offset=<n>` on success returns `ActionResponse{replace, assets-root, tree:<fresh root with updated section + empty modal slot>, feedback:<success snackbar>}`.
- [ ] `PATCH /actions/assets/<id>?...` idem.
- [ ] `DELETE /actions/assets/<id>?force=<bool>&...` idem; `force=true` sends `?force=true` to the backend.
- [ ] Mutation BE error → `ActionResponse{replace, assets-modal-slot, <modal repopulated + inline error text>}`.
- [ ] Filter value and offset persist across successful mutations.
- [ ] New "Actions" column in the table renders edit + delete icon buttons per row.
- [ ] `assets-new-btn` is rendered as the third column of the filter row.
- [ ] Missing JWT on any new endpoint → `401 {"error":"unauthorized","redirect":"/login"}`.
- [ ] Invalid id / unknown asset on edit/delete modal GET → `404 NOT_FOUND`.
- [ ] All new user-visible strings resolve via i18n en/es.
