package portfolio

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = i18n.Load(filepath.Join("..", "..", "locales"))
}

func sampleChartPoints(currency string) []EvolutionPoint {
	return []EvolutionPoint{
		{Currency: currency, RecordedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TotalValue: 10000},
		{Currency: currency, RecordedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TotalValue: 10500},
		{Currency: currency, RecordedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), TotalValue: 11000},
	}
}

func TestBuildValueOverTimeCard_RootCard(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	assert.Equal(t, "card", card.Type)
	assert.Equal(t, "chart-value-over-time-card", card.ID)
}

func TestBuildValueOverTimeCard_TimeframeControlsHaveSixButtons(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	tf := findDescendantByID(card, "timeframe-controls")
	require.NotNil(t, tf)
	ids := []string{"chart-timeframe-1m", "chart-timeframe-3m", "chart-timeframe-6m", "chart-timeframe-ytd", "chart-timeframe-1y", "chart-timeframe-all"}
	require.Len(t, tf.Children, 6)
	for i, id := range ids {
		assert.Equal(t, "button", tf.Children[i].Type, "button %d type", i)
		assert.Equal(t, id, tf.Children[i].ID, "button %d id", i)
	}
}

func TestBuildValueOverTimeCard_SelectedTimeframeHasSolidStyle(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "3m", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	selected := findDescendantByID(card, "chart-timeframe-3m")
	require.NotNil(t, selected)
	assert.Equal(t, "primary", selected.Props["variant"])
	assert.Equal(t, "solid", selected.Props["style"])
	unselected := findDescendantByID(card, "chart-timeframe-1y")
	require.NotNil(t, unselected)
	assert.Equal(t, "secondary", unselected.Props["variant"])
	assert.Equal(t, "ghost", unselected.Props["style"])
}

func TestBuildValueOverTimeCard_ButtonURLCarriesFullState(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "3m", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	btn := findDescendantByID(card, "chart-timeframe-6m")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "click", a.Trigger)
	assert.Equal(t, "reload", a.Type)
	assert.Equal(t, "chart-value-over-time-card", a.TargetID)
	assert.Contains(t, a.Endpoint, "timeframe=6m")
	assert.Contains(t, a.Endpoint, "mode=abs")
	assert.Contains(t, a.Endpoint, "currency=USD")
}

func TestBuildValueOverTimeCard_ModeControlsTwoButtons(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	m := findDescendantByID(card, "mode-controls")
	require.NotNil(t, m)
	require.Len(t, m.Children, 2)
	assert.Equal(t, "chart-mode-abs", m.Children[0].ID)
	assert.Equal(t, "chart-mode-pct", m.Children[1].ID)
}

func TestBuildValueOverTimeCard_CurrencyControlsHiddenWhenSingle(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	assert.Nil(t, findDescendantByID(card, "currency-controls"))
}

func TestBuildValueOverTimeCard_CurrencyControlsShownWhenMulti(t *testing.T) {
	points := append(sampleChartPoints("USD"), sampleChartPoints("EUR")...)
	card := BuildValueOverTimeCard(points, ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD", "EUR"}, "en")
	cc := findDescendantByID(card, "currency-controls")
	require.NotNil(t, cc)
	require.Len(t, cc.Children, 2)
	assert.Equal(t, "chart-currency-USD", cc.Children[0].ID)
	assert.Equal(t, "chart-currency-EUR", cc.Children[1].ID)
}

func TestBuildValueOverTimeCard_AbsDataMapping(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, data, 3)
	assert.Equal(t, 10000.0, data[0]["value"])
	assert.Equal(t, 10500.0, data[1]["value"])
	assert.Equal(t, 11000.0, data[2]["value"])
}

func TestBuildValueOverTimeCard_CurrencyFilters(t *testing.T) {
	points := append(sampleChartPoints("USD"), sampleChartPoints("EUR")...)
	card := BuildValueOverTimeCard(points, ChartState{Timeframe: "all", Mode: "abs", Currency: "EUR"}, []string{"USD", "EUR"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, data, 3)
}

func TestBuildValueOverTimeCard_NotEnoughData(t *testing.T) {
	single := []EvolutionPoint{{Currency: "USD", RecordedAt: time.Now(), TotalValue: 100}}
	card := BuildValueOverTimeCard(single, ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	assert.Empty(t, data)
	assert.Equal(t, "Record at least two snapshots to see the chart.", chart.Props["empty_message"])
}
