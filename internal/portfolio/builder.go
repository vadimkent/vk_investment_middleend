package portfolio

import (
	"sort"
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

// BuildScreen builds the portfolio tree for the given positions.
// now is used to format relative times.
func BuildScreen(positions []Position, lang string, now time.Time) components.Component {
	if len(positions) == 0 {
		return BuildEmpty(lang)
	}

	summary := buildSummary(positions, lang)
	table := buildTable(positions, lang, now)

	root := components.ColumnWithGap("portfolio-root", "lg", summary, table)
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

func buildSummary(ps []Position, lang string) components.Component {
	label := components.TextStyled("summary-label", i18n.T(lang, "portfolio.total_value"), "sm", "normal", "", "muted", "", "")

	totals := components.Column("total-values")
	byCurrency := totalsByCurrency(ps)
	if len(byCurrency) == 0 {
		totals.Children = append(totals.Children, components.Text("total-value-empty", "—", "xl", "bold"))
	} else {
		codes := make([]string, 0, len(byCurrency))
		for c := range byCurrency {
			codes = append(codes, c)
		}
		sort.Slice(codes, func(i, j int) bool {
			return byCurrency[codes[i]] > byCurrency[codes[j]]
		})
		for _, c := range codes {
			v := byCurrency[c]
			totals.Children = append(totals.Children,
				components.Text("total-value-"+c, FormatMoney(&v, c, lang), "xl", "bold"))
		}
	}

	inner := components.ColumnWithGap("portfolio-summary", "sm", label, totals)
	card := components.Card("portfolio-summary-card", inner)
	// Wrap in a row with auto + 1fr so the card shrinks to content width.
	return components.Row("portfolio-summary-row", []string{"auto", "1fr"},
		card,
		components.Column("portfolio-summary-spacer"),
	)
}

func totalsByCurrency(ps []Position) map[string]float64 {
	out := map[string]float64{}
	for _, p := range ps {
		if p.CurrentValue == nil {
			continue
		}
		out[p.Currency] += *p.CurrentValue
	}
	return out
}

func buildTable(ps []Position, lang string, now time.Time) components.Component {
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
		colored("cell-unrealized-pnl", FormatSignedMoney(p.UnrealizedPnL, p.Currency, lang), pnlColor(p.UnrealizedPnL)),
		colored("cell-pnl-pct", FormatSignedPercent(pct, lang), pnlColor(pct)),
		colored("cell-realized-pnl", FormatSignedMoney(&realized, p.Currency, lang), pnlColor(&realized)),
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

func colored(id, content, color string) components.Component {
	if color == "" {
		return components.Text(id, content, "sm", "normal")
	}
	return components.TextStyled(id, content, "sm", "normal", "", color, "", "")
}
