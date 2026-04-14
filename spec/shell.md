# Shell

The app shell is the persistent frame the frontend fetches once after authentication. It contains navigation slots and a content placeholder where screens render.

## Endpoint

| Method | Path     | Description                          | Response         |
|--------|----------|--------------------------------------|------------------|
| GET    | `/shell` | Returns the shell component tree for the current platform and language. | Component tree |

Headers read:
- `X-Platform` — `web | web_mobile | android | ios`. Missing or unknown → `web`.
- `Accept-Language` — BCP 47 tag. Missing or unsupported → `en`.

Auth: required (JWT passthrough).

## Platform adaptation

| Platform       | Nav type    | Slots rendered                                            |
|----------------|-------------|-----------------------------------------------------------|
| `web`          | `sidebar`   | `nav_header`, `nav_main`, `nav_footer`, `content_slot`    |
| `web_mobile`   | `bottombar` | `nav_header`, `content_slot`, `bottombar`                 |
| `android`      | `bottombar` | `content_slot`, `bottombar`                               |
| `ios`          | `bottombar` | `content_slot`, `bottombar`                               |

## Navigation items

Six entries, same order on every platform. Each item navigates (SDUI `navigate` action) to the corresponding screen endpoint.

| ID          | Label key         | Icon         | Target route              |
|-------------|-------------------|--------------|---------------------------|
| `portfolio` | `nav.portfolio`   | `pie-chart`  | `/screens/portfolio`      |
| `assets`    | `nav.assets`      | `coins`      | `/screens/assets`         |
| `trades`    | `nav.trades`      | `arrow-swap` | `/screens/trades`         |
| `snapshots` | `nav.snapshots`   | `camera`     | `/screens/snapshots`      |
| `import`    | `nav.import`      | `upload`     | `/screens/import`         |
| `analysis`  | `nav.analysis`    | `sparkles`   | `/screens/analysis`       |

## Slot contents

- **`nav_header`** (`web`, `web_mobile`): app logo + app name.
- **`nav_main`** (`web` only): the six nav items as a vertical list.
- **`nav_footer`** (`web` only): logout button (SDUI `logout` action).
- **`bottombar`** (`web_mobile`, `android`, `ios`): the six nav items as a horizontal list. Logout is owned by the Portfolio screen header on these platforms — out of shell scope.
- **`content_slot`**: always present. ID `content`.

## i18n

All user-facing strings resolve from `locales/<lang>.json` via label keys. Supported locales: `en`, `es`. Fallback: `en`.

Keys introduced by the shell:
- `app.name`
- `nav.portfolio`, `nav.assets`, `nav.trades`, `nav.snapshots`, `nav.import`, `nav.analysis`
- `nav.logout`

## Acceptance criteria

- [ ] `GET /shell` with `X-Platform: web` returns `nav_type: sidebar` and the four slots (`nav_header`, `nav_main`, `nav_footer`, `content_slot`).
- [ ] `GET /shell` with `X-Platform: web_mobile` returns `nav_type: bottombar` with `nav_header`, `content_slot`, `bottombar`.
- [ ] `GET /shell` with `X-Platform: android` or `ios` returns `nav_type: bottombar` with `content_slot` and `bottombar` (no header).
- [ ] Missing or unknown `X-Platform` defaults to `web`.
- [ ] Every nav item is present exactly once and carries a `navigate` action pointing to its screen route.
- [ ] All labels resolve via i18n — no hardcoded literals in the response. `Accept-Language: es` returns Spanish labels; unknown language falls back to English.
- [ ] `nav_footer` on `web` contains a logout control that carries the SDUI `logout` action.
- [ ] Unauthenticated requests to `/shell` return 401.
