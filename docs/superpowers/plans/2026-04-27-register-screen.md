# Register Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the public `GET /screens/register` SDUI screen plus an aligned `POST /actions/register` action, with a small SDUI base extension (`min_length` + `match_field` on `input`) that the form needs.

**Architecture:** Mirror the login screen. New package `internal/register/` for the screen GET (builder + handler). Existing `internal/auth/register_handler.go` refactored to follow the profile-style `ActionResponse` pattern (replace the form, never flat HTTP errors for user-recoverable cases) and to validate `confirm_password` server-side. SDUI `input` gains two optional client-side validation props.

**Tech Stack:** Go, Gin, project-internal SDUI components, project i18n (`internal/i18n`), Go stdlib `testing` + `testify`.

**Source-of-truth files (read-only context for implementers):**
- Spec contract to follow: `spec/screens/login.md` (mirror this).
- Profile replace-form pattern reference: `internal/profile/change_password_handler.go`.
- Existing register submit handler to refactor: `internal/auth/register_handler.go`.
- Existing screen builder pattern: `internal/login/builder.go`.
- Existing screen handler pattern: `internal/login/handler.go`.
- SDUI base spec to extend: `spec/sdui-base-components.md`.

**Commits:** Conventional Commits, no Claude Code co-author trailer (per repo CLAUDE.md).

---

## File Map

**Create:**
- `internal/register/builder.go` — `BuildScreen(lang string, errorMsg string) components.Component` and `BuildForm(lang string, prefillEmail string, errorMsg string, submitDisabled bool) components.Component`. Returns the screen tree; `errorMsg` empty means no banner; `submitDisabled` true greys out the submit (used by the disabled-registration outcome).
- `internal/register/builder_test.go` — table-driven tests verifying the rendered tree (form id, input names + props, action wiring, banner presence/absence, login-link nav).
- `internal/register/handler.go` — `type Handler struct{}` exposing `Get(c *gin.Context)`. Reads `Accept-Language`, returns `BuildScreen(lang, "")`.
- `internal/register/handler_test.go` — black-box test of `GET /screens/register`: 200, no auth required, `Accept-Language: es` returns Spanish, no shell slot ids in the tree.
- `spec/screens/register.md` — canonical screen spec (this is the contract that ships).

