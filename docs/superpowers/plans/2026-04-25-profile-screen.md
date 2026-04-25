# Profile Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the SDUI Profile screen described in `spec/screens/profile.md` — display name + default currency, change email, change password, delete account.

**Architecture:** A new `internal/profile/` package mirroring the structure of `internal/snapshots/` — typed BE client, builder, screen GET use case, one handler per action, JSON-body mutations. Routes are added under `protected.Group("")` in `internal/server/server.go`. A new `LogoutResponse` helper extends `components/actions.go` with the `logout` ActionResponse variant the delete-account flow needs. Currencies for the default-currency select are fetched via a new local `configClient` (calls `GET /v1/config`), kept inside the `profile` package — promote to `internal/shared/` only when a second screen needs it.

**Tech Stack:** Go, Gin, the project's existing `components`, `i18n`, and `shared` packages. Tests use `testify` + `httptest` (consistent with the existing handler tests).

---

## File Structure

**Created (all paths relative to repo root):**

| File | Responsibility |
|---|---|
| `internal/profile/types.go` | `User`, `Preferences`, `BackendValidationError`, `errCodeMissingFields` etc. constants |
| `internal/profile/errors.go` | Sentinel errors (`ErrUnauthorized`, `ErrBackend`) and the validation-error parser |
| `internal/profile/client.go` | BE HTTP client. Methods: `GetMe`, `UpdateProfile`, `UpdateEmail`, `ChangePassword`, `DeleteAccount` |
| `internal/profile/config_client.go` | `GET /v1/config` client (returns currencies). Local to the package for now. |
| `internal/profile/get_usecase.go` | Orchestrates `GetMe` + `GetConfig` and calls `BuildScreen` |
| `internal/profile/builder.go` | Top-level `BuildScreen(me, currencies, lang)` plus the four card builders (`buildProfileCard`, `buildEmailCard`, `buildPasswordCard`, `buildDangerCard`) and shared IDs |
| `internal/profile/modal_builder.go` | `BuildDeleteModal(lang, errMessage string)` |
| `internal/profile/handler.go` | Screen GET handler |
| `internal/profile/update_handler.go` | `POST /actions/profile/update` |
| `internal/profile/update_email_handler.go` | `POST /actions/profile/update_email` |
| `internal/profile/change_password_handler.go` | `POST /actions/profile/change_password` (with middleend-side validation) |
| `internal/profile/delete_modal_handler.go` | `GET /actions/profile/delete_modal` |
| `internal/profile/delete_handler.go` | `POST /actions/profile/delete_account` |
| `internal/profile/parsing.go` | `parseLang`, `parseJSONBody`, `respondBadRequest` (small wrappers — keep them package-local mirroring snapshots) |
| `internal/profile/<each>_test.go` | One test file per source file with stubs and table-driven tests |

**Modified:**

| File | Change |
|---|---|
| `internal/components/actions.go` | Add `LogoutResponse(redirectURL string) ActionResponse` returning `{Action: "logout", TargetID: redirectURL}` |
| `internal/components/actions_test.go` | Add a unit test for `LogoutResponse` |
| `internal/server/server.go` | Wire the six routes (1 GET screen + 5 actions). Also wire the `configClient`. |
| `locales/en.json`, `locales/es.json` | Add the `profile.*` namespace per spec |

---

## Conventions used throughout the plan

- All mutation endpoints accept JSON. Bad JSON → `400 BAD_REQUEST` via the same inline pattern snapshots uses.
- `BackendValidationError{Code, Message}` is the in-package validation-error type. Translation BE-code → i18n key happens in each handler (per spec mapping).
- Every handler that re-emits a card on validation error returns `200 ActionResponse{Action: "replace", TargetID: <card-id>, Tree: <subtree>, Feedback: nil}`. Banners are mounted as the **first child** of the card subtree (a `Text` with `color: error`).
- Every handler that returns success emits `Feedback: &Snackbar(...)` with `variant: success`.
- Tests use stubs that implement narrow interfaces (`type meFetcher interface { GetMe(...) ... }`) — same style as `internal/snapshots/handler_test.go`.
- After every task: run `make test` from the repo root. Then commit. Conventional Commits, no Claude Code trailer.
- After tasks that touch live behavior (handlers, routes, i18n): restart the dev server with `./cli run` (kill existing on `:8082`).

---

## Task 1: Add `LogoutResponse` to `components/actions.go`

**Files:**
- Modify: `internal/components/actions.go`
- Modify: `internal/components/actions_test.go`

- [ ] **Step 1: Read current actions.go to find the right insertion point**

Run: `grep -n '^func ' internal/components/actions.go`

Expected: list shows `Logout`, `Navigate`, `ReplaceResponse`, `NavigateResponse`, `RefreshResponse`, `ErrorResponse`. Insert `LogoutResponse` after `NavigateResponse`.

- [ ] **Step 2: Write the failing test**

Append to `internal/components/actions_test.go`:

```go
func TestLogoutResponse(t *testing.T) {
	resp := LogoutResponse("/screens/login")
	assert.Equal(t, "logout", resp.Action)
	assert.Equal(t, "/screens/login", resp.TargetID)
	assert.Nil(t, resp.Tree)
	assert.Nil(t, resp.Feedback)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/components/ -run TestLogoutResponse -v`
Expected: FAIL — `undefined: LogoutResponse`.

- [ ] **Step 4: Implement**

Add to `internal/components/actions.go` (after `NavigateResponse`):

```go
// LogoutResponse creates an action response that clears the auth token on the
// client and navigates to redirectURL. Used by destructive flows like
// delete-account where the session must end alongside the navigation.
func LogoutResponse(redirectURL string) ActionResponse {
	return ActionResponse{Action: "logout", TargetID: redirectURL}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/components/ -run TestLogoutResponse -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/components/actions.go internal/components/actions_test.go
git commit -m "feat(components): add LogoutResponse for destructive flows"
```

---

## Task 2: Profile package — types and BE client errors

**Files:**
- Create: `internal/profile/types.go`
- Create: `internal/profile/errors.go`
- Create: `internal/profile/types_test.go`

- [ ] **Step 1: Write `types.go`**

```go
package profile

// User mirrors GET /v1/user/me.
type User struct {
	ID          string      `json:"id"`
	Email       string      `json:"email"`
	DisplayName *string     `json:"display_name,omitempty"`
	Preferences Preferences `json:"preferences"`
	CreatedAt   string      `json:"created_at"`
}

// Preferences carries user preferences from /v1/user/me.
type Preferences struct {
	DefaultCurrency *string `json:"default_currency,omitempty"`
}

// AppConfig is the relevant slice of GET /v1/config we use here.
type AppConfig struct {
	Currencies []string `json:"currencies"`
}
```

- [ ] **Step 2: Write `errors.go`**

```go
package profile

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Sentinel errors emitted by the profile clients.
var (
	ErrUnauthorized = errors.New("backend unauthorized")
	ErrBackend      = errors.New("backend error")
)

// BackendValidationError is returned for the 4xx codes the BE documents:
// INVALID_DISPLAY_NAME, INVALID_CURRENCY, MISSING_FIELDS, INVALID_CREDENTIALS,
// EMAIL_ALREADY_EXISTS, INVALID_PASSWORD.
type BackendValidationError struct {
	Code    string
	Message string
}

func (e *BackendValidationError) Error() string {
	return fmt.Sprintf("backend validation: %s: %s", e.Code, e.Message)
}

// parseValidationError pulls {"error":{"code":"...","message":"..."}} out of a
// 4xx body. If the body is unrecognised, it returns a generic error wrapped
// around ErrBackend so the caller still maps to 502.
func parseValidationError(body []byte) error {
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.Error.Code == "" {
		return fmt.Errorf("%w: malformed validation error", ErrBackend)
	}
	return &BackendValidationError{Code: env.Error.Code, Message: env.Error.Message}
}
```

- [ ] **Step 3: Write `types_test.go` — coverage for `parseValidationError`**

```go
package profile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValidationError_KnownCode(t *testing.T) {
	body := []byte(`{"error":{"code":"INVALID_DISPLAY_NAME","message":"too long"}}`)
	err := parseValidationError(body)
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "INVALID_DISPLAY_NAME", be.Code)
	assert.Equal(t, "too long", be.Message)
}

func TestParseValidationError_Malformed(t *testing.T) {
	err := parseValidationError([]byte(`not json`))
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestParseValidationError_NoCode(t *testing.T) {
	err := parseValidationError([]byte(`{"error":{}}`))
	assert.True(t, errors.Is(err, ErrBackend))
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/profile/ -v`
Expected: PASS for all three.

- [ ] **Step 5: Commit**

```bash
git add internal/profile/types.go internal/profile/errors.go internal/profile/types_test.go
git commit -m "feat(profile): add types and validation-error parser"
```

