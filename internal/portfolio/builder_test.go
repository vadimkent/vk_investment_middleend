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
}

func TestBuildEmpty_NoSummaryCards(t *testing.T) {
	s := BuildEmpty("en")
	for _, id := range []string{"summary-card-total-value", "summary-card-total-pnl", "summary-card-performance", "summary-card-snapshot-change", "summary-card-open-positions"} {
		assert.Nil(t, findDescendantByID(s, id), "unexpected %s in empty tree", id)
	}
}

func TestBuildScreen_SummaryRowHasFiveCardsInOrder(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	row := findDescendantByID(s, "portfolio-summary-row")
	require.NotNil(t, row)

	widths, ok := row.Props["widths"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"1fr", "1fr", "1fr", "1fr", "1fr"}, widths)

	want := []string{
		"summary-card-total-value",
		"summary-card-total-pnl",
		"summary-card-performance",
		"summary-card-snapshot-change",
		"summary-card-open-positions",
	}
	require.Len(t, row.Children, 5)
	for i, id := range want {
		assert.Equal(t, "card", row.Children[i].Type, "child %d type", i)
		assert.Equal(t, id, row.Children[i].ID, "child %d id", i)
	}
}

func TestBuildScreen_SummaryCardLabelsLocalized(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "es", time.Now())
	cases := map[string]string{
		"summary-label-total-value":     "Valor total",
		"summary-label-total-pnl":       "G/P total",
		"summary-label-performance":     "Rendimiento total",
		"summary-label-snapshot-change": "Cambio último snapshot",
		"summary-label-open-positions":  "Posiciones abiertas",
	}
	for id, want := range cases {
		node := findDescendantByID(s, id)
		require.NotNil(t, node, "missing %s", id)
		assert.Equal(t, want, node.Props["content"])
		assert.Equal(t, "muted", node.Props["color"])
	}
}

func TestBuildScreen_TotalValueOneLinePerCurrencyDesc(t *testing.T) {
	u := 500.0
	e := 1500.0
	ps := []Position{
		{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &u},
		{AssetID: "b1", Ticker: "B", Currency: "EUR", CurrentValue: &e},
	}
	s := BuildScreen(&PortfolioResponse{Positions: ps}, nil, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-total-value")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 2)
	assert.Equal(t, "summary-value-total-value-EUR", vals.Children[0].ID)
	assert.Equal(t, "€1,500.00", vals.Children[0].Props["content"])
	assert.Equal(t, "summary-value-total-value-USD", vals.Children[1].ID)
	assert.Equal(t, "$500.00", vals.Children[1].Props["content"])
}

func TestBuildScreen_TotalValueEmptyShowsDash(t *testing.T) {
	ps := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD"}}
	s := BuildScreen(&PortfolioResponse{Positions: ps}, nil, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-total-value")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "summary-value-total-value-empty", vals.Children[0].ID)
	assert.Equal(t, "—", vals.Children[0].Props["content"])
}

func TestBuildScreen_TotalPnLSignedAndColored(t *testing.T) {
	tc := 1000.0
	cur := 1200.0
	pnl := 200.0
	positives := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD",
		TotalCost: &tc, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 50.0}}
	s := BuildScreen(&PortfolioResponse{Positions: positives}, nil, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-total-pnl")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "+$250.00", vals.Children[0].Props["content"])
	assert.Equal(t, "positive", vals.Children[0].Props["color"])
}

func TestBuildScreen_PerformanceFallsBackToDash(t *testing.T) {
	u := 100.0
	ps := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &u}}
	s := BuildScreen(&PortfolioResponse{Positions: ps}, nil, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-performance")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "—", vals.Children[0].Props["content"])
	_, hasColor := vals.Children[0].Props["color"]
	assert.False(t, hasColor)
}

func TestBuildScreen_SnapshotChangeBasic(t *testing.T) {
	u := 1000.0
	ps := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
	}
	s := BuildScreen(&PortfolioResponse{Positions: ps}, evo, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-snapshot-change")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "+10.00%", vals.Children[0].Props["content"])
	assert.Equal(t, "positive", vals.Children[0].Props["color"])
}

func TestBuildScreen_SnapshotChangeDashWhenNoEvolution(t *testing.T) {
	u := 1000.0
	ps := []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &u}}
	s := BuildScreen(&PortfolioResponse{Positions: ps}, nil, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-snapshot-change")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "—", vals.Children[0].Props["content"])
}

