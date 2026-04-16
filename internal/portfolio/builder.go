package portfolio

import (
	"strconv"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

func positionColumns(lang string) []components.TableColumn {
	return []components.TableColumn{
		{ID: "ticker", Header: i18n.T(lang, "portfolio.col.ticker"), Width: "80px"},
		{ID: "name", Header: i18n.T(lang, "portfolio.col.name"), Width: "1fr"},
		{ID: "type", Header: i18n.T(lang, "portfolio.col.type"), Width: "80px"},
		{ID: "quantity", Header: i18n.T(lang, "portfolio.col.quantity"), Width: "80px", Align: "right"},
		{ID: "avg_cost", Header: i18n.T(lang, "portfolio.col.avg_cost"), Width: "110px", Align: "right"},
		{ID: "total_cost", Header: i18n.T(lang, "portfolio.col.total_cost"), Width: "110px", Align: "right"},
		{ID: "market_value", Header: i18n.T(lang, "portfolio.col.market_value"), Width: "120px", Align: "right"},
		{ID: "unrealized_pnl", Header: i18n.T(lang, "portfolio.col.unrealized_pnl"), Width: "130px", Align: "right"},
		{ID: "pnl_pct", Header: i18n.T(lang, "portfolio.col.pnl_pct"), Width: "80px", Align: "right"},
		{ID: "realized_pnl", Header: i18n.T(lang, "portfolio.col.realized_pnl"), Width: "120px", Align: "right"},
		{ID: "last_snapshot", Header: i18n.T(lang, "portfolio.col.last_snapshot"), Width: "120px", Align: "right"},
	}
}

// BuildScreen builds the portfolio tree for the given response and evolution
// points. now is used to format relative times.
func BuildScreen(resp *PortfolioResponse, evolution []EvolutionPoint, chartPoints []EvolutionPoint, lang string, now time.Time) components.Component {
	if len(resp.Positions) == 0 {
		return BuildEmpty(lang)
	}

	positions := resp.Positions
	SortPositions(positions)

	metrics := ComputeMetrics(positions, evolution)
	currencies := metrics.CurrencyOrder

	liveState := LiveState{Live: resp.IsLive}
	headerRow := BuildPortfolioHeaderRow(liveState, lang)
	liveDataSection := BuildLiveDataSection(resp, metrics, liveState, currencies, lang, now)
	chartsSection := buildInitialChartsSection(chartPoints, positions, lang)
	allocationSection := buildInitialAllocationSection(positions, lang)

	root := components.ColumnWithGap("portfolio-root", "lg", headerRow, liveDataSection, chartsSection, allocationSection)
	return components.Screen("portfolio", i18n.T(lang, "portfolio.title"), root)
}

// BuildEmpty builds the screen for an empty portfolio.
func BuildEmpty(lang string) components.Component {
	title := components.Text("empty-title", i18n.T(lang, "portfolio.empty_title"), "lg", "bold")
	subtitle := components.TextStyled("empty-subtitle", i18n.T(lang, "portfolio.empty_subtitle"), "md", "normal", "", "muted", "", "")
	empty := components.ColumnWithGap("portfolio-empty", "sm", title, subtitle)
	root := components.ColumnWithGap("portfolio-root", "lg", empty)
	return components.Screen("portfolio", i18n.T(lang, "portfolio.title"), root)
}

func buildIncludeClosedForm(lang string) components.Component {
	checkbox := components.Component{
		Type: "checkbox",
		ID:   "include-closed-checkbox",
		Props: map[string]any{
			"name":  "include_closed",
			"label": i18n.T(lang, "portfolio.include_closed"),
		},
		Actions: []components.Action{
			{
				Trigger:  "change",
				Type:     "submit",
				Endpoint: "/actions/portfolio/include_closed",
				Method:   "POST",
				TargetID: "include-closed-form",
			},
		},
	}
	return components.Form("include-closed-form", checkbox)
}

func buildSummaryRow(m SummaryMetrics, lang string) components.Component {
	cards := []components.Component{
		buildTotalValueCard(m, lang),
		buildTotalPnLCard(m, lang),
		buildPerformanceCard(m, lang),
		buildSnapshotChangeCard(m, lang),
		buildOpenPositionsCard(m, lang),
	}
	row := components.Row("portfolio-summary-row", []string{"1fr", "1fr", "1fr", "1fr", "1fr"}, cards...)
	row.Props["gap"] = "md"
	return row
}

func buildTotalValueCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-total-value")
	if len(m.CurrencyOrder) == 0 || !anyHasValue(m.TotalValue, m.CurrencyOrder) {
		values.Children = append(values.Children, components.Text("summary-value-total-value-empty", "—", "xl", "bold"))
	} else {
		for _, c := range m.CurrencyOrder {
			v, ok := m.TotalValue[c]
			if !ok {
				continue
			}
			txt := components.Text("summary-value-total-value-"+c, FormatMoney(&v, c, lang), "xl", "bold")
			txt.Props["sensitive"] = true
			values.Children = append(values.Children, txt)
		}
	}
	return wrapCard("total-value", "portfolio.total_value", lang, values)
}

func buildTotalPnLCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-total-pnl")
	if len(m.CurrencyOrder) == 0 {
		values.Children = append(values.Children, components.Text("summary-value-total-pnl-empty", "—", "xl", "bold"))
	} else {
		for _, c := range m.CurrencyOrder {
			v := m.TotalPnL[c]
			txt := coloredValue("summary-value-total-pnl-"+c, FormatSignedMoney(&v, c, lang), pnlColor(&v))
			txt.Props["sensitive"] = true
			values.Children = append(values.Children, txt)
		}
	}
	return wrapCard("total-pnl", "portfolio.total_pnl", lang, values)
}

func buildPerformanceCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-performance")
	if len(m.CurrencyOrder) == 0 {
		values.Children = append(values.Children, components.Text("summary-value-performance-empty", "—", "xl", "bold"))
	} else {
		for _, c := range m.CurrencyOrder {
			pct := m.Performance[c]
			values.Children = append(values.Children,
				coloredValue("summary-value-performance-"+c, FormatSignedPercent(pct, lang), pnlColor(pct)))
		}
	}
	return wrapCard("performance", "portfolio.performance", lang, values)
}

func buildSnapshotChangeCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-snapshot-change")
	if len(m.CurrencyOrder) == 0 {
		values.Children = append(values.Children, components.Text("summary-value-snapshot-change-empty", "—", "xl", "bold"))
	} else {
		for _, c := range m.CurrencyOrder {
			pct := m.SnapshotChange[c]
			values.Children = append(values.Children,
				coloredValue("summary-value-snapshot-change-"+c, FormatSignedPercent(pct, lang), pnlColor(pct)))
		}
	}
	return wrapCard("snapshot-change", "portfolio.snapshot_change", lang, values)
}

func buildOpenPositionsCard(m SummaryMetrics, lang string) components.Component {
	values := components.Column("summary-values-open-positions",
		components.Text("summary-value-open-positions", strconv.Itoa(m.OpenPositions), "xl", "bold"),
	)
	return wrapCard("open-positions", "portfolio.open_positions", lang, values)
}

func wrapCard(id, labelKey, lang string, valuesCol components.Component) components.Component {
	label := components.TextStyled("summary-label-"+id, i18n.T(lang, labelKey), "sm", "normal", "", "muted", "", "")
	content := components.ColumnWithGap("summary-card-content-"+id, "sm", label, valuesCol)
	return components.Card("summary-card-"+id, content)
}

func anyHasValue(byCurrency map[string]float64, order []string) bool {
	for _, c := range order {
		if _, ok := byCurrency[c]; ok {
			return true
		}
	}
	return false
}

func coloredValue(id, content, color string) components.Component {
	if color == "" {
		return components.Text(id, content, "xl", "bold")
	}
	return components.TextStyled(id, content, "xl", "bold", "", color, "", "")
}

func BuildPositionsTable(ps []Position, lang string, now time.Time, isLive bool) components.Component {
	cols := positionColumns(lang)
	rows := make([]components.Component, 0, len(ps))
	for _, p := range ps {
		rows = append(rows, buildPositionRow(p, lang, now, isLive))
	}
	table := components.Table("positions-table", cols, rows...)
	return components.Card("positions-table-card", table)
}

// priceSourceDot returns the dot prefix for a live price source.
func priceSourceDot(source string) string {
	switch source {
	case "live", "snapshot", "none":
		return "●"
	default:
		return ""
	}
}

