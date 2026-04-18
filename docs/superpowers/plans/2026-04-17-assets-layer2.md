# Assets Layer 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `docs/superpowers/specs/2026-04-17-assets-layer2-design.md` — add create, edit, and delete flows to the assets screen via modal dialogs, plus three shared SDUI primitives (`visible_when`, `input.pattern`, `input.auto_uppercase`).

**Architecture:** Extend the existing `internal/assets/` package with a BE mutate client, a modal builder, three modal GET handlers (create/edit/delete modal trees), and three mutation handlers (POST/PATCH/DELETE). On mutation success, handlers rebuild `assets-root` fresh (updated list + empty modal slot) preserving filter/offset via query params. Modal errors are returned as `replace` on the modal slot with the same form pre-populated + inline error text.

**Tech Stack:** Go, Gin, testify, existing `internal/components`, `internal/i18n`, `internal/auth`, `internal/shared`.

---

## File Structure

**Create:**

| File | Responsibility |
|---|---|
| `internal/assets/mutate_client.go` | `GetAsset`, `CreateAsset`, `UpdateAsset`, `DeleteAsset` methods on `Client` |
| `internal/assets/mutate_client_test.go` | BE calls, forwarded headers, status mapping |
| `internal/assets/modal_builder.go` | `BuildCreateModal`, `BuildEditModal`, `BuildDeleteModal` |
| `internal/assets/modal_builder_test.go` | Tree shape per modal, visibility flags, action wiring |
| `internal/assets/create_modal_handler.go` | `GET /actions/assets/create_modal` |
| `internal/assets/create_modal_handler_test.go` | HTTP paths |
| `internal/assets/edit_modal_handler.go` | `GET /actions/assets/edit_modal?id=` |
| `internal/assets/edit_modal_handler_test.go` | HTTP paths |
| `internal/assets/delete_modal_handler.go` | `GET /actions/assets/delete_modal?id=` |
| `internal/assets/delete_modal_handler_test.go` | HTTP paths |
| `internal/assets/create_handler.go` | `POST /actions/assets/create` |
| `internal/assets/create_handler_test.go` | Happy path + BE error → modal replace |
| `internal/assets/update_handler.go` | `PATCH /actions/assets/:id` |
| `internal/assets/update_handler_test.go` | Idem |
| `internal/assets/delete_handler.go` | `DELETE /actions/assets/:id` |
| `internal/assets/delete_handler_test.go` | Idem |
| `spec/screens/assets/02-mutations.md` | Canonical L2 spec |

**Modify:**

| File | Change |
|---|---|
| `internal/components/base.go` | Add `VisibleWhen` struct type |
| `spec/sdui-base-components.md` | Document `visible_when`, `input.pattern`, `input.auto_uppercase` |
| `internal/assets/builder.go` | Add "New Asset" button to filter row; add "Actions" column with edit/delete buttons per row; add empty `assets-modal-slot` at the end of `assets-root` |
| `internal/assets/builder_test.go` | Assertions for the new button, column, and modal slot |
| `internal/server/server.go` | Register six new protected routes |
| `locales/en.json`, `locales/es.json` | Add `assets.new`, `assets.create.*`, `assets.edit.*`, `assets.delete.*`, `assets.form.*`, `common.cancel` |
| `spec/spec.md` | Point the assets row at the screens/assets directory (unchanged format; L2 file joins L1) |

---

### Task 1: SDUI types + spec additions

**Files:**
- Modify: `internal/components/base.go`
- Modify: `spec/sdui-base-components.md`

- [ ] **Step 1: Add `VisibleWhen` struct to base.go**

At the end of `internal/components/base.go` (after `RadioGroup`), append:

```go

// VisibleWhen expresses conditional visibility of a form component based on
// another control's current value. When the expression evaluates false in the
// current form state, the frontend hides the component.
//
// Supported ops: "eq" (equals), "ne" (not equals).
// Field must match the `name` prop of another control in the same form.
type VisibleWhen struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}
```

- [ ] **Step 2: Run `go build ./...`**

Expected: clean build. No test yet — this is a type-only addition; tests come when modal_builder uses it.

- [ ] **Step 3: Update `spec/sdui-base-components.md` — modal section stays, but three additions.**

Find the `### input` section's prop table and append two rows:

```
| `pattern` | string | no | ECMAScript regex validated client-side on change/blur; non-matching values block submission |
| `auto_uppercase` | bool | no | Frontend transforms entered value to uppercase as the user types |
```

- [ ] **Step 4: Add a new subsection at the end of the document**

After the last component section, add:

```markdown

---

## Form component visibility: `visible_when`

Form components (`input`, `select`, `checkbox`, `textarea`, `radio_group`) accept an optional `visible_when` prop that expresses conditional visibility based on another control's current value.

Structure:

```json
{
  "field": "is_complex",
  "op": "eq",
  "value": false
}
```

When the expression evaluates `false` against the current form state, the frontend hides the component. Hidden components do not contribute to form data on submit.

| Field | Type | Description |
|-------|------|-------------|
| `field` | string | `name` of another form control in the same form |
| `op` | string | `eq` (equals) or `ne` (not equals) |
| `value` | any | String, bool, or number to compare against |

Compound expressions (`and`/`or`) are not defined. If more complex reactive logic is needed, do a server-side round-trip instead.
```

- [ ] **Step 5: Commit**

```bash
git add internal/components/base.go spec/sdui-base-components.md
git commit -m "feat(sdui): VisibleWhen type + input.pattern + input.auto_uppercase"
```

---

### Task 2: i18n keys

**Files:**
- Modify: `locales/en.json`, `locales/es.json`

- [ ] **Step 1: Extend the `assets` block and add `common.cancel` in `locales/en.json`**

Locate the `assets` block. Inside it, after `empty_filtered_subtitle`, insert:

```json
    "new": "New Asset",
    "create": {
      "title": "Create Asset",
      "submit": "Create",
      "success": "Asset created"
    },
    "edit": {
      "title": "Edit {ticker}",
      "submit": "Save",
      "success": "Asset updated"
    },
    "delete": {
      "title": "Delete Asset",
      "confirm": "Delete {ticker}? This cannot be undone.",
      "force_label": "Also delete associated trades and snapshots",
      "submit": "Delete",
      "success": "Asset deleted",
      "success_force": "Asset and associated data deleted"
    },
    "form": {
      "is_complex": "Complex asset",
      "external_ticker": "External ticker",
      "external_ticker_placeholder": "Defaults to ticker if empty"
    },
```

(Note: the existing `empty_filtered_subtitle` line now needs a trailing comma.)

At the top level, add (before `home`):

```json
  "common": {
    "cancel": "Cancel"
  },
```

- [ ] **Step 2: Mirror in `locales/es.json`**

Same structure with Spanish strings:

```json
    "new": "Nuevo Activo",
    "create": {
      "title": "Crear Activo",
      "submit": "Crear",
      "success": "Activo creado"
    },
    "edit": {
      "title": "Editar {ticker}",
      "submit": "Guardar",
      "success": "Activo actualizado"
    },
    "delete": {
      "title": "Eliminar Activo",
      "confirm": "¿Eliminar {ticker}? Esta acción no se puede deshacer.",
      "force_label": "Eliminar también trades y snapshots asociados",
      "submit": "Eliminar",
      "success": "Activo eliminado",
      "success_force": "Activo y datos asociados eliminados"
    },
    "form": {
      "is_complex": "Activo complejo",
      "external_ticker": "Ticker externo",
      "external_ticker_placeholder": "Usa el ticker si queda vacío"
    },
```

```json
  "common": {
    "cancel": "Cancelar"
  },
```

- [ ] **Step 3: Verify JSON parses**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && python3 -c "import json; json.load(open('locales/en.json')); json.load(open('locales/es.json')); print('ok')"`
Expected: `ok`.

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "i18n: add assets mutation keys (new/create/edit/delete/form) + common.cancel"
```

---

### Task 3: Mutate client

**Files:**
- Create: `internal/assets/mutate_client.go`
- Create: `internal/assets/mutate_client_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/mutate_client_test.go`:

