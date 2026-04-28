# Profile Screen

User profile management. Lets the authenticated user view and update their display name and default currency, change their email, change their password, and permanently delete their account.

## Purpose

Single screen with four independent sections:

1. **Profile** — display name + default currency preference.
2. **Email** — change email (requires current password).
3. **Password** — change password (requires current password + new password confirmation).
4. **Danger Zone** — permanently delete account (requires password confirmation, opens a modal).

Each section is its own card with its own form and Save button. Submits replace only the affected card, except `delete_account` which logs the user out and redirects to login.

## Endpoints

All endpoints are **protected** (JWT). Missing / invalid / expired JWT on the screen GET, or a `401` from the BE on the screen GET, returns `401 {"error":"unauthorized","redirect":"/screens/login"}`. Backend 5xx / network / malformed JSON returns `502 BACKEND_ERROR`. Invalid body on a middleend endpoint returns `400 BAD_REQUEST`.

| Method | Path | Purpose |
|---|---|---|
| `GET`  | `/screens/profile` | Full screen render. Parallel `GET /v1/user/me` + `GET /v1/config`. |
| `POST` | `/actions/profile/update` | Profile section save. Calls `PATCH /v1/user/me`. |
| `POST` | `/actions/profile/update_email` | Email section save. Calls `PATCH /v1/user/me/email`. |
| `POST` | `/actions/profile/change_password` | Password section save. Calls `POST /v1/user/me/password`. |
| `GET`  | `/actions/profile/delete_modal` | Returns the delete-account confirmation modal into the modal slot. |
| `POST` | `/actions/profile/delete_account` | Submits the modal. Calls `DELETE /v1/user/me`. |

Every non-screen endpoint returns an `ActionResponse`. Mutation endpoints accept JSON (`Content-Type: application/json`).

### Validation-error handling — the 401-from-BE rule

`PATCH /v1/user/me/email`, `POST /v1/user/me/password`, and `DELETE /v1/user/me` return `401 INVALID_CREDENTIALS` when the supplied `current_password` / `password` is wrong. This is **not** session expiration — it is a validation error. The middleend translates these BE-401s into a normal `200 ActionResponse` with a `replace` of the affected card or modal and an inline error banner. Only a `401` originating from the middleend itself (token absent / invalid / expired) or a BE-401 on `GET /v1/user/me` during the screen GET counts as a session-expiration `401` and triggers the documented redirect.

## Backend dependencies

- `GET /v1/user/me` — user profile. Critical path on screen GET. Response: `{ id, email, display_name?, preferences:{ default_currency? }, created_at }`.
- `GET /v1/config` — for the `default_currency` select options. Critical path on screen GET (without it the select cannot render). Response includes `currencies: string[]`.
- `PATCH /v1/user/me` — profile update. Body: `{ display_name?, preferences?:{ default_currency? } }`. Errors: `INVALID_DISPLAY_NAME` (422), `INVALID_CURRENCY` (422).
- `PATCH /v1/user/me/email` — email update. Body: `{ new_email, current_password }`. Errors: `MISSING_FIELDS` (400), `INVALID_CREDENTIALS` (401, see rule above), `EMAIL_ALREADY_EXISTS` (409).
- `POST /v1/user/me/password` — password change. Body: `{ current_password, new_password }`. Returns `204`. Errors: `MISSING_FIELDS` (400), `INVALID_CREDENTIALS` (401, see rule above), `INVALID_PASSWORD` (422).
- `DELETE /v1/user/me` — delete account. Body: `{ password }`. Returns `204`. Errors: `MISSING_FIELDS` (400), `INVALID_CREDENTIALS` (401, see rule above).

## Layout

A vertical column of four cards plus an empty modal slot. From top to bottom:

1. **Screen header** — title `profile.title`.
2. **`profile-card`** — Profile section.
3. **`email-card`** — Email section.
4. **`password-card`** — Password section.
5. **`danger-card`** — Danger Zone (visually differentiated; destructive variant).
6. **`profile-modal-slot`** — empty by default; the delete confirmation modal renders here.

Each card is identified by a stable `id` so its action endpoint can target only its own subtree on `replace`. The modal slot is a sibling of the cards: card mutations never touch it; the modal lifecycle is owned by `delete_modal` and `delete_account`.

There is no empty state — an authenticated user always has a `User` record. There is no loading / live / hide-values / pagination concern on this screen. No cell carries `sensitive: true` (no monetary values).

## Sections

### Profile card (`profile-card`)