**Modify:**
- `internal/components/base.go:357-385` — extend `InputFull` with `min_length` (int) and `match_field` (string) handling. Strategy: add a new constructor `InputAdvanced(opts InputOptions) Component` and have `Input` / `InputFull` delegate to it; existing callers stay binary-compatible. The `InputOptions` struct is the new public shape.
- `internal/components/base.go` (same file, separate diff) — add `InputOptions` struct + `InputAdvanced` constructor.
- `internal/components/base_test.go` (or whichever existing test file covers `Input`/`InputFull`; create if absent) — verify new props serialize when set, are omitted when zero.
- `internal/auth/register_handler.go` — full rewrite of the body of `Post`: parse `confirm_password`, validate length + match middleend-side, translate every BE outcome to a `200 ActionResponse`, fix `/login` → `/screens/login`, strip `confirm_password` before forwarding to BE.
- `internal/auth/register_handler_test.go` — replace existing assertions to match the new ActionResponse shape; add cases for length, match, disabled, email-exists, transient.
- `internal/server/server.go:42` (block above the `protected` group) — register `s.router.GET("/screens/register", register.NewHandler().Get)` next to the existing login GET line. Add `register` import.
- `internal/server/server_test.go` — extend the public-route test (or add) so `GET /screens/register` returns 200 without an Authorization header.
- `locales/en.json:20-29` (the `"auth"` block) — add the new keys listed in Task 2.
- `locales/es.json:20-29` (the `"auth"` block) — Spanish counterparts.
- `spec/sdui-base-components.md` — document `min_length` and `match_field` in the `input` props table.
- `spec/spec.md` — add the Register row to the screens index (it's missing today; Login is there).

**Out of repo (not changed):** the backend.

---

## Task 1: Extend SDUI `input` with `min_length` and `match_field` (constructor + spec)

**Files:**
- Modify: `internal/components/base.go` (add `InputOptions` + `InputAdvanced`; leave existing `Input` / `InputFull` untouched as thin wrappers)
- Test: `internal/components/base_test.go` (add file if absent)
- Modify: `spec/sdui-base-components.md`

- [ ] **Step 1: Write the failing tests for `InputAdvanced`**

Append to `internal/components/base_test.go` (create if file does not exist with appropriate package header):

```go
package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInputAdvanced_RequiredFieldsOnly(t *testing.T) {
	c := InputAdvanced(InputOptions{ID: "p", Name: "p", InputType: "password"})

	assert.Equal(t, "input", c.Type)
	assert.Equal(t, "p", c.ID)
	assert.Equal(t, "p", c.Props["name"])
	assert.Equal(t, "password", c.Props["input_type"])
	_, hasMin := c.Props["min_length"]
	_, hasMatch := c.Props["match_field"]
	assert.False(t, hasMin, "min_length must be omitted when zero")
	assert.False(t, hasMatch, "match_field must be omitted when empty")
}

func TestInputAdvanced_MinLengthAndMatchField(t *testing.T) {
	c := InputAdvanced(InputOptions{
		ID: "confirm", Name: "confirm_password", InputType: "password",
		Required: true, MinLength: 8, MatchField: "password",
	})

	assert.Equal(t, true, c.Props["required"])
	assert.Equal(t, 8, c.Props["min_length"])
	assert.Equal(t, "password", c.Props["match_field"])
}

func TestInputFull_StillWorksUnchanged(t *testing.T) {
	c := InputFull("e", "email", "email", "Email", "you@example.com", "", true, false, 0)
	assert.Equal(t, "input", c.Type)
	assert.Equal(t, "email", c.Props["name"])
	assert.Equal(t, true, c.Props["required"])
	_, hasMin := c.Props["min_length"]
	assert.False(t, hasMin)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/components/ -run InputAdvanced -v`
Expected: FAIL with "undefined: InputAdvanced" and/or "undefined: InputOptions".

- [ ] **Step 3: Add `InputOptions` + `InputAdvanced` and refactor `Input` / `InputFull` to delegate**

In `internal/components/base.go`, immediately after the existing `InputFull` (around line 385), add:

```go
// InputOptions is the full prop surface for the input component.
// Zero-valued fields are omitted from the rendered props.
type InputOptions struct {
	ID           string
	Name         string
	InputType    string
	Label        string
	Placeholder  string
	DefaultValue string
	Required     bool
	Disabled     bool
	MaxLength    int
	Pattern      string
	AutoUpper    bool
	MinLength    int    // client-side: blocks submit if value is shorter
	MatchField   string // client-side: blocks submit unless value equals the named sibling field's value
}

// InputAdvanced builds an input component from an options struct.
func InputAdvanced(o InputOptions) Component {
	props := map[string]any{
		"name":       o.Name,
		"input_type": o.InputType,
	}
	if o.Label != "" {
		props["label"] = o.Label
	}
	if o.Placeholder != "" {
		props["placeholder"] = o.Placeholder
	}
	if o.DefaultValue != "" {
		props["default_value"] = o.DefaultValue
	}
	if o.Required {
		props["required"] = true
	}
	if o.Disabled {
		props["disabled"] = true
	}
	if o.MaxLength > 0 {
		props["max_length"] = o.MaxLength
	}
	if o.Pattern != "" {
		props["pattern"] = o.Pattern
	}
	if o.AutoUpper {
		props["auto_uppercase"] = true
	}
	if o.MinLength > 0 {
		props["min_length"] = o.MinLength
	}
	if o.MatchField != "" {
		props["match_field"] = o.MatchField
	}
	return Component{Type: "input", ID: o.ID, Props: props}
}
```

Leave `Input` and `InputFull` as-is (they already work; adding `InputAdvanced` is purely additive). No callers need to change in this task.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/components/ -v -run "Input"`
Expected: PASS for all three new tests; existing input tests still PASS.

- [ ] **Step 5: Update `spec/sdui-base-components.md`**

In the `### input` props table (just below `pattern` and `auto_uppercase`), append two rows so the table reads:

```
| `pattern` | string | no | ECMAScript regex validated client-side on change/blur; non-matching values block submission |
| `auto_uppercase` | bool | no | Frontend transforms entered value to uppercase as the user types |
| `min_length` | int | no | Minimum character count; values shorter than this block submission. Validated client-side on change/blur |
| `match_field` | string | no | Name of another input within the same `form`; submission is blocked unless this field's value equals that sibling's value. Validated client-side on change/blur of either field |
```

Also append a third Go example after the existing two:

```go
c := components.InputAdvanced(components.InputOptions{
    ID: "confirm", Name: "confirm_password", InputType: "password",
    Required: true, MinLength: 8, MatchField: "password",
})
```

- [ ] **Step 6: Commit**

```bash
git add internal/components/base.go internal/components/base_test.go spec/sdui-base-components.md
git commit -m "feat(sdui): add min_length and match_field props to input"
```

---

## Task 2: Add register i18n keys (en + es)

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

- [ ] **Step 1: Update `locales/en.json` `auth` block**

Replace the existing `"auth": { ... }` block (currently at lines 20-29) with:

```json
  "auth": {
    "login_title": "Log in",
    "register_title": "Create account",
    "email_label": "Email",
    "email_placeholder": "you@example.com",
    "password_label": "Password",
    "password_placeholder": "Your password",
    "confirm_password_label": "Confirm password",
    "confirm_password_placeholder": "Repeat your password",
    "submit": "Log in",
    "register_submit": "Create account",
    "no_account_prompt": "Don't have an account?",
    "register_link": "Sign up",
    "have_account_prompt": "Already have an account?",
    "login_link": "Log in",
    "register_success": "Account created. Please log in.",
    "error_email_exists": "An account with this email already exists.",
    "error_registration_disabled": "Registration is currently disabled.",
    "error_validation": "Please check the form and try again.",
    "error_transient": "Something went wrong. Please try again."
  },
```

- [ ] **Step 2: Update `locales/es.json` `auth` block**

Replace the existing `"auth": { ... }` block (currently at lines 20-29) with:

```json
  "auth": {
    "login_title": "Iniciar sesión",
    "register_title": "Crear cuenta",
    "email_label": "Correo",
    "email_placeholder": "vos@ejemplo.com",
    "password_label": "Contraseña",
    "password_placeholder": "Tu contraseña",
    "confirm_password_label": "Confirmar contraseña",
    "confirm_password_placeholder": "Repetí tu contraseña",
    "submit": "Ingresar",
    "register_submit": "Crear cuenta",
    "no_account_prompt": "¿No tenés cuenta?",
    "register_link": "Registrarme",
    "have_account_prompt": "¿Ya tenés cuenta?",
    "login_link": "Ingresar",
    "register_success": "Cuenta creada. Iniciá sesión.",
    "error_email_exists": "Ya existe una cuenta con ese email.",
    "error_registration_disabled": "El registro está deshabilitado.",
    "error_validation": "Revisá el formulario e intentá de nuevo.",
    "error_transient": "Hubo un problema. Intentá de nuevo."
  },
```

- [ ] **Step 3: Run i18n + components tests as smoke**

Run: `go test ./internal/i18n/... ./internal/components/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "i18n(auth): add register screen keys (en, es)"
```

---

## Task 3: Write the canonical screen spec

**Files:**
- Create: `spec/screens/register.md`
- Modify: `spec/spec.md`

- [ ] **Step 1: Create `spec/screens/register.md`**

Content:

```markdown
# Register Screen

Standalone account-creation screen for unauthenticated users. Like login, it renders without the shell — no nav sidebar, no bottombar.

## Purpose

Collect an email, a password, and a password confirmation; submit them to `POST /actions/register`; on success, send the user to the login screen with a success snackbar; on recoverable failures, re-render the form with an inline error banner.

## Endpoint

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET`  | `/screens/register` | no | Returns the register screen component tree. Public. |
| `POST` | `/actions/register` | no | Submits the form. Returns an `ActionResponse`. |

Headers read on `GET`:
- `Accept-Language` — BCP-47 tag. Missing or unsupported → `en`.

`X-Platform` is not consulted.

## Layout

A centered card with the app name, a title, the form, and a row at the bottom with a prompt + a button that navigates to `/screens/login`. Centering is via shared alignment props (`align_items` / `justify_items` `center`) on the column root. No shell slots are emitted.

The form's `id` is `register-form`. The card's `id` is `register-card`. The action handler replaces `register-form` (not the card) on recoverable errors — same pattern as `internal/profile/change_password_handler.go`.

## Form behavior

Three inputs:

| Name | Type | Props |
|---|---|---|
| `email`            | `email`    | `required: true` |
| `password`         | `password` | `required: true`, `min_length: 8` |
| `confirm_password` | `password` | `required: true`, `match_field: "password"` |

Submit button: label `auth.register_submit`, action `submit POST /actions/register` targeting `register-form`.

Submit body collected by FE: `{ email, password, confirm_password }`.

Login-link button: label `auth.login_link`, action `navigate /screens/login` (client-side, `target: self`).

## `POST /actions/register` outcomes

| Outcome | HTTP | `action` | `target_id` | Tree | Feedback |
|---|---|---|---|---|---|
| Success (BE `201`) | `200` | `navigate` | `/screens/login` | — | snackbar `auth.register_success` (variant: success) |
| BE `409 EMAIL_ALREADY_EXISTS` | `200` | `replace` | `register-form` | form rebuilt: email pre-filled, passwords cleared, banner `auth.error_email_exists` | — |
| BE `403 REGISTRATION_DISABLED` | `200` | `replace` | `register-form` | form rebuilt: all fields cleared, banner `auth.error_registration_disabled`, submit button disabled | — |
| Middleend validation (`email`/`password`/`confirm_password` empty, `password` shorter than 8, `confirm_password` mismatch) | `200` | `replace` | `register-form` | form rebuilt: email pre-filled, passwords cleared, banner `auth.error_validation` | — |
| BE 5xx / network / malformed JSON | `200` | `none` | — | — | snackbar `auth.error_transient` (variant: error) |
| Middleend internal error (panic, unexpected) | `502` | flat error envelope `{"error":{"code":"BACKEND_ERROR",...}}` | — | — | — |

The `confirm_password` field is **stripped** before forwarding to the backend; only `{ email, password }` is sent to `POST /v1/auth/register`.

## Banner placement

Inside the form, above the inputs. Rebuilt each time the form is re-rendered. Variant `error` for all error keys. When `errorMsg` is empty the banner is omitted entirely (zero-valued).

## i18n keys

Under `auth.*` (defined in `locales/{en,es}.json`):

- `auth.register_title`
- `auth.register_submit`
- `auth.email_label`, `auth.email_placeholder` (reused)
- `auth.password_label`, `auth.password_placeholder` (reused)
- `auth.confirm_password_label`, `auth.confirm_password_placeholder`
- `auth.have_account_prompt`, `auth.login_link`
- `auth.register_success`
- `auth.error_email_exists`
- `auth.error_registration_disabled`
- `auth.error_validation`
- `auth.error_transient`

App name reused: `app.name`.

## Out of scope

- Email verification / confirmation link.
- Auto-login after register (BE returns `201` with no token).
- Password-strength meter beyond `min_length: 8`.
- `REGISTRATION_ENABLED` probing on `GET` — the form always renders; the disabled state surfaces only on submit (BE 403 → banner).

## Acceptance criteria

### `GET /screens/register`

- [ ] Returns `200` without any `Authorization` header.
- [ ] Body is a `screen` with `id: register`. No shell slot components (`nav_header`, `nav_main`, `nav_footer`, `bottombar`, `content_slot`).
- [ ] Tree contains `register-card` with: `app.name` text, `auth.register_title` text, `register-form`, submit button, and a row with `auth.have_account_prompt` text + `auth.login_link` button.
- [ ] `register-form` contains exactly three inputs, in order: `email` (type `email`, `required`), `password` (type `password`, `required`, `min_length: 8`), `confirm_password` (type `password`, `required`, `match_field: "password"`).
- [ ] Submit button action: `{trigger: click, type: submit, endpoint: /actions/register, method: POST, target_id: register-form}`.
- [ ] Login-link button action: `{trigger: click, type: navigate, url: /screens/login, target: self}`.
- [ ] Root container is centered both axes via shared alignment props.
- [ ] All user-facing strings resolve via i18n; `Accept-Language: es` returns Spanish; unknown languages fall back to English.

### `POST /actions/register`

- [ ] Success → `200 ActionResponse{action: navigate, target_id: /screens/login, feedback: snackbar (auth.register_success)}`.
- [ ] BE `409 EMAIL_ALREADY_EXISTS` → `200 ActionResponse{action: replace, target_id: register-form}`; tree contains banner `auth.error_email_exists`; email input has `default_value` matching the submitted email; password inputs are empty.
- [ ] BE `403 REGISTRATION_DISABLED` → `200 ActionResponse{action: replace, target_id: register-form}`; tree contains banner `auth.error_registration_disabled`; submit button has `disabled: true`.
- [ ] Middleend validation failures (empty fields, `password` < 8, mismatch) → `200 ActionResponse{action: replace, target_id: register-form}`; banner `auth.error_validation`; email pre-filled, passwords cleared.
- [ ] BE 5xx → `200 ActionResponse{action: none, feedback: snackbar (auth.error_transient, error)}`. Body forwarded to BE never includes `confirm_password`.
- [ ] BE call body is exactly `{ "email": <string>, "password": <string> }`.
```

- [ ] **Step 2: Update `spec/spec.md` screens index**

In the screens index table (currently lines 39-49), insert the Register row immediately after the Login row:

```
| Login | [`screens/login.md`](screens/login.md) |
| Register | [`screens/register.md`](screens/register.md) |
| Portfolio | [`screens/portfolio.md`](screens/portfolio.md) |
```

- [ ] **Step 3: Commit**

```bash
git add spec/screens/register.md spec/spec.md
git commit -m "docs(spec): add register screen contract"
```

---

## Task 4: Register screen builder

**Files:**
- Create: `internal/register/builder.go`
- Test: `internal/register/builder_test.go`

- [ ] **Step 1: Write the failing builder test**

Create `internal/register/builder_test.go`:

```go
package register

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/project/vk-investment-middleend/internal/components"
)

func findByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if got := findByID(c.Children[i], id); got != nil {
			return got
		}
	}
	return nil
}

func TestBuildScreen_HasFormWithThreeInputs(t *testing.T) {
	tree := BuildScreen("en", "")

	assert.Equal(t, "screen", tree.Type)
	assert.Equal(t, "register", tree.ID)

	form := findByID(tree, "register-form")
	if assert.NotNil(t, form, "register-form must exist") {
		emailIn := findByID(*form, "register-email")
		passIn := findByID(*form, "register-password")
		confirmIn := findByID(*form, "register-confirm-password")
		assert.Equal(t, "email", emailIn.Props["name"])
		assert.Equal(t, true, emailIn.Props["required"])
		assert.Equal(t, "password", passIn.Props["name"])
		assert.Equal(t, true, passIn.Props["required"])
		assert.Equal(t, 8, passIn.Props["min_length"])
		assert.Equal(t, "confirm_password", confirmIn.Props["name"])
		assert.Equal(t, "password", confirmIn.Props["match_field"])
	}
}

func TestBuildScreen_NoBannerByDefault(t *testing.T) {
	tree := BuildScreen("en", "")
	banner := findByID(tree, "register-banner")
	assert.Nil(t, banner, "banner must be omitted when errorMsg is empty")
}