func TestBuildScreen_OpenPositionsCount(t *testing.T) {
	u1, u2, u3 := 1.0, 2.0, 3.0
	ps := []Position{
		{AssetID: "a", Ticker: "A", Currency: "USD", CurrentValue: &u1},
		{AssetID: "b", Ticker: "B", Currency: "USD", CurrentValue: &u2},
		{AssetID: "c", Ticker: "C", Currency: "USD", CurrentValue: &u3},
	}
	s := BuildScreen(&PortfolioResponse{Positions: ps}, nil, nil, "en", time.Now())
	vals := findDescendantByID(s, "summary-values-open-positions")
	require.NotNil(t, vals)
	require.Len(t, vals.Children, 1)
	assert.Equal(t, "summary-value-open-positions", vals.Children[0].ID)
	assert.Equal(t, "3", vals.Children[0].Props["content"])
}

func TestBuildScreen_PositionsTablePreservedFromLayer1(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	assert.NotNil(t, findDescendantByID(s, "positions-table-card"))
	assert.NotNil(t, findDescendantByID(s, "positions-table"))
}

func TestBuildScreen_TableHas11Columns(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	table := findDescendantByID(s, "positions-table")
	require.NotNil(t, table)
	cols, ok := table.Props["columns"].([]components.TableColumn)
	require.True(t, ok)
	assert.Len(t, cols, 11)
}

func TestBuildScreen_TableHasOneRowPerPosition(t *testing.T) {
	ps := samplePositions()
	s := BuildScreen(&PortfolioResponse{Positions: ps}, nil, nil, "en", time.Now())
	table := findDescendantByID(s, "positions-table")
	require.NotNil(t, table)
	assert.Equal(t, "table", table.Type)
	assert.Len(t, table.Children, len(ps))
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

	s := BuildScreen(&PortfolioResponse{Positions: ps}, nil, nil, "en", now)
	row := findDescendantByID(s, "position-a1")
	require.NotNil(t, row)
	assert.Equal(t, "table_row", row.Type)
	require.Len(t, row.Children, 11)

	want := []string{"AAPL", "Apple Inc", "STOCK", "10", "$153.33", "$1,533.33", "$1,855.00", "+$321.67", "+20.98%", "+$175.00", "2 days ago"}
	for i, w := range want {
		assert.Equal(t, w, row.Children[i].Props["content"], "col %d", i)
	}
}

// helpers

func samplePositions() []Position {
	qty, avg, total, cur, pnl := 10.0, 100.0, 1000.0, 1200.0, 200.0
	return []Position{
		{AssetID: "s1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD",
			Quantity: &qty, AvgCost: &avg, TotalCost: &total, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 0},
	}
}


func TestBuildPositionsTable_ReturnsCardWithExpectedID(t *testing.T) {
	ps := samplePositions()
	card := BuildPositionsTable(ps, "en", time.Now(), false)

	assert.Equal(t, "card", card.Type)
	assert.Equal(t, "positions-table-card", card.ID)

	table := findDescendantByID(card, "positions-table")
	require.NotNil(t, table)
	assert.Equal(t, "table", table.Type)

	cols, ok := table.Props["columns"].([]components.TableColumn)
	require.True(t, ok)
	assert.Len(t, cols, 11)

	require.Len(t, table.Children, len(ps))
	for _, child := range table.Children {
		assert.Equal(t, "table_row", child.Type)
	}
}

func TestBuildPositionsTable_MonetaryCellsAreSensitive(t *testing.T) {
	ps := samplePositions()
	card := BuildPositionsTable(ps, "en", time.Now(), false)
	table := findDescendantByID(card, "positions-table")
	require.NotNil(t, table)
	require.Len(t, table.Children, 1)
	row := table.Children[0]

	sensitiveIDs := []string{"cell-avg-cost", "cell-total-cost", "cell-market-value", "cell-unrealized-pnl", "cell-realized-pnl"}
	for _, id := range sensitiveIDs {
		cell := findDescendantByID(row, id)
		require.NotNil(t, cell, "missing %s", id)
		assert.Equal(t, true, cell.Props["sensitive"], "%s should be sensitive", id)
	}

	notSensitiveIDs := []string{"cell-ticker", "cell-name", "cell-type", "cell-quantity", "cell-pnl-pct", "cell-last-snapshot"}
	for _, id := range notSensitiveIDs {
		cell := findDescendantByID(row, id)
		require.NotNil(t, cell, "missing %s", id)
		_, has := cell.Props["sensitive"]
		assert.False(t, has, "%s should not be sensitive", id)
	}
}