**Inputs:**
- `display_name` — text, label `profile.display_name`, placeholder `profile.display_name_placeholder`, `defaultValue = me.display_name ?? ""`, `maxLength: 100`, not required.
- `default_currency` — select, label `profile.default_currency`. Options: a first item `"— None —"` (`profile.default_currency_none`) with empty value, followed by every entry in `config.currencies` in BE order. `defaultValue = me.preferences.default_currency ?? ""`, not required.
- `[ Save ]` button — primary, submits the form to `POST /actions/profile/update`.

**Submit body** (`POST /actions/profile/update`) — flat, mirrors form input names:
```json
{ "display_name": <string>, "default_currency": <string> }
```

Mapping rules:
- Each string field is trimmed before validation.
- Trimmed `display_name == ""` → forwarded as `null` (clear).
- `default_currency == ""` → forwarded as `null` (clear).

**Forwarded to BE:** `PATCH /v1/user/me` with the BE-expected nested shape — the middleend lifts `default_currency` under `preferences`:
```json
{ "display_name": <string|null>, "preferences": { "default_currency": <string|null> } }
```

**Responses:**
- `200` from BE → `ActionResponse{ replace profile-card }` re-rendering the card with values from the BE response, plus `feedback: snackbar success` (`profile.update.success`).
- `422 INVALID_DISPLAY_NAME` → `ActionResponse{ replace profile-card }`. Inputs preserved with the user's submitted values. Inline error banner (first child of the card, above the form) using `profile.update.error.invalid_display_name`.
- `422 INVALID_CURRENCY` → same shape, banner `profile.update.error.invalid_currency`.

### Email card (`email-card`)

**Display strip** (always visible, above the form): a muted text reading `profile.email.current` with `{email}` interpolated to `me.email`.

**Inputs:**
- `new_email` — email type, label `profile.email.new`, required, no `defaultValue`.
- `current_password` — password type, label `profile.email.current_password`, required.
- `[ Save ]` button — primary, submits to `POST /actions/profile/update_email`.

**Submit body** (`POST /actions/profile/update_email`):
```json
{ "new_email": <string>, "current_password": <string> }
```

**Forwarded to BE:** `PATCH /v1/user/me/email`.

**Responses:**
- `200` from BE → `ActionResponse{ replace email-card }` re-rendering the card with the new email visible in the `Current: {email}` strip and **all inputs cleared**, plus snackbar `profile.email.success`.
- `400 MISSING_FIELDS` / `401 INVALID_CREDENTIALS` / `409 EMAIL_ALREADY_EXISTS` → `ActionResponse{ replace email-card }`. `new_email` preserved with the user's submitted value; `current_password` **always cleared** (passwords are never echoed back, even the "current" one). Inline banner above the form, key per the table below.

### Password card (`password-card`)

**Inputs:**
- `current_password` — password type, label `profile.password.current`, required.
- `new_password` — password type, label `profile.password.new`, required, `maxLength: 128`.
- `confirm_password` — password type, label `profile.password.confirm`, required.
- `[ Save ]` button — primary, submits to `POST /actions/profile/change_password`.

**Submit body** (`POST /actions/profile/change_password`):
```json
{ "current_password": <string>, "new_password": <string>, "confirm_password": <string> }
```

**Middleend-side validation, in order, before any BE call:**
1. Any of the three fields empty → `ActionResponse{ replace password-card }` with all inputs cleared and inline banner `profile.password.error.missing_fields`. No BE call.
2. `new_password != confirm_password` → `ActionResponse{ replace password-card }` with all inputs cleared and inline banner `profile.password.error.do_not_match`. No BE call.

If both pass, the middleend calls `POST /v1/user/me/password` with `{ current_password, new_password }` (the `confirm_password` field is dropped before forwarding).

**Responses:**
- `204` from BE → `ActionResponse{ replace password-card }` with **all three inputs cleared**, plus snackbar `profile.password.success`.
- `400 MISSING_FIELDS` / `401 INVALID_CREDENTIALS` / `422 INVALID_PASSWORD` → `ActionResponse{ replace password-card }` with **all three inputs cleared** (passwords are never echoed back, even on error), inline banner per the table below.

### Danger Zone (`danger-card` + `profile-modal-slot`)

**Card display:**
- Heading `profile.danger.title` rendered with `color: error`.
- Body text `profile.danger.body`.
- `[ Delete account ]` button — destructive variant. Action: `GET /actions/profile/delete_modal`.

**`GET /actions/profile/delete_modal`** → `ActionResponse{ replace profile-modal-slot }` with a `ModalFull`:
- `title`: `profile.danger.modal.title`.
- `dismissible: true`.
- Body: warning text `profile.danger.modal.body` followed by a `Form`:
  - `password` — password type, label `profile.danger.modal.password_label`, required.
  - `[ Cancel ]` button — closes the modal (consistent with `dismissible: true`; the implementation may dismiss client-side or emit an empty `replace` of `profile-modal-slot`).
  - `[ Delete account ]` button — destructive, submits the form to `POST /actions/profile/delete_account`.