func TestBuildScreen_BannerWhenErrorMsgPresent(t *testing.T) {
	tree := BuildScreen("en", "Something failed")
	banner := findByID(tree, "register-banner")
	if assert.NotNil(t, banner, "banner must be present when errorMsg is non-empty") {
		assert.Equal(t, "Something failed", banner.Props["text"])
	}
}

func TestBuildScreen_LoginLinkNavigates(t *testing.T) {
	tree := BuildScreen("en", "")
	link := findByID(tree, "register-login-link")
	if assert.NotNil(t, link) {
		assert.Len(t, link.Actions, 1)
		assert.Equal(t, "navigate", link.Actions[0].Type)
		assert.Equal(t, "/screens/login", link.Actions[0].URL)
	}
}

func TestBuildScreen_NoShellSlots(t *testing.T) {
	tree := BuildScreen("en", "")
	for _, id := range []string{"nav_header", "nav_main", "nav_footer", "bottombar", "content_slot"} {
		assert.Nil(t, findByID(tree, id), "shell slot %s must not be present", id)
	}
}

func TestBuildScreen_Spanish(t *testing.T) {
	tree := BuildScreen("es", "")
	title := findByID(tree, "register-title")
	if assert.NotNil(t, title) {
		assert.Equal(t, "Crear cuenta", title.Props["text"])
	}
}