// priceSourceColor returns the SDUI color for a price-source dot.
func priceSourceColor(source string) string {
	switch source {
	case "live":
		return "positive"
	case "snapshot":
		return "muted"
	default:
		return "negative"
	}
}

func buildPositionRow(p Position, lang string, now time.Time, isLive bool) components.Component {
	realized := p.RealizedPnL
	pct := PnLPct(p.UnrealizedPnL, p.TotalCost)

	marketValueContent := FormatMoney(p.CurrentValue, p.Currency, lang)
	marketValueColor := ""
	if isLive && p.PriceSource != nil {
		marketValueContent = priceSourceDot(*p.PriceSource) + " " + marketValueContent
		marketValueColor = priceSourceColor(*p.PriceSource)
	}

	cells := []components.Component{
		components.Text("cell-ticker", p.Ticker, "sm", "bold"),
		components.Text("cell-name", p.Name, "sm", "normal"),
		components.Text("cell-type", p.AssetType, "sm", "normal"),
		components.Text("cell-quantity", FormatQuantity(p.Quantity, lang), "sm", "normal"),
		sensitiveText("cell-avg-cost", FormatMoney(p.AvgCost, p.Currency, lang), ""),
		sensitiveText("cell-total-cost", FormatMoney(p.TotalCost, p.Currency, lang), ""),
		sensitiveColoredText("cell-market-value", marketValueContent, marketValueColor),
		sensitiveColoredText("cell-unrealized-pnl", FormatSignedMoney(p.UnrealizedPnL, p.Currency, lang), pnlColor(p.UnrealizedPnL)),
		coloredCell("cell-pnl-pct", FormatSignedPercent(pct, lang), pnlColor(pct)),
		sensitiveColoredText("cell-realized-pnl", FormatSignedMoney(&realized, p.Currency, lang), pnlColor(&realized)),
		components.Text("cell-last-snapshot", FormatRelativeTime(p.LastSnapshotAt, now, lang), "sm", "normal"),
	}
	return components.TableRow("position-"+p.AssetID, cells...)
}

func sensitiveText(id, content, color string) components.Component {
	c := components.Text(id, content, "sm", "normal")
	c.Props["sensitive"] = true
	if color != "" {
		c.Props["color"] = color
	}
	return c
}

func sensitiveColoredText(id, content, color string) components.Component {
	var c components.Component
	if color == "" {
		c = components.Text(id, content, "sm", "normal")
	} else {
		c = components.TextStyled(id, content, "sm", "normal", "", color, "", "")
	}
	c.Props["sensitive"] = true
	return c
}

// pnlColor returns "positive", "negative" or "" (no color) based on v.
func pnlColor(v *float64) string {
	if v == nil || *v == 0 {
		return ""
	}
	if *v > 0 {
		return "positive"
	}
	return "negative"
}

func coloredCell(id, content, color string) components.Component {
	if color == "" {
		return components.Text(id, content, "sm", "normal")
	}
	return components.TextStyled(id, content, "sm", "normal", "", color, "", "")
}

// buildInitialChartsSection produces the charts-section for the initial
// screen render. Chooses default currency from positions (highest total value).
func buildInitialChartsSection(chartPoints []EvolutionPoint, positions []Position, lang string) components.Component {
	metrics := ComputeMetrics(positions, nil)
	currencies := metrics.CurrencyOrder
	defaultCurrency := ""
	if len(currencies) > 0 {
		defaultCurrency = currencies[0]
	}
	state := ChartState{Timeframe: "all", Mode: "abs", Currency: defaultCurrency}
	return BuildChartsSection(chartPoints, state, currencies, lang)
}

// buildInitialAllocationSection produces the allocation-section for the initial
// screen render. Defaults group_by=asset and currency=<first by total value DESC>.
func buildInitialAllocationSection(positions []Position, lang string) components.Component {
	metrics := ComputeMetrics(positions, nil)
	currencies := metrics.CurrencyOrder
	defaultCurrency := ""
	if len(currencies) > 0 {
		defaultCurrency = currencies[0]
	}
	state := AllocationState{GroupBy: "asset", Currency: defaultCurrency}
	return BuildAllocationSection(positions, state, currencies, lang)
}