**`POST /actions/profile/delete_account`:**

Body:
```json
{ "password": <string> }
```

Forwarded to `DELETE /v1/user/me` with the same body.

**Responses:**
- `204` from BE → `ActionResponse{ action: "logout", redirect: "/screens/login" }`. No snackbar — the redirect is the feedback.
  - The `logout` action is the new SDUI primitive added by this screen: the frontend clears the stored auth token and navigates to `redirect`.
- `400 MISSING_FIELDS` / `401 INVALID_CREDENTIALS` → `ActionResponse{ replace profile-modal-slot }` re-rendering the modal with `password` cleared and an inline banner inside the modal body, key per the table below.

## Cross-section rules

- **Card isolation** — every mutation `replace`s only its own card subtree. `update_email`'s success path also refreshes the `Current: {email}` strip, but that strip lives inside `email-card`, so the rule still holds. No mutation ever touches a sibling card.
- **Modal slot ownership** — the modal slot is opened only by `delete_modal` and closed by `delete_account` success (via `logout` + `redirect`) or by Cancel (empty `replace`). No card mutation touches the slot.
- **Password echo-back** — every input of `type: password` is always returned empty in any re-emission of a card or modal (success or error).
- **Plaintext error banners** — banners are mounted as the first child of the affected card or modal, before the form. The visual treatment is `text` with `color: error` (concretized in implementation; the spec asserts "inline red banner above the form").
- **JSON bodies** — every `POST /actions/profile/*` accepts a JSON body. Malformed JSON returns `400 BAD_REQUEST`.
- **No persisted state** — the screen has no query params; every `GET /screens/profile` renders a clean tree (modal closed, forms reset to current BE values).
- **HideValues / sensitive** — not applicable. No `text` on this screen carries `sensitive: true`.

## i18n keys

Namespace `profile.*`. Languages `en` and `es`. Missing-key fallback: `en`, then the key itself.

### Screen
`profile.title`

### Profile card
`profile.section.profile`,
`profile.display_name`, `profile.display_name_placeholder`,
`profile.default_currency`, `profile.default_currency_none`,
`profile.update.save`,
`profile.update.success`,
`profile.update.error.invalid_display_name`,
`profile.update.error.invalid_currency`.

### Email card
`profile.section.email`,
`profile.email.current` (with `{email}` interpolation),
`profile.email.new`,
`profile.email.current_password`,
`profile.email.save`,
`profile.email.success`,
`profile.email.error.missing_fields`,
`profile.email.error.invalid_credentials`,
`profile.email.error.email_exists`.

### Password card
`profile.section.password`,
`profile.password.current`,
`profile.password.new`,
`profile.password.confirm`,
`profile.password.save`,
`profile.password.success`,
`profile.password.error.missing_fields`,
`profile.password.error.do_not_match`,
`profile.password.error.invalid_credentials`,
`profile.password.error.invalid_password`.

### Danger Zone
`profile.danger.title`,
`profile.danger.body`,
`profile.danger.delete_button`,
`profile.danger.modal.title`,
`profile.danger.modal.body`,
`profile.danger.modal.password_label`,
`profile.danger.modal.cancel`,
`profile.danger.modal.confirm`,
`profile.danger.error.missing_fields`,
`profile.danger.error.invalid_credentials`.

Where `common.*` already provides Save / Cancel labels, the screen reuses them; the keys above are the screen's owned set.

## Error handling

| Situation | HTTP from middleend | Body |
|---|---|---|
| Screen GET: missing/invalid/expired JWT, or BE returns 401 on `/v1/user/me` | 401 | `{"error":"unauthorized","redirect":"/screens/login"}` |
| Screen GET: BE 5xx / network / malformed on `/v1/user/me` or `/v1/config` | 502 | `{"error":{"code":"BACKEND_ERROR","message":"..."}}` |
| Mutation: BE returns a known validation code (`INVALID_DISPLAY_NAME`, `INVALID_CURRENCY`, `MISSING_FIELDS`, `INVALID_CREDENTIALS`, `EMAIL_ALREADY_EXISTS`, `INVALID_PASSWORD`) | 200 | `ActionResponse` with `replace` of the affected card or modal and inline banner |
| Mutation: BE 5xx / network / malformed | 502 | `{"error":{"code":"BACKEND_ERROR","message":"..."}}` |
| Mutation: malformed JSON body, non-JSON `Content-Type`, missing required JSON field shape | 400 | `{"error":{"code":"BAD_REQUEST","message":"..."}}` |