---

## Task 3: BE client — `GetMe`, `UpdateProfile`, `UpdateEmail`, `ChangePassword`, `DeleteAccount`

**Files:**
- Create: `internal/profile/client.go`
- Create: `internal/profile/client_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/profile/client_test.go`:

```go
package profile

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(handler http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	return NewClient(srv.URL, 2*time.Second), srv
}

func TestClient_GetMe_Happy(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/me", r.URL.Path)
		assert.Equal(t, "Bearer t", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"u1","email":"a@b.c","display_name":"Vadim","preferences":{"default_currency":"USD"},"created_at":"2026-01-01T00:00:00Z"}`))
	})
	defer srv.Close()

	me, err := c.GetMe(context.Background(), "Bearer t")
	require.NoError(t, err)
	assert.Equal(t, "u1", me.ID)
	require.NotNil(t, me.DisplayName)
	assert.Equal(t, "Vadim", *me.DisplayName)
	require.NotNil(t, me.Preferences.DefaultCurrency)
	assert.Equal(t, "USD", *me.Preferences.DefaultCurrency)
}

func TestClient_GetMe_Unauthorized(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	_, err := c.GetMe(context.Background(), "Bearer t")
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_UpdateProfile_ForwardsBodyAndAuth(t *testing.T) {
	var got map[string]any
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/v1/user/me", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		decodeJSON(t, r, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"u1","email":"a@b.c","preferences":{}}`))
	})
	defer srv.Close()

	body := map[string]any{"display_name": "Vadim", "preferences": map[string]any{"default_currency": "USD"}}
	_, err := c.UpdateProfile(context.Background(), "Bearer t", body)
	require.NoError(t, err)
	assert.Equal(t, "Vadim", got["display_name"])
}

func TestClient_UpdateProfile_ValidationError(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"INVALID_DISPLAY_NAME","message":"too long"}}`))
	})
	defer srv.Close()

	_, err := c.UpdateProfile(context.Background(), "Bearer t", map[string]any{})
	var be *BackendValidationError
	require.True(t, errors.As(err, &be))
	assert.Equal(t, "INVALID_DISPLAY_NAME", be.Code)
}

func TestClient_UpdateEmail_Happy(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/me/email", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"u1","email":"new@b.c"}`))
	})
	defer srv.Close()

	err := c.UpdateEmail(context.Background(), "Bearer t", "new@b.c", "pw")
	require.NoError(t, err)
}

func TestClient_ChangePassword_Returns204(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/me/password", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	err := c.ChangePassword(context.Background(), "Bearer t", "old", "new12345")
	require.NoError(t, err)
}

func TestClient_DeleteAccount_Returns204(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/me", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	err := c.DeleteAccount(context.Background(), "Bearer t", "pw")
	require.NoError(t, err)
}
```

Add helper at the bottom of the test file (used above):

```go
import "encoding/json"

func decodeJSON(t *testing.T, r *http.Request, v any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(r.Body).Decode(v))
}
```

(Place the helper after the test functions; merge the `import` block at the top — Go requires single grouped imports.)

- [ ] **Step 2: Run tests to confirm failure**

Run: `go test ./internal/profile/ -run TestClient -v`
Expected: build error — `undefined: NewClient`, `undefined: Client`, etc.

- [ ] **Step 3: Implement `client.go`**

```go
package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// GetMe → GET /v1/user/me.
func (c *Client) GetMe(ctx context.Context, authorization string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/user/me", nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var me User
		if err := json.Unmarshal(body, &me); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &me, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// UpdateProfile → PATCH /v1/user/me.
func (c *Client) UpdateProfile(ctx context.Context, authorization string, body map[string]any) (*User, error) {
	raw, err := c.doJSON(ctx, http.MethodPatch, "/v1/user/me", authorization, body, http.StatusOK)
	if err != nil {
		return nil, err
	}
	var me User
	if err := json.Unmarshal(raw, &me); err != nil {
		return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
	}
	return &me, nil
}

// UpdateEmail → PATCH /v1/user/me/email.
func (c *Client) UpdateEmail(ctx context.Context, authorization, newEmail, currentPassword string) error {
	body := map[string]any{"new_email": newEmail, "current_password": currentPassword}
	_, err := c.doJSON(ctx, http.MethodPatch, "/v1/user/me/email", authorization, body, http.StatusOK)
	return err
}

// ChangePassword → POST /v1/user/me/password.
func (c *Client) ChangePassword(ctx context.Context, authorization, currentPassword, newPassword string) error {
	body := map[string]any{"current_password": currentPassword, "new_password": newPassword}
	_, err := c.doJSON(ctx, http.MethodPost, "/v1/user/me/password", authorization, body, http.StatusNoContent)
	return err
}

// DeleteAccount → DELETE /v1/user/me.
func (c *Client) DeleteAccount(ctx context.Context, authorization, password string) error {
	body := map[string]any{"password": password}
	_, err := c.doJSON(ctx, http.MethodDelete, "/v1/user/me", authorization, body, http.StatusNoContent)
	return err
}

// doJSON handles the request/response envelope for every JSON mutation. It
// returns the raw body on success (so callers can parse a typed result if they
// need one), maps 401 → ErrUnauthorized, 4xx with a known envelope →
// BackendValidationError, and everything else → ErrBackend.
func (c *Client) doJSON(ctx context.Context, method, path, authorization string, body map[string]any, successStatus int) ([]byte, error) {
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
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case successStatus:
		return raw, nil
	case http.StatusUnauthorized, http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusConflict:
		// All four are validation outcomes per the BE spec for these endpoints.
		// 401 from these mutation endpoints is INVALID_CREDENTIALS, not session
		// expiry — translate it the same way as 4xx.
		return nil, parseValidationError(raw)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

**Note:** This intentionally diverges from snapshots' `doSnapshotWithBody` (where `401` returns `ErrUnauthorized`). Per the spec's "401-from-BE rule", every BE-401 on these mutation endpoints is `INVALID_CREDENTIALS` and must surface as a `BackendValidationError`. `GetMe` is the **only** profile call that maps `401 → ErrUnauthorized`.

Update `TestClient_GetMe_Unauthorized` is already correct (it tests `GetMe` → `ErrUnauthorized`).

Add a test that proves the divergence for mutations:

```go
func TestClient_UpdateEmail_BE401IsValidationError(t *testing.T) {
	c, srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"INVALID_CREDENTIALS","message":"wrong"}}`))
	})
	defer srv.Close()

	err := c.UpdateEmail(context.Background(), "Bearer t", "n@x.y", "pw")
	var be *BackendValidationError
	require.True(t, errors.As(err, &be), "got %v", err)
	assert.Equal(t, "INVALID_CREDENTIALS", be.Code)
}
```

- [ ] **Step 4: Run tests to confirm pass**

Run: `go test ./internal/profile/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/profile/client.go internal/profile/client_test.go
git commit -m "feat(profile): add backend client (me, profile, email, password, delete)"
```

---

## Task 4: `configClient` for `/v1/config`

**Files:**
- Create: `internal/profile/config_client.go`
- Create: `internal/profile/config_client_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package profile

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigClient_Get_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/config", r.URL.Path)
		assert.Equal(t, "Bearer t", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"asset_types":[],"currencies":["USD","EUR","ARS"],"price_providers":[],"sources":[]}`))
	}))
	defer srv.Close()

	c := NewConfigClient(srv.URL, 2*time.Second)
	cfg, err := c.GetConfig(context.Background(), "Bearer t")
	require.NoError(t, err)
	assert.Equal(t, []string{"USD", "EUR", "ARS"}, cfg.Currencies)
}

func TestConfigClient_Get_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewConfigClient(srv.URL, 2*time.Second)
	_, err := c.GetConfig(context.Background(), "Bearer t")
	assert.True(t, errors.Is(err, ErrUnauthorized))
}
```

- [ ] **Step 2: Confirm failure**

Run: `go test ./internal/profile/ -run TestConfigClient -v`
Expected: undefined `NewConfigClient` / `GetConfig`.

- [ ] **Step 3: Implement**

`internal/profile/config_client.go`:

```go
package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ConfigClient calls GET /v1/config. Local to the profile package today; lift to
// internal/shared/configcatalog if a second screen needs it.
type ConfigClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewConfigClient(baseURL string, timeout time.Duration) *ConfigClient {
	return &ConfigClient{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

func (c *ConfigClient) GetConfig(ctx context.Context, authorization string) (*AppConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/config", nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var cfg AppConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &cfg, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/profile/ -run TestConfigClient -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/profile/config_client.go internal/profile/config_client_test.go
git commit -m "feat(profile): add /v1/config client for currency options"
```

---

## Task 5: Builders — full screen and the four cards

