package portfolio

import (
	"strconv"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

var columnWidths = []string{
	"80px",  // ticker
	"1fr",   // name
	"80px",  // type
	"80px",  // quantity
	"110px", // avg cost
	"110px", // total cost
	"120px", // market value
	"130px", // unrealized pnl
	"80px",  // % pnl
	"120px", // realized pnl
	"120px", // last snapshot
}

var columnKeys = []string{
	"portfolio.col.ticker",
	"portfolio.col.name",
	"portfolio.col.type",
	"portfolio.col.quantity",
	"portfolio.col.avg_cost",
	"portfolio.col.total_cost",
	"portfolio.col.market_value",
	"portfolio.col.unrealized_pnl",
	"portfolio.col.pnl_pct",
	"portfolio.col.realized_pnl",
	"portfolio.col.last_snapshot",
}

// BuildScreen builds the portfolio tree for the given positions and evolution
// points. now is used to format relative times.
func BuildScreen(positions []Position, evolution []EvolutionPoint, chartPoints []EvolutionPoint, lang string, now time.Time) components.Component {
	if len(positions) == 0 {
		return BuildEmpty(lang)
	}

	metrics := ComputeMetrics(positions, evolution)
	summary := buildSummaryRow(metrics, lang)
	controls := buildIncludeClosedForm(lang)
	table := BuildPositionsTable(positions, lang, now)

	chartsSection := buildInitialChartsSection(chartPoints, positions, lang)
	root := components.ColumnWithGap("portfolio-root", "lg", summary, controls, table, chartsSection)
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
			values.Children = append(values.Children,
				components.Text("summary-value-total-value-"+c, FormatMoney(&v, c, lang), "xl", "bold"))
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
			values.Children = append(values.Children,
				coloredValue("summary-value-total-pnl-"+c, FormatSignedMoney(&v, c, lang), pnlColor(&v)))
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

func BuildPositionsTable(ps []Position, lang string, now time.Time) components.Component {
	headerCells := make([]components.Component, 0, 11)
	for i, key := range columnKeys {
		cell := components.Text("col-"+columnShortID(i), i18n.T(lang, key), "sm", "bold")
		headerCells = append(headerCells, cell)
	}
	header := components.Row("positions-header", columnWidths, headerCells...)

	listChildren := make([]components.Component, 0, len(ps))
	for _, p := range ps {
		listChildren = append(listChildren, buildPositionItem(p, lang, now))
	}
	body := components.List("positions-body", listChildren...)

	inner := components.ColumnWithGap("positions-table", "sm", header, body)
	return components.Card("positions-table-card", inner)
}

func columnShortID(i int) string {
	names := []string{"ticker", "name", "type", "quantity", "avg-cost", "total-cost", "market-value", "unrealized-pnl", "pnl-pct", "realized-pnl", "last-snapshot"}
	return names[i]
}

func buildPositionItem(p Position, lang string, now time.Time) components.Component {
	realized := p.RealizedPnL
	pct := PnLPct(p.UnrealizedPnL, p.TotalCost)

	cells := []components.Component{
		components.Text("cell-ticker", p.Ticker, "sm", "bold"),
		components.Text("cell-name", p.Name, "sm", "normal"),
		components.Text("cell-type", p.AssetType, "sm", "normal"),
		components.Text("cell-quantity", FormatQuantity(p.Quantity, lang), "sm", "normal"),
		components.Text("cell-avg-cost", FormatMoney(p.AvgCost, p.Currency, lang), "sm", "normal"),
		components.Text("cell-total-cost", FormatMoney(p.TotalCost, p.Currency, lang), "sm", "normal"),
		components.Text("cell-market-value", FormatMoney(p.CurrentValue, p.Currency, lang), "sm", "normal"),
		coloredCell("cell-unrealized-pnl", FormatSignedMoney(p.UnrealizedPnL, p.Currency, lang), pnlColor(p.UnrealizedPnL)),
		coloredCell("cell-pnl-pct", FormatSignedPercent(pct, lang), pnlColor(pct)),
		coloredCell("cell-realized-pnl", FormatSignedMoney(&realized, p.Currency, lang), pnlColor(&realized)),
		components.Text("cell-last-snapshot", FormatRelativeTime(p.LastSnapshotAt, now, lang), "sm", "normal"),
	}
	row := components.Row("position-"+p.AssetID+"-row", columnWidths, cells...)
	return components.ListItem("position-"+p.AssetID, row)
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
