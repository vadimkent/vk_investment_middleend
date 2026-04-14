package components

// Series is one line in a line_chart.
type Series struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Color       string `json:"color"`
	ValueFormat string `json:"value_format"`
}

// Axis describes an axis. Key is optional (x-axis only); Format applies to
// both axes' tick labels.
type Axis struct {
	Key    string `json:"key,omitempty"`
	Format string `json:"format,omitempty"`
}

// LineChart creates a line_chart custom component. See
// spec/sdui-custom-components.md §1.
//
// Pass empty string for height / emptyMessage to omit those props.
func LineChart(id, height string, series []Series, xAxis, yAxis Axis, data []map[string]any, emptyMessage string) Component {
	props := map[string]any{
		"series": series,
		"x_axis": xAxis,
		"y_axis": yAxis,
		"data":   data,
	}
	if height != "" {
		props["height"] = height
	}
	if emptyMessage != "" {
		props["empty_message"] = emptyMessage
	}
	return Component{
		Type:  "line_chart",
		ID:    id,
		Props: props,
	}
}
