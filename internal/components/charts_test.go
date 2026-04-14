package components

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLineChart_EmitsTypeAndID(t *testing.T) {
	c := LineChart("x",
		"md",
		[]Series{{Key: "v", Label: "Value", Color: "chart_1", ValueFormat: "currency_compact"}},
		Axis{Key: "date", Format: "month_year"},
		Axis{Format: "currency_compact"},
		[]map[string]any{{"date": "2026-01-01", "v": 100.0}},
		"Not enough data",
	)
	assert.Equal(t, "line_chart", c.Type)
	assert.Equal(t, "x", c.ID)
}

func TestLineChart_AllPropsPresent(t *testing.T) {
	data := []map[string]any{{"date": "2026-01-01", "v": 100.0}}
	c := LineChart("x",
		"md",
		[]Series{{Key: "v", Label: "Value", Color: "chart_1", ValueFormat: "currency_compact"}},
		Axis{Key: "date", Format: "month_year"},
		Axis{Format: "currency_compact"},
		data,
		"Not enough data",
	)
	assert.Equal(t, "md", c.Props["height"])
	assert.Equal(t, "Not enough data", c.Props["empty_message"])
	assert.Equal(t, data, c.Props["data"])
	_, ok := c.Props["series"].([]Series)
	assert.True(t, ok)
	_, ok = c.Props["x_axis"].(Axis)
	assert.True(t, ok)
	_, ok = c.Props["y_axis"].(Axis)
	assert.True(t, ok)
}

func TestLineChart_OmitsEmptyHeight(t *testing.T) {
	c := LineChart("x",
		"",
		[]Series{{Key: "v", Label: "V", Color: "chart_1", ValueFormat: "currency"}},
		Axis{Key: "date", Format: "date"},
		Axis{},
		[]map[string]any{},
		"",
	)
	_, hasHeight := c.Props["height"]
	assert.False(t, hasHeight)
	_, hasEmpty := c.Props["empty_message"]
	assert.False(t, hasEmpty)
}

func TestLineChart_JSONShape(t *testing.T) {
	c := LineChart("chart-value-over-time",
		"md",
		[]Series{{Key: "value", Label: "Value", Color: "chart_1", ValueFormat: "currency_compact"}},
		Axis{Key: "date", Format: "month_year"},
		Axis{Format: "currency_compact"},
		[]map[string]any{{"date": "2026-01-01", "value": 100.0}},
		"Not enough data",
	)
	b, err := json.Marshal(c)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"type":"line_chart"`)
	assert.Contains(t, s, `"id":"chart-value-over-time"`)
	assert.Contains(t, s, `"series":[{"key":"value","label":"Value","color":"chart_1","value_format":"currency_compact"}]`)
	assert.Contains(t, s, `"x_axis":{"key":"date","format":"month_year"}`)
	assert.Contains(t, s, `"y_axis":{"format":"currency_compact"}`)
	assert.Contains(t, s, `"data":[{"date":"2026-01-01","value":100}]`)
	assert.Contains(t, s, `"empty_message":"Not enough data"`)
}
