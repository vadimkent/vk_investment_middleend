# SDUI Base Components — Go Reference

All components are built using helpers from `internal/components`. Every helper returns a `components.Component`.

```go
type Component struct {
    Type     string            `json:"type"`
    ID       string            `json:"id"`
    Props    map[string]any    `json:"props"`
    Children []Component       `json:"children,omitempty"`
    Actions  []Action          `json:"actions,omitempty"`
}
```

---

## 1. Screen & Layout

### screen

Top-level container. Every screen endpoint returns one. The frontend renders it as a full-viewport container (`min-h-screen`). To center content (e.g. a login card), use a `column` child with `align_items: center` and `justify_items: center`.

Screen props are **metadata** — they are not rendered as visible UI by the screen itself. If you want a visible heading inside the screen, add a `text` component to the tree. The shell may read `title`/`subtitle`/`icon` to display them in a header area next to the content slot.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `title` | string | no | Screen title. Metadata — used for `document.title` (browser tab) and accessibility; optionally rendered by the shell header. Never rendered by the screen itself. |
| `subtitle` | string | no | Secondary title. Metadata; same rules as `title`. |
| `icon` | string | no | Icon identifier. Metadata; same rules as `title`. |
| `back_action` | bool | no | Whether back navigation is present (action attached). Shell hint. |

```go
Screen(id, title string, children ...Component) Component
ScreenFull(id, title, subtitle, icon string, backAction *Action, children ...Component) Component
```

```go
c := components.Screen("home", "Dashboard",
    components.Text("welcome", "Hello", "lg", "bold"),
)

back := components.NavigateBack()
c := components.ScreenFull("detail", "Order #42", "Pending", "package", &back,
    components.Text("info", "Details here", "md", "normal"),
)
```

### row

Horizontal layout container. Width distribution via CSS grid values. Only sent for `web` platform.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `widths` | []string | yes | Width distribution (e.g. `"1fr"`, `"300px"`) |
| `gap` | string | no | Space between children |

```go
Row(id string, widths []string, children ...Component) Component
RowWithGap(id string, widths []string, gap string, children ...Component) Component
```

```go
c := components.Row("layout", []string{"1fr", "2fr"},
    components.Card("sidebar", ...),
    components.Card("main", ...),
)
```

### column

Vertical layout container. Used across all platforms.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `gap` | string | no | Space between children |

```go
Column(id string, children ...Component) Component
ColumnWithGap(id, gap string, children ...Component) Component
```

```go
c := components.ColumnWithGap("stack", "16px",
    components.Text("t1", "First", "md", "normal"),
    components.Text("t2", "Second", "md", "normal"),
)
```

### group

Generic container without visual styling. Groups children structurally or as data source for submit.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| — | — | — | No specific props |

```go
Group(id string, children ...Component) Component
```

---

## 2. Content

### text

Displays text with configurable size, weight, color, and decoration.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `content` | string | yes | Text to display |
| `size` | string | yes | `xs` / `sm` / `md` / `lg` / `xl` / `2xl` |
| `weight` | string | yes | `light` / `normal` / `medium` / `bold` |
| `display` | string | no | `block` / `inline` |
| `color` | enum | no | `primary` / `secondary` / `muted` / `error` / `positive` / `negative`. `error` is reserved for validation / system failures. Use `positive` (gain, up) and `negative` (loss, down) for deltas and P&L. |
| `hex_color` | string | no | Custom hex color (e.g. `#FF0000`). Deprecated-candidate — prefer `color` tokens for portability. |
| `decoration` | string | no | `underline` / `strikethrough` / `none` |

```go
Text(id, content, size, weight string) Component
TextStyled(id, content, size, weight, display, color, hexColor, decoration string) Component
```

```go
c := components.Text("title", "Welcome", "xl", "bold")
c := components.TextStyled("err", "Invalid", "sm", "normal", "", "error", "", "")
```

### image

Displays an image from a URL.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `src` | string | yes | Image URL |
| `alt` | string | yes | Alt text |
| `width` | string | no | Free-form CSS width (`"120px"`, `"50%"`). Exact dimensions are legitimate for logos/avatars/illustrations. |
| `height` | string | no | Free-form CSS height. Same rules as `width`. |
| `fit` | enum | no | `cover` / `contain` / `fill` / `none` / `scale-down`. Other values are ignored by the frontend. |
| `border_radius` | enum | no | `none` / `sm` (4) / `md` (8) / `lg` (16) / `full` (9999). Raw px values are ignored. |

