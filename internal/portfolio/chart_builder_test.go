package portfolio

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
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

func TestBuildChartsSection_RootIsColumn(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	assert.Equal(t, "column", s.Type)
	assert.Equal(t, "charts-section", s.ID)
}

func TestBuildChartsSection_ContainsControlsThenValueCard(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	require.GreaterOrEqual(t, len(s.Children), 2)
	assert.Equal(t, "controls-row", s.Children[0].ID)
	assert.Equal(t, "chart-value-over-time-card", s.Children[1].ID)
}

func TestBuildValueOverTimeCard_HasTitleAndChart(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, "en")
	assert.Equal(t, "card", card.Type)
	assert.Equal(t, "chart-value-over-time-card", card.ID)

	title := findDescendantByID(card, "chart-value-over-time-title")
	require.NotNil(t, title)
	assert.Equal(t, "Portfolio Value Over Time", title.Props["content"])

	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	assert.Equal(t, "line_chart", chart.Type)
}

func TestBuildValueOverTimeCard_DoesNotContainControls(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, "en")
	assert.Nil(t, findDescendantByID(card, "controls-row"))
	assert.Nil(t, findDescendantByID(card, "timeframe-controls"))
	assert.Nil(t, findDescendantByID(card, "mode-controls"))
	assert.Nil(t, findDescendantByID(card, "currency-controls"))
}

func TestBuildChartsSection_TimeframeControlsHaveSixButtons(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	tf := findDescendantByID(s, "timeframe-controls")
	require.NotNil(t, tf)
	require.Len(t, tf.Children, 6)
	ids := []string{"chart-timeframe-1m", "chart-timeframe-3m", "chart-timeframe-6m", "chart-timeframe-ytd", "chart-timeframe-1y", "chart-timeframe-all"}
	for i, id := range ids {
		assert.Equal(t, id, tf.Children[i].ID)
	}
}

func TestBuildChartsSection_SelectedTimeframeHasSolidStyle(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "3m", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	selected := findDescendantByID(s, "chart-timeframe-3m")
	require.NotNil(t, selected)
	assert.Equal(t, "primary", selected.Props["variant"])
	assert.Equal(t, "solid", selected.Props["style"])
}

func TestBuildChartsSection_ButtonActionTargetsChartsSection(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "3m", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	btn := findDescendantByID(s, "chart-timeframe-6m")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "reload", a.Type)
	assert.Equal(t, "charts-section", a.TargetID)
	assert.Contains(t, a.Endpoint, "timeframe=6m")
	assert.Contains(t, a.Endpoint, "mode=abs")
	assert.Contains(t, a.Endpoint, "currency=USD")
}