**Files:**
- Create: `internal/profile/builder.go`
- Create: `internal/profile/builder_test.go`

The IDs and i18n keys all match `spec/screens/profile.md`. Banner is mounted as the first child of a card when an error string is non-empty.

- [ ] **Step 1: Write `builder.go` (full file)**

```go
package profile

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// Stable ids referenced by the screen tree and partial-update endpoints.
const (
	ScreenID         = "profile"
	ProfileCardID    = "profile-card"
	EmailCardID      = "email-card"
	PasswordCardID   = "password-card"
	DangerCardID     = "danger-card"
	ModalSlotID      = "profile-modal-slot"
	DeleteModalID    = "profile-delete-modal"
)

// BuildScreen assembles the full profile screen tree.
func BuildScreen(me *User, cfg *AppConfig, lang string) components.Component {
	col := components.Column("profile-column",
		BuildProfileCard(me, cfg, lang, ""),
		BuildEmailCard(me, lang, "", ""),
		BuildPasswordCard(lang, ""),
		BuildDangerCard(lang),
		components.Group(ModalSlotID),
	)
	return components.Screen(ScreenID, i18n.T(lang, "profile.title"), col)
}

// BuildProfileCard renders the Profile section. errMessage is non-empty when
// re-emitting after a validation error; it shows as the first child banner.
// If preserved values are passed via *Form, defaults are taken from the User.
func BuildProfileCard(me *User, cfg *AppConfig, lang, errMessage string) components.Component {
	return buildProfileCardWith(
		strDeref(me.DisplayName),
		strDeref(me.Preferences.DefaultCurrency),
		cfg, lang, errMessage,
	)
}

// buildProfileCardWith allows the update handler to re-emit with preserved
// (possibly invalid) inputs.
func buildProfileCardWith(displayName, currency string, cfg *AppConfig, lang, errMessage string) components.Component {
	children := []components.Component{}
	if errMessage != "" {
		children = append(children, components.TextStyled("profile-card-error", errMessage, "sm", "regular", "block", "error", "", ""))
	}
	currencyOptions := []components.SelectOption{{Value: "", Label: i18n.T(lang, "profile.default_currency.none")}}
	for _, code := range cfg.Currencies {
		currencyOptions = append(currencyOptions, components.SelectOption{Value: code, Label: code})
	}
	form := components.Form("profile-form",
		components.InputFull("input-display-name", "display_name", "text",
			i18n.T(lang, "profile.display_name"),
			i18n.T(lang, "profile.display_name.placeholder"),
			displayName, false, false, 100),
		components.SelectFull("input-default-currency", "default_currency",
			i18n.T(lang, "profile.default_currency"), "", currency,
			currencyOptions, false, false),
		buildSubmitButton("profile-save", i18n.T(lang, "profile.update.save"),
			"/actions/profile/update"),
	)
	children = append(children,
		components.Text("profile-section-title", i18n.T(lang, "profile.section.profile"), "lg", "bold"),
		form,
	)
	return components.Card(ProfileCardID, children...)
}

func BuildEmailCard(me *User, lang, newEmail, errMessage string) components.Component {
	return buildEmailCardWith(me.Email, newEmail, lang, errMessage)
}

func buildEmailCardWith(currentEmail, newEmail, lang, errMessage string) components.Component {
	children := []components.Component{
		components.Text("email-section-title", i18n.T(lang, "profile.section.email"), "lg", "bold"),
		components.TextStyled("email-current", interpolate(i18n.T(lang, "profile.email.current"), map[string]string{"email": currentEmail}), "sm", "regular", "block", "muted", "", ""),
	}
	if errMessage != "" {
		children = append(children, components.TextStyled("email-card-error", errMessage, "sm", "regular", "block", "error", "", ""))
	}
	children = append(children, components.Form("email-form",
		components.InputFull("input-new-email", "new_email", "email",
			i18n.T(lang, "profile.email.new"), "", newEmail, true, false, 0),
		components.InputFull("input-current-password", "current_password", "password",
			i18n.T(lang, "profile.email.current_password"), "", "", true, false, 0),
		buildSubmitButton("email-save", i18n.T(lang, "profile.email.save"),
			"/actions/profile/update_email"),
	))
	return components.Card(EmailCardID, children...)
}

func BuildPasswordCard(lang, errMessage string) components.Component {
	children := []components.Component{
		components.Text("password-section-title", i18n.T(lang, "profile.section.password"), "lg", "bold"),
	}
	if errMessage != "" {
		children = append(children, components.TextStyled("password-card-error", errMessage, "sm", "regular", "block", "error", "", ""))
	}
	children = append(children, components.Form("password-form",
		components.InputFull("input-current-password", "current_password", "password",
			i18n.T(lang, "profile.password.current"), "", "", true, false, 0),
		components.InputFull("input-new-password", "new_password", "password",
			i18n.T(lang, "profile.password.new"), "", "", true, false, 128),
		components.InputFull("input-confirm-password", "confirm_password", "password",
			i18n.T(lang, "profile.password.confirm"), "", "", true, false, 128),
		buildSubmitButton("password-save", i18n.T(lang, "profile.password.save"),
			"/actions/profile/change_password"),
	))
	return components.Card(PasswordCardID, children...)
}

func BuildDangerCard(lang string) components.Component {
	return components.Card(DangerCardID,
		components.TextStyled("danger-title", i18n.T(lang, "profile.danger.title"), "lg", "bold", "block", "error", "", ""),
		components.Text("danger-body", i18n.T(lang, "profile.danger.body"), "sm", "regular"),
		components.ButtonFull("danger-delete-btn",
			i18n.T(lang, "profile.danger.delete_button"),
			"", "destructive", "solid",
			components.Action{Trigger: "click", Type: "fetch", Method: "GET",
				Endpoint: "/actions/profile/delete_modal", TargetID: ModalSlotID},
		),
	)
}

func buildSubmitButton(id, label, endpoint string) components.Component {
	return components.ButtonFull(id, label, "", "primary", "solid",
		components.Action{Trigger: "click", Type: "submit", Method: "POST", Endpoint: endpoint, TargetID: ""},
	)
}

func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// interpolate replaces `{key}` tokens in a template. Tiny helper — shared with
// other screens via i18n if needed later.
func interpolate(tmpl string, vars map[string]string) string {
	out := tmpl
	for k, v := range vars {
		out = replace(out, "{"+k+"}", v)
	}
	return out
}

func replace(s, old, new string) string {
	// strings.ReplaceAll without importing strings just to make the helper local.
	// Acceptable because we already pull strings elsewhere; keep this minimal:
	for i := 0; ; {
		j := indexOf(s[i:], old)
		if j < 0 {
			return s
		}
		s = s[:i+j] + new + s[i+j+len(old):]
		i += j + len(new)
	}
}

func indexOf(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

**Note about `Action`**: Look at how snapshots' submit buttons emit their action — copy the exact field set the frontend expects (`Trigger`, `Type`, `Method`, `Endpoint`, `TargetID`). If the snapshots wizard uses a different field combination, mirror it.

**Inspection step before continuing:** open `internal/snapshots/builder.go` and find the function that builds the `[ Save ]` button on the create wizard. Copy its exact `Action{}` literal style into `buildSubmitButton` here. Update if the fields differ.

Run: `grep -n 'Action{' internal/snapshots/wizard_builder.go internal/snapshots/builder.go`

Expected: a small set of literals — pick the matching one (form submit) and align.

**Replace the simplified `replace`/`indexOf` helpers above with `strings.ReplaceAll`:**

```go
import "strings"

func interpolate(tmpl string, vars map[string]string) string {
	for k, v := range vars {
		tmpl = strings.ReplaceAll(tmpl, "{"+k+"}", v)
	}
	return tmpl
}
```

(Drop the `replace`/`indexOf` helpers; use `strings.ReplaceAll` in the import block.)

- [ ] **Step 2: Write tests**

`internal/profile/builder_test.go`:

```go
package profile

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr(s string) *string { return &s }

func sampleUser() *User {
	return &User{
		ID:          "u1",
		Email:       "vadim@example.com",
		DisplayName: ptr("Vadim"),
		Preferences: Preferences{DefaultCurrency: ptr("USD")},
		CreatedAt:   "2026-01-01T00:00:00Z",
	}
}

func sampleConfig() *AppConfig {
	return &AppConfig{Currencies: []string{"USD", "EUR", "ARS"}}
}

