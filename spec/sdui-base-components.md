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

### table

Tabular data with aligned columns across header and rows. The table owns column widths and alignment; rows and cells inherit them via CSS subgrid so the header and every body row line up automatically. Use `table` for data with parallel columns (positions, orders, transactions). Use `list` for uniform feeds without column structure.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `columns` | `Column[]` | yes | Column configuration. Each column defines an id, header label, optional width, and optional alignment. |

**`Column`**: `{ id: string, header: string, width?: string, align?: "left" | "center" | "right" }`. Missing `width` defaults to `"1fr"`. `align` defaults to `"left"`.

Children must be `table_row` components. Each row's children are placed into the columns in order; the number of children per row should match `columns.length`. The header row is rendered automatically by the frontend from `columns[].header`.

```go
Table(id string, columns []TableColumn, children ...Component) Component
```

```go
cols := []components.TableColumn{
    {ID: "ticker", Header: "Ticker", Width: "80px"},
    {ID: "name",   Header: "Name",   Width: "1fr"},
    {ID: "value",  Header: "Value",  Width: "120px", Align: "right"},
}
c := components.Table("positions-table", cols,
    components.TableRow("row-1",
        components.Text("t1", "AAPL", "sm", "bold"),
        components.Text("n1", "Apple Inc", "sm", "normal"),
        components.Text("v1", "$1,855.00", "sm", "normal"),
    ),
)
```

### table_row

A row inside a `table`. Uses CSS subgrid so every row shares the same column tracks as the table. Supports click actions for row-level interaction (e.g. navigate to a detail screen).

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| — | — | — | Shape comes from the table's `columns` |
| `expandable` | bool | no | Default `false`. When `true`, the row is toggleable — clicking the main row expands / collapses a full-width `details` panel rendered beneath it. The frontend renders a chevron indicator. |

Each child of `table_row` is rendered into a cell aligned according to the column's `align`. Use `text`, `badge`, `image`, or any component as a cell.

In addition to the cell `children`, expandable rows carry a `details` slot:

| Slot | Type | Description |
|------|------|-------------|
| `details` | `Component[]` | Subtree rendered as a full-width panel directly beneath the row when expanded. Pre-emitted in the tree (not fetched on expand). Breaks the subgrid only for its own row. |

When **any** row in a `table` is `expandable: true`, the frontend automatically prepends a 24px chevron column to the header (and to all rows — expandable or not — to preserve column alignment). This column is not part of `columns`; it is purely presentational.

Expand state is client-side, keyed per `row.id`. Multiple rows may be expanded simultaneously. State resets on any `replace` that rebuilds the table subtree.

```go
TableRow(id string, children ...Component) Component
TableRowExpandable(id string, cells []Component, details ...Component) Component
```

The existing `TableRow(id, children...)` signature is unchanged — non-expandable rows remain the default.

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
| `label` | string | no | Button text (required if no `icon` or `image_src`) |
| `icon` | string | no | Icon token (e.g. `refresh`, `plus`, `delete`). Looked up in the frontend's icon registry and rendered as an SVG component. Takes priority over `image_src`. |
| `image_src` | string | no | Fallback image URL. Only used when `icon` is absent. |
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
| `pattern` | string | no | ECMAScript regex validated client-side on change/blur; non-matching values block submission |
| `auto_uppercase` | bool | no | Frontend transforms entered value to uppercase as the user types |
| `min_length` | int | no | Minimum character count; values shorter than this block submission. Validated client-side on change/blur |
| `match_field` | string | no | Name of another input within the same `form`; submission is blocked unless this field's value equals that sibling's value. Validated client-side on change/blur of either field |
| `required_message` | string | no | Localized message shown when `required` is set and the field is empty. If absent, frontend falls back to a default. |
| `pattern_message` | string | no | Localized message shown when `pattern` is set and the value does not match. |
| `min_length_message` | string | no | Localized message shown when `min_length` is set and the value is shorter. May contain `{min}` interpolated to the threshold. |
| `max_length_message` | string | no | Localized message shown when `max_length` is set and the value is longer. May contain `{max}`. |
| `match_field_message` | string | no | Localized message shown when `match_field` is set and the values differ. |

**Validation message resolution.** When multiple rules fail simultaneously, the frontend renders the message for the highest-priority failure in this order: `required` > `match_field` > `pattern` > `min_length` > `max_length`. Only one message is shown at a time. The middleend decides the localized copy; the frontend only picks which prop to display based on which rule failed.

```go
Input(id, placeholder, name, inputType string) Component
InputFull(id, name, inputType, label, placeholder, defaultValue string, required, disabled bool, maxLength int) Component
InputAdvanced(o InputOptions) Component
```

```go
c := components.Input("email-input", "Enter email", "email", "email")
c := components.InputFull("name-input", "name", "text", "Full Name", "Enter name", "", true, false, 100)
```

```go
c := components.InputAdvanced(components.InputOptions{
    ID: "confirm", Name: "confirm_password", InputType: "password",
    Required: true, MinLength: 8, MatchField: "password",
})
```

### form