```go
package assets

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetAsset_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/assets/a1", r.URL.Path)
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD","is_complex":false,"price_provider":"TWELVE_DATA","external_ticker":"AAPL"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	a, err := c.GetAsset(context.Background(), "Bearer tok", "a1")
	require.NoError(t, err)
	assert.Equal(t, "AAPL", a.Ticker)
	require.NotNil(t, a.PriceProvider)
	assert.Equal(t, "TWELVE_DATA", *a.PriceProvider)
	require.NotNil(t, a.ExternalTicker)
	assert.Equal(t, "AAPL", *a.ExternalTicker)
}

func TestClient_GetAsset_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.GetAsset(context.Background(), "Bearer tok", "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAssetNotFound))
}

func TestClient_CreateAsset_ForwardsBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/assets", r.URL.Path)
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"a2","ticker":"TSLA","name":"Tesla","asset_type":"STOCK","currency":"USD","is_complex":false}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	body := map[string]any{"ticker": "TSLA", "name": "Tesla", "asset_type": "STOCK", "currency": "USD"}
	a, err := c.CreateAsset(context.Background(), "Bearer tok", body)
	require.NoError(t, err)
	assert.Equal(t, "TSLA", a.Ticker)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &parsed))
	assert.Equal(t, "TSLA", parsed["ticker"])
}

func TestClient_CreateAsset_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"ASSET_ALREADY_EXISTS","message":"Asset exists"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateAsset(context.Background(), "Bearer tok", map[string]any{})
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be), "want BackendValidationError, got %T", err)
	assert.Equal(t, "ASSET_ALREADY_EXISTS", be.Code)
}

func TestClient_UpdateAsset_PATCH(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		assert.Equal(t, "/v1/assets/a1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"a1","ticker":"AAPL","name":"Apple Inc","asset_type":"STOCK","currency":"USD","is_complex":false}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	body := map[string]any{"name": "Apple Inc"}
	a, err := c.UpdateAsset(context.Background(), "Bearer tok", "a1", body)
	require.NoError(t, err)
	assert.Equal(t, "Apple Inc", a.Name)
	assert.Equal(t, http.MethodPatch, gotMethod)
}

func TestClient_DeleteAsset_PassesForce(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteAsset(context.Background(), "Bearer tok", "a1", true)
	require.NoError(t, err)
	assert.Contains(t, gotQuery, "force=true")
}

func TestClient_DeleteAsset_AssetHasData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"ASSET_HAS_DATA","message":"Has data"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	err := c.DeleteAsset(context.Background(), "Bearer tok", "a1", false)
	require.Error(t, err)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "ASSET_HAS_DATA", be.Code)
}

func TestClient_CreateAsset_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateAsset(context.Background(), "", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_CreateAsset_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second)
	_, err := c.CreateAsset(context.Background(), "Bearer tok", map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

// silence "imported and not used: strings" if the file above doesn't otherwise use it.
var _ = strings.TrimSpace
```

- [ ] **Step 2: Run tests to verify failure**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/assets/... -run 'TestClient_(GetAsset|CreateAsset|UpdateAsset|DeleteAsset)' -v`
Expected: FAIL — `GetAsset`, `CreateAsset`, `UpdateAsset`, `DeleteAsset`, `ErrAssetNotFound`, `BackendValidationError`, `ExternalTicker` undefined.

- [ ] **Step 3: Add `ExternalTicker` field to `Asset` struct (and parser)**

Edit `internal/assets/types.go`. In the `Asset` struct, add the field:

```go
type Asset struct {
	ID             string
	Ticker         string
	Name           string
	AssetType      string
	Currency       string
	IsComplex      bool
	PriceProvider  *string
	ExternalTicker *string
}
```

In the `rawAsset` struct, add the tag:

```go
type rawAsset struct {
	ID             string  `json:"id"`
	Ticker         string  `json:"ticker"`
	Name           string  `json:"name"`
	AssetType      string  `json:"asset_type"`
	Currency       string  `json:"currency"`
	IsComplex      bool    `json:"is_complex"`
	PriceProvider  *string `json:"price_provider"`
	ExternalTicker *string `json:"external_ticker"`
}
```

In `ParseListResponse`, propagate the new field when mapping raw → domain:

```go
out.Assets = append(out.Assets, Asset{
	ID:             ra.ID,
	Ticker:         ra.Ticker,
	Name:           ra.Name,
	AssetType:      ra.AssetType,
	Currency:       ra.Currency,
	IsComplex:      ra.IsComplex,
	PriceProvider:  ra.PriceProvider,
	ExternalTicker: ra.ExternalTicker,
})
```

(If the existing code uses `Asset(ra)` direct cast, change it back to the explicit map since the structs now have the new field in the same order and the cast still works — but explicit is safer.)

- [ ] **Step 4: Implement the mutate client**

Create `internal/assets/mutate_client.go`:

```go
package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// ErrAssetNotFound is returned when the backend responds 404 to a single-asset lookup.
var ErrAssetNotFound = errors.New("asset not found")

// BackendValidationError carries a 4xx validation error from the backend.
// Includes the error code (e.g. ASSET_ALREADY_EXISTS, ASSET_HAS_DATA) and
// a human-readable message.
type BackendValidationError struct {
	Code    string
	Message string
}

func (e *BackendValidationError) Error() string {
	return fmt.Sprintf("backend validation: %s: %s", e.Code, e.Message)
}

type backendErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// GetAsset fetches a single asset by id.
func (c *Client) GetAsset(ctx context.Context, authorization, id string) (*Asset, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/assets/"+id, nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	return c.doAsset(req, http.StatusOK)
}

// CreateAsset posts the given fields to /v1/assets.
func (c *Client) CreateAsset(ctx context.Context, authorization string, body map[string]any) (*Asset, error) {
	return c.doAssetWithBody(ctx, authorization, http.MethodPost, "/v1/assets", body, http.StatusCreated)
}

// UpdateAsset patches an existing asset.
func (c *Client) UpdateAsset(ctx context.Context, authorization, id string, body map[string]any) (*Asset, error) {
	return c.doAssetWithBody(ctx, authorization, http.MethodPatch, "/v1/assets/"+id, body, http.StatusOK)
}