### Mutation error → i18n banner key

| Endpoint | BE error code | Banner key |
|---|---|---|
| `update` | `INVALID_DISPLAY_NAME` | `profile.update.error.invalid_display_name` |
| `update` | `INVALID_CURRENCY` | `profile.update.error.invalid_currency` |
| `update_email` | `MISSING_FIELDS` | `profile.email.error.missing_fields` |
| `update_email` | `INVALID_CREDENTIALS` | `profile.email.error.invalid_credentials` |
| `update_email` | `EMAIL_ALREADY_EXISTS` | `profile.email.error.email_exists` |
| `change_password` | (middleend) any field empty | `profile.password.error.missing_fields` |
| `change_password` | (middleend) `new_password != confirm_password` | `profile.password.error.do_not_match` |
| `change_password` | `MISSING_FIELDS` | `profile.password.error.missing_fields` |
| `change_password` | `INVALID_CREDENTIALS` | `profile.password.error.invalid_credentials` |
| `change_password` | `INVALID_PASSWORD` | `profile.password.error.invalid_password` |
| `delete_account` | `MISSING_FIELDS` | `profile.danger.error.missing_fields` |
| `delete_account` | `INVALID_CREDENTIALS` | `profile.danger.error.invalid_credentials` |

## Acceptance criteria

- [ ] `GET /screens/profile` without a valid JWT returns `401` with the documented redirect.
- [ ] With a valid JWT the middleend issues `GET /v1/user/me` and `GET /v1/config` in parallel, forwarding `Authorization`.
- [ ] The response is a `screen` with `id: profile` and `props.title` resolved per `Accept-Language`.
- [ ] The screen tree contains, in order: `profile-card`, `email-card`, `password-card`, `danger-card`, and a sibling `profile-modal-slot` that is empty on initial render.
- [ ] `profile-card`: inputs `display_name` (text, max 100, defaultValue from BE) and `default_currency` (select with first item `"— None —"` and the `config.currencies` list, defaultValue from BE).
- [ ] `email-card`: a `Current: {email}` strip is visible above the form; inputs are `new_email` (email, required) and `current_password` (password, required).
- [ ] `password-card`: three password inputs (`current_password`, `new_password`, `confirm_password`), all required.
- [ ] `danger-card`: heading rendered with `color: error`, body text, and a destructive-variant `Delete account` button that fires `GET /actions/profile/delete_modal`.
- [ ] **Profile update**: `POST /actions/profile/update` accepts a JSON body. Empty trimmed strings are forwarded as `null`. Replaces only the `profile-card`. Snackbar success on `200`. Inline banner on `INVALID_DISPLAY_NAME` / `INVALID_CURRENCY`.
- [ ] **Email update**: `POST /actions/profile/update_email` replaces only the `email-card`. Success: `Current: {email}` reflects the new email and inputs are cleared. Error: `new_email` preserved, `current_password` cleared, inline banner per the i18n table.
- [ ] **Password change**: `POST /actions/profile/change_password` validates middleend-side first — empty fields or mismatched `new_password`/`confirm_password` short-circuit without a BE call and return an inline banner with all three inputs cleared. On success or BE error, all three inputs are cleared.
- [ ] **Delete modal**: `GET /actions/profile/delete_modal` replaces `profile-modal-slot` with a `ModalFull` containing a single-input password form. Cancel closes the modal.
- [ ] **Delete account**: `POST /actions/profile/delete_account` calls `DELETE /v1/user/me` with `{ password }`. On `204`, returns an `ActionResponse` whose action is `logout` and `redirect: /screens/login`, with no snackbar. On error, replaces the modal with `password` cleared and an inline banner.
- [ ] Validation errors from BE (including the BE's `401 INVALID_CREDENTIALS` on email/password/delete) are returned as `200 ActionResponse` with a partial `replace`. Only middleend-side `401`s and the BE-`401` on `GET /v1/user/me` produce a session-expiration `401` with redirect.
- [ ] BE 5xx / network / malformed JSON in any path returns `502 BACKEND_ERROR`.
- [ ] Every card has a stable `id`; only its own action replaces it. The modal slot is a sibling and is never touched by card mutations.
- [ ] Every input of `type: password` is empty in every re-emission of a card or modal (success or error).
- [ ] All user-facing strings resolve via i18n `en`/`es`; no hardcoded literals in the response.
- [ ] No `text` on this screen carries `sensitive: true`.
- [ ] Every `GET /screens/profile` renders the modal closed and the forms populated with the current BE values.
