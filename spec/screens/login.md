# Login Screen

Entry point for unauthenticated users. The only screen on the BFF that does **not** live behind the shell — it renders standalone, without the nav sidebar or bottom bar, because the user has no session yet.

## Purpose

Collect an email and password, submit them to the authentication action, and either navigate the user into the app (on success, with a persisted JWT) or show a form error (on failure). Also provide a path to registration.

## Endpoint

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET` | `/screens/login` | no | Returns the login screen component tree. Public — reachable without any token. |

Headers read:
- `Accept-Language` — BCP-47 tag. Missing or unsupported → `en`.

`X-Platform` is **not** consulted — the form has the same shape on every platform. No platform-specific branching here.

## Layout

A centered card with the app logo, a title, two fields (email + password), and a submit button, followed by a short prompt that links to the register screen. Centering is handled by the `screen → column` container using the SDUI shared alignment props (`align_items` / `justify_items` set to `center`), so the card sits in the middle of the viewport regardless of screen size.

No shell slots are emitted (no `nav_header`, `nav_main`, `nav_footer`, `bottombar`, `content_slot`). The screen is self-contained.

## Form behavior

- Two inputs: `email` (type `email`, required) and `password` (type `password`, required). Both have localized labels and placeholders.
- A single submit button carries a `submit` action to `POST /actions/login` with `target_id` pointing at the form so the frontend collects the two fields into a JSON body (`{email, password}`).
- Submit response (handled by `POST /actions/login`, out of scope for this screen):
  - Success → `ActionResponse{action: navigate, target_id: "/screens/portfolio"}` with a success feedback snackbar, and an `auth` payload carrying the new JWT + its expiry. The frontend reads `auth` and persists the token before applying the navigation instruction.
  - Failure → `ActionResponse{action: none, feedback: <error snackbar>}`. The user stays on the login screen. No navigation happens.

## Register link

A small row under the form: a prompt text plus a button that navigates to `/screens/register` (client-side navigation, no round-trip to the submit action).

## i18n keys

All user-facing strings resolve from the `auth.*` namespace. Concrete strings live in `locales/en.json` / `locales/es.json`. Missing-key fallback: `en`, then the key itself.

- `auth.login_title`
- `auth.email_label`, `auth.email_placeholder`
- `auth.password_label`, `auth.password_placeholder`
- `auth.submit`
- `auth.no_account_prompt`
- `auth.register_link`

## Out of scope

- **Forgot-password flow** — the backend does not expose it.
- **"Remember me" option** — token TTL is backend-controlled.
- **Redirect when already authenticated** — this endpoint is public and always returns the form; the frontend is expected to skip requesting it when it already holds a valid token.

## Acceptance criteria

- [ ] `GET /screens/login` returns `200` without any `Authorization` header.
- [ ] The response is a `screen` with `id: login`.
- [ ] The tree contains a card with an app logo, a title, a form with an email input and a password input (both `required: true`), and a submit button.
- [ ] The submit button's single action is `{trigger: "click", type: "submit", endpoint: "/actions/login", method: "POST", target_id: <the form's id>}`.
- [ ] The tree contains a register link button whose single action is `{trigger: "click", type: "navigate", url: "/screens/register", target: "self"}`.
- [ ] The root container is centered both horizontally and vertically inside the viewport via shared alignment props on a column.
- [ ] Every user-facing string resolves via i18n — no hardcoded literals. `Accept-Language: es` returns Spanish labels; unknown languages fall back to English.
- [ ] The response contains no shell slot components (`nav_header`, `nav_main`, `nav_footer`, `bottombar`, `content_slot`).