// asJSON marshals a Component to a generic map for tree assertions.
func asJSON(t *testing.T, c any) map[string]any {
	t.Helper()
	b, err := json.Marshal(c)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func TestBuildScreen_RootShape(t *testing.T) {
	tree := BuildScreen(sampleUser(), sampleConfig(), "en")
	m := asJSON(t, tree)
	assert.Equal(t, "screen", m["type"])
	assert.Equal(t, ScreenID, m["id"])
	props := m["props"].(map[string]any)
	assert.Equal(t, "Profile", props["title"]) // proves i18n is wired (en locale)
	// Has a column with the four cards + modal slot.
	body, _ := json.Marshal(tree)
	bodyStr := string(body)
	for _, id := range []string{ProfileCardID, EmailCardID, PasswordCardID, DangerCardID, ModalSlotID} {
		assert.Contains(t, bodyStr, id)
	}
}

func TestBuildProfileCard_DefaultsFromUser(t *testing.T) {
	c := BuildProfileCard(sampleUser(), sampleConfig(), "en", "")
	body, _ := json.Marshal(c)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, `"defaultValue":"Vadim"`)
	assert.Contains(t, bodyStr, `"defaultValue":"USD"`)
	assert.Contains(t, bodyStr, `"value":""`)        // — None — option present
	assert.NotContains(t, bodyStr, "profile-card-error")
}

func TestBuildProfileCard_WithError(t *testing.T) {
	c := BuildProfileCard(sampleUser(), sampleConfig(), "en", "Display name must be between 1 and 100 characters")
	bodyStr, _ := json.Marshal(c)
	assert.Contains(t, string(bodyStr), "profile-card-error")
	assert.Contains(t, string(bodyStr), "must be between 1 and 100")
}

func TestBuildEmailCard_CurrentEmailInterpolated(t *testing.T) {
	c := BuildEmailCard(sampleUser(), "en", "", "")
	bodyStr, _ := json.Marshal(c)
	// The "Current: {email}" template (en) should now contain the actual email.
	assert.Contains(t, string(bodyStr), "vadim@example.com")
	assert.False(t, strings.Contains(string(bodyStr), "{email}"), "interpolation token leaked")
}

func TestBuildEmailCard_PreservesNewEmail(t *testing.T) {
	c := BuildEmailCard(sampleUser(), "en", "preserved@x.y", "wrong password")
	bodyStr, _ := json.Marshal(c)
	assert.Contains(t, string(bodyStr), `"defaultValue":"preserved@x.y"`)
	assert.Contains(t, string(bodyStr), "email-card-error")
}

func TestBuildPasswordCard_AlwaysEmpty(t *testing.T) {
	c := BuildPasswordCard("en", "")
	bodyStr, _ := json.Marshal(c)
	// All three password inputs must have empty defaultValue.
	count := strings.Count(string(bodyStr), `"defaultValue":""`)
	assert.GreaterOrEqual(t, count, 3)
}

func TestBuildDangerCard_HasDestructiveButton(t *testing.T) {
	c := BuildDangerCard("en")
	bodyStr, _ := json.Marshal(c)
	assert.Contains(t, string(bodyStr), `"variant":"destructive"`)
	assert.Contains(t, string(bodyStr), "/actions/profile/delete_modal")
}
```

**Note on i18n in tests:** the tests assume `locales/en.json` already has the `profile.*` keys. Task 11 wires those. Until Task 11 lands, these tests will fail with the **key** rendering (e.g. `"profile.title"`) instead of `"Profile"`. To unblock this task, **either** (a) land Task 11 first, or (b) write the assertions against the keys (`"profile.title"`), then tighten in Task 11 once locales exist.

**Pick (a):** reorder Task 11 (i18n locales) before this task. Adjust the plan as you go — when you start this task, run Task 11 first.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/profile/ -run TestBuild -v`
Expected: PASS (after i18n keys exist).

- [ ] **Step 4: Commit**

```bash
git add internal/profile/builder.go internal/profile/builder_test.go
git commit -m "feat(profile): add screen and card builders"
```

---

## Task 6: Modal builder

**Files:**
- Create: `internal/profile/modal_builder.go`
- Create: `internal/profile/modal_builder_test.go`

- [ ] **Step 1: Write tests**

```go
package profile

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDeleteModal_HasPasswordInput(t *testing.T) {
	m := BuildDeleteModal("en", "")
	b, err := json.Marshal(m)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, DeleteModalID)
	assert.Contains(t, s, `"name":"password"`)
	assert.Contains(t, s, `"inputType":"password"`)
	assert.Contains(t, s, "/actions/profile/delete_account")
}

func TestBuildDeleteModal_WithError_BannerInside(t *testing.T) {
	m := BuildDeleteModal("en", "Incorrect password")
	b, _ := json.Marshal(m)
	s := string(b)
	assert.Contains(t, s, "Incorrect password")
}

func TestBuildDeleteModal_PasswordAlwaysEmpty(t *testing.T) {
	m := BuildDeleteModal("en", "")
	b, _ := json.Marshal(m)
	assert.Contains(t, string(b), `"defaultValue":""`)
}
```

- [ ] **Step 2: Implement**

```go
package profile

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildDeleteModal renders the delete-account confirmation modal. errMessage is
// non-empty when re-rendering after a validation error.
func BuildDeleteModal(lang, errMessage string) components.Component {
	body := []components.Component{
		components.Text("delete-modal-body", i18n.T(lang, "profile.danger.modal.body"), "sm", "regular"),
	}
	if errMessage != "" {
		body = append(body, components.TextStyled("delete-modal-error", errMessage, "sm", "regular", "block", "error", "", ""))
	}
	form := components.Form("delete-form",
		components.InputFull("input-delete-password", "password", "password",
			i18n.T(lang, "profile.danger.modal.password_label"), "", "", true, false, 0),
		components.ButtonFull("delete-cancel-btn",
			i18n.T(lang, "profile.danger.modal.cancel"),
			"", "secondary", "ghost",
			components.Action{Trigger: "click", Type: "dismiss_modal", TargetID: ModalSlotID},
		),
		components.ButtonFull("delete-confirm-btn",
			i18n.T(lang, "profile.danger.modal.confirm"),
			"", "destructive", "solid",
			components.Action{Trigger: "click", Type: "submit", Method: "POST",
				Endpoint: "/actions/profile/delete_account", TargetID: ""},
		),
	)
	body = append(body, form)
	return components.ModalFull(DeleteModalID,
		i18n.T(lang, "profile.danger.modal.title"),
		"dialog", true, true, body...)
}
```

**Cancel-button note:** the spec says "closes the modal (consistent with `dismissible: true`; the implementation may dismiss client-side or emit an empty `replace` of `profile-modal-slot`)". The `dismiss_modal` action type used here is the most lightweight option. If the existing frontend doesn't support `dismiss_modal` yet, swap to a fetch action that returns an empty replace of `ModalSlotID`. Decide this when wiring routes — verify via a test request or by checking the frontend's action handler list.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/profile/ -run TestBuildDeleteModal -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/profile/modal_builder.go internal/profile/modal_builder_test.go
git commit -m "feat(profile): add delete-account modal builder"
```

---

## Task 7: Screen GET — use case + handler + parsing helpers

**Files:**
- Create: `internal/profile/parsing.go`
- Create: `internal/profile/get_usecase.go`
- Create: `internal/profile/handler.go`
- Create: `internal/profile/get_usecase_test.go`
- Create: `internal/profile/handler_test.go`

- [ ] **Step 1: Write `parsing.go`**

Mirror snapshots' helpers. Keep them package-local.

```go
package profile

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/gin-gonic/gin"
)

func parseLang(c *gin.Context) string {
	lang := c.GetHeader("Accept-Language")
	switch lang {
	case "es", "es-ES", "es-AR":
		return "es"
	default:
		return "en"
	}
}

