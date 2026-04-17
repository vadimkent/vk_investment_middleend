# Sidebar Collapse — Design

**Date:** 2026-04-17
**Status:** Draft

## 1. Goal

Add support for collapsing the sidebar on the `web` platform. State lives on the frontend (localStorage); the middleend declares how components should render in each state via a new shared prop, and provides a new action to toggle the state.

## 2. Contract Changes (SDUI)

### 2.1 New action: `toggle_sidebar`

Client-side only. No params. No round-trip to the middleend. Symmetric with `toggle_theme`.

```go
func ToggleSidebar() Action  // {"trigger":"click", "type":"toggle_sidebar"}
```

Any component that accepts actions can fire it (`button`, `icon_toggle`, `nav_item`, etc.) — the placement is an implementation choice, not part of the contract.

### 2.2 New shared prop: `sidebar_visibility`

Controls whether a component renders based on the sidebar's collapse state.

| Value | Behavior |
|-------|----------|
| `always` | Always rendered. Default when the prop is absent. |
| `expanded` | Rendered only when the sidebar is expanded. |
| `collapsed` | Rendered only when the sidebar is collapsed. |

**Scope:** the prop takes effect only when the component lives inside a shell whose `nav_type` is `sidebar` (today: `web` platform). It is a no-op under `bottombar`, `burger`, `header_only`, `header_footer`.

**Backward compatible:** omitted = `always`.

### 2.3 `nav_item` auto-collapse

When the sidebar is collapsed, the frontend automatically:
- Hides the `label`.
- Centers the `icon`.
- Uses `label` as the tooltip.

Middleend contract: every `nav_item` must set a non-empty `icon`. `DefaultNavItems()` already satisfies this.

No JSON shape change for `nav_item`.

## 3. Middleend Implementation

### 3.1 New files / functions

- **`internal/components/actions.go`** — add `ToggleSidebar()` constructor.
- **`internal/components/actions_test.go`** — add test for the new constructor (mirror the existing no-param action constructors like `Logout()`, `Dismiss()`).

### 3.2 Shell changes (`internal/shell/builder.go`)

**`buildNavHeader`** — mark the existing app-name text as expanded-only and add a short variant for collapsed:

```go
func buildNavHeader(lang string) components.Component {
    appName := components.Text("app-name", i18n.T(lang, "app.name"), "lg", "bold")
    appName.Props["sidebar_visibility"] = "expanded"

    appNameShort := components.Text("app-name-short", i18n.T(lang, "app.name_short"), "lg", "bold")
    appNameShort.Props["sidebar_visibility"] = "collapsed"

    return components.NavHeader("shell-header", appName, appNameShort)
}
```

**`buildNavFooter`** — add sidebar toggle as the first item; split logout into two Buttons (expanded + collapsed):

```go
func buildNavFooter(lang string) components.Component {
    sidebarToggle := components.IconToggle("sidebar-toggle", false,
        "panel-left-close", "panel-left-open",
        i18n.T(lang, "nav.sidebar_collapse"), i18n.T(lang, "nav.sidebar_expand"),
        components.ToggleSidebar(), components.ToggleSidebar(),
    )

    themeToggle := components.IconToggle("theme-toggle", false,
        "sun", "moon",
        i18n.T(lang, "nav.theme_light"), i18n.T(lang, "nav.theme_dark"),
        components.Action{Trigger: "click", Type: "toggle_theme"},
        components.Action{Trigger: "click", Type: "toggle_theme"},
    )

    logoutExpanded := components.Button("logout-btn", i18n.T(lang, "nav.logout"), components.Logout())
    logoutExpanded.Props["icon"] = "logout"
    logoutExpanded.Props["sidebar_visibility"] = "expanded"

    logoutCollapsed := components.Button("logout-btn-collapsed", "", components.Logout())
    logoutCollapsed.Props["icon"] = "logout"
    logoutCollapsed.Props["sidebar_visibility"] = "collapsed"

    return components.NavFooter("shell-footer",
        sidebarToggle, themeToggle, logoutExpanded, logoutCollapsed,
    )
}
```

Order in the footer: sidebar toggle → theme toggle → logout.

### 3.3 i18n keys

Add to `locales/en.json` and `locales/es.json`:

- `app.name_short` — `"VK"` (same in both locales).
- `nav.sidebar_collapse` — EN: `"Collapse sidebar"` / ES: `"Colapsar sidebar"`.
- `nav.sidebar_expand` — EN: `"Expand sidebar"` / ES: `"Expandir sidebar"`.

### 3.4 Spec updates

- **`spec/sdui-actions.md`** — add a `toggle_sidebar` subsection under §2 Action Types, after `toggle_theme`.
- **`spec/sdui-shared-props.md`** — add a new §6 "Sidebar Visibility" section documenting `sidebar_visibility`, its three values, scope (only under `nav_type: sidebar`), and backward-compat note.
- **`spec/sdui-shell.md`** — under §3 Named Slots or §6 BuildShell, add a short note that on collapsed sidebar, `nav_item` hides the label and uses it as a tooltip (frontend-handled), and that middleend must guarantee an `icon` on every `nav_item`.

### 3.5 Tests

- `internal/components/actions_test.go` — test `ToggleSidebar()` returns the expected Action.
- `internal/shell/builder_test.go` — update shell tests to cover:
  - `nav_header` contains both app-name (expanded) and app-name-short (collapsed) with correct `sidebar_visibility`.
  - `nav_footer` contains sidebar-toggle at index 0 with `toggle_sidebar` actions.
  - `nav_footer` contains both logout buttons with correct `sidebar_visibility` and `icon`.
  - `buildBottomBar` (non-web platforms) is unchanged.

## 4. Non-goals

- No changes to `nav_main` or any screen. Nav items already carry `icon`, so auto-collapse handling is purely frontend.
- No new endpoint. Shell JSON shape is unchanged beyond the new props.
- No server-side awareness of collapse state.
- No changes to `bottombar`, `burger`, or mobile platforms.

## 5. Risks / Open Questions

- **Icon availability:** `panel-left-close` / `panel-left-open` / `logout` must exist in the frontend's icon registry. If not, swap for `menu` / `chevron-left` etc. — purely a naming detail, no contract impact.
- **`nav_footer` layout under collapse:** the collapsed footer will contain `sidebar-toggle`, `theme-toggle`, `logout-btn-collapsed` — three icons stacked vertically. If the footer's default orientation doesn't fit collapsed width, that's a frontend styling concern, not middleend.
