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