func parseJSONBody(c *gin.Context) (map[string]any, error) {
	if c.GetHeader("Content-Type") != "application/json" {
		return nil, errors.New("expected application/json")
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func respondBadRequest(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": msg}})
}

func respondBackendError(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(502, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": msg}})
}
```

(Verify `parseLang` matches the convention used elsewhere; if `internal/i18n` exposes a `LangFromRequest`, prefer that and remove the local helper.)

- [ ] **Step 2: Write `get_usecase.go`**

```go
package profile

import (
	"context"

	"github.com/project/vk-investment-middleend/internal/components"
)

// Narrow interfaces — easier to stub in tests.
type meFetcher interface {
	GetMe(ctx context.Context, authorization string) (*User, error)
}

type configFetcher interface {
	GetConfig(ctx context.Context, authorization string) (*AppConfig, error)
}

type GetUseCase struct {
	me  meFetcher
	cfg configFetcher
}

func NewGetUseCase(me meFetcher, cfg configFetcher) *GetUseCase {
	return &GetUseCase{me: me, cfg: cfg}
}

// Execute fetches the current user and the config in sequence (matching the
// pattern used by snapshots — sequential with short-circuit on error). If the
// project standardises on parallel fetches later, lift this to a goroutine pair.
func (uc *GetUseCase) Execute(ctx context.Context, authorization, lang string) (components.Component, error) {
	me, err := uc.me.GetMe(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	cfg, err := uc.cfg.GetConfig(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	return BuildScreen(me, cfg, lang), nil
}
```

- [ ] **Step 3: Write `handler.go`**

```go
package profile

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
)

type Handler struct{ uc *GetUseCase }

func NewHandler(uc *GetUseCase) *Handler { return &Handler{uc: uc} }

// Get serves GET /screens/profile.
func (h *Handler) Get(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)
	tree, err := h.uc.Execute(c.Request.Context(), auth, lang)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		respondBackendError(c, "could not load profile")
		return
	}
	c.JSON(http.StatusOK, tree)
}
```

- [ ] **Step 4: Tests**

`get_usecase_test.go`:

```go
package profile

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMe struct {
	res     *User
	err     error
	calls   int
	gotAuth string
}

func (s *stubMe) GetMe(_ context.Context, auth string) (*User, error) {
	s.calls++
	s.gotAuth = auth
	return s.res, s.err
}

type stubCfg struct {
	res     *AppConfig
	err     error
	calls   int
	gotAuth string
}

func (s *stubCfg) GetConfig(_ context.Context, auth string) (*AppConfig, error) {
	s.calls++
	s.gotAuth = auth
	return s.res, s.err
}

func TestGetUseCase_Happy(t *testing.T) {
	m := &stubMe{res: sampleUser()}
	cfg := &stubCfg{res: sampleConfig()}
	uc := NewGetUseCase(m, cfg)
	tree, err := uc.Execute(context.Background(), "Bearer t", "en")
	require.NoError(t, err)
	assert.Equal(t, "Bearer t", m.gotAuth)
	assert.Equal(t, "Bearer t", cfg.gotAuth)
	assert.Equal(t, "screen", asJSON(t, tree)["type"])
}

func TestGetUseCase_MeUnauthorized_ShortCircuits(t *testing.T) {
	m := &stubMe{err: ErrUnauthorized}
	cfg := &stubCfg{}
	uc := NewGetUseCase(m, cfg)
	_, err := uc.Execute(context.Background(), "", "en")
	assert.True(t, errors.Is(err, ErrUnauthorized))
	assert.Equal(t, 0, cfg.calls, "config should not be called after me failed")
}

func TestGetUseCase_ConfigError(t *testing.T) {
	m := &stubMe{res: sampleUser()}
	cfg := &stubCfg{err: ErrBackend}
	uc := NewGetUseCase(m, cfg)
	_, err := uc.Execute(context.Background(), "", "en")
	assert.True(t, errors.Is(err, ErrBackend))
}
```

`handler_test.go`:

```go
package profile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/screens/profile", h.Get)
	return r
}

func TestHandler_Get_Happy(t *testing.T) {
	uc := NewGetUseCase(&stubMe{res: sampleUser()}, &stubCfg{res: sampleConfig()})
	r := newRouter(NewHandler(uc))
	req := httptest.NewRequest(http.MethodGet, "/screens/profile", nil)
	req.Header.Set("Authorization", "Bearer t")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, ScreenID, body["id"])
}

func TestHandler_Get_MeUnauthorized_RedirectsToLogin(t *testing.T) {
	uc := NewGetUseCase(&stubMe{err: ErrUnauthorized}, &stubCfg{})
	r := newRouter(NewHandler(uc))
	req := httptest.NewRequest(http.MethodGet, "/screens/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "/screens/login", body["redirect"])
}

func TestHandler_Get_ConfigBackendError_502(t *testing.T) {
	uc := NewGetUseCase(&stubMe{res: sampleUser()}, &stubCfg{err: ErrBackend})
	r := newRouter(NewHandler(uc))
	req := httptest.NewRequest(http.MethodGet, "/screens/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj := body["error"].(map[string]any)
	assert.Equal(t, "BACKEND_ERROR", errObj["code"])
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/profile/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/profile/parsing.go internal/profile/get_usecase.go internal/profile/handler.go internal/profile/get_usecase_test.go internal/profile/handler_test.go
git commit -m "feat(profile): add screen GET handler and use case"
```

---

## Task 8: Profile update action (`POST /actions/profile/update`)

**Files:**
- Create: `internal/profile/update_handler.go`
- Create: `internal/profile/update_handler_test.go`

- [ ] **Step 1: Write tests**

```go
package profile

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

type stubProfileUpdater struct {
	res    *User
	err    error
	gotBody map[string]any
}

func (s *stubProfileUpdater) UpdateProfile(_ context.Context, _ string, body map[string]any) (*User, error) {
	s.gotBody = body
	return s.res, s.err
}

func newUpdateRouter(updater *stubProfileUpdater, cfg *stubCfg) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/profile/update", NewUpdateHandler(updater, cfg).Post)
	return r
}

func postJSON(t *testing.T, r http.Handler, path, jsonBody string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer t")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateHandler_Happy(t *testing.T) {
	updated := &User{ID: "u1", Email: "vadim@example.com", DisplayName: ptr("Vadim"), Preferences: Preferences{DefaultCurrency: ptr("EUR")}}
	upd := &stubProfileUpdater{res: updated}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)

	w := postJSON(t, r, "/actions/profile/update", `{"display_name":"Vadim","preferences":{"default_currency":"EUR"}}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ProfileCardID, resp["target_id"])
	require.NotNil(t, resp["feedback"])
	assert.Equal(t, "Vadim", upd.gotBody["display_name"])
}

func TestUpdateHandler_EmptyDisplayNameSentAsNull(t *testing.T) {
	upd := &stubProfileUpdater{res: sampleUser()}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)

	w := postJSON(t, r, "/actions/profile/update", `{"display_name":"  ","preferences":{"default_currency":""}}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, upd.gotBody["display_name"])
	prefs := upd.gotBody["preferences"].(map[string]any)
	assert.Nil(t, prefs["default_currency"])
}

func TestUpdateHandler_BackendValidationError_BannerInline(t *testing.T) {
	upd := &stubProfileUpdater{err: &BackendValidationError{Code: "INVALID_DISPLAY_NAME", Message: "too long"}}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)

	w := postJSON(t, r, "/actions/profile/update", `{"display_name":"x","preferences":{"default_currency":"USD"}}`)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ProfileCardID, resp["target_id"])
	assert.Nil(t, resp["feedback"])
	assert.Contains(t, w.Body.String(), "profile-card-error")
}

func TestUpdateHandler_BadJSON_400(t *testing.T) {
	upd := &stubProfileUpdater{}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)
	w := postJSON(t, r, "/actions/profile/update", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHandler_BackendError_502(t *testing.T) {
	upd := &stubProfileUpdater{err: ErrBackend}
	cfg := &stubCfg{res: sampleConfig()}
	r := newUpdateRouter(upd, cfg)
	w := postJSON(t, r, "/actions/profile/update", `{}`)
	require.Equal(t, http.StatusBadGateway, w.Code)
}
```

- [ ] **Step 2: Implement `update_handler.go`**

```go
package profile

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type profileUpdater interface {
	UpdateProfile(ctx context.Context, authorization string, body map[string]any) (*User, error)
}

type UpdateHandler struct {
	updater profileUpdater
	cfg     configFetcher
}

func NewUpdateHandler(updater profileUpdater, cfg configFetcher) *UpdateHandler {
	return &UpdateHandler{updater: updater, cfg: cfg}
}

// Post handles POST /actions/profile/update.
func (h *UpdateHandler) Post(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	submitted, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}

	displayName, currency := readProfileFields(submitted)
	body := buildProfileUpdateBody(displayName, currency)

	updated, err := h.updater.UpdateProfile(c.Request.Context(), auth, body)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			h.respondValidation(c, lang, displayName, currency, be)
			return
		}
		respondBackendError(c, "could not update profile")
		return
	}

	cfg, err := h.cfg.GetConfig(c.Request.Context(), auth)
	if err != nil {
		respondBackendError(c, "could not load currencies")
		return
	}
	tree := BuildProfileCard(updated, cfg, lang, "")
	fb := components.Snackbar("feedback", i18n.T(lang, "profile.update.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ProfileCardID,
		Tree:     &tree,
		Feedback: &fb,
	})
}

// readProfileFields pulls the two fields we care about out of an arbitrary
// JSON body. Empty / whitespace strings are normalised to empty.
func readProfileFields(in map[string]any) (displayName, currency string) {
	if v, ok := in["display_name"].(string); ok {
		displayName = strings.TrimSpace(v)
	}
	if prefs, ok := in["preferences"].(map[string]any); ok {
		if v, ok := prefs["default_currency"].(string); ok {
			currency = strings.TrimSpace(v)
		}
	}
	return
}

