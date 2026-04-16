package portfolio

import (
	"testing"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func samplePositionsForAllocation() []Position {
	v1, v2, v3 := 1200.0, 800.0, 400.0
	return []Position{
		{AssetID: "a1", Ticker: "AAPL", AssetType: "STOCK", Currency: "USD", CurrentValue: &v1},
		{AssetID: "a2", Ticker: "BND", AssetType: "BOND", Currency: "USD", CurrentValue: &v2},
		{AssetID: "a3", Ticker: "TSLA", AssetType: "STOCK", Currency: "USD", CurrentValue: &v3},
	}
}

func TestBuildAllocationSection_RootIsColumn(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	assert.Equal(t, "column", s.Type)
	assert.Equal(t, "allocation-section", s.ID)
}

func TestBuildAllocationSection_HasControlsThenCard(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	require.Len(t, s.Children, 2)
	assert.Equal(t, "allocation-controls-row", s.Children[0].ID)
	assert.Equal(t, "chart-allocation-card", s.Children[1].ID)
}

func TestBuildAllocationSection_GroupByControlsHaveTwoButtons(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	gb := findDescendantByID(s, "allocation-group-by-controls")
	require.NotNil(t, gb)
	require.Len(t, gb.Children, 2)
	assert.Equal(t, "allocation-group-by-asset", gb.Children[0].ID)
	assert.Equal(t, "allocation-group-by-type", gb.Children[1].ID)
}

func TestBuildAllocationSection_SelectedGroupByHasSolidStyle(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "type", Currency: "USD"}, []string{"USD"}, "en")
	selected := findDescendantByID(s, "allocation-group-by-type")
	require.NotNil(t, selected)
	assert.Equal(t, "primary", selected.Props["variant"])
	assert.Equal(t, "solid", selected.Props["style"])
	unselected := findDescendantByID(s, "allocation-group-by-asset")
	require.NotNil(t, unselected)
	assert.Equal(t, "secondary", unselected.Props["variant"])
	assert.Equal(t, "ghost", unselected.Props["style"])
}

func TestBuildAllocationSection_ButtonActionTargetsAllocationSection(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	btn := findDescendantByID(s, "allocation-group-by-type")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "click", a.Trigger)
	assert.Equal(t, "reload", a.Type)
	assert.Equal(t, "allocation-section", a.TargetID)
	assert.Contains(t, a.Endpoint, "group_by=type")
	assert.Contains(t, a.Endpoint, "currency=USD")
}

func TestBuildAllocationSection_CurrencyControlsHiddenWhenSingle(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	assert.Nil(t, findDescendantByID(s, "currency-controls"))
}

func TestBuildAllocationSection_CurrencyControlsShownWhenMulti(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD", "EUR"}, "en")
	cc := findDescendantByID(s, "currency-controls")
	require.NotNil(t, cc)
	require.Len(t, cc.Children, 2)
}

func TestBuildAllocationSection_GroupByAssetSlices(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(s, "chart-allocation")
	require.NotNil(t, chart)
	slices, ok := chart.Props["slices"].([]components.Slice)
	require.True(t, ok)
	// Expect 3 slices sorted DESC by value: AAPL(1200), BND(800), TSLA(400).
	require.Len(t, slices, 3)
	assert.Equal(t, "a1", slices[0].Key)
	assert.Equal(t, "AAPL", slices[0].Label)
	assert.InDelta(t, 1200.0, slices[0].Value, 1e-9)
	assert.Equal(t, "chart_1", slices[0].Color)

	assert.Equal(t, "a2", slices[1].Key)
	assert.Equal(t, "BND", slices[1].Label)
	assert.InDelta(t, 800.0, slices[1].Value, 1e-9)
	assert.Equal(t, "chart_2", slices[1].Color)

	assert.Equal(t, "a3", slices[2].Key)
	assert.Equal(t, "TSLA", slices[2].Label)
	assert.InDelta(t, 400.0, slices[2].Value, 1e-9)
	assert.Equal(t, "chart_3", slices[2].Color)
}

func TestBuildAllocationSection_GroupByTypeSlices(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "type", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(s, "chart-allocation")
	require.NotNil(t, chart)
	slices, ok := chart.Props["slices"].([]components.Slice)
	require.True(t, ok)
	// STOCK = AAPL+TSLA = 1600, BOND = BND = 800. DESC by value.
	require.Len(t, slices, 2)
	assert.Equal(t, "STOCK", slices[0].Key)
	assert.Equal(t, "STOCK", slices[0].Label)
	assert.InDelta(t, 1600.0, slices[0].Value, 1e-9)
	assert.Equal(t, "BOND", slices[1].Key)
	assert.InDelta(t, 800.0, slices[1].Value, 1e-9)
}

func TestBuildAllocationSection_FiltersByCurrency(t *testing.T) {
	v := 2000.0
	positions := append(samplePositionsForAllocation(),
		Position{AssetID: "e1", Ticker: "SAP", AssetType: "STOCK", Currency: "EUR", CurrentValue: &v},
	)
	s := BuildAllocationSection(positions, AllocationState{GroupBy: "asset", Currency: "EUR"}, []string{"USD", "EUR"}, "en")
	chart := findDescendantByID(s, "chart-allocation")
	require.NotNil(t, chart)
	slices, ok := chart.Props["slices"].([]components.Slice)
	require.True(t, ok)
	require.Len(t, slices, 1)
	assert.Equal(t, "e1", slices[0].Key)
	assert.Equal(t, "SAP", slices[0].Label)
	assert.InDelta(t, 2000.0, slices[0].Value, 1e-9)
}

func TestBuildAllocationSection_EmptyWhenNoPositionsWithValue(t *testing.T) {
	positions := []Position{
		{AssetID: "n1", Ticker: "NULL1", AssetType: "COMPLEX", Currency: "USD"}, // no current_value
	}
	s := BuildAllocationSection(positions, AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	chart := findDescendantByID(s, "chart-allocation")
	require.NotNil(t, chart)
	slices, ok := chart.Props["slices"].([]components.Slice)
	require.True(t, ok)
	assert.Empty(t, slices)
	assert.Equal(t, "No positions with known value", chart.Props["empty_message"])
}

func TestBuildAllocationSection_CardContainsTitleAndPie(t *testing.T) {
	s := BuildAllocationSection(samplePositionsForAllocation(), AllocationState{GroupBy: "asset", Currency: "USD"}, []string{"USD"}, "en")
	card := findDescendantByID(s, "chart-allocation-card")
	require.NotNil(t, card)
	title := findDescendantByID(*card, "chart-allocation-title")
	require.NotNil(t, title)
	assert.Equal(t, "Allocation", title.Props["content"])
	chart := findDescendantByID(*card, "chart-allocation")
	require.NotNil(t, chart)
	assert.Equal(t, "pie_chart", chart.Type)
	assert.Equal(t, "donut", chart.Props["shape"])
	assert.Equal(t, "currency_compact", chart.Props["value_format"])
	assert.Equal(t, true, chart.Props["show_legend"])
}
