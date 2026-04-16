package components

// TableColumn defines a single column in a table.
type TableColumn struct {
	ID     string `json:"id"`
	Header string `json:"header"`
	Width  string `json:"width,omitempty"`
	Align  string `json:"align,omitempty"`
}

// Table creates a table component. The frontend renders the header
// automatically from columns[].Header. Children must be table_row components.
func Table(id string, columns []TableColumn, children ...Component) Component {
	return Component{
		Type:     "table",
		ID:       id,
		Props:    map[string]any{"columns": columns},
		Children: children,
	}
}

// TableRow creates a table_row component. Each child maps to a column
// in order via CSS subgrid.
func TableRow(id string, children ...Component) Component {
	return Component{
		Type:     "table_row",
		ID:       id,
		Props:    map[string]any{},
		Children: children,
	}
}