// buildProfileUpdateBody maps the form values to the BE payload, sending null
// for cleared fields.
func buildProfileUpdateBody(displayName, currency string) map[string]any {
	body := map[string]any{}
	if displayName == "" {
		body["display_name"] = nil
	} else {
		body["display_name"] = displayName
	}
	prefs := map[string]any{}
	if currency == "" {
		prefs["default_currency"] = nil
	} else {
		prefs["default_currency"] = currency
	}
	body["preferences"] = prefs
	return body
}

// respondValidation re-emits the profile card with the user's submitted values
// and an inline banner translated from the BE error code.
func (h *UpdateHandler) respondValidation(c *gin.Context, lang, displayName, currency string, be *BackendValidationError) {
	auth := c.GetHeader("Authorization")
	cfg, err := h.cfg.GetConfig(c.Request.Context(), auth)
	if err != nil {
		respondBackendError(c, "could not load currencies")
		return
	}
	msg := i18n.T(lang, profileErrorKey(be.Code))
	tree := buildProfileCardWith(displayName, currency, cfg, lang, msg)
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ProfileCardID,
		Tree:     &tree,
	})
}

// profileErrorKey maps BE validation codes to i18n banner keys.
func profileErrorKey(code string) string {
	switch code {
	case "INVALID_DISPLAY_NAME":
		return "profile.update.error.invalid_display_name"
	case "INVALID_CURRENCY":
		return "profile.update.error.invalid_currency"
	default:
		return "profile.update.error.invalid_display_name" // fallback; same surface
	}
}
```

**Note on the import:** add `"context"` to the import block (referenced in the `profileUpdater` interface).

- [ ] **Step 3: Run tests**

Run: `go test ./internal/profile/ -run TestUpdateHandler -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/profile/update_handler.go internal/profile/update_handler_test.go
git commit -m "feat(profile): add profile update action"
```

---

## Task 9: Email update action (`POST /actions/profile/update_email`)

**Files:**
- Create: `internal/profile/update_email_handler.go`
- Create: `internal/profile/update_email_handler_test.go`

- [ ] **Step 1: Write tests** (mirror Task 8's structure)

```go
package profile

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

type stubEmailUpdater struct {
	err error
}

func (s *stubEmailUpdater) UpdateEmail(_ context.Context, _, _, _ string) error { return s.err }

func newEmailRouter(upd *stubEmailUpdater, me *stubMe) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/profile/update_email", NewUpdateEmailHandler(upd, me).Post)
	return r
}

func TestUpdateEmail_Happy_RebuildsCardWithNewEmail(t *testing.T) {
	updated := &User{ID: "u1", Email: "new@example.com"}
	me := &stubMe{res: updated}
	r := newEmailRouter(&stubEmailUpdater{}, me)
	w := postJSON(t, r, "/actions/profile/update_email", `{"new_email":"new@example.com","current_password":"pw"}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, EmailCardID, resp["target_id"])
	require.NotNil(t, resp["feedback"])
	assert.Contains(t, w.Body.String(), "new@example.com")
	// new_email and current_password inputs are cleared (defaultValue:"")
	assert.Contains(t, w.Body.String(), `"defaultValue":""`)
}

func TestUpdateEmail_InvalidCredentials_PreservesNewEmailClearsPassword(t *testing.T) {
	r := newEmailRouter(
		&stubEmailUpdater{err: &BackendValidationError{Code: "INVALID_CREDENTIALS", Message: "wrong"}},
		&stubMe{res: sampleUser()},
	)
	w := postJSON(t, r, "/actions/profile/update_email", `{"new_email":"preserved@x.y","current_password":"pw"}`)
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, `"defaultValue":"preserved@x.y"`)
	assert.Contains(t, body, "email-card-error")
	// At least one input has empty defaultValue (current_password).
	assert.Contains(t, body, `"defaultValue":""`)
}

func TestUpdateEmail_EmailExists(t *testing.T) {
	r := newEmailRouter(
		&stubEmailUpdater{err: &BackendValidationError{Code: "EMAIL_ALREADY_EXISTS", Message: "in use"}},
		&stubMe{res: sampleUser()},
	)
	w := postJSON(t, r, "/actions/profile/update_email", `{"new_email":"taken@x.y","current_password":"pw"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "email-card-error")
}

func TestUpdateEmail_BadJSON_400(t *testing.T) {
	r := newEmailRouter(&stubEmailUpdater{}, &stubMe{res: sampleUser()})
	w := postJSON(t, r, "/actions/profile/update_email", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
```

- [ ] **Step 2: Implement**

```go
package profile

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type emailUpdater interface {
	UpdateEmail(ctx context.Context, authorization, newEmail, currentPassword string) error
}

type UpdateEmailHandler struct {
	updater emailUpdater
	me      meFetcher
}

func NewUpdateEmailHandler(updater emailUpdater, me meFetcher) *UpdateEmailHandler {
	return &UpdateEmailHandler{updater: updater, me: me}
}

func (h *UpdateEmailHandler) Post(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	in, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}
	newEmail, _ := in["new_email"].(string)
	currentPassword, _ := in["current_password"].(string)

	if err := h.updater.UpdateEmail(c.Request.Context(), auth, newEmail, currentPassword); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			tree := buildEmailCardWith(currentEmailFromMe(c, h.me, auth), newEmail, lang, i18n.T(lang, emailErrorKey(be.Code)))
			c.JSON(http.StatusOK, components.ActionResponse{
				Action: "replace", TargetID: EmailCardID, Tree: &tree,
			})
			return
		}
		respondBackendError(c, "could not update email")
		return
	}

	// Success: re-fetch /v1/user/me to get the updated email.
	updated, err := h.me.GetMe(c.Request.Context(), auth)
	if err != nil {
		respondBackendError(c, "could not refresh profile")
		return
	}
	tree := BuildEmailCard(updated, lang, "", "")
	fb := components.Snackbar("feedback", i18n.T(lang, "profile.email.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: EmailCardID, Tree: &tree, Feedback: &fb,
	})
}

// currentEmailFromMe returns the user's current email; on fetch failure it
// returns "" — the banner is the dominant signal in this code path.
func currentEmailFromMe(c *gin.Context, me meFetcher, auth string) string {
	u, err := me.GetMe(c.Request.Context(), auth)
	if err != nil || u == nil {
		return ""
	}
	return u.Email
}

func emailErrorKey(code string) string {
	switch code {
	case "MISSING_FIELDS":
		return "profile.email.error.missing_fields"
	case "INVALID_CREDENTIALS":
		return "profile.email.error.invalid_credentials"
	case "EMAIL_ALREADY_EXISTS":
		return "profile.email.error.email_exists"
	default:
		return "profile.email.error.missing_fields"
	}
}
```

- [ ] **Step 3: Run tests, commit**

Run: `go test ./internal/profile/ -run TestUpdateEmail -v`
Expected: PASS.

```bash
git add internal/profile/update_email_handler.go internal/profile/update_email_handler_test.go
git commit -m "feat(profile): add email update action"
```

---

## Task 10: Password change action (`POST /actions/profile/change_password`)

**Files:**
- Create: `internal/profile/change_password_handler.go`
- Create: `internal/profile/change_password_handler_test.go`

The middleend-side validation (empty fields, mismatch) short-circuits before any BE call. Both error and success rebuild the password card with all three inputs cleared.

- [ ] **Step 1: Write tests**

```go
package profile

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

type stubPasswordChanger struct {
	called bool
	err    error
}

func (s *stubPasswordChanger) ChangePassword(_ context.Context, _, _, _ string) error {
	s.called = true
	return s.err
}

func newPasswordRouter(c *stubPasswordChanger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/profile/change_password", NewChangePasswordHandler(c).Post)
	return r
}

func TestChangePassword_Happy(t *testing.T) {
	pc := &stubPasswordChanger{}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"old","new_password":"newPassword!","confirm_password":"newPassword!"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, pc.called)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, PasswordCardID, resp["target_id"])
	require.NotNil(t, resp["feedback"])
}

func TestChangePassword_MissingFields_NoBECall(t *testing.T) {
	pc := &stubPasswordChanger{}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"","new_password":"","confirm_password":""}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, pc.called)
	assert.Contains(t, w.Body.String(), "password-card-error")
}

func TestChangePassword_DoNotMatch_NoBECall(t *testing.T) {
	pc := &stubPasswordChanger{}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"a","new_password":"b","confirm_password":"c"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, pc.called)
	assert.Contains(t, w.Body.String(), "password-card-error")
}

func TestChangePassword_BEInvalidCredentials(t *testing.T) {
	pc := &stubPasswordChanger{err: &BackendValidationError{Code: "INVALID_CREDENTIALS", Message: "wrong"}}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"old","new_password":"newPassword!","confirm_password":"newPassword!"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, pc.called)
	assert.Contains(t, w.Body.String(), "password-card-error")
}