Groups inputs for submission. Collects child input values for submit actions.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `loading` | bool | no | Disables all children when true |

A `form` SHOULD contain at least one descendant `button` whose action is `type: "submit"`. The frontend uses that button as the Enter-key submit target: pressing Enter on any input within the form triggers a click on the first such button (in DOM order). Forms without a submit button still work for explicit clicks on whatever submit triggers exist elsewhere, but they will not respond to Enter.

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

Dropdown selection field. Typically used inside a `form`, but can also stand alone with an attached action (e.g. a filter that triggers a `reload` on change).

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | Field name |
| `options` | []SelectOption | yes | Array of `{value, label}` |
| `label` | string | no | Field label |
| `placeholder` | string | no | Placeholder text |
| `default_value` | string | no | Pre-selected value |
| `required` | bool | no | Required validation |
| `disabled` | bool | no | Disable the field |

**Placeholder source:** When `select` triggers an action, it exposes `value` — the `value` of the currently selected option — for URL placeholder substitution (see [sdui-actions.md § URL Placeholders](sdui-actions.md)).

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

### icon_toggle

Binary toggle rendered as a clickable icon. Has two states (inactive / active), each with its own icon token and tooltip. On click the frontend flips the visual state instantly (optimistic) and fires the corresponding action. Does not require a form wrapper.

The component carries **two actions** in its `actions` array:
- `actions[0]` — fired when transitioning from inactive → active (click while `active=false`).
- `actions[1]` — fired when transitioning from active → inactive (click while `active=true`).

Both actions use `trigger: "click"`. The frontend selects based on the current visual state.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `active` | bool | yes | Initial state. `false` = inactive, `true` = active. |
| `icon_inactive` | string | yes | Icon token shown when `active=false`. |
| `icon_active` | string | yes | Icon token shown when `active=true`. |
| `tooltip_inactive` | string | no | Tooltip text when inactive. |
| `tooltip_active` | string | no | Tooltip text when active. |

```go
IconToggle(id string, active bool, iconInactive, iconActive, tooltipInactive, tooltipActive string, actionOn, actionOff Action) Component
```

```go
c := components.IconToggle("live-toggle", false,
    "radio", "radio",
    "Activate live prices", "Deactivate live prices",
    components.Reload("/actions/portfolio/live_data?live=true", "live-data-section"),
    components.Reload("/actions/portfolio/live_data?live=false", "live-data-section"),
)
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

#### Modal slot pattern

A **modal slot** is a project convention used by every screen with create/edit/delete flows (`assets`, `trades`, `snapshots`, `profile`). It is the stable target a screen's mutation/modal actions point at via `replace`.

**Shape.** Each screen tree has three siblings under its root, in this order:

```
screen
└── column "<screen>-root"  (gap: lg)
    ├── header              (title + global actions)
    ├── section             (filter + table/list + pagination — id "<screen>-section")
    └── modal-slot          (column id="<screen>-modal-slot", initially empty)
```

The modal slot is a `column` with a known id ending in `-modal-slot` (e.g. `snapshots-modal-slot`). It starts with no children.

**Replace targeting.** Actions that open a modal/wizard emit:

```json
{
  "action": "replace",
  "target_id": "<screen>-modal-slot",
  "tree": <subtree to inject>
}
```

The frontend swaps the slot's children for the new subtree.

**Why a sibling of section.**
- Filter/pagination of the section can `replace` `<screen>-section` without disturbing an open modal.
- Mutation success replaces the screen root (`<screen>-root`) entirely — the fresh tree carries an empty modal slot, so the modal closes and the list refreshes in one response.
- The frontend doesn't need to know what kind of subtree lands in the slot — `modal`, `wizard`, or any other component.

**Frontend rendering.** The slot is a presentational container: when it has children, the frontend renders them as an overlay layer above the section (dialog on desktop, drawer/sheet on mobile). Empty slot → no overlay. Components placed inside the slot may carry their own chrome (`modal` has its own title bar and dismiss button) or rely on the slot's overlay container (`wizard` does this).

**Closing a modal.**
- A `dismiss` action on a button → frontend closes the overlay locally and clears the slot.
- A `replace` action on `<screen>-modal-slot` with an empty `tree` → equivalent server-driven close.
- A successful mutation that replaces the screen root → slot is empty in the fresh tree, overlay closes.

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

---

## Form component visibility: `visible_when`

Form components (`input`, `select`, `checkbox`, `textarea`, `radio_group`) accept an optional `visible_when` prop that expresses conditional visibility based on another control's current value.

Structure:

```json
{
  "field": "is_complex",
  "op": "eq",
  "value": false
}
```

When the expression evaluates to `true`, the component is visible; when it evaluates to `false`, the frontend hides it. Hidden components do not contribute to form data on submit.

| Field | Type | Description |
|-------|------|-------------|
| `field` | string | `name` of another form control in the same form |
| `op` | string | `eq` (equals) or `ne` (not equals) |
| `value` | any | String, bool, or number to compare against |

Compound expressions (`and`/`or`) are not defined. If more complex reactive logic is needed, do a server-side round-trip instead.
