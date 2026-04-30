# SDUI Actions — Go Reference

Actions define what happens when a user interacts with a component. The middleend declares actions; the frontend executes them.

---

## 1. Action Structure

```go
type Action struct {
    Trigger  string `json:"trigger"`
    Type     string `json:"type"`
    URL      string `json:"url,omitempty"`
    Target   string `json:"target,omitempty"`
    Endpoint string `json:"endpoint,omitempty"`
    Method   string `json:"method,omitempty"`
    TargetID string `json:"target_id,omitempty"`
    Loading  string `json:"loading,omitempty"`
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `trigger` | string | yes | What triggers the action: `click`, `submit`, `change`, `longpress` |
| `type` | string | yes | Action type identifier |
| `...params` | varies | no | Action-specific parameters (see below) |

## 2a. URL Placeholders

Any action with a `url` or `endpoint` field may contain placeholders of the form `{name}`. When the action is dispatched, the frontend substitutes each placeholder with a named value provided by the component (or form) that triggered the action.

The set of names a component exposes is defined in each component's spec (see [sdui-base-components.md](sdui-base-components.md)). For example, `select` exposes `value` (the `value` of the currently selected option), so `endpoint: "/api/list?asset_type={value}"` on a select becomes `/api/list?asset_type=STOCK` when the option with `value:"STOCK"` is chosen.

The substituted value is URL-encoded by the frontend before being spliced into the string. Placeholders whose name is not exposed by the triggering component are a middleend authoring error (the spec does not define a behavior for them).

## 2b. Loading Indicators

Any action that hits the middleend (`submit`, `reload`) can declare a `loading` field to show a visual indicator while the request is in flight. Two equivalent forms are accepted:

**Form A — string token (default for short waits):**

| Value | Behavior |
|---|---|
| `"section"` | Renders a semi-transparent overlay with spinner on the subtree whose `id` matches `target_id`. |
| `"full"` | Renders a fullscreen overlay (`z-50`) with spinner over the entire viewport. |
| (absent) | No loading indicator. The action completes silently. |

**Form B — object with cycling messages (for long waits):**

```json
"loading": {
  "scope": "section" | "full",
  "messages": ["Detecting columns…", "Mapping tickers…", "Resolving currencies…"]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `scope` | enum | yes | `"section"` or `"full"` — same semantics as Form A. |
| `messages` | string[] | no | Localized phrases the frontend rotates through every **2 seconds** in order, looping at the end. Empty / absent → behaves like Form A. |

The frontend renders the spinner (unchanged) plus, when `messages` is non-empty, a single line of text below the spinner that cycles with a brief cross-fade. Messages are purely cosmetic — they have no relationship to actual server-side progress.

The middleend decides **when** to show loading and **what scope** — the frontend only implements the visual. Loading clears automatically when the action response arrives.

Client-side-only actions (`toggle_sensitive`, `navigate`, `refresh`, etc.) ignore `loading`.

```json
{
  "trigger": "click",
  "type": "reload",
  "endpoint": "/actions/portfolio/live_data?live=true",
  "target_id": "live-data-section",
  "loading": "section"
}
```

```json
{
  "trigger": "click",
  "type": "submit",
  "endpoint": "/actions/import/analyze",
  "method": "POST",
  "target_id": "import-modal-slot",
  "loading": {
    "scope": "full",
    "messages": [
      "Detecting columns…",
      "Mapping tickers…",
      "Resolving currencies…",
      "Building preview…",
      "Validating consistency…"
    ]
  }
}
```

---

## 2. Action Types

### navigate

Navigate to a screen or route.

| Param | Type | Description |
|-------|------|-------------|
| `url` | string | Target URL/route |
| `target` | string | `self` (default) or `blank` (new tab) |

```go
Navigate(url string) Action              // target: "self"
NavigateBlank(url string) Action          // target: "blank"
```

```go
components.Button("go", "View Order", components.Navigate("/screens/order/42"))
```

### navigate_back

Go back to the previous screen. No parameters.

```go
NavigateBack() Action
```

```go
back := components.NavigateBack()
components.ScreenFull("detail", "Detail", "", "", &back, ...)
```

### submit

Collect form data and send to an endpoint. The response is an `ActionResponse`.

| Param | Type | Description |
|-------|------|-------------|
| `endpoint` | string | API endpoint to POST/PUT/DELETE to |
| `method` | string | HTTP method |
| `target_id` | string | Component ID whose children provide form data |

```go
Submit(endpoint, method, targetID string) Action
```

```go
components.Button("save", "Save",
    components.Submit("/api/users", "POST", "user-form"),
)
```

### reload

GET an endpoint and replace a component with the response. The response is an `ActionResponse`.

| Param | Type | Description |
|-------|------|-------------|
| `endpoint` | string | API endpoint to GET |
| `target_id` | string | Component ID to replace with response |

```go
Reload(endpoint, targetID string) Action
```

```go
components.Button("refresh-stats", "Refresh",
    components.Reload("/api/stats", "stats-panel"),
)
```

### refresh

Re-fetch the current screen. No parameters.

```go
Refresh() Action
```

### open_url

Open an external URL in the browser. Not for internal navigation.

| Param | Type | Description |
|-------|------|-------------|
| `url` | string | External URL to open |

```go
OpenURL(url string) Action
```

```go
components.Button("docs", "Documentation", components.OpenURL("https://docs.example.com"))
```

### dismiss

Close a modal or overlay. No parameters.

```go
Dismiss() Action
```

```go
components.Button("cancel", "Cancel", components.Dismiss())
```

### logout

Clear auth token and navigate to login. No parameters.

```go
Logout() Action
```

```go
components.Button("logout", "Sign Out", components.Logout())
```

### toggle_theme

Toggles light/dark mode. Client-side only, no round-trip. No parameters.

```go
ToggleTheme() Action
```

### toggle_sidebar

Toggles sidebar collapsed/expanded state. Client-side only, no round-trip. No parameters. Only meaningful under `nav_type: sidebar`. Ignored by other nav types.

```go
ToggleSidebar() Action
```

---

## 2c. Submitting on Enter

`Form` is a `<div data-sdui-form="true">`, not an HTML `<form>` — so the browser does not natively translate Enter-key presses on inputs into a form submit. The frontend reproduces the standard "press Enter to submit" behavior in user-space:

- Every `<button>` whose action has `type: "submit"` is rendered with `data-sdui-submit="true"`.
- `Input` listens for `keydown`. When the key is `Enter` (no IME composition in progress), the input walks up to the nearest `[data-sdui-form="true"]` ancestor, finds the first descendant `[data-sdui-submit="true"]:not([disabled])`, and calls `.click()`.
- The clicked button runs its existing `submit` flow — `hasInvalidFields` check, `revealErrors` if blocked, `collectFormData`, dispatch. Nothing in the submit path is duplicated.
- `Textarea` does **not** trap Enter; the browser default (insert newline) is preserved.
- If the form has no `data-sdui-submit="true"` button (e.g. forms that auto-submit via `change` actions), Enter does nothing.
- If the form has multiple submit buttons (uncommon — typically only one alongside `cancel` / `navigate` siblings), the first in DOM order wins.

**Middleend takeaway:** no contract change required. To opt into Enter-to-submit, simply emit a `button` with a `submit` action inside the form — the same shape used today. To opt out, omit the submit button (rare).

---

## 3. Action Response

Standard response from the middleend when the frontend executes a `submit` or `reload` action.

```go
type ActionResponse struct {
    Action   string     `json:"action"`
    TargetID string     `json:"target_id,omitempty"`
    Tree     *Component `json:"tree,omitempty"`
    Feedback *Component `json:"feedback,omitempty"`
    Auth     *AuthInfo  `json:"auth,omitempty"`
}

type AuthInfo struct {
    Token     string `json:"token"`
    ExpiresAt string `json:"expires_at,omitempty"`
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `action` | string | yes | `replace` / `navigate` / `refresh` / `none` / `logout` |
| `target_id` | string | no | Component ID to replace, or URL to navigate to |
| `tree` | Component | no | New component tree (required for `replace`) |
| `feedback` | Component | no | Temporary feedback (e.g. `snackbar`) |
| `auth` | AuthInfo | no | Auth data to persist. Any response can include it. |

### Action Values

| Action | Behavior | Required fields |
|--------|----------|-----------------|
| `replace` | Replace component `target_id` with `tree` | `target_id`, `tree` |
| `navigate` | Navigate to URL in `target_id` | `target_id` |
| `refresh` | Re-fetch current screen | — |
| `none` | No navigation/replacement, show feedback only | — |
| `logout` | Clear the stored auth token, then navigate to `target_id`. Used by destructive flows (e.g. delete account) where the session must end alongside the navigation. | `target_id` |

---

## 4. Response Helpers

### ReplaceResponse

Replace a component with a new tree, optionally showing feedback.

```go
ReplaceResponse(targetID string, tree Component, feedback *Component) ActionResponse
```

```go
newCard := components.Card("stats-panel",
    components.Text("updated", "Stats refreshed", "md", "bold"),
)
fb := components.Snackbar("fb", "Updated successfully", "success")
resp := components.ReplaceResponse("stats-panel", newCard, &fb)
```

### NavigateResponse

Navigate to a URL after the action, optionally showing feedback.

```go
NavigateResponse(url string, feedback *Component) ActionResponse
```

```go
fb := components.Snackbar("fb", "Order created", "success")
resp := components.NavigateResponse("/screens/orders", &fb)
```

### RefreshResponse

Re-fetch the current screen, optionally showing feedback.

```go
RefreshResponse(feedback *Component) ActionResponse
```

```go
fb := components.Snackbar("fb", "Saved", "success")
resp := components.RefreshResponse(&fb)
```

### LogoutResponse

End the session: the frontend clears the stored auth token, then navigates to `redirectURL`. Used by destructive flows (e.g. delete account) where the session must end alongside the navigation. Emits `{action: "logout", target_id: redirectURL}`.

```go
LogoutResponse(redirectURL string) ActionResponse
```

```go
resp := components.LogoutResponse("/screens/login")
```

### ErrorResponse

Show error feedback without any navigation or replacement.

```go
ErrorResponse(message string) ActionResponse
```

Internally creates a `Snackbar` with variant `"error"` and action `"none"`.

```go
resp := components.ErrorResponse("Failed to save changes")
```

### WithAuth

Attach authentication info to any action response. Chainable.

```go
func (r ActionResponse) WithAuth(token, expiresAt string) ActionResponse
```

```go
resp := components.NavigateResponse("/screens/home", nil).
    WithAuth(token, expiresAt)
```

---

## 5. Examples

### Form Submit Flow

```go
// Screen with form
func buildCreateUserScreen(lang string) components.Component {
    return components.Screen("create-user", i18n.T(lang, "user.create"),
        components.Form("user-form",
            components.InputFull("name", "name", "text", i18n.T(lang, "user.name"), "", "", true, false, 100),
            components.InputFull("email", "email", "email", i18n.T(lang, "user.email"), "", "", true, false, 0),
            components.Button("submit", i18n.T(lang, "action.save"),
                components.Submit("/api/users", "POST", "user-form"),
            ),
        ),
    )
}

// Handler for POST /api/users
func handleCreateUser(w http.ResponseWriter, r *http.Request) {
    // ... parse body, call backend ...
    if err != nil {
        resp := components.ErrorResponse("Failed to create user")
        json.NewEncoder(w).Encode(resp)
        return
    }
    fb := components.Snackbar("fb", "User created", "success")
    resp := components.NavigateResponse("/screens/users", &fb)
    json.NewEncoder(w).Encode(resp)
}
```

### Reload Flow

```go
// Button that reloads a panel
components.Button("refresh", "Refresh Data",
    components.Reload("/api/dashboard/stats", "stats-panel"),
)

// Handler for GET /api/dashboard/stats
func handleStats(w http.ResponseWriter, r *http.Request) {
    stats := fetchStats()
    tree := components.Card("stats-panel",
        components.Text("count", fmt.Sprintf("%d orders", stats.Count), "lg", "bold"),
    )
    resp := components.ReplaceResponse("stats-panel", tree, nil)
    json.NewEncoder(w).Encode(resp)
}
```

### Login with Auth

```go
// Login screen (standalone, no shell)
func buildLoginScreen(lang string) components.Component {
    return components.Screen("login", i18n.T(lang, "auth.login"),
        components.Form("login-form",
            components.Input("email", i18n.T(lang, "auth.email"), "email", "email"),
            components.Input("pass", i18n.T(lang, "auth.password"), "password", "password"),
            components.Button("login-btn", i18n.T(lang, "auth.login"),
                components.Submit("/api/login", "POST", "login-form"),
            ),
        ),
    )
}

// Handler for POST /api/login
func handleLogin(w http.ResponseWriter, r *http.Request) {
    // ... validate credentials, get token ...
    if err != nil {
        resp := components.ErrorResponse("Invalid credentials")
        json.NewEncoder(w).Encode(resp)
        return
    }
    resp := components.NavigateResponse("/screens/home", nil).
        WithAuth(token, expiresAt)
    json.NewEncoder(w).Encode(resp)
}
```