func TestChangePassword_BEInvalidPassword(t *testing.T) {
	pc := &stubPasswordChanger{err: &BackendValidationError{Code: "INVALID_PASSWORD", Message: "too short"}}
	r := newPasswordRouter(pc)
	w := postJSON(t, r, "/actions/profile/change_password", `{"current_password":"old","new_password":"abc","confirm_password":"abc"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, pc.called)
	assert.Contains(t, w.Body.String(), "password-card-error")
}
```

- [ ] **Step 2: Implement**

```go
package profile

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type passwordChanger interface {
	ChangePassword(ctx context.Context, authorization, currentPassword, newPassword string) error
}

type ChangePasswordHandler struct {
	changer passwordChanger
}

func NewChangePasswordHandler(changer passwordChanger) *ChangePasswordHandler {
	return &ChangePasswordHandler{changer: changer}
}

func (h *ChangePasswordHandler) Post(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	in, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}
	current, _ := in["current_password"].(string)
	newPw, _ := in["new_password"].(string)
	confirm, _ := in["confirm_password"].(string)

	// Middleend-side validation. No BE call on these paths.
	if current == "" || newPw == "" || confirm == "" {
		respondPasswordError(c, lang, "profile.password.error.missing_fields")
		return
	}
	if newPw != confirm {
		respondPasswordError(c, lang, "profile.password.error.do_not_match")
		return
	}

	if err := h.changer.ChangePassword(c.Request.Context(), auth, current, newPw); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			respondPasswordError(c, lang, passwordErrorKey(be.Code))
			return
		}
		respondBackendError(c, "could not change password")
		return
	}

	tree := BuildPasswordCard(lang, "")
	fb := components.Snackbar("feedback", i18n.T(lang, "profile.password.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: PasswordCardID, Tree: &tree, Feedback: &fb,
	})
}

func respondPasswordError(c *gin.Context, lang, key string) {
	tree := BuildPasswordCard(lang, i18n.T(lang, key))
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: PasswordCardID, Tree: &tree,
	})
}

func passwordErrorKey(code string) string {
	switch code {
	case "MISSING_FIELDS":
		return "profile.password.error.missing_fields"
	case "INVALID_CREDENTIALS":
		return "profile.password.error.invalid_credentials"
	case "INVALID_PASSWORD":
		return "profile.password.error.invalid_password"
	default:
		return "profile.password.error.invalid_credentials"
	}
}
```

- [ ] **Step 3: Run tests, commit**

```bash
go test ./internal/profile/ -run TestChangePassword -v
git add internal/profile/change_password_handler.go internal/profile/change_password_handler_test.go
git commit -m "feat(profile): add password change action with client-side validation"
```

---

## Task 11: Delete modal handler + delete account handler

**Files:**
- Create: `internal/profile/delete_modal_handler.go`
- Create: `internal/profile/delete_handler.go`
- Create: `internal/profile/delete_modal_handler_test.go`
- Create: `internal/profile/delete_handler_test.go`

- [ ] **Step 1: Write `delete_modal_handler.go`**

```go
package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
)

type DeleteModalHandler struct{}

func NewDeleteModalHandler() *DeleteModalHandler { return &DeleteModalHandler{} }

func (h *DeleteModalHandler) Get(c *gin.Context) {
	lang := parseLang(c)
	modal := BuildDeleteModal(lang, "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: ModalSlotID, Tree: &modal,
	})
}
```

- [ ] **Step 2: Write `delete_handler.go`**

```go
package profile

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type accountDeleter interface {
	DeleteAccount(ctx context.Context, authorization, password string) error
}

type DeleteHandler struct {
	deleter accountDeleter
}

func NewDeleteHandler(deleter accountDeleter) *DeleteHandler {
	return &DeleteHandler{deleter: deleter}
}

func (h *DeleteHandler) Post(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	in, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}
	password, _ := in["password"].(string)

	if err := h.deleter.DeleteAccount(c.Request.Context(), auth, password); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			modal := BuildDeleteModal(lang, i18n.T(lang, dangerErrorKey(be.Code)))
			c.JSON(http.StatusOK, components.ActionResponse{
				Action: "replace", TargetID: ModalSlotID, Tree: &modal,
			})
			return
		}
		respondBackendError(c, "could not delete account")
		return
	}

	c.JSON(http.StatusOK, components.LogoutResponse("/screens/login"))
}

func dangerErrorKey(code string) string {
	switch code {
	case "MISSING_FIELDS":
		return "profile.danger.error.missing_fields"
	case "INVALID_CREDENTIALS":
		return "profile.danger.error.invalid_credentials"
	default:
		return "profile.danger.error.invalid_credentials"
	}
}
```

- [ ] **Step 3: Tests**

`delete_modal_handler_test.go`:

```go
package profile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteModalHandler_ReturnsModal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/actions/profile/delete_modal", NewDeleteModalHandler().Get)

	req := httptest.NewRequest(http.MethodGet, "/actions/profile/delete_modal", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ModalSlotID, resp["target_id"])
	assert.Contains(t, w.Body.String(), DeleteModalID)
}
```

`delete_handler_test.go`:

```go
package profile

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDeleter struct{ err error }

func (s *stubDeleter) DeleteAccount(_ context.Context, _, _ string) error { return s.err }

func newDeleteRouter(d *stubDeleter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/profile/delete_account", NewDeleteHandler(d).Post)
	return r
}

func TestDeleteHandler_Happy_LogoutResponse(t *testing.T) {
	r := newDeleteRouter(&stubDeleter{})
	w := postJSON(t, r, "/actions/profile/delete_account", `{"password":"pw"}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "logout", resp["action"])
	assert.Equal(t, "/screens/login", resp["target_id"])
	assert.Nil(t, resp["feedback"])
}

func TestDeleteHandler_InvalidCredentials_RemodalsWithError(t *testing.T) {
	r := newDeleteRouter(&stubDeleter{err: &BackendValidationError{Code: "INVALID_CREDENTIALS", Message: "wrong"}})
	w := postJSON(t, r, "/actions/profile/delete_account", `{"password":"x"}`)
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	var resp map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &resp))
	assert.Equal(t, "replace", resp["action"])
	assert.Equal(t, ModalSlotID, resp["target_id"])
	assert.Contains(t, body, "delete-modal-error")
	assert.Contains(t, body, `"defaultValue":""`) // password cleared
}