func TestBuildForm_PrefillEmailAndDisableSubmit(t *testing.T) {
	form := BuildForm("en", "user@example.com", "boom", true)
	email := findByID(form, "register-email")
	pass := findByID(form, "register-password")
	confirm := findByID(form, "register-confirm-password")
	submit := findByID(form, "register-submit")

	assert.Equal(t, "user@example.com", email.Props["default_value"])
	_, hasPassDefault := pass.Props["default_value"]
	_, hasConfirmDefault := confirm.Props["default_value"]
	assert.False(t, hasPassDefault, "password must be cleared on rebuild")
	assert.False(t, hasConfirmDefault, "confirm_password must be cleared on rebuild")
	assert.Equal(t, true, submit.Props["disabled"])
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/register/...`
Expected: FAIL with build error "package internal/register does not exist" / "undefined BuildScreen".

- [ ] **Step 3: Create the builder**

Create `internal/register/builder.go`:

```go
package register

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

const (
	ScreenID    = "register"
	CardID      = "register-card"
	FormID      = "register-form"
	BannerID    = "register-banner"
	EmailID     = "register-email"
	PasswordID  = "register-password"
	ConfirmID   = "register-confirm-password"
	SubmitID    = "register-submit"
	LoginLinkID = "register-login-link"
	TitleID     = "register-title"
)

// BuildScreen builds the standalone register screen component tree.
// errorMsg empty means no banner is rendered.
func BuildScreen(lang string, errorMsg string) components.Component {
	form := BuildForm(lang, "", errorMsg, false)

	loginLink := components.Component{
		Type: "button",
		ID:   LoginLinkID,
		Props: map[string]any{
			"label": i18n.T(lang, "auth.login_link"),
			"style": "ghost",
		},
		Actions: []components.Action{components.Navigate("/screens/login")},
	}

	loginRow := components.RowWithGap("register-login-row", []string{"1fr", "auto", "auto"}, "sm",
		components.Column("register-login-row-spacer"),
		components.Text("register-have-prompt", i18n.T(lang, "auth.have_account_prompt"), "sm", "normal"),
		loginLink,
	)
	loginRow.Props["align_items"] = "center"

	appName := components.Text("register-app-name", i18n.T(lang, "app.name"), "xl", "bold")
	title := components.Text(TitleID, i18n.T(lang, "auth.register_title"), "lg", "normal")

	content := components.ColumnWithGap("register-content", "lg",
		appName,
		title,
		form,
		loginRow,
	)

	padded := components.Row("register-padded", []string{"40px", "360px", "40px"},
		components.Column("register-pad-left"),
		content,
		components.Column("register-pad-right"),
	)

	card := components.Card(CardID,
		components.Column("register-card-inner",
			components.Spacer("register-pad-top", "xl"),
			padded,
			components.Spacer("register-pad-bottom", "xl"),
		),
	)

	root := components.Column("register-root", card)
	root.Props["align_items"] = "center"
	root.Props["justify_items"] = "center"

	return components.Screen(ScreenID, i18n.T(lang, "auth.register_title"), root)
}

// BuildForm rebuilds just the form subtree. Used by the action handler to
// produce a `replace` payload for register-form on validation/error outcomes.
//
//	prefillEmail   — value to put back into the email input (passwords always cleared)
//	errorMsg       — when non-empty, an error banner is rendered above the inputs
//	submitDisabled — when true, the submit button has disabled: true (used for REGISTRATION_DISABLED)
func BuildForm(lang, prefillEmail, errorMsg string, submitDisabled bool) components.Component {
	emailInput := components.InputAdvanced(components.InputOptions{
		ID: EmailID, Name: "email", InputType: "email",
		Label:        i18n.T(lang, "auth.email_label"),
		Placeholder:  i18n.T(lang, "auth.email_placeholder"),
		DefaultValue: prefillEmail,
		Required:     true,
	})

	passwordInput := components.InputAdvanced(components.InputOptions{
		ID: PasswordID, Name: "password", InputType: "password",
		Label:       i18n.T(lang, "auth.password_label"),
		Placeholder: i18n.T(lang, "auth.password_placeholder"),
		Required:    true,
		MinLength:   8,
	})

	confirmInput := components.InputAdvanced(components.InputOptions{
		ID: ConfirmID, Name: "confirm_password", InputType: "password",
		Label:       i18n.T(lang, "auth.confirm_password_label"),
		Placeholder: i18n.T(lang, "auth.confirm_password_placeholder"),
		Required:    true,
		MatchField:  "password",
	})

	submitProps := map[string]any{"label": i18n.T(lang, "auth.register_submit")}
	if submitDisabled {
		submitProps["disabled"] = true
	}
	submit := components.Component{
		Type:    "button",
		ID:      SubmitID,
		Props:   submitProps,
		Actions: []components.Action{components.Submit("/actions/register", "POST", FormID)},
	}

	submitRow := components.Row("register-submit-row", []string{"1fr", "auto"},
		components.Column("register-submit-spacer"),
		submit,
	)

	stack := components.ColumnWithGap("register-form-stack", "lg")
	if errorMsg != "" {
		stack.Children = append(stack.Children, components.Component{
			Type:  "banner",
			ID:    BannerID,
			Props: map[string]any{"variant": "error", "text": errorMsg},
		})
	}
	stack.Children = append(stack.Children,
		components.ColumnWithGap("register-fields", "sm",
			emailInput,
			passwordInput,
			confirmInput,
		),
		submitRow,
	)

	return components.Form(FormID, stack)
}
```

- [ ] **Step 4: Run builder tests**

Run: `go test ./internal/register/... -v`
Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/register/builder.go internal/register/builder_test.go
git commit -m "feat(register): add screen builder"
```

---

## Task 5: Register screen GET handler

**Files:**
- Create: `internal/register/handler.go`
- Test: `internal/register/handler_test.go`

- [ ] **Step 1: Write the failing handler test**

Create `internal/register/handler_test.go`:

```go
package register

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandler_GetReturns200WithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/screens/register", NewHandler().Get)

	req := httptest.NewRequest(http.MethodGet, "/screens/register", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, `"id":"register"`))
	assert.False(t, strings.Contains(body, `"id":"nav_header"`), "shell slot must not be present")
}

func TestHandler_RespectsAcceptLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/screens/register", NewHandler().Get)

	req := httptest.NewRequest(http.MethodGet, "/screens/register", nil)
	req.Header.Set("Accept-Language", "es-AR,es;q=0.9")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Spanish title should appear somewhere in the body
	assert.True(t, strings.Contains(w.Body.String(), "Crear cuenta"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/register/... -run TestHandler -v`
Expected: FAIL — `NewHandler` undefined.

- [ ] **Step 3: Create the handler**

Create `internal/register/handler.go`:

```go
package register

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Get(c *gin.Context) {
	lang := parseLang(c)
	c.JSON(http.StatusOK, BuildScreen(lang, ""))
}

func parseLang(c *gin.Context) string {
	header := c.GetHeader("Accept-Language")
	if header == "" {
		return "en"
	}
	parts := strings.SplitN(header, ",", 2)
	lang := strings.SplitN(parts[0], "-", 2)[0]
	lang = strings.SplitN(lang, ";", 2)[0]
	return strings.TrimSpace(lang)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/register/... -v`
Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/register/handler.go internal/register/handler_test.go
git commit -m "feat(register): add screen GET handler"
```

---

## Task 6: Wire `GET /screens/register` in the server

**Files:**
- Modify: `internal/server/server.go` (around line 42, the public-routes block)
- Modify: `internal/server/server_test.go`

- [ ] **Step 1: Write the failing route test**

Append to `internal/server/server_test.go` (use existing helpers / patterns in the file):

```go
func TestPublicRoute_ScreensRegister(t *testing.T) {
	srv, _ := newTestServer(t) // use whatever bootstrap helper the file already exposes
	req := httptest.NewRequest(http.MethodGet, "/screens/register", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"register"`)
}
```

If `newTestServer` / `srv.Router()` are not the actual names in this file, adapt to whatever the existing tests use — read the file first.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/server/... -run TestPublicRoute_ScreensRegister -v`
Expected: FAIL with 404 on the route.

- [ ] **Step 3: Wire the route**

In `internal/server/server.go`:

(a) Add the import next to the existing `internal/login` import:

```go
"github.com/project/vk-investment-middleend/internal/register"
```

(b) Just below the existing `s.router.GET("/screens/login", ...)` line (currently line 42), add:

```go
s.router.GET("/screens/register", register.NewHandler().Get)
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/server/... -v`
Expected: PASS, including the new test and all pre-existing.

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go internal/server/server_test.go
git commit -m "feat(register): wire GET /screens/register"
```

---

## Task 7: Refactor `POST /actions/register` to ActionResponse pattern

**Files:**
- Modify: `internal/auth/register_handler.go`
- Modify: `internal/auth/register_handler_test.go`

The current handler returns flat HTTP 4xx envelopes for `409` / `403` and navigates to `/login` (wrong path). This task aligns it with the spec.

- [ ] **Step 1: Replace the handler tests**

Read `internal/auth/register_handler_test.go` to learn the existing helpers (mock client interface, etc.), then rewrite the file's test functions so that:

```go
package auth

// ... existing imports + any test-helper auth.Client mock already in the file

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeRegistrar struct {
	gotEmail, gotPassword string
	err                   error
}

func (f *fakeRegistrar) Register(_ context.Context, email, password string) error {
	f.gotEmail, f.gotPassword = email, password
	return f.err
}

func registerPost(t *testing.T, body any, reg registrar) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/actions/register", NewRegisterHandler(reg).Post)
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/actions/register", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRegister_Success_NavigatesToScreensLogin(t *testing.T) {
	w := registerPost(t, map[string]string{
		"email": "u@e.com", "password": "longenough", "confirm_password": "longenough",
	}, &fakeRegistrar{})

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "navigate", resp["action"])
	assert.Equal(t, "/screens/login", resp["target_id"])
}

func TestRegister_StripsConfirmPassword(t *testing.T) {
	reg := &fakeRegistrar{}
	registerPost(t, map[string]string{
		"email": "u@e.com", "password": "longenough", "confirm_password": "longenough",
	}, reg)
	assert.Equal(t, "u@e.com", reg.gotEmail)
	assert.Equal(t, "longenough", reg.gotPassword)
}

func TestRegister_PasswordTooShort_ReplaceForm(t *testing.T) {
	w := registerPost(t, map[string]string{
		"email": "u@e.com", "password": "short", "confirm_password": "short",
	}, &fakeRegistrar{})

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"action":"replace"`)
	assert.Contains(t, body, `"target_id":"register-form"`)
	assert.Contains(t, body, "Please check the form") // auth.error_validation in en
}

func TestRegister_Mismatch_ReplaceForm(t *testing.T) {
	w := registerPost(t, map[string]string{
		"email": "u@e.com", "password": "longenough", "confirm_password": "different1",
	}, &fakeRegistrar{})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"action":"replace"`)
}

func TestRegister_EmailAlreadyExists_ReplaceFormWithBanner(t *testing.T) {
	w := registerPost(t, map[string]string{
		"email": "u@e.com", "password": "longenough", "confirm_password": "longenough",
	}, &fakeRegistrar{err: ErrEmailAlreadyExists})

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"action":"replace"`)
	assert.Contains(t, body, "already exists")
}

