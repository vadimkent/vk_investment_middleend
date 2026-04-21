package components

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTable_EmitsTypeAndColumns(t *testing.T) {
	cols := []TableColumn{
		{ID: "ticker", Header: "Ticker", Width: "80px"},
		{ID: "name", Header: "Name", Width: "1fr"},
		{ID: "value", Header: "Value", Width: "120px", Align: "right"},
	}
	c := Table("t1", cols,
		TableRow("r1", Text("c1", "AAPL", "sm", "bold")),
	)
	assert.Equal(t, "table", c.Type)
	assert.Equal(t, "t1", c.ID)
	assert.Equal(t, cols, c.Props["columns"])
	require.Len(t, c.Children, 1)
	assert.Equal(t, "table_row", c.Children[0].Type)
}

func TestTableRow_EmitsTypeAndChildren(t *testing.T) {
	r := TableRow("r1",
		Text("c1", "AAPL", "sm", "bold"),
		Text("c2", "Apple", "sm", "normal"),
	)
	assert.Equal(t, "table_row", r.Type)
	assert.Equal(t, "r1", r.ID)
	require.Len(t, r.Children, 2)
}

func TestTableColumn_OmitsEmptyWidthAndAlign(t *testing.T) {
	col := TableColumn{ID: "x", Header: "X"}
	b, err := json.Marshal(col)
	require.NoError(t, err)
	s := string(b)
	assert.NotContains(t, s, "width")
	assert.NotContains(t, s, "align")
}

func TestTableColumn_IncludesWidthAndAlign(t *testing.T) {
	col := TableColumn{ID: "v", Header: "Value", Width: "120px", Align: "right"}
	b, err := json.Marshal(col)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"width":"120px"`)
	assert.Contains(t, s, `"align":"right"`)
}

func TestTable_JSONShape(t *testing.T) {
	cols := []TableColumn{
		{ID: "ticker", Header: "Ticker", Width: "80px"},
	}
	c := Table("tbl", cols,
		TableRow("r1", Text("c1", "AAPL", "sm", "bold")),
	)
	b, err := json.Marshal(c)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"type":"table"`)
	assert.Contains(t, s, `"columns":[{"id":"ticker","header":"Ticker","width":"80px"}]`)
	assert.Contains(t, s, `"type":"table_row"`)
}

func TestTableRowExpandable_SetsProps(t *testing.T) {
	details := []Component{Text("entry-1", "detail cell", "sm", "normal")}
	row := TableRowExpandable("row-1",
		[]Component{Text("c1", "Date", "sm", "normal")},
		details...,
	)

	if row.Type != "table_row" {
		t.Fatalf("type = %q, want table_row", row.Type)
	}
	if row.ID != "row-1" {
		t.Fatalf("id = %q, want row-1", row.ID)
	}
	if got := row.Props["expandable"]; got != true {
		t.Fatalf("props.expandable = %v, want true", got)
	}
	got, ok := row.Props["details"].([]Component)
	if !ok {
		t.Fatalf("props.details not []Component, got %T", row.Props["details"])
	}
	if len(got) != 1 || got[0].ID != "entry-1" {
		t.Fatalf("details mismatch: %+v", got)
	}
	if len(row.Children) != 1 || row.Children[0].ID != "c1" {
		t.Fatalf("cells mismatch: %+v", row.Children)
	}
}

func TestTableRowExpandable_JSONShape(t *testing.T) {
	row := TableRowExpandable("row-1",
		[]Component{Text("c1", "hello", "sm", "normal")},
		Text("d1", "detail", "sm", "normal"),
	)
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		Props struct {
			Expandable bool        `json:"expandable"`
			Details    []Component `json:"details"`
		} `json:"props"`
		Children []Component `json:"children"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Props.Expandable {
		t.Fatalf("json props.expandable not true: %s", string(b))
	}
	if len(out.Props.Details) != 1 || out.Props.Details[0].ID != "d1" {
		t.Fatalf("json props.details bad: %s", string(b))
	}
	if len(out.Children) != 1 || out.Children[0].ID != "c1" {
		t.Fatalf("json children bad: %s", string(b))
	}
}

func TestTableRow_UnchangedByExpandableAddition(t *testing.T) {
	// Regression: the original TableRow helper must not set expandable/details.
	row := TableRow("row-1", Text("c1", "hello", "sm", "normal"))
	if _, ok := row.Props["expandable"]; ok {
		t.Fatalf("TableRow should not set expandable prop: %+v", row.Props)
	}
	if _, ok := row.Props["details"]; ok {
		t.Fatalf("TableRow should not set details prop: %+v", row.Props)
	}
}
