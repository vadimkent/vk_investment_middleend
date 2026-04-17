# Sidebar Collapse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a collapsible sidebar to the web shell via a new client-side action (`toggle_sidebar`) and a new shared prop (`sidebar_visibility`), with no changes to endpoints or shell JSON shape.

**Architecture:** The middleend declares a toggle action and per-component visibility intent; the frontend owns collapse state (localStorage) and rendering. `nav_item` auto-collapses (label → tooltip) on the frontend; `image`/`text`/`button` use `sidebar_visibility` to swap between expanded and collapsed variants. The prop is a no-op outside `nav_type: sidebar`.

**Tech Stack:** Go 1.x, Gin, testify, project's SDUI components package, JSON i18n.

**Spec reference:** `docs/superpowers/specs/2026-04-17-sidebar-collapse-design.md`

---

## File Structure

- **Modify** `internal/components/actions.go` — add `ToggleSidebar()` constructor.
- **Modify** `internal/components/actions_test.go` — add test for `ToggleSidebar()`.
- **Modify** `internal/shell/builder.go` — update `buildNavHeader` and `buildNavFooter`.
- **Modify** `internal/shell/builder_test.go` — extend tests for header/footer changes.
- **Modify** `locales/en.json` — add i18n keys.
- **Modify** `locales/es.json` — add i18n keys.
- **Modify** `spec/sdui-actions.md` — document `toggle_sidebar`.
- **Modify** `spec/sdui-shared-props.md` — add `sidebar_visibility` section.
- **Modify** `spec/sdui-shell.md` — add note about `nav_item` auto-collapse.

---

## Task 1: Add i18n keys

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

- [ ] **Step 1: Add keys to `locales/en.json`**

Add `app.name_short` to the `app` object and `nav.sidebar_collapse` / `nav.sidebar_expand` to the `nav` object. After edit, the top of the file should read:

```json
{
  "app": {
    "name": "VK Investments",
    "name_short": "VK"
  },
  "nav": {
    "portfolio": "Portfolio",
    "assets": "Assets",
    "trades": "Trades",
    "snapshots": "Snapshots",
    "import": "Import",
    "analysis": "Analysis",
    "logout": "Log out",
    "theme_light": "Switch to dark mode",
    "theme_dark": "Switch to light mode",
    "sidebar_collapse": "Collapse sidebar",
    "sidebar_expand": "Expand sidebar"
  },
```

- [ ] **Step 2: Add keys to `locales/es.json`**

Mirror the structure:

```json
{
  "app": {
    "name": "VK Investments",
    "name_short": "VK"
  },
  "nav": {
    "portfolio": "Portafolio",
    "assets": "Activos",
    "trades": "Operaciones",
    "snapshots": "Snapshots",
    "import": "Importar",
    "analysis": "Análisis",
    "logout": "Cerrar sesión",
    "theme_light": "Cambiar a modo oscuro",
    "theme_dark": "Cambiar a modo claro",
    "sidebar_collapse": "Colapsar sidebar",
    "sidebar_expand": "Expandir sidebar"
  },
```

- [ ] **Step 3: Verify JSON is valid**

Run: `python3 -c "import json; json.load(open('locales/en.json')); json.load(open('locales/es.json')); print('ok')"`
Expected: `ok`

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "i18n: add sidebar collapse/expand and short app name keys"
```

---

## Task 2: Add `ToggleSidebar()` action constructor (TDD)

**Files:**
- Modify: `internal/components/actions.go`
- Modify: `internal/components/actions_test.go`

- [ ] **Step 1: Write failing test**

Append to `internal/components/actions_test.go`:

```go
func TestToggleSidebar_ReturnsClickToggleAction(t *testing.T) {
	action := ToggleSidebar()
	assert.Equal(t, "click", action.Trigger)
	assert.Equal(t, "toggle_sidebar", action.Type)
	assert.Empty(t, action.URL)
	assert.Empty(t, action.Endpoint)
	assert.Empty(t, action.TargetID)
}