func TestRegister_RegistrationDisabled_ReplaceFormDisabled(t *testing.T) {
	w := registerPost(t, map[string]string{
		"email": "u@e.com", "password": "longenough", "confirm_password": "longenough",
	}, &fakeRegistrar{err: ErrRegistrationDisabled})

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"action":"replace"`)
	assert.Contains(t, body, "disabled")
	assert.Contains(t, body, `"disabled":true`) // submit button disabled
}

func TestRegister_Transient_ActionNoneSnackbar(t *testing.T) {
	w := registerPost(t, map[string]string{
		"email": "u@e.com", "password": "longenough", "confirm_password": "longenough",
	}, &fakeRegistrar{err: errors.New("boom")})

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"action":"none"`)
	assert.True(t, strings.Contains(body, "snackbar"))
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/auth/... -run TestRegister -v`
Expected: most tests FAIL — handler still returns `/login`, flat 4xx envelopes, doesn't validate `confirm_password`, no `registrar` interface.

- [ ] **Step 3: Refactor the handler**

Replace the body of `internal/auth/register_handler.go`:

```go
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/register"
)

const minPasswordLen = 8

// registrar is the contract the handler depends on. *Client implements it.
type registrar interface {
	Register(ctx context.Context, email, password string) error
}

type RegisterHandler struct {
	reg registrar
}