```go
Image(id, src, alt string) Component
ImageStyled(id, src, alt, width, height, fit, borderRadius string) Component
```

```go
c := components.Image("avatar", "https://cdn.example.com/u/1.jpg", "User avatar")
c := components.ImageStyled("thumb", "/img/product.jpg", "Product", "200px", "200px", "cover", "md")
```

### card

Visual grouping container with border/shadow.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `elevation` | enum | no | `none` / `sm` / `md` / `lg` |
| `border_radius` | enum | no | `none` / `sm` (4) / `md` (8) / `lg` (16) / `full` (9999). Raw px values are ignored. |

```go
Card(id string, children ...Component) Component
CardStyled(id, elevation, borderRadius string, children ...Component) Component
```

```go
c := components.Card("info-card",
    components.Text("t", "Card content", "md", "normal"),
)
```

### list

Scrollable container for repeated items.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `orientation` | string | no | `vertical` (default) / `horizontal` |

```go
List(id string, children ...Component) Component
ListHorizontal(id string, children ...Component) Component
```

### list_item

Single entry within a `list`. Any children. Clickable when `actions` present.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| — | — | — | No specific props |

```go
ListItem(id string, children ...Component) Component
```

```go
c := components.List("orders",
    components.ListItem("o-1",
        components.Text("o1-name", "Order #1", "md", "bold"),
    ),
    components.ListItem("o-2",
        components.Text("o2-name", "Order #2", "md", "bold"),
    ),
)
```

### divider

Separator line. No children.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `orientation` | string | no | `horizontal` (default) / `vertical` |

```go
Divider(id string) Component
DividerVertical(id string) Component
```

### spacer

Invisible spacing element. No children.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `size` | enum | yes | `none` / `xs` (4) / `sm` (8) / `md` (16) / `lg` (24) / `xl` (32) / `2xl` (48). Raw px values are ignored. |

```go
Spacer(id, size string) Component
```

---

## 3. Interactive

### button

Clickable element. Must have actions.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `label` | string | no | Button text (required if no `image_src`) |
| `image_src` | string | no | Button image (required if no `label`) |
| `variant` | string | yes | `primary` / `secondary` |
| `style` | string | yes | `solid` / `ghost` / `outline` |
| `disabled` | bool | no | Disable the button |
| `loading` | bool | no | Show loading state |
| `size` | enum | no | `xs` / `sm` / `md` / `lg`. Controls label text size and button padding. Default `md`. |

```go
Button(id, label string, actions ...Action) Component
ButtonFull(id, label, imageSrc, variant, style string, actions ...Action) Component
```

```go
c := components.Button("save-btn", "Save", components.Submit("/api/save", "POST", "form-1"))
c := components.ButtonFull("icon-btn", "", "/icons/add.svg", "secondary", "ghost",
    components.Navigate("/items/new"),
)
```

### input

Text input field. Must be inside a `form`.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | Field name for form data |
| `input_type` | string | yes | HTML input type (text, email, password, number, etc.) |
| `placeholder` | string | no | Placeholder text |
| `label` | string | no | Field label |
| `default_value` | string | no | Pre-filled value |
| `required` | bool | no | Required validation |
| `disabled` | bool | no | Disable the field |
| `max_length` | int | no | Maximum character count |

```go
Input(id, placeholder, name, inputType string) Component
InputFull(id, name, inputType, label, placeholder, defaultValue string, required, disabled bool, maxLength int) Component
```

```go
c := components.Input("email-input", "Enter email", "email", "email")
c := components.InputFull("name-input", "name", "text", "Full Name", "Enter name", "", true, false, 100)
```

### form

Groups inputs for submission. Collects child input values for submit actions.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `loading` | bool | no | Disables all children when true |

```go
Form(id string, children ...Component) Component
```

```go
c := components.Form("login-form",
    components.Input("email", "Email", "email", "email"),
    components.Input("pass", "Password", "password", "password"),
    components.Button("login-btn", "Login",
        components.Submit("/api/login", "POST", "login-form"),
    ),
)
```

### select

Dropdown selection field. Must be inside a `form`.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | Field name |
| `options` | []SelectOption | yes | Array of `{value, label}` |
| `label` | string | no | Field label |
| `placeholder` | string | no | Placeholder text |
| `default_value` | string | no | Pre-selected value |
| `required` | bool | no | Required validation |
| `disabled` | bool | no | Disable the field |

```go
type SelectOption struct {
    Value string `json:"value"`
    Label string `json:"label"`
}

Select(id, name string, options []SelectOption) Component
SelectFull(id, name, label, placeholder, defaultValue string, options []SelectOption, required, disabled bool) Component
```