func TestBuildScreen_IncludeClosedFormPresent(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())

	form := findDescendantByID(s, "include-closed-form")
	require.NotNil(t, form)
	assert.Equal(t, "form", form.Type)

	cb := findDescendantByID(s, "include-closed-checkbox")
	require.NotNil(t, cb)
	assert.Equal(t, "checkbox", cb.Type)
	assert.Equal(t, "include_closed", cb.Props["name"])
	assert.Equal(t, "Include closed positions", cb.Props["label"])

	require.Len(t, cb.Actions, 1)
	a := cb.Actions[0]
	assert.Equal(t, "change", a.Trigger)
	assert.Equal(t, "submit", a.Type)
	assert.Equal(t, "/actions/portfolio/include_closed", a.Endpoint)
	assert.Equal(t, "POST", a.Method)
	assert.Equal(t, "include-closed-form", a.TargetID)
}

func TestBuildScreen_CheckboxOutsidePositionsTableCard(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	tableCard := findDescendantByID(s, "positions-table-card")
	require.NotNil(t, tableCard)
	assert.Nil(t, findDescendantByID(*tableCard, "include-closed-checkbox"))
	assert.Nil(t, findDescendantByID(*tableCard, "include-closed-form"))
}

func TestBuildScreen_IncludeClosedLocalizedEs(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "es", time.Now())
	cb := findDescendantByID(s, "include-closed-checkbox")
	require.NotNil(t, cb)
	assert.Equal(t, "Incluir posiciones cerradas", cb.Props["label"])
}

func TestBuildScreen_EmptyHasNoIncludeClosedForm(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{}, nil, nil, "en", time.Now())
	assert.Nil(t, findDescendantByID(s, "include-closed-form"))
}

func TestBuildScreen_ChartsSectionPresentWhenPositions(t *testing.T) {
	ps := samplePositions()
	s := BuildScreen(&PortfolioResponse{Positions: ps}, nil, nil, "en", time.Now())
	assert.NotNil(t, findDescendantByID(s, "charts-section"))
	assert.NotNil(t, findDescendantByID(s, "chart-value-over-time-card"))
}

func TestBuildScreen_ChartsSectionAbsentWhenEmpty(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{}, nil, nil, "en", time.Now())
	assert.Nil(t, findDescendantByID(s, "charts-section"))
}

func TestBuildScreen_AllocationSectionPresentWhenPositions(t *testing.T) {
	ps := samplePositions()
	s := BuildScreen(&PortfolioResponse{Positions: ps}, nil, nil, "en", time.Now())
	assert.NotNil(t, findDescendantByID(s, "allocation-section"))
}

func TestBuildScreen_AllocationSectionAbsentWhenEmpty(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{}, nil, nil, "en", time.Now())
	assert.Nil(t, findDescendantByID(s, "allocation-section"))
}

func TestBuildScreen_AllocationSectionIsLastInRoot(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	root := findDescendantByID(s, "portfolio-root")
	require.NotNil(t, root)
	require.Len(t, root.Children, 4)
	last := root.Children[len(root.Children)-1]
	assert.Equal(t, "allocation-section", last.ID)
}

func TestBuildScreen_LiveDataSectionPresentWhenPositions(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	assert.NotNil(t, findDescendantByID(s, "live-data-section"))
}

func TestBuildScreen_PortfolioRootHasFourTopChildren(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	root := findDescendantByID(s, "portfolio-root")
	require.NotNil(t, root)
	ids := []string{}
	for _, c := range root.Children {
		ids = append(ids, c.ID)
	}
	assert.Equal(t, []string{"live-header-row", "live-data-section", "charts-section", "allocation-section"}, ids)
}

func TestBuildScreen_SummaryTotalValueIsSensitive(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	tv := findDescendantByID(s, "summary-value-total-value-USD")
	require.NotNil(t, tv)
	assert.Equal(t, true, tv.Props["sensitive"])
}

func TestBuildScreen_SummaryTotalPnLIsSensitive(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	pnl := findDescendantByID(s, "summary-value-total-pnl-USD")
	require.NotNil(t, pnl)
	assert.Equal(t, true, pnl.Props["sensitive"])
}

func TestBuildScreen_SummaryPerformanceNotSensitive(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	perf := findDescendantByID(s, "summary-value-performance-USD")
	require.NotNil(t, perf)
	_, has := perf.Props["sensitive"]
	assert.False(t, has)
}

func TestBuildScreen_SummaryOpenPositionsNotSensitive(t *testing.T) {
	s := BuildScreen(&PortfolioResponse{Positions: samplePositions()}, nil, nil, "en", time.Now())
	op := findDescendantByID(s, "summary-value-open-positions")
	require.NotNil(t, op)
	_, has := op.Props["sensitive"]
	assert.False(t, has)
}