func NewRegisterHandler(reg registrar) *RegisterHandler {
	return &RegisterHandler{reg: reg}
}

type registerRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (h *RegisterHandler) Post(c *gin.Context) {
	lang := parseLang(c)

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondReplace(c, lang, "", "auth.error_validation", false)
		return
	}
	email := strings.TrimSpace(req.Email)

	// Middleend-side defense in depth — these errors are also gated client-side.
	if email == "" || req.Password == "" || req.ConfirmPassword == "" {
		respondReplace(c, lang, email, "auth.error_validation", false)
		return
	}
	if len(req.Password) < minPasswordLen {
		respondReplace(c, lang, email, "auth.error_validation", false)
		return
	}
	if req.Password != req.ConfirmPassword {
		respondReplace(c, lang, email, "auth.error_validation", false)
		return
	}

	err := h.reg.Register(c.Request.Context(), email, req.Password)
	switch {
	case err == nil:
		fb := components.Snackbar("feedback", i18n.T(lang, "auth.register_success"), "success")
		c.JSON(http.StatusOK, components.ActionResponse{
			Action: "navigate", TargetID: "/screens/login", Feedback: &fb,
		})
	case errors.Is(err, ErrEmailAlreadyExists):
		respondReplace(c, lang, email, "auth.error_email_exists", false)
	case errors.Is(err, ErrRegistrationDisabled):
		respondReplace(c, lang, "", "auth.error_registration_disabled", true)
	default:
		fb := components.Snackbar("feedback", i18n.T(lang, "auth.error_transient"), "error")
		c.JSON(http.StatusOK, components.ActionResponse{
			Action: "none", Feedback: &fb,
		})
	}
}

