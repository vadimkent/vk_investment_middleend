package components

// Component represents an SDUI component node.
type Component struct {
	Type     string            `json:"type"`
	ID       string            `json:"id"`
	Props    map[string]any    `json:"props"`
	Children []Component       `json:"children,omitempty"`
	Actions  []Action          `json:"actions,omitempty"`
}

// Screen creates a top-level screen component.
func Screen(id, title string, children ...Component) Component {
	return Component{
		Type:     "screen",
		ID:       id,
		Props:    map[string]any{"title": title},
		Children: children,
	}
}

// ScreenFull creates a screen with optional subtitle, icon, and back navigation.
func ScreenFull(id, title, subtitle, icon string, backAction *Action, children ...Component) Component {
	props := map[string]any{"title": title}
	if subtitle != "" {
		props["subtitle"] = subtitle
	}
	if icon != "" {
		props["icon"] = icon
	}
	var actions []Action
	if backAction != nil {
		props["back_action"] = true
		actions = append(actions, *backAction)
	}
	return Component{
		Type:     "screen",
		ID:       id,
		Props:    props,
		Children: children,
		Actions:  actions,
	}
}

// Text creates a text component with size and weight.
func Text(id, content, size, weight string) Component {
	return Component{
		Type: "text",
		ID:   id,
		Props: map[string]any{
			"content": content,
			"size":    size,
			"weight":  weight,
		},
	}
}

// TextStyled creates a text component with full visual control.
// Pass empty string for display/color/hexColor/decoration to use defaults.
func TextStyled(id, content, size, weight, display, color, hexColor, decoration string) Component {
	props := map[string]any{
		"content": content,
		"size":    size,
		"weight":  weight,
	}
	if display != "" {
		props["display"] = display
	}
	if color != "" {
		props["color"] = color
	}
	if hexColor != "" {
		props["hex_color"] = hexColor
	}
	if decoration != "" {
		props["decoration"] = decoration
	}
	return Component{
		Type:  "text",
		ID:    id,
		Props: props,
	}
}

// Button creates a button component.
func Button(id, label string, actions ...Action) Component {
	return Component{
		Type:    "button",
		ID:      id,
		Props:   map[string]any{"label": label, "variant": "primary", "style": "solid"},
		Actions: actions,
	}
}

// ButtonFull creates a button with all visual options.
func ButtonFull(id, label, imageSrc, variant, style string, actions ...Action) Component {
	props := map[string]any{"variant": variant, "style": style}
	if label != "" {
		props["label"] = label
	}
	if imageSrc != "" {
		props["image_src"] = imageSrc
	}
	return Component{
		Type:    "button",
		ID:      id,
		Props:   props,
		Actions: actions,
	}
}

