# Register Screen — Design

Date: 2026-04-27
Branch: `feature/register-screen`

## Context

The login screen (`spec/screens/login.md`, `internal/login/`) already renders a button that navigates to `/screens/register`. The submit-action handler at `internal/auth/register_handler.go` is wired and proxies to backend `POST /v1/auth/register`. What's missing:

- The screen contract: `spec/screens/register.md`.
- The screen builder + handler: `internal/register/`.
- A small SDUI base extension to support client-side `min_length` and `match_field` validation on the `input` component.
- An alignment of the existing register submit handler with the project's `ActionResponse` conventions (currently it returns flat HTTP 4xx envelopes; profile-style cards return `200` with `replace` + inline banner).

## Goals

- Authenticated users with no account can self-register: email + password, with confirmation.
- Validation feedback is immediate (client-side) for the obvious cases; the server is authoritative on conflicts and BE-side rules.
- Visual and structural parity with the login screen so the two feel like one flow.
- Clean SDUI semantics throughout: no flat HTTP error envelopes for user-recoverable submit failures.

## Non-Goals

- Email verification flow.
- Captcha / rate-limit UI (BE may rate-limit; UI just relays the message).
- Forgot-password.
- Auto-login after register (BE returns `201` with no token; user is sent to `/screens/login`).
- Localization beyond `en` and `es` (project default).

## Approach

Three coordinated changes, one screen contract.

### 1. SDUI base extension — `input` props

Extend the `input` base component (in `spec/sdui-base-components.md`) with two new optional props:

| Prop | Type | Description |
|---|---|---|
| `min_length` | int | Minimum character count. Validated client-side on blur/change; values shorter than this block submission. Server-side handlers MAY also enforce. |
| `match_field` | string | Name of another input within the same `form`. Submission is blocked unless both fields hold identical values. Validated client-side on blur/change of either field. |

Both validations follow the existing convention used by `pattern`: client-side gate; on failed validation the form does not submit and the field is marked invalid. No new component types, no error-message props (the FE renders a default validation message; copy is via i18n by key — see § i18n below).

### 2. Register screen — contract

`GET /screens/register` renders a standalone screen mirroring login: centered card with logo, title, form (email + password + confirm), submit, and a row at the bottom with a prompt + link back to `/screens/login`. No shell slots are emitted (screen is reachable while unauthenticated, same as login). Public route — no JWT.

The form posts to `POST /actions/register`. The action handler returns a single `ActionResponse` shape for every outcome (success, validation error, BE error, transient error). The screen never returns a flat HTTP error envelope on the submit endpoint.

### 3. Register submit handler — alignment

Adjust `internal/auth/register_handler.go` to produce `ActionResponse` consistently:

| Outcome | Status | Action | Behavior |
|---|---|---|---|
| Success (`201` from BE) | `200` | `navigate` | TargetID `/screens/login`, success snackbar `auth.register_success`. |
| `409 EMAIL_ALREADY_EXISTS` | `200` | `replace` | Re-render `register-card` with email input pre-filled (password fields cleared) and inline error banner `auth.error_email_exists`. |
| `403 REGISTRATION_DISABLED` | `200` | `replace` | Re-render `register-card` with all inputs cleared, banner `auth.error_registration_disabled`, submit button disabled. |
| `400` from middleend (missing fields, mismatch) | `200` | `replace` | Re-render with banner `auth.error_validation`. Defense-in-depth — UI also gates this client-side. |
| BE 5xx / network / malformed JSON | `200` | `none` | Snackbar (error) `auth.error_transient`. Form values preserved. |
| Middleend internal error (panic, unexpected) | `502` | flat error | Standard `BACKEND_ERROR` envelope. Out of band; not user-recoverable. |

The fix to the existing `TargetID: "/login"` → `/screens/login` rides along.

## Layout

```
screen (id: register, no shell slots)
└─ column (root, align_items: center, justify_items: center)
   └─ card (id: register-card)
      └─ column
         ├─ spacer xl
         ├─ row [40px / 360px / 40px]
         │  └─ column (id: register-content, gap: lg)
         │     ├─ text app.name (xl bold)
         │     ├─ text auth.register_title (lg)
         │     ├─ form (id: register-form)
         │     │  └─ column (gap: lg)
         │     │     ├─ column (gap: sm)
         │     │     │  ├─ input email      (name: email,            type: email,    required, pattern: HTML email)
         │     │     │  ├─ input password   (name: password,         type: password, required, min_length: 8)
         │     │     │  └─ input confirm    (name: confirm_password, type: password, required, match_field: password)
         │     │     └─ row [1fr / auto]
         │     │        └─ button submit (auth.register_submit → POST /actions/register, target: register-form)
         │     └─ row [1fr / auto / auto] (gap: sm, align_items: center)
         │        ├─ text  auth.have_account_prompt (sm)
         │        └─ button auth.login_link (style: ghost, action: navigate /screens/login)
         └─ spacer xl
```

