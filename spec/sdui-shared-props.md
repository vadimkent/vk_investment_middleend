# SDUI Shared Props — Go Reference

Shared props are optional properties that can appear on any component. The frontend reads and applies them automatically. The middleend sets them directly in the `Props` map.

---

## 1. Container Alignment

Available on components with children: `row`, `column`, `group`, `card`, `list_item`, `form`, `nav_header`, `nav_main`, `nav_footer`, `bottombar`.

| Prop | Values | Description |
|------|--------|-------------|
| `align_items` | `left` / `center` / `right` / `stretch` | Horizontal alignment of children |
| `justify_items` | `top` / `center` / `bottom` / `stretch` | Vertical alignment of children |

```go
col := components.Column("centered")
col.Props["align_items"] = "center"
col.Props["justify_items"] = "center"
```

---

## 2. Self Alignment

Available on any component. Overrides the parent's alignment for this component.

| Prop | Values | Description |
|------|--------|-------------|
| `align_self` | `left` / `center` / `right` | Override parent's horizontal alignment |
| `justify_self` | `top` / `center` / `bottom` | Override parent's vertical alignment |

```go
btn := components.Button("submit", "Save", components.Submit("/api/save", "POST", "form-1"))
btn.Props["align_self"] = "right"
```

---

## 3. Spacing

Available on components with children.

| Prop | Values | Description |
|------|--------|-------------|
| `gap` | enum token (see below) | Space between children |

`gap` is a semantic token. Raw CSS values (`"16px"`, `"1rem"`, etc.) are ignored by the frontend.

| Token | px |
|-------|-----|
| `none` | 0 |
| `xs`   | 4 |
| `sm`   | 8 |
| `md`   | 16 |
| `lg`   | 24 |
| `xl`   | 32 |
| `2xl`  | 48 |

The `RowWithGap` and `ColumnWithGap` helpers take a gap value. For other containers, set it directly:

```go
card := components.Card("info")
card.Props["gap"] = "md"
```

---

## 4. Positioning

Available on any component.

| Prop | Values | Description |
|------|--------|-------------|
| `position` | `static` / `fixed` / `absolute` | Positioning mode |

- `static` — Default flow positioning.
- `fixed` — Positioned relative to the viewport. Use `align_self`/`justify_self` for placement.
- `absolute` — Positioned relative to the nearest positioned parent. Use `align_self`/`justify_self` for placement.

```go
fab := components.Button("fab", "+", components.Navigate("/items/new"))
fab.Props["position"] = "fixed"
fab.Props["align_self"] = "right"
fab.Props["justify_self"] = "bottom"
```

---

## 5. Usage Pattern

Since shared props are not part of the component helpers' signatures, set them after construction:

```go
// Build the component
row := components.Row("header-row", []string{"1fr", "auto"},
    components.Text("title", "Dashboard", "xl", "bold"),
    components.Button("action", "New", components.Navigate("/new")),
)

// Apply shared props
row.Props["align_items"] = "center"
row.Props["justify_items"] = "center"
```

The frontend handles these props at render time. The middleend only needs to include them in the props map when the default behavior needs to be overridden.