func respondReplace(c *gin.Context, lang, prefillEmail, errorKey string, submitDisabled bool) {
	tree := register.BuildForm(lang, prefillEmail, i18n.T(lang, errorKey), submitDisabled)
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: register.FormID, Tree: &tree,
	})
}

func parseLang(c *gin.Context) string {
	header := c.GetHeader("Accept-Language")
	if header == "" {
		return "en"
	}
	parts := strings.SplitN(header, ",", 2)
	lang := strings.SplitN(parts[0], "-", 2)[0]
	lang = strings.SplitN(lang, ";", 2)[0]
	return strings.TrimSpace(lang)
}
```

If a `parseLang` already exists in package `auth` (collision), drop the local one and import from where it lives — read the auth package before adding to confirm.

- [ ] **Step 4: Verify wiring still compiles**

Confirm that `internal/server/server.go` line 46 — `auth.NewRegisterHandler(authClient)` — still compiles. `*auth.Client.Register` already implements `registrar`. If the server file currently passes `*Client` through a different exported type, adapt the constructor signature accordingly (read the file).

- [ ] **Step 5: Run tests**

Run: `go test ./internal/auth/... ./internal/register/... ./internal/server/... -v`
Expected: PASS.

- [ ] **Step 6: Run the full test suite**

Run: `go test ./...`
Expected: PASS (no regressions).

- [ ] **Step 7: Commit**

```bash
git add internal/auth/register_handler.go internal/auth/register_handler_test.go
git commit -m "refactor(auth): register submit returns ActionResponse, validates confirm_password"
```

---

## Task 8: Restart the dev server and smoke-test

- [ ] **Step 1: Restart the middleend**

```bash
lsof -ti:8082 | xargs -r kill -9 2>/dev/null; ./cli run
```

Run in background. Confirm `:8082` is up:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8082/health
```
Expected: `200`.

- [ ] **Step 2: Smoke `GET /screens/register`**

```bash
curl -s -H 'Accept-Language: en' http://localhost:8082/screens/register | head -c 400
curl -s -H 'Accept-Language: es' http://localhost:8082/screens/register | head -c 400
```

Expected: both return JSON with `"id":"register"` and the appropriate-language title (`Create account` / `Crear cuenta`). No `nav_header`, no `bottombar`, no `content_slot` strings in the body.

- [ ] **Step 3: Smoke `POST /actions/register` validation paths**

Validation (mismatch):

```bash
curl -s -X POST http://localhost:8082/actions/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","password":"longenough","confirm_password":"different1"}'
```

Expected: `200` body containing `"action":"replace"`, `"target_id":"register-form"`, and the `auth.error_validation` text.

(The success path is not smoke-tested here — it depends on the BE. Acceptance is via the unit tests in Task 7.)

- [ ] **Step 4: No commit (smoke only)**

---

## Self-Review

Spec coverage check:

- ✅ `min_length` + `match_field` props on `input` → Task 1.
- ✅ i18n keys → Task 2.
- ✅ Canonical screen spec → Task 3.
- ✅ Screen builder + tests → Task 4.
- ✅ Screen handler + tests → Task 5.
- ✅ Server wiring + test → Task 6.
- ✅ Submit handler aligned to `ActionResponse`, `confirm_password` validated and stripped, `/login` → `/screens/login` → Task 7.
- ✅ Smoke restart → Task 8.

Type/name consistency:

- `register.FormID` used in Task 4 (defined) and Task 7 (consumed) — match.
- `register.BuildForm(lang, prefillEmail, errorMsg, submitDisabled)` defined in Task 4, called from Task 7 with `i18n.T(lang, errorKey)` as the third arg — match (BuildForm takes a resolved string, not a key).
- `registrar` interface in Task 7 matches `*auth.Client.Register`'s existing signature `Register(ctx, email, password) error`.

No placeholders. No "TBD". No "similar to Task N".