The `register-card` is the replace target; mutations from the action handler swap the card subtree, never the screen root.

## Form behavior

- Submit body sent by FE (collected from the form): `{ email, password, confirm_password }`.
- The middleend strips `confirm_password` before forwarding to BE; it forwards `{ email, password }` to `POST /v1/auth/register`.
- Defense-in-depth at the middleend on submit:
  - Trimmed `email` empty → `400` (translated to inline banner per the table above).
  - `password` shorter than 8 → `400`.
  - `confirm_password != password` → `400`.
- BE responses are translated per the table.

## i18n keys

Added under the `auth.*` namespace (en + es):

- `auth.register_title` — "Create account" / "Crear cuenta"
- `auth.register_submit` — "Create account" / "Crear cuenta"
- `auth.confirm_password_label` — "Confirm password" / "Confirmar contraseña"
- `auth.confirm_password_placeholder`
- `auth.have_account_prompt` — "Already have an account?" / "¿Ya tenés cuenta?"
- `auth.login_link` — "Log in" / "Iniciar sesión"
- `auth.register_success` — "Account created. Please log in." / "Cuenta creada. Iniciá sesión."
- `auth.error_email_exists` — "An account with this email already exists." / "Ya existe una cuenta con ese email."
- `auth.error_registration_disabled` — "Registration is currently disabled." / "El registro está deshabilitado."
- `auth.error_validation` — "Please check the form and try again." / "Revisá el formulario e intentá de nuevo."
- `auth.error_transient` — "Something went wrong. Please try again." / "Hubo un problema. Intentá de nuevo."

Reused from existing `auth.*`:
- `auth.email_label`, `auth.email_placeholder`, `auth.password_label`, `auth.password_placeholder`, `app.name`.

## Public-route configuration

`GET /screens/register` and `POST /actions/register` are already public per `spec/security.md`. No routing/auth changes needed — only ensuring the new GET handler is registered on the public router group.

`REGISTRATION_ENABLED` is **not** consulted by the middleend GET; the form always renders. The disabled state is surfaced only on submit (via the BE 403). This matches the existing principle "the middleend does not duplicate the check".

## Out of scope (non-goals reaffirmed)

- Email verification, "send me a confirmation" link, or any post-registration step beyond the redirect to login.
- Auto-login: BE returns `201` with no token, so we cannot auto-login without a second BE call. Explicitly punted.
- Password-strength meter beyond min length 8.

## Acceptance criteria

### Screen GET (`GET /screens/register`)

- [ ] Returns `200` without any `Authorization` header.
- [ ] Body is a `screen` with `id: register` and no shell slot components.
- [ ] Tree contains a card (`register-card`) with: app-name text, title text, form (`register-form`), submit button, and a row with prompt + login link.
- [ ] Form contains exactly three inputs: `email` (type `email`, required), `password` (type `password`, required, `min_length: 8`), `confirm_password` (type `password`, required, `match_field: "password"`).
- [ ] Submit button action: `{trigger: click, type: submit, endpoint: /actions/register, method: POST, target_id: register-form}`.
- [ ] Login-link button action: `{trigger: click, type: navigate, url: /screens/login, target: self}`.
- [ ] Root container is centered both axes via shared alignment props.
- [ ] All user-facing strings resolve via i18n; `Accept-Language: es` returns Spanish.
- [ ] Response contains no `nav_header`, `nav_main`, `nav_footer`, `bottombar`, or `content_slot`.

### Submit (`POST /actions/register`)

- [ ] Success path: BE `201` → `200 ActionResponse{action: navigate, target_id: /screens/login, feedback: success snackbar (auth.register_success)}`.
- [ ] Email conflict: BE `409 EMAIL_ALREADY_EXISTS` → `200 ActionResponse{action: replace, target_id: register-card, ...}` with `auth.error_email_exists` banner; password fields cleared, email pre-filled.
- [ ] Disabled: BE `403 REGISTRATION_DISABLED` → `200 ActionResponse{action: replace, ...}` with `auth.error_registration_disabled` banner; submit disabled.
- [ ] Mismatch / missing fields detected at middleend → `200 ActionResponse{action: replace, ...}` with `auth.error_validation` banner; FE also gates client-side via `min_length` + `match_field`.
- [ ] BE 5xx / network error → `200 ActionResponse{action: none, feedback: snackbar error (auth.error_transient)}`. Form values preserved.
- [ ] Forwarded body to BE never includes `confirm_password`.

### SDUI base

- [ ] `spec/sdui-base-components.md` documents `min_length` (int) and `match_field` (string) on the `input` component.
- [ ] Component constructor exposes the new props (Go side); existing `InputFull` stays binary-compatible (overload or builder).
- [ ] Existing inputs in other screens unchanged (no new validation introduced where not specified).
