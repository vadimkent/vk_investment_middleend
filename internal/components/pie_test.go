package components

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPieChart_EmitsTypeAndID(t *testing.T) {
	c := PieChart("x", "md", "donut", "currency_compact",
		[]Slice{{Key: "s1", Label: "S1", Value: 100, Color: "chart_1"}},
		true,
		"",
	)
	assert.Equal(t, "pie_chart", c.Type)
	assert.Equal(t, "x", c.ID)
}

func TestPieChart_AllPropsPresent(t *testing.T) {
	slices := []Slice{{Key: "s1", Label: "S1", Value: 100, Color: "chart_1"}}
	c := PieChart("x", "md", "donut", "currency_compact", slices, true, "No data")
	assert.Equal(t, "md", c.Props["height"])
	assert.Equal(t, "donut", c.Props["shape"])
	assert.Equal(t, "currency_compact", c.Props["value_format"])
	assert.Equal(t, true, c.Props["show_legend"])
	assert.Equal(t, "No data", c.Props["empty_message"])
	_, ok := c.Props["slices"].([]Slice)
	assert.True(t, ok)
}

func TestPieChart_OmitsEmptyHeightAndMessage(t *testing.T) {
	c := PieChart("x", "", "pie", "integer",
		[]Slice{{Key: "s", Label: "S", Value: 1, Color: "chart_1"}},
		false,
		"",
	)
	_, hasHeight := c.Props["height"]
	assert.False(t, hasHeight)
	_, hasEmpty := c.Props["empty_message"]
	assert.False(t, hasEmpty)
	assert.Equal(t, false, c.Props["show_legend"])
}

func TestPieChart_JSONShape(t *testing.T) {
	c := PieChart("chart-allocation", "md", "donut", "currency_compact",
		[]Slice{
			{Key: "AAPL", Label: "AAPL", Value: 12500, Color: "chart_1"},
			{Key: "MSFT", Label: "MSFT", Value: 8200, Color: "chart_2"},
		},
		true,
		"No positions with known value",
	)
	b, err := json.Marshal(c)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"type":"pie_chart"`)
	assert.Contains(t, s, `"id":"chart-allocation"`)
	assert.Contains(t, s, `"shape":"donut"`)
	assert.Contains(t, s, `"value_format":"currency_compact"`)
	assert.Contains(t, s, `"show_legend":true`)
	assert.Contains(t, s, `"slices":[{"key":"AAPL","label":"AAPL","value":12500,"color":"chart_1"},{"key":"MSFT","label":"MSFT","value":8200,"color":"chart_2"}]`)
	assert.Contains(t, s, `"empty_message":"No positions with known value"`)
}
