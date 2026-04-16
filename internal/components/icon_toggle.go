package components

// IconToggle creates an icon_toggle base component — a binary toggle rendered
// as a clickable icon with two states. See sdui-base-components.md.
//
// actionOn fires when transitioning inactive→active (actions[0]).
// actionOff fires when transitioning active→inactive (actions[1]).
func IconToggle(id string, active bool, iconInactive, iconActive, tooltipInactive, tooltipActive string, actionOn, actionOff Action) Component {
	props := map[string]any{
		"active":        active,
		"icon_inactive": iconInactive,
		"icon_active":   iconActive,
	}
	if tooltipInactive != "" {
		props["tooltip_inactive"] = tooltipInactive
	}
	if tooltipActive != "" {
		props["tooltip_active"] = tooltipActive
	}
	return Component{
		Type:    "icon_toggle",
		ID:      id,
		Props:   props,
		Actions: []Action{actionOn, actionOff},
	}
}