// DeleteAsset deletes an asset, optionally with force.
func (c *Client) DeleteAsset(ctx context.Context, authorization, id string, force bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/assets/"+id, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Set("force", strconv.FormatBool(force))
	req.URL.RawQuery = q.Encode()
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrAssetNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return parseValidationError(rawBody)
	default:
		return fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func (c *Client) doAsset(req *http.Request, successStatus int) (*Asset, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case successStatus:
		var ra rawAsset
		if err := json.Unmarshal(rawBody, &ra); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		a := Asset{
			ID:             ra.ID,
			Ticker:         ra.Ticker,
			Name:           ra.Name,
			AssetType:      ra.AssetType,
			Currency:       ra.Currency,
			IsComplex:      ra.IsComplex,
			PriceProvider:  ra.PriceProvider,
			ExternalTicker: ra.ExternalTicker,
		}
		return &a, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrAssetNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return nil, parseValidationError(rawBody)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func (c *Client) doAssetWithBody(ctx context.Context, authorization, method, path string, body map[string]any, successStatus int) (*Asset, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	return c.doAsset(req, successStatus)
}

func parseValidationError(body []byte) error {
	var b backendErrorBody
	if err := json.Unmarshal(body, &b); err != nil || b.Error.Code == "" {
		return fmt.Errorf("%w: status 4xx", ErrBackend)
	}
	return &BackendValidationError{Code: b.Error.Code, Message: b.Error.Message}
}
```

- [ ] **Step 5: Run full test suite**

Run: `go test ./internal/assets/... -count=1 -v`
Expected: all existing L1 tests still pass + 9 new mutate_client tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/assets/mutate_client.go internal/assets/mutate_client_test.go internal/assets/types.go
git commit -m "feat(assets): mutate client (Get/Create/Update/Delete) + ExternalTicker field"
```

---

### Task 4: Section builder updates (filter button + actions column + modal slot)

**Files:**
- Modify: `internal/assets/builder.go`
- Modify: `internal/assets/builder_test.go`

- [ ] **Step 1: Update `BuildScreen` to include the modal slot**

In `internal/assets/builder.go`, change `BuildScreen` to:

```go
// BuildScreen returns the full SDUI tree for GET /screens/assets.
func BuildScreen(result *ListResult, params ListParams, lang string) components.Component {
	section := BuildAssetsSection(result, params, lang)
	modalSlot := components.Column("assets-modal-slot")
	root := components.ColumnWithGap("assets-root", "lg", section, modalSlot)
	return components.Screen("assets", i18n.T(lang, "assets.title"), root)
}
```

- [ ] **Step 2: Update `buildFilter` to add the "New Asset" button**

Replace the current `buildFilter` function body with:

```go
func buildFilter(params ListParams, lang string) components.Component {
	opts := []components.SelectOption{
		{Value: "", Label: i18n.T(lang, "assets.filter.type_any")},
		{Value: "STOCK", Label: "STOCK"},
		{Value: "ETF", Label: "ETF"},
		{Value: "CRYPTO", Label: "CRYPTO"},
		{Value: "BOND", Label: "BOND"},
	}
	sel := components.Component{
		Type: "select",
		ID:   "asset-type-select",
		Props: map[string]any{
			"name":          "asset_type",
			"label":         i18n.T(lang, "assets.filter.type"),
			"default_value": params.AssetType,
			"options":       opts,
		},
		Actions: []components.Action{
			{
				Trigger:  "change",
				Type:     "reload",
				Endpoint: "/actions/assets/list?asset_type={value}",
				TargetID: "assets-section",
				Loading:  "section",
			},
		},
	}
	filler := components.Spacer("filter-spacer", "none")
	newBtn := components.ButtonFull("assets-new-btn", i18n.T(lang, "assets.new"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/assets/create_modal",
			TargetID: "assets-modal-slot",
			Loading:  "section",
		},
	)
	return components.Row("assets-filter-row", []string{"240px", "1fr", "auto"}, sel, filler, newBtn)
}
```

- [ ] **Step 3: Add the "Actions" column to the table**

Replace the `buildTable` function body with:

```go
func buildTable(assets []Asset, lang string) components.Component {
	cols := []components.TableColumn{
		{ID: "ticker", Header: i18n.T(lang, "assets.col.ticker"), Width: "120px"},
		{ID: "name", Header: i18n.T(lang, "assets.col.name"), Width: "1fr"},
		{ID: "type", Header: i18n.T(lang, "assets.col.type"), Width: "100px"},
		{ID: "currency", Header: i18n.T(lang, "assets.col.currency"), Width: "100px"},
		{ID: "complex", Header: i18n.T(lang, "assets.col.complex"), Width: "100px", Align: "center"},
		{ID: "price_provider", Header: i18n.T(lang, "assets.col.price_provider"), Width: "160px"},
		{ID: "actions", Header: "", Width: "120px", Align: "right"},
	}
	rows := make([]components.Component, 0, len(assets))
	for _, a := range assets {
		rows = append(rows, buildRow(a))
	}
	return components.Table("assets-table", cols, rows...)
}
```

- [ ] **Step 4: Add row-level action buttons**

Replace `buildRow` with:

```go
func buildRow(a Asset) components.Component {
	cell := func(id, content string) components.Component {
		return components.Text(id, content, "sm", "normal")
	}
	ticker := components.Text("asset-"+a.ID+"-ticker", strings.ToUpper(a.Ticker), "sm", "bold")
	complexCell := "—"
	if a.IsComplex {
		complexCell = "✓"
	}
	providerCell := "—"
	if !a.IsComplex && a.PriceProvider != nil {
		providerCell = *a.PriceProvider
	}

	editBtn := components.ButtonFull("edit-"+a.ID, "", "", "secondary", "ghost",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/assets/edit_modal?id=" + a.ID,
			TargetID: "assets-modal-slot",
			Loading:  "section",
		},
	)
	editBtn.Props["icon"] = "pencil"
	deleteBtn := components.ButtonFull("delete-"+a.ID, "", "", "secondary", "ghost",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/assets/delete_modal?id=" + a.ID,
			TargetID: "assets-modal-slot",
			Loading:  "section",
		},
	)
	deleteBtn.Props["icon"] = "trash"
	actionsRow := components.RowWithGap("actions-"+a.ID, []string{"auto", "auto"}, "sm", editBtn, deleteBtn)

	return components.TableRow("asset-"+a.ID,
		ticker,
		cell("asset-"+a.ID+"-name", a.Name),
		cell("asset-"+a.ID+"-type", a.AssetType),
		cell("asset-"+a.ID+"-currency", strings.ToUpper(a.Currency)),
		cell("asset-"+a.ID+"-complex", complexCell),
		cell("asset-"+a.ID+"-price_provider", providerCell),
		actionsRow,
	)
}
```

- [ ] **Step 5: Update the tests**

In `internal/assets/builder_test.go`, update:

- `TestBuildScreen_ShapeAndTitle`: add assertion that `assets-modal-slot` exists under `assets-root`.

After the existing assertions in that test:

```go
	modalSlot := findByID(tree, "assets-modal-slot")
	require.NotNil(t, modalSlot)
	assert.Equal(t, "column", modalSlot.Type)
	assert.Empty(t, modalSlot.Children)
```

- `TestBuildAssetsSection_TableColumnsAndRows`: update the columns assertion to expect 7 columns (including `actions`), and assert that each row has an `actions-<id>` row with two buttons.

Change:

```go
	require.Len(t, cols, 6)
	assert.Equal(t, []string{"ticker", "name", "type", "currency", "complex", "price_provider"},
		[]string{cols[0].ID, cols[1].ID, cols[2].ID, cols[3].ID, cols[4].ID, cols[5].ID})
```

to:

```go
	require.Len(t, cols, 7)
	assert.Equal(t, []string{"ticker", "name", "type", "currency", "complex", "price_provider", "actions"},
		[]string{cols[0].ID, cols[1].ID, cols[2].ID, cols[3].ID, cols[4].ID, cols[5].ID, cols[6].ID})
```

And update the row cell count from 6 to 7 in the loop assertions:

```go
	require.Len(t, r1.Children, 7)
```

Apply to r2 and r3 as well.

Add assertions for the actions row:

```go
	actionsR1 := findByID(r1, "actions-a1")
	require.NotNil(t, actionsR1)
	editBtn := findByID(*actionsR1, "edit-a1")
	require.NotNil(t, editBtn)
	require.Len(t, editBtn.Actions, 1)
	assert.Equal(t, "reload", editBtn.Actions[0].Type)
	assert.Equal(t, "/actions/assets/edit_modal?id=a1", editBtn.Actions[0].Endpoint)
	assert.Equal(t, "assets-modal-slot", editBtn.Actions[0].TargetID)

	deleteBtn := findByID(*actionsR1, "delete-a1")
	require.NotNil(t, deleteBtn)
	assert.Equal(t, "/actions/assets/delete_modal?id=a1", deleteBtn.Actions[0].Endpoint)
```

Add a new test for the new button:

```go
func TestBuildAssetsSection_NewAssetButton(t *testing.T) {
	section := BuildAssetsSection(&ListResult{Size: 10}, ListParams{}, "en")
	btn := findByID(section, "assets-new-btn")
	require.NotNil(t, btn)
	assert.Equal(t, "button", btn.Type)
	assert.Equal(t, "New Asset", btn.Props["label"])
	require.Len(t, btn.Actions, 1)
	assert.Equal(t, "reload", btn.Actions[0].Type)
	assert.Equal(t, "/actions/assets/create_modal", btn.Actions[0].Endpoint)
	assert.Equal(t, "assets-modal-slot", btn.Actions[0].TargetID)
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/assets/... -count=1 -v`
Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/assets/builder.go internal/assets/builder_test.go
git commit -m "feat(assets): L2 wiring in list — New Asset button, actions column, modal slot"
```

---

### Task 5: Modal builder

**Files:**
- Create: `internal/assets/modal_builder.go`
- Create: `internal/assets/modal_builder_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/modal_builder_test.go`:

```go
package assets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCreateModal_Shape(t *testing.T) {
	m := BuildCreateModal(ListParams{AssetType: "STOCK", Offset: 10}, "en", "")

	assert.Equal(t, "modal", m.Type)
	assert.Equal(t, "assets-create-modal", m.ID)
	assert.Equal(t, true, m.Props["visible"])
	assert.Equal(t, "Create Asset", m.Props["title"])
	assert.Equal(t, "dialog", m.Props["presentation"])

	form := findByID(m, "assets-create-form")
	require.NotNil(t, form)

	ticker := findByID(m, "create-ticker")
	require.NotNil(t, ticker)
	assert.Equal(t, "input", ticker.Type)
	assert.Equal(t, "ticker", ticker.Props["name"])
	assert.Equal(t, true, ticker.Props["required"])
	assert.Equal(t, 20, ticker.Props["max_length"])
	assert.Equal(t, `^[A-Z0-9.\-]+$`, ticker.Props["pattern"])
	assert.Equal(t, true, ticker.Props["auto_uppercase"])

	pp := findByID(m, "create-price-provider")
	require.NotNil(t, pp)
	vw, ok := pp.Props["visible_when"].(VisibleWhenValue)
	require.True(t, ok, "visible_when must be set on price_provider")
	assert.Equal(t, "is_complex", vw.Field)
	assert.Equal(t, "eq", vw.Op)
	assert.Equal(t, false, vw.Value)

	ext := findByID(m, "create-external-ticker")
	require.NotNil(t, ext)
	vw2, ok := ext.Props["visible_when"].(VisibleWhenValue)
	require.True(t, ok)
	assert.Equal(t, "price_provider", vw2.Field)
	assert.Equal(t, "ne", vw2.Op)
	assert.Equal(t, "", vw2.Value)

	submit := findByID(m, "create-submit")
	require.NotNil(t, submit)
	require.Len(t, submit.Actions, 1)
	act := submit.Actions[0]
	assert.Equal(t, "submit", act.Type)
	assert.Equal(t, "POST", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/assets/create")
	assert.Contains(t, act.Endpoint, "asset_type=STOCK")
	assert.Contains(t, act.Endpoint, "offset=10")
	assert.Equal(t, "assets-create-form", act.TargetID)
}

func TestBuildCreateModal_WithError(t *testing.T) {
	m := BuildCreateModal(ListParams{}, "en", "Ticker already registered")
	err := findByID(m, "modal-error")
	require.NotNil(t, err)
	assert.Equal(t, "Ticker already registered", err.Props["content"])
	assert.Equal(t, "negative", err.Props["color"])
}

func TestBuildEditModal_ImmutableFieldsAsText(t *testing.T) {
	provider := "TWELVE_DATA"
	ext := "AAPL"
	a := &Asset{
		ID: "a1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK",
		Currency: "USD", IsComplex: false,
		PriceProvider: &provider, ExternalTicker: &ext,
	}
	m := BuildEditModal(a, ListParams{}, "en", "")

	assert.Equal(t, "modal", m.Type)
	assert.Equal(t, "assets-edit-modal", m.ID)
	assert.Equal(t, "Edit AAPL", m.Props["title"])

	// Immutable fields as text, not input
	tickerStatic := findByID(m, "edit-ticker-static")
	require.NotNil(t, tickerStatic)
	assert.Equal(t, "text", tickerStatic.Type)

	// Mutable name as input
	nameInput := findByID(m, "edit-name")
	require.NotNil(t, nameInput)
	assert.Equal(t, "input", nameInput.Type)
	assert.Equal(t, "Apple", nameInput.Props["default_value"])

	submit := findByID(m, "edit-submit")
	require.NotNil(t, submit)
	act := submit.Actions[0]
	assert.Equal(t, "PATCH", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/assets/a1")
}

func TestBuildDeleteModal_Shape(t *testing.T) {
	m := BuildDeleteModal("a1", "AAPL", ListParams{AssetType: "STOCK", Offset: 0}, "en", "")

	assert.Equal(t, "modal", m.Type)
	assert.Equal(t, "assets-delete-modal", m.ID)
	assert.Equal(t, "Delete Asset", m.Props["title"])

	msg := findByID(m, "delete-message")
	require.NotNil(t, msg)
	assert.Equal(t, "Delete AAPL? This cannot be undone.", msg.Props["content"])

	force := findByID(m, "delete-force")
	require.NotNil(t, force)
	assert.Equal(t, "checkbox", force.Type)
	assert.Equal(t, "force", force.Props["name"])

	submit := findByID(m, "delete-submit")
	require.NotNil(t, submit)
	act := submit.Actions[0]
	assert.Equal(t, "DELETE", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/assets/a1")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/assets/... -run 'TestBuild(Create|Edit|Delete)Modal' -v`
Expected: FAIL — `BuildCreateModal`, `BuildEditModal`, `BuildDeleteModal`, `VisibleWhenValue` undefined.

- [ ] **Step 3: Add a type alias for test convenience**

In `internal/assets/modal_builder.go` (create new file):

```go
package assets

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// VisibleWhenValue is an alias exposing components.VisibleWhen so tests in this
// package can type-assert without importing components in the test file.
type VisibleWhenValue = components.VisibleWhen

// BuildCreateModal returns the tree for the create-asset modal.
// listParams is the current filter/offset so the submit endpoint preserves list state.
// errMsg, when non-empty, is rendered at the top of the form.
func BuildCreateModal(listParams ListParams, lang, errMsg string) components.Component {
	submitEndpoint := "/actions/assets/create?" + mutationQuery(listParams)

	assetTypeOpts := []components.SelectOption{
		{Value: "STOCK", Label: "STOCK"},
		{Value: "ETF", Label: "ETF"},
		{Value: "CRYPTO", Label: "CRYPTO"},
		{Value: "BOND", Label: "BOND"},
	}
	currencyOpts := []components.SelectOption{
		{Value: "USD", Label: "USD"},
		{Value: "EUR", Label: "EUR"},
		{Value: "ARS", Label: "ARS"},
		{Value: "MXN", Label: "MXN"},
		{Value: "GBP", Label: "GBP"},
	}
	providerOpts := []components.SelectOption{
		{Value: "", Label: i18n.T(lang, "assets.filter.type_any")},
		{Value: "COINGECKO", Label: "COINGECKO"},
		{Value: "TWELVE_DATA", Label: "TWELVE_DATA"},
		{Value: "ALPHA_VANTAGE", Label: "ALPHA_VANTAGE"},
	}

	fields := []components.Component{}
	if errMsg != "" {
		fields = append(fields, components.TextStyled("modal-error", errMsg, "sm", "normal", "", "negative", "", ""))
	}
	fields = append(fields,
		input("create-ticker", "ticker", "text", i18n.T(lang, "assets.col.ticker"), "", "", true, 20, map[string]any{
			"pattern":        `^[A-Z0-9.\-]+$`,
			"auto_uppercase": true,
		}, nil),
		input("create-name", "name", "text", i18n.T(lang, "assets.col.name"), "", "", true, 100, nil, nil),
		selectField("create-asset-type", "asset_type", i18n.T(lang, "assets.col.type"), "", assetTypeOpts, true, nil),
		selectField("create-currency", "currency", i18n.T(lang, "assets.col.currency"), "", currencyOpts, true, nil),
		checkboxField("create-is-complex", "is_complex", i18n.T(lang, "assets.form.is_complex"), false, nil),
		selectField("create-price-provider", "price_provider", i18n.T(lang, "assets.col.price_provider"), "", providerOpts, false,
			&components.VisibleWhen{Field: "is_complex", Op: "eq", Value: false}),
		input("create-external-ticker", "external_ticker", "text", i18n.T(lang, "assets.form.external_ticker"),
			i18n.T(lang, "assets.form.external_ticker_placeholder"), "", false, 100, nil,
			&components.VisibleWhen{Field: "price_provider", Op: "ne", Value: ""}),
	)

	fieldsCol := components.ColumnWithGap("assets-create-fields", "md", fields...)

	cancelBtn := components.ButtonFull("create-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		components.Dismiss())
	submitBtn := components.ButtonFull("create-submit", i18n.T(lang, "assets.create.submit"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Method:   "POST",
			Endpoint: submitEndpoint,
			TargetID: "assets-create-form",
			Loading:  "section",
		},
	)
	actionsRow := components.RowWithGap("create-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("create-actions-spacer", "none"),
		cancelBtn,
		submitBtn,
	)

	form := components.Form("assets-create-form", fieldsCol, actionsRow)
	return components.ModalFull("assets-create-modal", i18n.T(lang, "assets.create.title"), "dialog", true, true, form)
}

// BuildEditModal returns the tree for the edit-asset modal.
// `a` must be non-nil (handler is responsible for fetching and 404-mapping).
func BuildEditModal(a *Asset, listParams ListParams, lang, errMsg string) components.Component {
	submitEndpoint := "/actions/assets/" + a.ID + "?" + mutationQuery(listParams)

	providerOpts := []components.SelectOption{
		{Value: "", Label: i18n.T(lang, "assets.filter.type_any")},
		{Value: "COINGECKO", Label: "COINGECKO"},
		{Value: "TWELVE_DATA", Label: "TWELVE_DATA"},
		{Value: "ALPHA_VANTAGE", Label: "ALPHA_VANTAGE"},
	}

	fields := []components.Component{}
	if errMsg != "" {
		fields = append(fields, components.TextStyled("modal-error", errMsg, "sm", "normal", "", "negative", "", ""))
	}

	// Immutable fields as static text (each a labeled line).
	fields = append(fields,
		staticField("edit-ticker-static", i18n.T(lang, "assets.col.ticker"), strings.ToUpper(a.Ticker)),
		staticField("edit-asset-type-static", i18n.T(lang, "assets.col.type"), a.AssetType),
		staticField("edit-currency-static", i18n.T(lang, "assets.col.currency"), strings.ToUpper(a.Currency)),
		staticField("edit-complex-static", i18n.T(lang, "assets.col.complex"), complexText(a.IsComplex)),
	)

	// Mutable fields.
	fields = append(fields,
		input("edit-name", "name", "text", i18n.T(lang, "assets.col.name"), "", a.Name, true, 100, nil, nil),
	)
	if !a.IsComplex {
		defaultProvider := ""
		if a.PriceProvider != nil {
			defaultProvider = *a.PriceProvider
		}
		fields = append(fields,
			selectField("edit-price-provider", "price_provider", i18n.T(lang, "assets.col.price_provider"), defaultProvider, providerOpts, false, nil),
		)
		defaultExt := ""
		if a.ExternalTicker != nil {
			defaultExt = *a.ExternalTicker
		}
		fields = append(fields,
			input("edit-external-ticker", "external_ticker", "text", i18n.T(lang, "assets.form.external_ticker"),
				i18n.T(lang, "assets.form.external_ticker_placeholder"), defaultExt, false, 100, nil,
				&components.VisibleWhen{Field: "price_provider", Op: "ne", Value: ""}),
		)
	}

	fieldsCol := components.ColumnWithGap("assets-edit-fields", "md", fields...)

	cancelBtn := components.ButtonFull("edit-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		components.Dismiss())
	submitBtn := components.ButtonFull("edit-submit", i18n.T(lang, "assets.edit.submit"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Method:   "PATCH",
			Endpoint: submitEndpoint,
			TargetID: "assets-edit-form",
			Loading:  "section",
		},
	)
	actionsRow := components.RowWithGap("edit-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("edit-actions-spacer", "none"),
		cancelBtn,
		submitBtn,
	)

	title := strings.ReplaceAll(i18n.T(lang, "assets.edit.title"), "{ticker}", strings.ToUpper(a.Ticker))
	form := components.Form("assets-edit-form", fieldsCol, actionsRow)
	return components.ModalFull("assets-edit-modal", title, "dialog", true, true, form)
}

// BuildDeleteModal returns the tree for the delete-asset confirmation modal.
func BuildDeleteModal(assetID, ticker string, listParams ListParams, lang, errMsg string) components.Component {
	submitEndpoint := "/actions/assets/" + assetID + "?" + mutationQuery(listParams)

	message := strings.ReplaceAll(i18n.T(lang, "assets.delete.confirm"), "{ticker}", strings.ToUpper(ticker))

	children := []components.Component{}
	if errMsg != "" {
		children = append(children, components.TextStyled("modal-error", errMsg, "sm", "normal", "", "negative", "", ""))
	}
	children = append(children,
		components.Text("delete-message", message, "md", "normal"),
		checkboxField("delete-force", "force", i18n.T(lang, "assets.delete.force_label"), false, nil),
	)

	cancelBtn := components.ButtonFull("delete-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		components.Dismiss())
	submitBtn := components.ButtonFull("delete-submit", i18n.T(lang, "assets.delete.submit"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Method:   "DELETE",
			Endpoint: submitEndpoint,
			TargetID: "assets-delete-form",
			Loading:  "section",
		},
	)
	actionsRow := components.RowWithGap("delete-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("delete-actions-spacer", "none"),
		cancelBtn,
		submitBtn,
	)
	children = append(children, actionsRow)

	form := components.Form("assets-delete-form", children...)
	return components.ModalFull("assets-delete-modal", i18n.T(lang, "assets.delete.title"), "dialog", true, true, form)
}

// -------- helpers --------

func mutationQuery(p ListParams) string {
	q := url.Values{}
	if p.AssetType != "" {
		q.Set("asset_type", p.AssetType)
	}
	q.Set("offset", strconv.Itoa(p.Offset))
	return q.Encode()
}

// input builds an input component with all optional props and a visible_when.
func input(id, name, inputType, label, placeholder, defaultValue string, required bool, maxLength int, extra map[string]any, vw *components.VisibleWhen) components.Component {
	props := map[string]any{
		"name":       name,
		"input_type": inputType,
	}
	if label != "" {
		props["label"] = label
	}
	if placeholder != "" {
		props["placeholder"] = placeholder
	}
	if defaultValue != "" {
		props["default_value"] = defaultValue
	}
	if required {
		props["required"] = true
	}
	if maxLength > 0 {
		props["max_length"] = maxLength
	}
	for k, v := range extra {
		props[k] = v
	}
	if vw != nil {
		props["visible_when"] = *vw
	}
	return components.Component{Type: "input", ID: id, Props: props}
}

func selectField(id, name, label, defaultValue string, opts []components.SelectOption, required bool, vw *components.VisibleWhen) components.Component {
	props := map[string]any{
		"name":    name,
		"options": opts,
	}
	if label != "" {
		props["label"] = label
	}
	if defaultValue != "" {
		props["default_value"] = defaultValue
	}
	if required {
		props["required"] = true
	}
	if vw != nil {
		props["visible_when"] = *vw
	}
	return components.Component{Type: "select", ID: id, Props: props}
}

func checkboxField(id, name, label string, checked bool, vw *components.VisibleWhen) components.Component {
	props := map[string]any{
		"name":  name,
		"label": label,
	}
	if checked {
		props["checked"] = true
	}
	if vw != nil {
		props["visible_when"] = *vw
	}
	return components.Component{Type: "checkbox", ID: id, Props: props}
}

func staticField(id, label, value string) components.Component {
	content := fmt.Sprintf("%s: %s", label, value)
	return components.Text(id, content, "sm", "normal")
}

func complexText(isComplex bool) string {
	if isComplex {
		return "✓"
	}
	return "—"
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/assets/... -count=1`
Expected: all pass (L1 + mutate_client + modal_builder tests).

- [ ] **Step 5: Commit**

```bash
git add internal/assets/modal_builder.go internal/assets/modal_builder_test.go
git commit -m "feat(assets): modal builder (create/edit/delete trees)"
```

---

### Task 6: Create modal handler

**Files:**
- Create: `internal/assets/create_modal_handler.go`
- Create: `internal/assets/create_modal_handler_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/create_modal_handler_test.go`:

```go
package assets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateModalHandler_HappyPath(t *testing.T) {
	h := NewCreateModalHandler()
	r := gin.New()
	r.GET("/actions/assets/create_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/create_modal?asset_type=STOCK&offset=10", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "replace", body["action"])
	assert.Equal(t, "assets-modal-slot", body["target_id"])
	tree, ok := body["tree"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "modal", tree["type"])
	assert.Equal(t, "assets-create-modal", tree["id"])
}

func TestCreateModalHandler_InvalidParams(t *testing.T) {
	h := NewCreateModalHandler()
	r := gin.New()
	r.GET("/actions/assets/create_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/create_modal?asset_type=BOGUS", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/assets/... -run 'TestCreateModalHandler' -v`
Expected: FAIL — `NewCreateModalHandler` undefined.

- [ ] **Step 3: Implement the handler**

Create `internal/assets/create_modal_handler.go`:

```go
package assets

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
)

type CreateModalHandler struct{}

func NewCreateModalHandler() *CreateModalHandler { return &CreateModalHandler{} }

func (h *CreateModalHandler) Get(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	lang := parseLang(c)
	modal := BuildCreateModal(params, lang, "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "assets-modal-slot",
		Tree:     &modal,
	})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/assets/... -count=1`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/assets/create_modal_handler.go internal/assets/create_modal_handler_test.go
git commit -m "feat(assets): GET /actions/assets/create_modal handler"
```

---

### Task 7: Edit modal handler

**Files:**
- Create: `internal/assets/edit_modal_handler.go`
- Create: `internal/assets/edit_modal_handler_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/edit_modal_handler_test.go`:

```go
package assets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAssetFetcher struct {
	asset *Asset
	err   error
}

func (s *stubAssetFetcher) GetAsset(_ context.Context, _ string, _ string) (*Asset, error) {
	return s.asset, s.err
}

func TestEditModalHandler_HappyPath(t *testing.T) {
	a := &Asset{ID: "a1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD"}
	h := NewEditModalHandler(&stubAssetFetcher{asset: a})
	r := gin.New()
	r.GET("/actions/assets/edit_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/edit_modal?id=a1", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	tree := body["tree"].(map[string]any)
	assert.Equal(t, "assets-edit-modal", tree["id"])
}

func TestEditModalHandler_MissingID(t *testing.T) {
	h := NewEditModalHandler(&stubAssetFetcher{})
	r := gin.New()
	r.GET("/actions/assets/edit_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/edit_modal", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEditModalHandler_NotFound(t *testing.T) {
	h := NewEditModalHandler(&stubAssetFetcher{err: ErrAssetNotFound})
	r := gin.New()
	r.GET("/actions/assets/edit_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/edit_modal?id=missing", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEditModalHandler_Unauthorized(t *testing.T) {
	h := NewEditModalHandler(&stubAssetFetcher{err: ErrUnauthorized})
	r := gin.New()
	r.GET("/actions/assets/edit_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/edit_modal?id=a1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/assets/... -run 'TestEditModalHandler' -v`
Expected: FAIL.

- [ ] **Step 3: Implement the handler**

Create `internal/assets/edit_modal_handler.go`:

```go
package assets

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// assetByIDFetcher is the narrow interface the edit/delete modal handlers need.
type assetByIDFetcher interface {
	GetAsset(ctx context.Context, authorization, id string) (*Asset, error)
}

type EditModalHandler struct {
	client assetByIDFetcher
}

func NewEditModalHandler(client assetByIDFetcher) *EditModalHandler {
	return &EditModalHandler{client: client}
}

func (h *EditModalHandler) Get(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "id is required"}})
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}

	a, err := h.client.GetAsset(c.Request.Context(), c.GetHeader("Authorization"), id)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		if errors.Is(err, ErrAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "asset not found"}})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load asset"}})
		return
	}

	modal := BuildEditModal(a, params, parseLang(c), "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "assets-modal-slot",
		Tree:     &modal,
	})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/assets/... -count=1`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/assets/edit_modal_handler.go internal/assets/edit_modal_handler_test.go
git commit -m "feat(assets): GET /actions/assets/edit_modal handler"
```

---

### Task 8: Delete modal handler

**Files:**
- Create: `internal/assets/delete_modal_handler.go`
- Create: `internal/assets/delete_modal_handler_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/delete_modal_handler_test.go`:

```go
package assets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteModalHandler_HappyPath(t *testing.T) {
	a := &Asset{ID: "a1", Ticker: "AAPL"}
	h := NewDeleteModalHandler(&stubAssetFetcher{asset: a})
	r := gin.New()
	r.GET("/actions/assets/delete_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/delete_modal?id=a1", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	tree := body["tree"].(map[string]any)
	assert.Equal(t, "assets-delete-modal", tree["id"])
}

func TestDeleteModalHandler_NotFound(t *testing.T) {
	h := NewDeleteModalHandler(&stubAssetFetcher{err: ErrAssetNotFound})
	r := gin.New()
	r.GET("/actions/assets/delete_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/delete_modal?id=missing", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteModalHandler_MissingID(t *testing.T) {
	h := NewDeleteModalHandler(&stubAssetFetcher{})
	r := gin.New()
	r.GET("/actions/assets/delete_modal", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/assets/delete_modal", nil)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/assets/... -run 'TestDeleteModalHandler' -v`
Expected: FAIL.

- [ ] **Step 3: Implement the handler**

Create `internal/assets/delete_modal_handler.go`:

```go
package assets

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type DeleteModalHandler struct {
	client assetByIDFetcher
}

func NewDeleteModalHandler(client assetByIDFetcher) *DeleteModalHandler {
	return &DeleteModalHandler{client: client}
}

func (h *DeleteModalHandler) Get(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "id is required"}})
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}

	a, err := h.client.GetAsset(c.Request.Context(), c.GetHeader("Authorization"), id)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		if errors.Is(err, ErrAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "asset not found"}})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load asset"}})
		return
	}

	modal := BuildDeleteModal(a.ID, a.Ticker, params, parseLang(c), "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "assets-modal-slot",
		Tree:     &modal,
	})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/assets/... -count=1`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/assets/delete_modal_handler.go internal/assets/delete_modal_handler_test.go
git commit -m "feat(assets): GET /actions/assets/delete_modal handler"
```

---

### Task 9: Create handler (POST)

**Files:**
- Create: `internal/assets/create_handler.go`
- Create: `internal/assets/create_handler_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/create_handler_test.go`:

```go
package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMutator struct {
	created *Asset
	createErr error
	list    *ListResult
	listErr error
}

func (s *stubMutator) CreateAsset(_ context.Context, _ string, _ map[string]any) (*Asset, error) {
	return s.created, s.createErr
}
func (s *stubMutator) UpdateAsset(_ context.Context, _, _ string, _ map[string]any) (*Asset, error) {
	return nil, nil
}
func (s *stubMutator) DeleteAsset(_ context.Context, _, _ string, _ bool) error {
	return nil
}
func (s *stubMutator) GetAsset(_ context.Context, _, _ string) (*Asset, error) {
	return nil, nil
}
func (s *stubMutator) List(_ context.Context, _ string, _ ListParams) (*ListResult, error) {
	return s.list, s.listErr
}

func TestCreateHandler_HappyPath(t *testing.T) {
	sc := &stubMutator{
		created: &Asset{ID: "a1", Ticker: "TSLA"},
		list:    &ListResult{Assets: []Asset{{ID: "a1", Ticker: "TSLA"}}, Total: 1, Size: 10},
	}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{"ticker": "TSLA", "name": "Tesla", "asset_type": "STOCK", "currency": "USD"})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create?asset_type=STOCK&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var respBody map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Equal(t, "replace", respBody["action"])
	assert.Equal(t, "assets-root", respBody["target_id"])
	fb := respBody["feedback"].(map[string]any)
	assert.Equal(t, "snackbar", fb["type"])
	assert.Equal(t, "Asset created", fb["props"].(map[string]any)["message"])
}

func TestCreateHandler_ValidationError(t *testing.T) {
	sc := &stubMutator{createErr: &BackendValidationError{Code: "ASSET_ALREADY_EXISTS", Message: "Ticker already registered"}}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{"ticker": "AAPL"})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create?asset_type=&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code) // handler returns 200 with replace pointing at modal
	var respBody map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Equal(t, "replace", respBody["action"])
	assert.Equal(t, "assets-modal-slot", respBody["target_id"])
}

func TestCreateHandler_Unauthorized(t *testing.T) {
	sc := &stubMutator{createErr: ErrUnauthorized}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateHandler_BackendError(t *testing.T) {
	sc := &stubMutator{createErr: ErrBackend}
	h := NewCreateHandler(sc)
	r := gin.New()
	r.POST("/actions/assets/create", h.Post)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/actions/assets/create", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/assets/... -run 'TestCreateHandler' -v`
Expected: FAIL.

- [ ] **Step 3: Implement the handler**

Create `internal/assets/create_handler.go`:

```go
package assets

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// assetMutator is the narrow interface mutation handlers depend on.
type assetMutator interface {
	CreateAsset(ctx context.Context, authorization string, body map[string]any) (*Asset, error)
	UpdateAsset(ctx context.Context, authorization, id string, body map[string]any) (*Asset, error)
	DeleteAsset(ctx context.Context, authorization, id string, force bool) error
	GetAsset(ctx context.Context, authorization, id string) (*Asset, error)
	List(ctx context.Context, authorization string, params ListParams) (*ListResult, error)
}

type CreateHandler struct {
	client assetMutator
}

func NewCreateHandler(client assetMutator) *CreateHandler {
	return &CreateHandler{client: client}
}

func (h *CreateHandler) Post(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	var body map[string]any
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(raw) == 0 {
		body = map[string]any{}
	} else if err := json.Unmarshal(raw, &body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid JSON body"}})
		return
	}

	_, err = h.client.CreateAsset(c.Request.Context(), auth, body)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			modal := BuildCreateModal(params, lang, be.Message)
			c.JSON(http.StatusOK, components.ActionResponse{
				Action:   "replace",
				TargetID: "assets-modal-slot",
				Tree:     &modal,
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not create asset"}})
		return
	}

	respondPostMutation(c, h.client, params, lang, i18n.T(lang, "assets.create.success"))
}

// respondPostMutation rebuilds assets-root with fresh list + empty modal slot + success feedback.
func respondPostMutation(c *gin.Context, client assetMutator, params ListParams, lang, successMsg string) {
	res, err := client.List(c.Request.Context(), c.GetHeader("Authorization"), params)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refresh list"}})
		return
	}
	section := BuildAssetsSection(res, params, lang)
	modalSlot := components.Column("assets-modal-slot")
	root := components.ColumnWithGap("assets-root", "lg", section, modalSlot)
	fb := components.Snackbar("feedback", successMsg, "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "assets-root",
		Tree:     &root,
		Feedback: &fb,
	})
}
```

Add the missing `json` import at the top: `"encoding/json"`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/assets/... -count=1`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/assets/create_handler.go internal/assets/create_handler_test.go
git commit -m "feat(assets): POST /actions/assets/create handler"
```

---

### Task 10: Update handler (PATCH)

**Files:**
- Create: `internal/assets/update_handler.go`
- Create: `internal/assets/update_handler_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/update_handler_test.go`:

```go
package assets

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type updateStub struct {
	*stubMutator
	updated *Asset
	updErr  error
}

func (s *updateStub) UpdateAsset(_ context.Context, _, _ string, _ map[string]any) (*Asset, error) {
	return s.updated, s.updErr
}

func TestUpdateHandler_HappyPath(t *testing.T) {
	sc := &stubMutator{list: &ListResult{Assets: []Asset{}, Total: 0, Size: 10}}
	h := NewUpdateHandler(&updateStub{stubMutator: sc, updated: &Asset{ID: "a1", Ticker: "AAPL"}})
	r := gin.New()
	r.PATCH("/actions/assets/:id", h.Patch)

	body, _ := json.Marshal(map[string]any{"name": "Apple Inc"})
	req := httptest.NewRequest(http.MethodPatch, "/actions/assets/a1?asset_type=&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "assets-root", resp["target_id"])
	fb := resp["feedback"].(map[string]any)
	assert.Equal(t, "Asset updated", fb["props"].(map[string]any)["message"])
}

func TestUpdateHandler_ValidationError_ReplacesModalSlot(t *testing.T) {
	sc := &stubMutator{asset: &Asset{ID: "a1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD"}}
	h := NewUpdateHandler(&updateStub{stubMutator: sc, updErr: &BackendValidationError{Code: "INVALID_PRICE_PROVIDER", Message: "bad provider"}})

	r := gin.New()
	r.PATCH("/actions/assets/:id", h.Patch)

	body, _ := json.Marshal(map[string]any{"name": "x"})
	req := httptest.NewRequest(http.MethodPatch, "/actions/assets/a1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "assets-modal-slot", resp["target_id"])
}
```

Note: `updateStub` wraps `stubMutator` so it can override `UpdateAsset` while inheriting the other methods. `GetAsset` is served by the embedded `stubMutator` via its `asset` field.

- [ ] **Step 2: Add `asset` field + GetAsset to `stubMutator`**

In `internal/assets/create_handler_test.go`, add an `asset` field:

```go
type stubMutator struct {
	created *Asset
	createErr error
	list    *ListResult
	listErr error
	asset   *Asset
	assetErr error
}
```

And update `GetAsset`:

```go
func (s *stubMutator) GetAsset(_ context.Context, _, _ string) (*Asset, error) {
	return s.asset, s.assetErr
}
```

- [ ] **Step 3: Run tests to verify failure**

Run: `go test ./internal/assets/... -run 'TestUpdateHandler' -v`
Expected: FAIL — `NewUpdateHandler` undefined.

- [ ] **Step 4: Implement the handler**

Create `internal/assets/update_handler.go`:

```go
package assets

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type UpdateHandler struct {
	client assetMutator
}

func NewUpdateHandler(client assetMutator) *UpdateHandler {
	return &UpdateHandler{client: client}
}

func (h *UpdateHandler) Patch(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "id is required"}})
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid body"}})
		return
	}
	var body map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid JSON"}})
			return
		}
	} else {
		body = map[string]any{}
	}

	_, err = h.client.UpdateAsset(c.Request.Context(), auth, id, body)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		if errors.Is(err, ErrAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND"}})
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			// Re-fetch asset to repopulate the edit modal.
			a, gerr := h.client.GetAsset(c.Request.Context(), auth, id)
			if gerr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refetch asset"}})
				return
			}
			modal := BuildEditModal(a, params, lang, be.Message)
			c.JSON(http.StatusOK, components.ActionResponse{
				Action:   "replace",
				TargetID: "assets-modal-slot",
				Tree:     &modal,
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not update asset"}})
		return
	}

	respondPostMutation(c, h.client, params, lang, i18n.T(lang, "assets.edit.success"))
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/assets/... -count=1`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/assets/update_handler.go internal/assets/update_handler_test.go internal/assets/create_handler_test.go
git commit -m "feat(assets): PATCH /actions/assets/:id handler"
```

---

### Task 11: Delete handler

**Files:**
- Create: `internal/assets/delete_handler.go`
- Create: `internal/assets/delete_handler_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/assets/delete_handler_test.go`:

```go
package assets

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deleteStub struct {
	*stubMutator
	delErr   error
	delForce *bool
}

func (s *deleteStub) DeleteAsset(_ context.Context, _, _ string, force bool) error {
	s.delForce = &force
	return s.delErr
}

func TestDeleteHandler_HappyPath_NoForce(t *testing.T) {
	sc := &stubMutator{
		list:  &ListResult{Assets: []Asset{}, Total: 0, Size: 10},
		asset: &Asset{ID: "a1", Ticker: "AAPL"},
	}
	h := NewDeleteHandler(&deleteStub{stubMutator: sc})
	r := gin.New()
	r.DELETE("/actions/assets/:id", h.Delete)

	body, _ := json.Marshal(map[string]any{"force": false})
	req := httptest.NewRequest(http.MethodDelete, "/actions/assets/a1?asset_type=&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	fb := resp["feedback"].(map[string]any)
	assert.Equal(t, "Asset deleted", fb["props"].(map[string]any)["message"])
}

func TestDeleteHandler_HappyPath_Force(t *testing.T) {
	sc := &stubMutator{
		list:  &ListResult{Assets: []Asset{}, Total: 0, Size: 10},
		asset: &Asset{ID: "a1", Ticker: "AAPL"},
	}
	h := NewDeleteHandler(&deleteStub{stubMutator: sc})
	r := gin.New()
	r.DELETE("/actions/assets/:id", h.Delete)

	body, _ := json.Marshal(map[string]any{"force": true})
	req := httptest.NewRequest(http.MethodDelete, "/actions/assets/a1?asset_type=&offset=0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	fb := resp["feedback"].(map[string]any)
	assert.Equal(t, "Asset and associated data deleted", fb["props"].(map[string]any)["message"])
}

func TestDeleteHandler_AssetHasData_ReplacesModal(t *testing.T) {
	sc := &stubMutator{asset: &Asset{ID: "a1", Ticker: "AAPL"}}
	stub := &deleteStub{stubMutator: sc, delErr: &BackendValidationError{Code: "ASSET_HAS_DATA", Message: "Has data"}}
	h := NewDeleteHandler(stub)
	r := gin.New()
	r.DELETE("/actions/assets/:id", h.Delete)

	body, _ := json.Marshal(map[string]any{"force": false})
	req := httptest.NewRequest(http.MethodDelete, "/actions/assets/a1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, "assets-modal-slot", resp["target_id"])
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/assets/... -run 'TestDeleteHandler' -v`
Expected: FAIL — `NewDeleteHandler` undefined.

- [ ] **Step 3: Implement the handler**

Create `internal/assets/delete_handler.go`:

```go
package assets

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type DeleteHandler struct {
	client assetMutator
}

func NewDeleteHandler(client assetMutator) *DeleteHandler {
	return &DeleteHandler{client: client}
}

func (h *DeleteHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "id is required"}})
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid body"}})
		return
	}
	var body struct {
		Force bool `json:"force"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid JSON"}})
			return
		}
	}

	err = h.client.DeleteAsset(c.Request.Context(), auth, id, body.Force)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		if errors.Is(err, ErrAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND"}})
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			// Re-fetch asset for modal redisplay.
			a, gerr := h.client.GetAsset(c.Request.Context(), auth, id)
			if gerr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refetch asset"}})
				return
			}
			modal := BuildDeleteModal(a.ID, a.Ticker, params, lang, be.Message)
			c.JSON(http.StatusOK, components.ActionResponse{
				Action:   "replace",
				TargetID: "assets-modal-slot",
				Tree:     &modal,
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not delete asset"}})
		return
	}

	msgKey := "assets.delete.success"
	if body.Force {
		msgKey = "assets.delete.success_force"
	}
	respondPostMutation(c, h.client, params, lang, i18n.T(lang, msgKey))
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/assets/... -count=1`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/assets/delete_handler.go internal/assets/delete_handler_test.go
git commit -m "feat(assets): DELETE /actions/assets/:id handler (with force)"
```

---

### Task 12: Route registration

**Files:**
- Modify: `internal/server/server.go`

- [ ] **Step 1: Add the new routes**

In `internal/server/server.go`, in `setupRoutes`, after the existing assets registration, append:

```go
	protected.GET("/actions/assets/create_modal", assets.NewCreateModalHandler().Get)
	protected.GET("/actions/assets/edit_modal", assets.NewEditModalHandler(assetsClient).Get)
	protected.GET("/actions/assets/delete_modal", assets.NewDeleteModalHandler(assetsClient).Get)
	protected.POST("/actions/assets/create", assets.NewCreateHandler(assetsClient).Post)
	protected.PATCH("/actions/assets/:id", assets.NewUpdateHandler(assetsClient).Patch)
	protected.DELETE("/actions/assets/:id", assets.NewDeleteHandler(assetsClient).Delete)
```

- [ ] **Step 2: Build + test**

Run: `go build ./... && go test ./... -count=1`
Expected: clean + all green.

- [ ] **Step 3: Commit**

```bash
git add internal/server/server.go
git commit -m "feat(server): register assets L2 routes (create/edit/delete modals + mutations)"
```

---

### Task 13: Canonical spec

**Files:**
- Create: `spec/screens/assets/02-mutations.md`
- Modify: `spec/spec.md`

- [ ] **Step 1: Write the canonical spec**

Create `spec/screens/assets/02-mutations.md` as the L2 contract. Copy the content from `docs/superpowers/specs/2026-04-17-assets-layer2-design.md` and adjust as follows:

- Change the top heading to `# Assets — Layer 2: Mutations`.
- Remove the first line ("Second and final layer..."), replace with:

```
Second and final layer of the assets screen. Adds create, edit, and delete operations via modal dialogs. Read-only listing (Layer 1) ships in `01-list.md`.
```

- Mark all acceptance criteria with `[x]` (they are all met after Task 12).
- Change any cross-references from `docs/superpowers/specs/...` to `spec/sdui-actions.md` / `spec/sdui-base-components.md`.

- [ ] **Step 2: Update `spec/spec.md` index**

The existing row:

```
| Assets | [`screens/assets/`](screens/assets/) — decomposed into 2 layers; layer 1 live in [`01-list.md`](screens/assets/01-list.md) |
```

Change to:

```
| Assets | [`screens/assets/`](screens/assets/) — decomposed into 2 layers; both live: [`01-list.md`](screens/assets/01-list.md), [`02-mutations.md`](screens/assets/02-mutations.md) |
```

- [ ] **Step 3: Commit**

```bash
git add spec/screens/assets/02-mutations.md spec/spec.md
git commit -m "docs(spec): canonical assets L2 spec + update index"
```

---

### Task 14: Smoke test

**Files:** (verification only — no commits)

- [ ] **Step 1: Build**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && make build`
Expected: `bin/server` built cleanly.

- [ ] **Step 2: Restart server**

Kill existing on :8082, start fresh:

```bash
lsof -i :8082 -sTCP:LISTEN -nP 2>/dev/null | awk 'NR>1 {print $2}' | xargs -r kill -9
sleep 1
./cli run > /tmp/me.log 2>&1 &
sleep 2
curl -s http://localhost:8082/health
```

Expected: healthy response.

- [ ] **Step 3: Unauth checks**

```bash
for path in /actions/assets/create_modal /actions/assets/edit_modal?id=x /actions/assets/delete_modal?id=x; do
  curl -s -o /dev/null -w "$path → %{http_code}\n" "http://localhost:8082$path"
done
```

Expected: all three return `401`.

```bash
curl -s -o /dev/null -w "POST /actions/assets/create → %{http_code}\n" -X POST http://localhost:8082/actions/assets/create
curl -s -o /dev/null -w "PATCH /actions/assets/x → %{http_code}\n"  -X PATCH http://localhost:8082/actions/assets/x
curl -s -o /dev/null -w "DELETE /actions/assets/x → %{http_code}\n" -X DELETE http://localhost:8082/actions/assets/x
```

Expected: all three return `401`.

- [ ] **Step 4: Full test suite + lint**

```bash
go test ./... -count=1
make lint
```

Expected: all green.

- [ ] **Step 5: Leave the server running**

Per project convention, do not kill the server. Verify it is still listening:

```bash
lsof -i :8082 -sTCP:LISTEN -nP 2>/dev/null | tail -1
```

No commit — this task is verification only.

---

## Self-Review Notes

**Spec coverage:**
- [x] `visible_when` type + docs → Task 1.
- [x] `input.pattern` / `input.auto_uppercase` docs → Task 1.
- [x] Screen tree changes (new button, actions column, modal slot) → Task 4.
- [x] Six new endpoints → Tasks 6–11 + Task 12 for registration.
- [x] Create modal tree → Task 5.
- [x] Edit modal tree (immutable as static text) → Task 5.
- [x] Delete modal with force checkbox → Task 5.
- [x] Response pattern (replace on assets-root + feedback snackbar) → Tasks 9–11 (`respondPostMutation`).
- [x] BE error → modal replace with inline error → Tasks 9–11.
- [x] i18n keys → Task 2.
- [x] Filter/offset preservation via mutation query params → all mutation handlers use `parseListParams` and `respondPostMutation` passes them to List.
- [x] Canonical spec → Task 13.

**Placeholder scan:** none found. All code blocks are concrete. Acceptance criteria are verifiable.

**Type consistency:** `assetMutator`, `assetByIDFetcher`, `VisibleWhen` / `VisibleWhenValue` alias, `Asset` (with new `ExternalTicker` field), `stubMutator` / `stubAssetFetcher` / `updateStub` / `deleteStub` — all defined where first referenced.
