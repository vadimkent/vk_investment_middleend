package components

// Slice is one slice in a pie_chart.
type Slice struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Color string  `json:"color"`
}

// PieChart creates a pie_chart custom component. See
// spec/sdui-custom-components.md §2.
//
// Pass empty string for height / emptyMessage to omit those props.
// show_legend is always included in the payload.
func PieChart(id, height, shape, valueFormat string, slices []Slice, showLegend bool, emptyMessage string) Component {
	props := map[string]any{
		"shape":        shape,
		"value_format": valueFormat,
		"slices":       slices,
		"show_legend":  showLegend,
	}
	if height != "" {
		props["height"] = height
	}
	if emptyMessage != "" {
		props["empty_message"] = emptyMessage
	}
	return Component{
		Type:  "pie_chart",
		ID:    id,
		Props: props,
	}
}