// Card creates a card container component.
func Card(id string, children ...Component) Component {
	return Component{
		Type:     "card",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// CardStyled creates a card with elevation and border radius control.
func CardStyled(id, elevation, borderRadius string, children ...Component) Component {
	props := map[string]any{}
	if elevation != "" {
		props["elevation"] = elevation
	}
	if borderRadius != "" {
		props["border_radius"] = borderRadius
	}
	return Component{
		Type:     "card",
		ID:       id,
		Props:    props,
		Children: children,
	}
}

// Row creates a horizontal layout container with width distribution.
func Row(id string, widths []string, children ...Component) Component {
	return Component{
		Type:     "row",
		ID:       id,
		Props:    map[string]any{"widths": widths},
		Children: children,
	}
}

// RowWithGap creates a row with a gap between children.
func RowWithGap(id string, widths []string, gap string, children ...Component) Component {
	return Component{
		Type:     "row",
		ID:       id,
		Props:    map[string]any{"widths": widths, "gap": gap},
		Children: children,
	}
}

// Column creates a vertical layout container.
func Column(id string, children ...Component) Component {
	return Component{
		Type:     "column",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// ColumnWithGap creates a column with a gap between children.
func ColumnWithGap(id, gap string, children ...Component) Component {
	return Component{
		Type:     "column",
		ID:       id,
		Props:    map[string]any{"gap": gap},
		Children: children,
	}
}

// NavHeader creates a navigation header slot.
func NavHeader(id string, children ...Component) Component {
	return Component{
		Type:     "nav_header",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// NavMain creates a primary navigation slot.
func NavMain(id string, children ...Component) Component {
	return Component{
		Type:     "nav_main",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// NavFooter creates a navigation footer slot.
func NavFooter(id string, children ...Component) Component {
	return Component{
		Type:     "nav_footer",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// BottomBar creates a fixed bottom bar slot.
func BottomBar(id string, children ...Component) Component {
	return Component{
		Type:     "bottombar",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// ContentSlot creates a content placeholder slot.
func ContentSlot(id string) Component {
	return Component{
		Type:  "content_slot",
		ID:    id,
		Props: map[string]any{},
	}
}

// NavItem creates a navigation link component.
func NavItem(id, label, icon, route string, actions ...Action) Component {
	return Component{
		Type:    "nav_item",
		ID:      id,
		Props:   map[string]any{"label": label, "icon": icon, "route": route},
		Actions: actions,
	}
}

// Image creates an image component.
func Image(id, src, alt string) Component {
	return Component{
		Type:  "image",
		ID:    id,
		Props: map[string]any{"src": src, "alt": alt},
	}
}

// ImageStyled creates an image with full visual control.
func ImageStyled(id, src, alt, width, height, fit, borderRadius string) Component {
	props := map[string]any{"src": src, "alt": alt}
	if width != "" {
		props["width"] = width
	}
	if height != "" {
		props["height"] = height
	}
	if fit != "" {
		props["fit"] = fit
	}
	if borderRadius != "" {
		props["border_radius"] = borderRadius
	}
	return Component{
		Type:  "image",
		ID:    id,
		Props: props,
	}
}

// Loading creates a loading indicator component.
func Loading(id string) Component {
	return Component{
		Type:  "loading",
		ID:    id,
		Props: map[string]any{},
	}
}

// LoadingStyled creates a loading indicator with size and variant.
func LoadingStyled(id, size, variant string) Component {
	props := map[string]any{}
	if size != "" {
		props["size"] = size
	}
	if variant != "" {
		props["variant"] = variant
	}
	return Component{
		Type:  "loading",
		ID:    id,
		Props: props,
	}
}

// ErrorComponent creates an error display component.
func ErrorComponent(id, message string) Component {
	return Component{
		Type:  "error",
		ID:    id,
		Props: map[string]any{"message": message},
	}
}

// ErrorWithRetry creates an error display with a retry action.
func ErrorWithRetry(id, message string, retryAction Action) Component {
	return Component{
		Type:    "error",
		ID:      id,
		Props:   map[string]any{"message": message, "retry_action": true},
		Actions: []Action{retryAction},
	}
}

// List creates a scrollable list container.
func List(id string, children ...Component) Component {
	return Component{
		Type:     "list",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// ListHorizontal creates a horizontal scrollable list.
func ListHorizontal(id string, children ...Component) Component {
	return Component{
		Type:     "list",
		ID:       id,
		Props:    map[string]any{"orientation": "horizontal"},
		Children: children,
	}
}

// ListItem creates a single item within a list.
func ListItem(id string, children ...Component) Component {
	return Component{
		Type:     "list_item",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// Input creates a text input field component.
func Input(id, placeholder, name, inputType string) Component {
	return Component{
		Type: "input",
		ID:   id,
		Props: map[string]any{
			"placeholder": placeholder,
			"name":        name,
			"input_type":  inputType,
		},
	}
}

// InputFull creates an input with all available options.
func InputFull(id, name, inputType, label, placeholder, defaultValue string, required, disabled bool, maxLength int) Component {
	props := map[string]any{
		"name":       name,
		"input_type": inputType,
	}
	if label != "" {
		props["label"] = label
	}
	if placeholder != "" {
		props["placeholder"] = placeholder
	}
	if defaultValue != "" {
		props["default_value"] = defaultValue
	}
	if required {
		props["required"] = true
	}
	if disabled {
		props["disabled"] = true
	}
	if maxLength > 0 {
		props["max_length"] = maxLength
	}
	return Component{
		Type:  "input",
		ID:    id,
		Props: props,
	}
}

// Form creates a form container that groups inputs for submission.
func Form(id string, children ...Component) Component {
	return Component{
		Type:     "form",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// Divider creates a horizontal separator line.
func Divider(id string) Component {
	return Component{
		Type:  "divider",
		ID:    id,
		Props: map[string]any{},
	}
}

// DividerVertical creates a vertical separator line.
func DividerVertical(id string) Component {
	return Component{
		Type:  "divider",
		ID:    id,
		Props: map[string]any{"orientation": "vertical"},
	}
}

// Spacer creates an invisible spacing element.
func Spacer(id, size string) Component {
	return Component{
		Type:  "spacer",
		ID:    id,
		Props: map[string]any{"size": size},
	}
}

// Group creates a generic container without visual styling.
func Group(id string, children ...Component) Component {
	return Component{
		Type:     "group",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}

// Snackbar creates a temporary feedback message component.
func Snackbar(id, message, variant string) Component {
	return Component{
		Type: "snackbar",
		ID:   id,
		Props: map[string]any{"message": message, "variant": variant},
	}
}

// SelectOption represents a single option in a select or radio_group.
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Select creates a dropdown selection field.
func Select(id, name string, options []SelectOption) Component {
	return Component{
		Type: "select",
		ID:   id,
		Props: map[string]any{
			"name":    name,
			"options": options,
		},
	}
}

// SelectFull creates a dropdown with all available options.
func SelectFull(id, name, label, placeholder, defaultValue string, options []SelectOption, required, disabled bool) Component {
	props := map[string]any{
		"name":    name,
		"options": options,
	}
	if label != "" {
		props["label"] = label
	}
	if placeholder != "" {
		props["placeholder"] = placeholder
	}
	if defaultValue != "" {
		props["default_value"] = defaultValue
	}
	if required {
		props["required"] = true
	}
	if disabled {
		props["disabled"] = true
	}
	return Component{
		Type:  "select",
		ID:    id,
		Props: props,
	}
}

// Checkbox creates a single boolean toggle with label.
func Checkbox(id, name, label string) Component {
	return Component{
		Type: "checkbox",
		ID:   id,
		Props: map[string]any{
			"name":  name,
			"label": label,
		},
	}
}

// Toggle creates an on/off switch with label.
func Toggle(id, name, label string) Component {
	return Component{
		Type: "toggle",
		ID:   id,
		Props: map[string]any{
			"name":  name,
			"label": label,
		},
	}
}

// Textarea creates a multi-line text input.
func Textarea(id, name string) Component {
	return Component{
		Type: "textarea",
		ID:   id,
		Props: map[string]any{
			"name": name,
		},
	}
}

// TextareaFull creates a textarea with all available options.
func TextareaFull(id, name, label, placeholder, defaultValue string, rows, maxLength int, required, disabled bool) Component {
	props := map[string]any{
		"name": name,
	}
	if label != "" {
		props["label"] = label
	}
	if placeholder != "" {
		props["placeholder"] = placeholder
	}
	if defaultValue != "" {
		props["default_value"] = defaultValue
	}
	if rows > 0 {
		props["rows"] = rows
	}
	if maxLength > 0 {
		props["max_length"] = maxLength
	}
	if required {
		props["required"] = true
	}
	if disabled {
		props["disabled"] = true
	}
	return Component{
		Type:  "textarea",
		ID:    id,
		Props: props,
	}
}

// RadioGroup creates a set of mutually exclusive options.
func RadioGroup(id, name string, options []SelectOption) Component {
	return Component{
		Type: "radio_group",
		ID:   id,
		Props: map[string]any{
			"name":    name,
			"options": options,
		},
	}
}

// Modal creates a basic overlay dialog.
func Modal(id string, visible bool, children ...Component) Component {
	return Component{
		Type:     "modal",
		ID:       id,
		Props:    map[string]any{"visible": visible},
		Children: children,
	}
}

// ModalFull creates a modal with all available options.
func ModalFull(id, title, presentation string, visible, dismissible bool, children ...Component) Component {
	props := map[string]any{"visible": visible}
	if title != "" {
		props["title"] = title
	}
	if presentation != "" {
		props["presentation"] = presentation
	}
	if !dismissible {
		props["dismissible"] = false
	}
	return Component{
		Type:     "modal",
		ID:       id,
		Props:    props,
		Children: children,
	}
}

// Badge creates a dot indicator overlaid on a child component.
func Badge(id string, child Component) Component {
	return Component{
		Type:     "badge",
		ID:       id,
		Props:    map[string]any{},
		Children: []Component{child},
	}
}

// BadgeCount creates a badge with a count number and variant.
func BadgeCount(id string, count int, variant string, child Component) Component {
	props := map[string]any{"count": count}
	if variant != "" {
		props["variant"] = variant
	}
	return Component{
		Type:     "badge",
		ID:       id,
		Props:    props,
		Children: []Component{child},
	}
}

// VisibleWhen expresses conditional visibility of a form component based on
// another control's current value. When the expression evaluates false in the
// current form state, the frontend hides the component.
//
// Supported ops: "eq" (equals), "ne" (not equals).
// Field must match the `name` prop of another control in the same form.
type VisibleWhen struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}