```go
opts := []components.SelectOption{
    {Value: "us", Label: "United States"},
    {Value: "mx", Label: "Mexico"},
}
c := components.Select("country", "country", opts)
```

### checkbox

Single boolean toggle with label. Must be inside a `form`.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | Field name |
| `label` | string | yes | Display label |
| `checked` | bool | no | Default state |
| `disabled` | bool | no | Disable the field |

```go
Checkbox(id, name, label string) Component
```

### toggle

On/off switch. Must be inside a `form`.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | Field name |
| `label` | string | yes | Display label |
| `checked` | bool | no | Default state |
| `disabled` | bool | no | Disable the field |

```go
Toggle(id, name, label string) Component
```

### textarea

Multi-line text input. Must be inside a `form`.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | Field name |
| `label` | string | no | Field label |
| `placeholder` | string | no | Placeholder text |
| `default_value` | string | no | Pre-filled value |
| `rows` | int | no | Visible row count |
| `max_length` | int | no | Maximum character count |
| `required` | bool | no | Required validation |
| `disabled` | bool | no | Disable the field |

```go
Textarea(id, name string) Component
TextareaFull(id, name, label, placeholder, defaultValue string, rows, maxLength int, required, disabled bool) Component
```

### radio_group

Set of mutually exclusive options. Must be inside a `form`.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | Field name |
| `options` | []SelectOption | yes | Array of `{value, label}` |
| `label` | string | no | Group label |
| `default_value` | string | no | Pre-selected value |
| `required` | bool | no | Required validation |
| `disabled` | bool | no | Disable all options |

```go
RadioGroup(id, name string, options []SelectOption) Component
```

---

## 4. State & Feedback

### loading

Platform-native loading indicator. No children.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `size` | string | no | `sm` / `md` / `lg` |
| `variant` | string | no | `spinner` / `skeleton` |

```go
Loading(id string) Component
LoadingStyled(id, size, variant string) Component
```

### error

Error message display.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `message` | string | yes | Error message text |
| `retry_action` | bool | no | Whether retry action is attached |

```go
ErrorComponent(id, message string) Component
ErrorWithRetry(id, message string, retryAction Action) Component
```

```go
c := components.ErrorWithRetry("err", "Failed to load",
    components.Reload("/api/orders", "order-list"),
)
```

### snackbar

Temporary feedback message (toast). Auto-dismisses.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `message` | string | yes | Feedback message |
| `variant` | string | yes | `success` / `error` / `info` / `warning` |

```go
Snackbar(id, message, variant string) Component
```

### modal

Overlay dialog. Renders children as content. Closed via `dismiss` action.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `visible` | bool | yes | Whether modal is shown |
| `title` | string | no | Modal title |
| `dismissible` | bool | no | Can be closed by user (default true) |
| `presentation` | string | no | `dialog` / `bottom_sheet` / `fullscreen` |

```go
Modal(id string, visible bool, children ...Component) Component
ModalFull(id, title, presentation string, visible, dismissible bool, children ...Component) Component
```

```go
c := components.ModalFull("confirm", "Confirm Delete", "dialog", true, true,
    components.Text("msg", "Are you sure?", "md", "normal"),
    components.Button("yes", "Delete", components.Submit("/api/delete", "DELETE", "confirm")),
    components.Button("no", "Cancel", components.Dismiss()),
)
```

### badge

Small indicator overlaid on a child component. Wraps a single child.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `count` | int | no | Numeric count to display |
| `variant` | string | no | `error` / `info` / `warning` / `success` |

```go
Badge(id string, child Component) Component
BadgeCount(id string, count int, variant string, child Component) Component
```

```go
c := components.BadgeCount("notif", 3, "error",
    components.NavItem("nav-inbox", "Inbox", "mail", "/screens/inbox", components.Navigate("/screens/inbox")),
)
```

---

## 5. Shell Slots

Shell slot components are documented in [sdui-shell.md](sdui-shell.md). For reference:

| Type | Helper | Description |
|------|--------|-------------|
| `nav_header` | `NavHeader(id, children...)` | Top zone: logo, user context |
| `nav_main` | `NavMain(id, children...)` | Primary navigation |
| `nav_footer` | `NavFooter(id, children...)` | Bottom zone of nav |
| `bottombar` | `BottomBar(id, children...)` | Fixed bottom bar |
| `content_slot` | `ContentSlot(id)` | Where the active screen renders |
| `nav_item` | `NavItem(id, label, icon, route, actions...)` | Navigation link |
