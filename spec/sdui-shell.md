# SDUI Shell — Go Reference

The app shell is the persistent application frame: navigation, layout regions, and user context. Screens render inside the shell's content slot.

---

## 1. Endpoint

```
GET /shell
Headers: Authorization: Bearer <token>, X-Platform: <platform>, Accept-Language: <lang>
```

Fetched once after authentication. The frontend caches and reuses it for the session.

---

## 2. Nav Types

The `nav_type` prop on the shell screen tells the frontend how to arrange slots.

| Nav type | Description |
|----------|-------------|
| `sidebar` | Persistent side navigation (collapsible) |
| `header_footer` | Top header + bottom footer navigation |
| `header_only` | Top header navigation only |
| `bottombar` | Fixed bottom bar navigation |
| `burger` | Collapsed navigation behind a hamburger menu |

---

## 3. Named Slots

| Slot | Purpose |
|------|---------|
| `nav_header` | Top zone: logo, user context, search, any component |
| `nav_main` | Primary navigation: typically `nav_item` components, accepts any component |
| `nav_footer` | Bottom zone of nav: secondary actions, logout |
| `bottombar` | Fixed bottom bar (independent of nav) |
| `content_slot` | Where the current screen renders |

Each slot is a generic container — it accepts any children.

---

## 4. Platform Adaptation

The middleend reads `X-Platform` and selects the appropriate `nav_type`.

| Platform | Supported nav types |
|----------|---------------------|
| `web` | `sidebar`, `header_footer`, `header_only` |
| `web_mobile` | `bottombar`, `sidebar`, `burger` |
| `android` | `bottombar`, `sidebar`, `burger` |
| `ios` | `bottombar`, `sidebar`, `burger` |

Default mapping in the scaffold:

```go
func navTypeForPlatform(platform string) string {
    switch platform {
    case "web":
        return "sidebar"
    default: // web_mobile, android, ios
        return "bottombar"
    }
}
```

---

## 5. Slots per Nav Type

| Nav type | nav_header | nav_main | nav_footer | bottombar |
|----------|------------|----------|------------|-----------|
| `sidebar` | yes | yes | yes | no |
| `burger` | yes | yes | yes | no |
| `header_footer` | yes | yes | yes | no |
| `header_only` | yes | yes | no | no |
| `bottombar` | yes (optional) | no | no | yes |

---

## 6. BuildShell

The `shell` package provides `BuildShell` which assembles the full shell tree:

```go
func BuildShell(lang, platform string) components.Component
```

It reads `DefaultNavItems()` and builds slots based on the platform's nav type.

### Nav Items

Define navigation entries by customizing `DefaultNavItems`:

```go
func DefaultNavItems() []NavItem {
    return []NavItem{
        {ID: "home", LabelKey: "nav.home", Icon: "home", Route: "/screens/home"},
        {ID: "orders", LabelKey: "nav.orders", Icon: "list", Route: "/screens/orders"},
        {ID: "profile", LabelKey: "nav.profile", Icon: "user", Route: "/screens/profile"},
    }
}
```

Each `NavItem` has:

| Field | Type | Description |
|-------|------|-------------|
| `ID` | string | Unique identifier (used as `nav-<ID>` in the component tree) |
| `LabelKey` | string | i18n key resolved via `i18n.T(lang, key)` |
| `Icon` | string | Icon name from standard icon set |
| `Route` | string | Screen endpoint fetched on click |

### Internal Build Functions

`BuildShell` delegates to internal helpers:

- `buildNavHeader(lang)` — Logo image + app name text
- `buildNavMain(lang)` — Iterates `DefaultNavItems`, creates `NavItem` components with `Navigate` actions
- `buildNavFooter(lang)` — Logout button
- `buildBottomBar(lang)` — Same items as nav_main but inside a `BottomBar` slot

### Usage in Handler

```go
func handleShell(w http.ResponseWriter, r *http.Request) {
    lang := i18n.LangFromRequest(r)
    platform := r.Header.Get("X-Platform")

    tree := shell.BuildShell(lang, platform)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tree)
}
```

---

## 7. Customization

To customize the shell beyond `DefaultNavItems`:

1. **Add items** — Add entries to `DefaultNavItems()`.
2. **Conditional items** — Replace `DefaultNavItems()` with a function that takes user context and returns role-specific items.
3. **Custom header** — Modify `buildNavHeader` to include search bars, user avatars, or other components.
4. **Custom footer** — Modify `buildNavFooter` to add settings links or version info.

Example — role-based nav:

```go
func NavItemsForRole(role string) []NavItem {
    items := []NavItem{
        {ID: "home", LabelKey: "nav.home", Icon: "home", Route: "/screens/home"},
    }
    if role == "admin" {
        items = append(items, NavItem{
            ID: "admin", LabelKey: "nav.admin", Icon: "shield", Route: "/screens/admin",
        })
    }
    return items
}
```

---

## 8. Rules

- Shell requires authentication. Return `401` if no token.
- Auth screens (login, register) render without the shell — they are standalone screens.
- Nav items are dynamic — the middleend decides what to show based on user context.
- All text in the shell uses i18n — no hardcoded strings. Use `i18n.T(lang, key)`.
- The shell is fetched once per session. Screens are fetched independently.
- `content_slot` is always present — it is the placeholder where screen content renders.
