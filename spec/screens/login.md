# Login Screen

Standalone SDUI screen (no shell) that renders the login form. Public — reachable without authentication.

## Endpoint

| Method | Path              | Auth | Description                                    |
|--------|-------------------|------|------------------------------------------------|
| GET    | `/screens/login`  | no   | Returns the login screen component tree        |

Headers read:
- `Accept-Language` — BCP 47 tag. Missing or unsupported → `en`.

No `X-Platform` adaptation: the form has the same structure on every platform.

## Component tree

```
screen id=login
  column id=login-root                         (align+justify center, fills viewport)
    card id=login-card                         (max-width constraint applied by the frontend style)
      column id=login-content (gap 16px)
        image id=login-logo                    (src /logo.svg, alt app.name)
        text  id=login-title                   (auth.login_title, size=xl, weight=bold)
        form  id=login-form
          column id=login-fields (gap 12px)
            input  id=login-email              (type=email, name=email, required, label+placeholder)
            input  id=login-password           (type=password, name=password, required, label+placeholder)
            button id=login-submit             (submit /actions/login POST login-form)
        row id=register-row
          text   id=register-prompt            (auth.no_account_prompt)
          button id=register-link              (navigate /screens/register, variant=link)
```

Centering: `login-root` is a `column` whose `Props` set `align_items: "center"` and `justify_items: "center"`. The frontend is expected to render the enclosing `screen` at viewport size so the column fills the available space.

## Actions

- Submit button carries a `submit` action: `endpoint=/actions/login`, `method=POST`, `target_id=login-form`. The form's inputs (`email`, `password`) are collected by the frontend and sent as JSON.
- Register link carries a `navigate` action to `/screens/register`.

## i18n keys

Added to `locales/<lang>.json`:

| Key | en | es |
|---|---|---|
| `auth.login_title` | Log in | Iniciar sesión |
| `auth.email_label` | Email | Correo |
| `auth.email_placeholder` | you@example.com | vos@ejemplo.com |
| `auth.password_label` | Password | Contraseña |
| `auth.password_placeholder` | Your password | Tu contraseña |
| `auth.submit` | Log in | Ingresar |
| `auth.no_account_prompt` | Don't have an account? | ¿No tenés cuenta? |
| `auth.register_link` | Sign up | Registrarme |

## Package layout

Mirrors `internal/home/` but without a backend call (the screen is static):

| File | Responsibility |
|---|---|
| `internal/login/builder.go` | `BuildScreen(lang string) components.Component` — builds the tree |
| `internal/login/handler.go` | `GET` handler; reads `Accept-Language`; returns the tree |
| `internal/login/builder_test.go` | Unit tests covering the acceptance criteria |

No `client.go` or `get_usecase.go` — this screen does not call the backend.

## Server wiring

Public route registered alongside `/health`, `/actions/login`, `/actions/register` — not under the protected group.

## Submit response handling

Out of scope for this screen. `POST /actions/login` (already implemented) returns:
- On success: `ActionResponse{action: navigate, target_id: /screens/home}.WithAuth(token, expires_at)` with a success snackbar. The navigate target is the default content screen (home today, portfolio once it exists); the shell is fetched independently by the frontend.
- On failure: `ActionResponse{action: none, feedback: error snackbar}`.

The frontend reads `auth` to persist the JWT and applies the `navigate` instruction.

## Out of scope

- Forgot-password flow — the backend does not expose it.
- "Remember me" option — tokens use the backend's TTL.
- Redirect when already authenticated — the endpoint is public and always returns the form. The frontend skips fetching it if a valid token is already held.

## Acceptance criteria

- [ ] `GET /screens/login` returns 200 without any `Authorization` header.
- [ ] The response is a component with `type: screen`, `id: login`.
- [ ] The tree contains: a `card` with a logo `image`, a title `text`, a `form` with one email `input` and one password `input` (both `required: true`), a submit `button`.
- [ ] The submit button's single action is `{trigger: click, type: submit, endpoint: /actions/login, method: POST, target_id: login-form}`.
- [ ] The tree contains a register `button` whose single action is `{trigger: click, type: navigate, url: /screens/register, target: self}`.
- [ ] Every user-facing string resolves via i18n — no hardcoded literals in the response.
- [ ] `Accept-Language: es` returns Spanish labels; unknown language falls back to English.
- [ ] The tree does NOT contain shell slots (`nav_header`, `nav_main`, `nav_footer`, `bottombar`, `content_slot`).
