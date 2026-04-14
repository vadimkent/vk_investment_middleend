package portfolio

import (
	"testing"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEmpty_HasEmptyBlock(t *testing.T) {
	s := BuildEmpty("en")
	assert.Equal(t, "screen", s.Type)
	assert.Equal(t, "portfolio", s.ID)

	empty := findDescendantByID(s, "portfolio-empty")
	require.NotNil(t, empty)
	title := findDescendantByID(*empty, "empty-title")
	require.NotNil(t, title)
	assert.Equal(t, "No positions yet", title.Props["content"])

	subtitle := findDescendantByID(*empty, "empty-subtitle")
	require.NotNil(t, subtitle)
	assert.Equal(t, "muted", subtitle.Props["color"])
}

func TestBuildEmpty_NoTable(t *testing.T) {
	s := BuildEmpty("en")
	assert.Nil(t, findDescendantByID(s, "positions-table"))
	assert.Nil(t, findDescendantByID(s, "positions-header"))
	assert.Nil(t, findDescendantByID(s, "positions-body"))
}

func TestBuildScreen_RootShape(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	ps := samplePositions()

	s := BuildScreen(ps, "en", now)
	assert.Equal(t, "screen", s.Type)
	assert.Equal(t, "portfolio", s.ID)
	assert.Equal(t, "Portfolio", s.Props["title"])

	summary := findDescendantByID(s, "portfolio-summary")
	require.NotNil(t, summary)
	table := findDescendantByID(s, "positions-table")
	require.NotNil(t, table)
}

func TestBuildScreen_TotalValueSingleCurrency(t *testing.T) {
	now := time.Now()
	v1, v2 := 1000.0, 500.0
	ps := []Position{
		{Ticker: "A", Currency: "USD", CurrentValue: &v1},
		{Ticker: "B", Currency: "USD", CurrentValue: &v2},
	}
	s := BuildScreen(ps, "en", now)

	totals := findDescendantByID(s, "total-values")
	require.NotNil(t, totals)
	require.Len(t, totals.Children, 1)
	assert.Equal(t, "$1,500.00", totals.Children[0].Props["content"])
}

func TestBuildScreen_TotalValueMultiCurrency(t *testing.T) {
	now := time.Now()
	u, e := 1000.0, 800.0
	ps := []Position{
		{Ticker: "A", Currency: "USD", CurrentValue: &u},
		{Ticker: "B", Currency: "EUR", CurrentValue: &e},
	}
	s := BuildScreen(ps, "en", now)

	totals := findDescendantByID(s, "total-values")
	require.NotNil(t, totals)
	require.Len(t, totals.Children, 2)
	assert.Equal(t, "$1,000.00", totals.Children[0].Props["content"])
	assert.Equal(t, "€800.00", totals.Children[1].Props["content"])
}

func TestBuildScreen_TotalValueAllNull(t *testing.T) {
	now := time.Now()
	ps := []Position{{Ticker: "A", Currency: "USD"}}
	s := BuildScreen(ps, "en", now)

	totals := findDescendantByID(s, "total-values")
	require.NotNil(t, totals)
	require.Len(t, totals.Children, 1)
	assert.Equal(t, "—", totals.Children[0].Props["content"])
}

func TestBuildScreen_HeaderHas11Columns(t *testing.T) {
	s := BuildScreen(samplePositions(), "en", time.Now())
	header := findDescendantByID(s, "positions-header")
	require.NotNil(t, header)
	widths, ok := header.Props["widths"].([]string)
	require.True(t, ok)
	assert.Len(t, widths, 11)
	assert.Len(t, header.Children, 11)
	labels := []string{"Ticker", "Name", "Type", "Quantity", "Avg Cost", "Total Cost", "Market Value", "Unrealized P&L", "% P&L", "Realized P&L", "Last Snapshot"}
	for i, want := range labels {
		assert.Equal(t, want, header.Children[i].Props["content"], "col %d", i)
	}
}

func TestBuildScreen_BodyUsesListWithOneItemPerPosition(t *testing.T) {
	ps := samplePositions()
	s := BuildScreen(ps, "en", time.Now())
	body := findDescendantByID(s, "positions-body")
	require.NotNil(t, body)
	assert.Equal(t, "list", body.Type)
	assert.Len(t, body.Children, len(ps))
	for _, child := range body.Children {
		assert.Equal(t, "list_item", child.Type)
	}
}

func TestBuildScreen_PositionRowValuesInOrder(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	qty, avg, total, cur, pnl, realized := 10.0, 153.33, 1533.33, 1855.0, 321.67, 175.0
	snap := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	ps := []Position{{
		AssetID: "a1", Ticker: "AAPL", Name: "Apple Inc", AssetType: "STOCK", Currency: "USD",
		Quantity: &qty, AvgCost: &avg, TotalCost: &total, CurrentValue: &cur,
		UnrealizedPnL: &pnl, RealizedPnL: realized, LastSnapshotAt: &snap,
	}}

	s := BuildScreen(ps, "en", now)
	item := findDescendantByID(s, "position-a1")
	require.NotNil(t, item)
	row := findDescendantByType(*item, "row")
	require.NotNil(t, row)
	require.Len(t, row.Children, 11)

	want := []string{"AAPL", "Apple Inc", "STOCK", "10", "$153.33", "$1,533.33", "$1,855.00", "+$321.67", "+20.98%", "+$175.00", "2 days ago"}
	for i, w := range want {
		assert.Equal(t, w, row.Children[i].Props["content"], "col %d", i)
	}
}

func TestBuildScreen_PositivePnLHasPositiveColor(t *testing.T) {
	now := time.Now()
	tc, cur, pnl := 1000.0, 1200.0, 200.0
	ps := []Position{{
		AssetID: "x1", Ticker: "X", Currency: "USD",
		TotalCost: &tc, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 50.0,
	}}
	s := BuildScreen(ps, "en", now)
	item := findDescendantByID(s, "position-x1")
	require.NotNil(t, item)
	row := findDescendantByType(*item, "row")
	require.NotNil(t, row)

	assert.Equal(t, "positive", row.Children[7].Props["color"])  // unrealized
	assert.Equal(t, "positive", row.Children[8].Props["color"])  // %
	assert.Equal(t, "positive", row.Children[9].Props["color"])  // realized
}

func TestBuildScreen_NegativePnLHasNegativeColor(t *testing.T) {
	now := time.Now()
	tc, cur, pnl := 1000.0, 900.0, -100.0
	ps := []Position{{
		AssetID: "x2", Ticker: "X", Currency: "USD",
		TotalCost: &tc, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: -25.0,
	}}
	s := BuildScreen(ps, "en", now)
	item := findDescendantByID(s, "position-x2")
	require.NotNil(t, item)
	row := findDescendantByType(*item, "row")
	require.NotNil(t, row)

	assert.Equal(t, "negative", row.Children[7].Props["color"])
	assert.Equal(t, "negative", row.Children[8].Props["color"])
	assert.Equal(t, "negative", row.Children[9].Props["color"])
}

func TestBuildScreen_ZeroOrNullPnLHasNoColor(t *testing.T) {
	now := time.Now()
	zero := 0.0
	ps := []Position{{
		AssetID: "x3", Ticker: "X", Currency: "USD",
		UnrealizedPnL: &zero, RealizedPnL: 0.0,
	}}
	s := BuildScreen(ps, "en", now)
	item := findDescendantByID(s, "position-x3")
	require.NotNil(t, item)
	row := findDescendantByType(*item, "row")
	require.NotNil(t, row)

	_, hasColor := row.Children[7].Props["color"]
	assert.False(t, hasColor)
	_, hasColor = row.Children[9].Props["color"]
	assert.False(t, hasColor)
}

// helpers

func samplePositions() []Position {
	qty, avg, total, cur, pnl := 10.0, 100.0, 1000.0, 1200.0, 200.0
	return []Position{
		{AssetID: "s1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD",
			Quantity: &qty, AvgCost: &avg, TotalCost: &total, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 0},
	}
}

func findDescendantByType(c components.Component, typ string) *components.Component {
	if c.Type == typ {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByType(c.Children[i], typ); found != nil {
			return found
		}
	}
	return nil
}

func findDescendantByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByID(c.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}