func TestToggleSidebar_JSONShape(t *testing.T) {
	b, err := json.Marshal(ToggleSidebar())
	require.NoError(t, err)
	assert.Equal(t, `{"trigger":"click","type":"toggle_sidebar"}`, string(b))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/components/ -run TestToggleSidebar -v`
Expected: FAIL — `undefined: ToggleSidebar`

- [ ] **Step 3: Implement `ToggleSidebar()` in `internal/components/actions.go`**

Append after the `Logout()` function:

```go
// ToggleSidebar creates a client-side action that toggles sidebar collapse state.
// No round-trip to the middleend. State is owned by the frontend (localStorage).
func ToggleSidebar() Action {
	return Action{Trigger: "click", Type: "toggle_sidebar"}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/components/ -run TestToggleSidebar -v`
Expected: PASS (both subtests)

- [ ] **Step 5: Run full components test suite to ensure no regression**

Run: `go test ./internal/components/`
Expected: PASS (all tests)

- [ ] **Step 6: Commit**

```bash
git add internal/components/actions.go internal/components/actions_test.go
git commit -m "feat(sdui): add ToggleSidebar action constructor"
```

---

## Task 3: Document `toggle_sidebar` action in `spec/sdui-actions.md`

**Files:**
- Modify: `spec/sdui-actions.md`

- [ ] **Step 1: Add `toggle_sidebar` subsection after `toggle_theme`**

After the `### toggle_theme` block (which ends after its code block on line ~180), insert:

```markdown

### toggle_sidebar

Toggles sidebar collapsed/expanded state. Client-side only, no round-trip. No parameters. Only meaningful under `nav_type: sidebar`; ignored by other nav types.

```go
ToggleSidebar() Action
```

```

- [ ] **Step 2: Verify the file still reads correctly**

Run: `grep -n "toggle_sidebar" spec/sdui-actions.md`
Expected: at least one hit under `### toggle_sidebar`.

- [ ] **Step 3: Commit**

```bash
git add spec/sdui-actions.md
git commit -m "docs(sdui): document toggle_sidebar action"
```

---

## Task 4: Document `sidebar_visibility` shared prop

**Files:**
- Modify: `spec/sdui-shared-props.md`

- [ ] **Step 1: Append a new section after the current §4 Positioning**

After the end of the Positioning section (the code example for `fab.Props["justify_self"] = "bottom"`) and before `## 5. Usage Pattern`, insert:

```markdown
---

## 5. Sidebar Visibility

Available on any component. Controls rendering based on the sidebar's collapse state.

| Prop | Values | Description |
|------|--------|-------------|
| `sidebar_visibility` | `always` / `expanded` / `collapsed` | Render only in the given sidebar state. Default `always`. |

**Scope:** this prop takes effect only when the component lives inside a shell whose `nav_type` is `sidebar` (today: `web`). It is a no-op under `bottombar`, `burger`, `header_only`, `header_footer`.

**Backward compatible:** when omitted, the component renders in every state (`always`).

```go
appName := components.Text("app-name", "VK Investments", "lg", "bold")
appName.Props["sidebar_visibility"] = "expanded"

appNameShort := components.Text("app-name-short", "VK", "lg", "bold")
appNameShort.Props["sidebar_visibility"] = "collapsed"
```

The frontend chooses which variant to render based on its own collapse state. The middleend sends both; only the matching one becomes visible.
```

Then renumber the following "Usage Pattern" section from `## 5.` to `## 6.`.

- [ ] **Step 2: Verify structure**

Run: `grep -n "^## " spec/sdui-shared-props.md`
Expected output (in order): `## 1. Container Alignment`, `## 2. Self Alignment`, `## 3. Spacing`, `## 4. Positioning`, `## 5. Sidebar Visibility`, `## 6. Usage Pattern`.

- [ ] **Step 3: Commit**

```bash
git add spec/sdui-shared-props.md
git commit -m "docs(sdui): document sidebar_visibility shared prop"
```

---

## Task 5: Document `nav_item` auto-collapse in shell spec

**Files:**
- Modify: `spec/sdui-shell.md`

- [ ] **Step 1: Add a note at the end of §3 Named Slots**

After the line `Each slot is a generic container — it accepts any children.` (around line 41) and before the `---` that starts `## 4. Platform Adaptation`, insert:

```markdown

### Collapsed sidebar behavior

When `nav_type` is `sidebar` and the user collapses it (via `toggle_sidebar`), the frontend automatically:

- Hides each `nav_item`'s `label` and centers its `icon`.
- Uses the `label` as a tooltip on hover.

The middleend must guarantee every `nav_item` has a non-empty `icon`. Other components in the sidebar tree can opt in/out of rendering in each state via the `sidebar_visibility` shared prop.
```

- [ ] **Step 2: Verify the file still lists the slots above the new subsection**

Run: `grep -n "Named Slots\|Collapsed sidebar behavior\|Platform Adaptation" spec/sdui-shell.md`
Expected: `Named Slots` comes before `Collapsed sidebar behavior` which comes before `Platform Adaptation`.

- [ ] **Step 3: Commit**

```bash
git add spec/sdui-shell.md
git commit -m "docs(sdui): note nav_item auto-collapse behavior in sidebar"
```

---

## Task 6: Update `buildNavHeader` — dual app-name (expanded/collapsed) (TDD)

**Files:**
- Modify: `internal/shell/builder_test.go`
- Modify: `internal/shell/builder.go`

- [ ] **Step 1: Write failing test**

Append to `internal/shell/builder_test.go`:

```go
func TestBuildShell_NavHeaderHasExpandedAndCollapsedAppName(t *testing.T) {
	shell := BuildShell("en", "web")
	header := findChild(shell, "nav_header")
	require.NotNil(t, header)
	require.Len(t, header.Children, 2, "nav_header should have expanded + collapsed app-name")

	expanded := header.Children[0]
	assert.Equal(t, "text", expanded.Type)
	assert.Equal(t, "app-name", expanded.ID)
	assert.Equal(t, "VK Investments", expanded.Props["content"])
	assert.Equal(t, "expanded", expanded.Props["sidebar_visibility"])

	collapsed := header.Children[1]
	assert.Equal(t, "text", collapsed.Type)
	assert.Equal(t, "app-name-short", collapsed.ID)
	assert.Equal(t, "VK", collapsed.Props["content"])
	assert.Equal(t, "collapsed", collapsed.Props["sidebar_visibility"])
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shell/ -run TestBuildShell_NavHeaderHasExpandedAndCollapsedAppName -v`
Expected: FAIL — header has only one child (`app-name`), or the `sidebar_visibility` prop is missing.

- [ ] **Step 3: Update `buildNavHeader` in `internal/shell/builder.go`**

Replace the current function body:

```go
func buildNavHeader(lang string) components.Component {
	appName := components.Text("app-name", i18n.T(lang, "app.name"), "lg", "bold")
	appName.Props["sidebar_visibility"] = "expanded"

	appNameShort := components.Text("app-name-short", i18n.T(lang, "app.name_short"), "lg", "bold")
	appNameShort.Props["sidebar_visibility"] = "collapsed"

	return components.NavHeader("shell-header", appName, appNameShort)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shell/ -run TestBuildShell_NavHeaderHasExpandedAndCollapsedAppName -v`
Expected: PASS

- [ ] **Step 5: Run full shell test suite to ensure no regressions**

Run: `go test ./internal/shell/`
Expected: PASS (all tests, including existing `TestBuildShell_WebSidebar` etc.)

- [ ] **Step 6: Commit**

```bash
git add internal/shell/builder.go internal/shell/builder_test.go
git commit -m "feat(shell): dual app-name in nav_header for sidebar collapse"
```

---

## Task 7: Update `buildNavFooter` — sidebar toggle + split logout (TDD)

**Files:**
- Modify: `internal/shell/builder_test.go`
- Modify: `internal/shell/builder.go`

- [ ] **Step 1: Write failing test — sidebar toggle present**

Append to `internal/shell/builder_test.go`:

```go
func TestBuildShell_NavFooterHasSidebarToggleFirst(t *testing.T) {
	shell := BuildShell("en", "web")
	footer := findChild(shell, "nav_footer")
	require.NotNil(t, footer)
	require.GreaterOrEqual(t, len(footer.Children), 1)

	toggle := footer.Children[0]
	assert.Equal(t, "icon_toggle", toggle.Type)
	assert.Equal(t, "sidebar-toggle", toggle.ID)
	assert.Equal(t, "panel-left-close", toggle.Props["icon_inactive"])
	assert.Equal(t, "panel-left-open", toggle.Props["icon_active"])
	assert.Equal(t, "Collapse sidebar", toggle.Props["tooltip_inactive"])
	assert.Equal(t, "Expand sidebar", toggle.Props["tooltip_active"])

	require.Len(t, toggle.Actions, 2)
	assert.Equal(t, "toggle_sidebar", toggle.Actions[0].Type)
	assert.Equal(t, "toggle_sidebar", toggle.Actions[1].Type)
}

func TestBuildShell_NavFooterLogoutSplitByVisibility(t *testing.T) {
	shell := BuildShell("en", "web")
	footer := findChild(shell, "nav_footer")
	require.NotNil(t, footer)

	var expanded, collapsed *components.Component
	for i, child := range footer.Children {
		if child.ID == "logout-btn" {
			expanded = &footer.Children[i]
		}
		if child.ID == "logout-btn-collapsed" {
			collapsed = &footer.Children[i]
		}
	}

	require.NotNil(t, expanded, "expanded logout button must exist")
	assert.Equal(t, "button", expanded.Type)
	assert.Equal(t, "Log out", expanded.Props["label"])
	assert.Equal(t, "logout", expanded.Props["icon"])
	assert.Equal(t, "expanded", expanded.Props["sidebar_visibility"])
	require.Len(t, expanded.Actions, 1)
	assert.Equal(t, "logout", expanded.Actions[0].Type)

	require.NotNil(t, collapsed, "collapsed logout button must exist")
	assert.Equal(t, "button", collapsed.Type)
	assert.Equal(t, "logout", collapsed.Props["icon"])
	assert.Equal(t, "collapsed", collapsed.Props["sidebar_visibility"])
	require.Len(t, collapsed.Actions, 1)
	assert.Equal(t, "logout", collapsed.Actions[0].Type)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/shell/ -run "TestBuildShell_NavFooterHasSidebarToggleFirst|TestBuildShell_NavFooterLogoutSplitByVisibility" -v`
Expected: FAIL — sidebar-toggle not found, logout button lacks `sidebar_visibility` / `icon`, and `logout-btn-collapsed` does not exist.

- [ ] **Step 3: Update `buildNavFooter` in `internal/shell/builder.go`**

Replace the current function body:

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
		sidebarToggle,
		themeToggle,
		logoutExpanded,
		logoutCollapsed,
	)
}
```

- [ ] **Step 4: Run new tests to verify they pass**

Run: `go test ./internal/shell/ -run "TestBuildShell_NavFooterHasSidebarToggleFirst|TestBuildShell_NavFooterLogoutSplitByVisibility" -v`
Expected: PASS

- [ ] **Step 5: Run full shell test suite (existing logout test should still pass)**

Run: `go test ./internal/shell/`
Expected: PASS — including `TestBuildShell_NavFooterHasLogoutOnWeb` (the existing test iterates all footer children looking for any `logout` action, so it still finds one).

- [ ] **Step 6: Commit**

```bash
git add internal/shell/builder.go internal/shell/builder_test.go
git commit -m "feat(shell): sidebar toggle + collapse-aware logout in nav_footer"
```

---

## Task 8: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Run full test suite**

Run: `make test`
Expected: all tests pass. No JSON parse errors, no unexpected failures.

- [ ] **Step 2: Run linter**

Run: `make lint`
Expected: no new warnings.

- [ ] **Step 3: Build the binary**

Run: `make build`
Expected: builds without errors.

- [ ] **Step 4: Smoke test — render shell locally (manual, optional)**

If the server is easy to start (`make run`), hit `GET /shell` with `X-Platform: web` and a valid token, and visually confirm:
- `nav_header` has two `text` children (`app-name` with `sidebar_visibility: expanded`, `app-name-short` with `sidebar_visibility: collapsed`).
- `nav_footer` has four children in order: `sidebar-toggle`, `theme-toggle`, `logout-btn` (expanded), `logout-btn-collapsed`.
- `sidebar-toggle` carries two `toggle_sidebar` actions.

Skip if the auth flow is non-trivial — all the invariants are covered by tests.

---

## Notes

- Every task is independently commit-able. The plan is additive: nothing changes the shell JSON shape for non-web platforms (`buildBottomBar` is untouched).
- The existing `buildNavFooter` theme-toggle uses an inline `Action{}` literal rather than a constructor. We match that style for `toggle_theme` and introduce a proper `ToggleSidebar()` constructor for the new action (the existing `toggle_theme` helper-less style is left as-is to avoid scope creep).
- Frontend contract change to extend Button auto-collapse was explicitly **rejected** in the design — we use two Button components with `sidebar_visibility` instead (same pattern as the logo example in the frontend's proposal).