func TestBuildChartsSection_CurrencyControlsHiddenWhenSingle(t *testing.T) {
	s := BuildChartsSection(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	assert.Nil(t, findDescendantByID(s, "currency-controls"))
}

func TestBuildChartsSection_CurrencyControlsShownWhenMulti(t *testing.T) {
	points := append(sampleChartPoints("USD"), sampleChartPoints("EUR")...)
	s := BuildChartsSection(points, ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD", "EUR"}, "en")
	cc := findDescendantByID(s, "currency-controls")
	require.NotNil(t, cc)
	require.Len(t, cc.Children, 2)
	assert.Equal(t, "chart-currency-USD", cc.Children[0].ID)
	assert.Equal(t, "chart-currency-EUR", cc.Children[1].ID)
}

func TestBuildValueOverTimeCard_AbsDataMapping(t *testing.T) {
	card := BuildValueOverTimeCard(sampleChartPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, data, 3)
	assert.Equal(t, 10000.0, data[0]["value"])
	assert.Equal(t, 10500.0, data[1]["value"])
	assert.Equal(t, 11000.0, data[2]["value"])
}

func TestBuildValueOverTimeCard_NotEnoughData(t *testing.T) {
	single := []EvolutionPoint{{Currency: "USD", RecordedAt: time.Now(), TotalValue: 100}}
	card := BuildValueOverTimeCard(single, ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	assert.Empty(t, data)
	assert.Equal(t, "Record at least two snapshots to see the chart.", chart.Props["empty_message"])
}

func sampleAssetPoints(currency string) []EvolutionPoint {
	return []EvolutionPoint{
		{Currency: currency, RecordedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TotalValue: 1500, Assets: []AssetValue{
			{AssetID: "u1", Ticker: "AAPL", Value: 500},
			{AssetID: "u2", Ticker: "GOOG", Value: 1000},
		}},
		{Currency: currency, RecordedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TotalValue: 2100, Assets: []AssetValue{
			{AssetID: "u1", Ticker: "AAPL", Value: 600},
			{AssetID: "u2", Ticker: "GOOG", Value: 1100},
			{AssetID: "u3", Ticker: "TSLA", Value: 400},
		}},
	}
}

func TestBuildAssetValueOverTimeCard_IsCard(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD"}, "en")
	assert.Equal(t, "card", card.Type)
	assert.Equal(t, "chart-asset-value-over-time-card", card.ID)
}

func TestBuildAssetValueOverTimeCard_HasTitle(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD"}, "en")
	title := findDescendantByID(card, "chart-asset-value-over-time-title")
	require.NotNil(t, title)
	assert.Equal(t, "Asset Value Over Time", title.Props["content"])
}

func TestBuildAssetValueOverTimeCard_SeriesPerTicker(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	series, ok := chart.Props["series"].([]components.Series)
	require.True(t, ok)
	// 3 distinct tickers, ordered by latest-point value desc: GOOG(1100), AAPL(600), TSLA(400).
	require.Len(t, series, 3)
	assert.Equal(t, "GOOG", series[0].Key)
	assert.Equal(t, "AAPL", series[1].Key)
	assert.Equal(t, "TSLA", series[2].Key)
	assert.Equal(t, "chart_1", series[0].Color)
	assert.Equal(t, "chart_2", series[1].Color)
	assert.Equal(t, "chart_3", series[2].Color)
	for _, s := range series {
		assert.Equal(t, "currency_compact", s.ValueFormat)
	}
}

func TestBuildAssetValueOverTimeCard_DataRowsHaveAllTickerKeys(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, data, 2)

	assert.Equal(t, 500.0, data[0]["AAPL"])
	assert.Equal(t, 1000.0, data[0]["GOOG"])
	assert.Nil(t, data[0]["TSLA"])

	assert.Equal(t, 600.0, data[1]["AAPL"])
	assert.Equal(t, 1100.0, data[1]["GOOG"])
	assert.Equal(t, 400.0, data[1]["TSLA"])
}

func TestBuildAssetValueOverTimeCard_IgnoresMode(t *testing.T) {
	card := BuildAssetValueOverTimeCard(sampleAssetPoints("USD"), ChartState{Currency: "USD", Mode: "pct"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	series := chart.Props["series"].([]components.Series)
	for _, s := range series {
		assert.Equal(t, "currency_compact", s.ValueFormat, "pct mode should not change asset chart")
	}
}

func TestBuildAssetValueOverTimeCard_FiltersByCurrency(t *testing.T) {
	points := append(sampleAssetPoints("USD"), EvolutionPoint{
		Currency: "EUR", RecordedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		TotalValue: 500, Assets: []AssetValue{{AssetID: "u9", Ticker: "SAP", Value: 500}},
	})
	card := BuildAssetValueOverTimeCard(points, ChartState{Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, data, 2)

	series := chart.Props["series"].([]components.Series)
	for _, s := range series {
		assert.NotEqual(t, "SAP", s.Key, "SAP is EUR and should not appear")
	}
}

func TestBuildAssetValueOverTimeCard_EmptyWhenLessThanTwoPoints(t *testing.T) {
	single := []EvolutionPoint{{Currency: "USD", RecordedAt: time.Now(), TotalValue: 100, Assets: []AssetValue{{AssetID: "u1", Ticker: "AAPL", Value: 100}}}}
	card := BuildAssetValueOverTimeCard(single, ChartState{Currency: "USD"}, "en")
	chart := findDescendantByID(card, "chart-asset-value-over-time")
	require.NotNil(t, chart)
	data, ok := chart.Props["data"].([]map[string]any)
	require.True(t, ok)
	assert.Empty(t, data)
	assert.Equal(t, "Record at least two snapshots to see the chart.", chart.Props["empty_message"])
}

func TestBuildChartsSection_ContainsAssetCard(t *testing.T) {
	s := BuildChartsSection(sampleAssetPoints("USD"), ChartState{Timeframe: "all", Mode: "abs", Currency: "USD"}, []string{"USD"}, "en")
	require.GreaterOrEqual(t, len(s.Children), 3)
	assert.Equal(t, "controls-row", s.Children[0].ID)
	assert.Equal(t, "chart-value-over-time-card", s.Children[1].ID)
	assert.Equal(t, "chart-asset-value-over-time-card", s.Children[2].ID)
}