func TestDeleteHandler_BadJSON_400(t *testing.T) {
	r := newDeleteRouter(&stubDeleter{})
	w := postJSON(t, r, "/actions/profile/delete_account", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/profile/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/profile/delete_modal_handler.go internal/profile/delete_handler.go internal/profile/delete_modal_handler_test.go internal/profile/delete_handler_test.go
git commit -m "feat(profile): add delete-account modal and action"
```

---

## Task 12: Wire i18n locales

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

- [ ] **Step 1: Locate the locale files**

Run: `ls locales/`
Expected: at least `en.json` and `es.json`. (If under a different path, e.g. `internal/i18n/locales/`, follow that — the i18n loader reads from a configured directory.)

- [ ] **Step 2: Add the `profile.*` namespace**

In `en.json`, add:

```json
"profile": {
  "title": "Profile",
  "section": {
    "profile": "Profile",
    "email": "Email",
    "password": "Password"
  },
  "display_name": "Display name",
  "display_name.placeholder": "How should we call you?",
  "default_currency": "Default currency",
  "default_currency.none": "— None —",
  "update": {
    "save": "Save",
    "success": "Profile updated.",
    "error": {
      "invalid_display_name": "Display name must be between 1 and 100 characters.",
      "invalid_currency": "Invalid currency."
    }
  },
  "email": {
    "current": "Current: {email}",
    "new": "New email",
    "current_password": "Current password",
    "save": "Save",
    "success": "Email updated.",
    "error": {
      "missing_fields": "Please fill in all required fields.",
      "invalid_credentials": "Incorrect password.",
      "email_exists": "This email is already in use."
    }
  },
  "password": {
    "current": "Current password",
    "new": "New password",
    "confirm": "Confirm new password",
    "save": "Save",
    "success": "Password updated.",
    "error": {
      "missing_fields": "Please fill in all required fields.",
      "do_not_match": "New password and confirmation do not match.",
      "invalid_credentials": "Incorrect current password.",
      "invalid_password": "New password must be at least 8 characters."
    }
  },
  "danger": {
    "title": "Danger Zone",
    "body": "Permanently delete your account and all data. This cannot be undone.",
    "delete_button": "Delete account",
    "modal": {
      "title": "Delete account",
      "body": "All your data will be erased. This cannot be undone.",
      "password_label": "Enter your password to confirm",
      "cancel": "Cancel",
      "confirm": "Delete account"
    },
    "error": {
      "missing_fields": "Please enter your password.",
      "invalid_credentials": "Incorrect password."
    }
  }
}
```

In `es.json`, add the same structure with Spanish strings:

```json
"profile": {
  "title": "Perfil",
  "section": {
    "profile": "Perfil",
    "email": "Correo electrónico",
    "password": "Contraseña"
  },
  "display_name": "Nombre",
  "display_name.placeholder": "¿Cómo te llamamos?",
  "default_currency": "Moneda por defecto",
  "default_currency.none": "— Ninguna —",
  "update": {
    "save": "Guardar",
    "success": "Perfil actualizado.",
    "error": {
      "invalid_display_name": "El nombre debe tener entre 1 y 100 caracteres.",
      "invalid_currency": "Moneda inválida."
    }
  },
  "email": {
    "current": "Actual: {email}",
    "new": "Nuevo correo",
    "current_password": "Contraseña actual",
    "save": "Guardar",
    "success": "Correo actualizado.",
    "error": {
      "missing_fields": "Completá todos los campos.",
      "invalid_credentials": "Contraseña incorrecta.",
      "email_exists": "Este correo ya está en uso."
    }
  },
  "password": {
    "current": "Contraseña actual",
    "new": "Nueva contraseña",
    "confirm": "Confirmar nueva contraseña",
    "save": "Guardar",
    "success": "Contraseña actualizada.",
    "error": {
      "missing_fields": "Completá todos los campos.",
      "do_not_match": "La nueva contraseña y la confirmación no coinciden.",
      "invalid_credentials": "Contraseña actual incorrecta.",
      "invalid_password": "La nueva contraseña debe tener al menos 8 caracteres."
    }
  },
  "danger": {
    "title": "Zona de peligro",
    "body": "Borrá permanentemente tu cuenta y todos los datos. No se puede deshacer.",
    "delete_button": "Eliminar cuenta",
    "modal": {
      "title": "Eliminar cuenta",
      "body": "Todos tus datos serán eliminados. Esta acción no se puede deshacer.",
      "password_label": "Ingresá tu contraseña para confirmar",
      "cancel": "Cancelar",
      "confirm": "Eliminar cuenta"
    },
    "error": {
      "missing_fields": "Ingresá tu contraseña.",
      "invalid_credentials": "Contraseña incorrecta."
    }
  }
}
```

**Caveat:** the i18n loader flattens keys with dots, but a key already containing a dot like `display_name.placeholder` becomes a nested map under `display_name`, not a single flat key. Verify against `internal/i18n/i18n.go`'s `flatten` to confirm the desired behavior. If flattening collides, rename the in-key dot to underscore (e.g. `display_name_placeholder`) and update the builder.

- [ ] **Step 3: Run all tests**

Run: `make test`
Expected: PASS — all profile tests now have real labels to assert against (the title test expects `"Profile"`).

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "feat(profile): add en/es locale strings"
```

---

## Task 13: Wire routes in `server.go`

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/server/server_test.go` (a smoke test that the routes exist)

- [ ] **Step 1: Read current `setupRoutes` to find the snapshots block**

Run: `grep -n 'snapshots' internal/server/server.go | head -20`

- [ ] **Step 2: Wire profile routes**

In `setupRoutes()`, near the snapshots block, add:

```go
// --- profile ---
profileClient := profile.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
profileConfigClient := profile.NewConfigClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
profileUC := profile.NewGetUseCase(profileClient, profileConfigClient)
protected.GET("/screens/profile", profile.NewHandler(profileUC).Get)
protected.POST("/actions/profile/update", profile.NewUpdateHandler(profileClient, profileConfigClient).Post)
protected.POST("/actions/profile/update_email", profile.NewUpdateEmailHandler(profileClient, profileClient).Post)
protected.POST("/actions/profile/change_password", profile.NewChangePasswordHandler(profileClient).Post)
protected.GET("/actions/profile/delete_modal", profile.NewDeleteModalHandler().Get)
protected.POST("/actions/profile/delete_account", profile.NewDeleteHandler(profileClient).Post)
```

Add the import at the top:

```go
import (
	// ...
	"github.com/project/vk-investment-middleend/internal/profile"
)
```

- [ ] **Step 3: Add a route smoke test**

In `internal/server/server_test.go`, append:

```go
func TestRouter_HasProfileRoutes(t *testing.T) {
	s := newTestServer(t) // existing helper used elsewhere; copy the established pattern
	r := s.Router()
	routes := r.Routes()
	wanted := map[string]bool{
		"GET /screens/profile":                   false,
		"POST /actions/profile/update":           false,
		"POST /actions/profile/update_email":     false,
		"POST /actions/profile/change_password":  false,
		"GET /actions/profile/delete_modal":      false,
		"POST /actions/profile/delete_account":   false,
	}
	for _, ri := range routes {
		key := ri.Method + " " + ri.Path
		if _, ok := wanted[key]; ok {
			wanted[key] = true
		}
	}
	for k, found := range wanted {
		assert.Truef(t, found, "route missing: %s", k)
	}
}
```

If `newTestServer` doesn't exist exactly with that name, mirror whatever `server_test.go` already uses to construct a router. Look at the existing top of the file for the helper.

- [ ] **Step 4: Run tests**

Run: `make test`
Expected: PASS.

- [ ] **Step 5: Lint**

Run: `make lint`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add internal/server/server.go internal/server/server_test.go
git commit -m "feat(profile): wire profile routes"
```

---

## Task 14: End-to-end smoke run

**Files:** none modified — manual verification.

- [ ] **Step 1: Restart the dev server**

```bash
./cli run
```

Expected: server listens on `:8082`.

- [ ] **Step 2: Hit the screen GET (you need a real Bearer token from the BE)**

```bash
TOKEN="<paste a valid token>"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/profile | jq '.id'
```

Expected: `"profile"`.

- [ ] **Step 3: Hit the screen GET without a token**

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8082/screens/profile
```

Expected: `401`.

- [ ] **Step 4: Hit `update` with a malformed body**

```bash
curl -s -o /dev/null -w "%{http_code}\n" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -X POST -d 'not json' http://localhost:8082/actions/profile/update
```

Expected: `400`.

- [ ] **Step 5: Verify the modal endpoint**

```bash
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/actions/profile/delete_modal | jq '.action, .target_id'
```

Expected: `"replace"` and `"profile-modal-slot"`.

- [ ] **Step 6: No commit** — this task is verification only.

---

## Task 15: Spec sync verification

- [ ] **Step 1: Re-read `spec/screens/profile.md` end-to-end** with the implementation in front of you. For every acceptance-criteria checkbox, point at the file/test that proves it. If any criterion has no implementation, open it as a follow-up task.

- [ ] **Step 2: Update `MEMORY.md` if anything surprising came up.** Otherwise no action.

- [ ] **Step 3: Final commit (only if needed).**

If the verification surfaced a small spec edit (e.g. a renamed key), commit it as `docs(spec): align profile spec with shipped behaviour`.

---

## Self-Review

**Spec coverage:** Each spec section has a task — endpoints (Tasks 7–11), four cards (Task 5), modal (Task 6), validation (Task 10 middleend rule), logout primitive (Task 1 + 11), i18n (Task 12), routes (Task 13). The 401-from-BE rule is enforced by the client's switch on validation status codes (Task 3).

**Placeholder scan:** No "TBD"/"TODO"/"implement later" inside steps. Every code step shows the code; every test step shows the test.

**Type consistency:** `User`, `Preferences`, `AppConfig` defined in Task 2. `BackendValidationError` defined in Task 2 and consumed in Tasks 3, 8–11. `meFetcher`/`configFetcher` interfaces in Task 7 and reused in Task 9. `LogoutResponse` defined in Task 1, called in Task 11. IDs (`ProfileCardID` etc.) defined in Task 5 and referenced everywhere.

**Open verification points the engineer needs to handle inline (not blockers):**

1. Action shape on submit buttons — copy the exact `Action{}` literal from `internal/snapshots/builder.go` so the frontend's existing form-submit handler accepts profile's buttons. (Task 5, Step 1.)
2. `parseLang` — replace the local helper with whatever the i18n package exposes if there's a canonical helper. (Task 7, Step 1.)
3. Modal Cancel — use `dismiss_modal` action type if supported, otherwise add a small `GET /actions/profile/close_modal` that emits an empty replace. (Task 6.)
4. Locale flattening — verify `display_name.placeholder` doesn't collide. Rename to `_` if needed. (Task 12.)
5. `newTestServer` helper in server_test — mirror existing pattern. (Task 13, Step 3.)

These are the kinds of small adjustments that surface only when the engineer is in the file. The plan flags each one in-place.
