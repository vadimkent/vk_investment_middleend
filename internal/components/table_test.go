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
