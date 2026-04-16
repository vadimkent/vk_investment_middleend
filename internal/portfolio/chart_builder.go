package portfolio

import (
	"net/url"
	"sort"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// ChartState is the user-selected state shared by all charts in the section.
type ChartState struct {
	Timeframe string // 1m, 3m, 6m, ytd, 1y, all
	Mode      string // abs, pct
	Currency  string // ISO code; "" when no currencies
}

var timeframes = []string{"1m", "3m", "6m", "ytd", "1y", "all"}

// BuildChartsSection wraps controls and the chart cards in a column that
// serves as the reload target. Pure — no BE dependency.
func BuildChartsSection(points []EvolutionPoint, state ChartState, currencies []string, lang string) components.Component {
	controls := buildChartControls(state, currencies, lang)
	value := BuildValueOverTimeCard(points, state, lang)
	asset := BuildAssetValueOverTimeCard(points, state, lang)
	col := components.ColumnWithGap("charts-section", "lg", controls, value, asset)
	return col
}

// BuildValueOverTimeCard builds the Value Over Time card: title + line_chart.
// Controls live outside (at the charts-section level).
func BuildValueOverTimeCard(points []EvolutionPoint, state ChartState, lang string) components.Component {
	title := components.Text("chart-value-over-time-title", i18n.T(lang, "portfolio.chart.value_over_time.title"), "md", "bold")
	chart := buildValueLineChart(points, state, lang)
	content := components.ColumnWithGap("chart-value-over-time-content", "md", title, chart)
	return components.Card("chart-value-over-time-card", content)
}

func buildChartControls(state ChartState, currencies []string, lang string) components.Component {
	tf := buildTimeframeControls(state, lang)
	md := buildModeControls(state, lang)
	spacer := components.Column("controls-spacer")

	var children []components.Component
	var widths []string
	if len(currencies) > 1 {
		children = []components.Component{buildCurrencyControls(state, currencies, lang), spacer, md, tf}
		widths = []string{"auto", "1fr", "auto", "auto"}
	} else {
		children = []components.Component{spacer, md, tf}
		widths = []string{"1fr", "auto", "auto"}
	}
	row := components.Row("controls-row", widths, children...)
	row.Props["gap"] = "lg"
	return row
}

func buildTimeframeControls(state ChartState, lang string) components.Component {
	btns := make([]components.Component, 0, len(timeframes))
	for _, tf := range timeframes {
		btns = append(btns, chartButton(
			"chart-timeframe-"+tf,
			i18n.T(lang, "portfolio.chart.timeframe."+tf),
			tf == state.Timeframe,
			evolutionURL(tf, state.Mode, state.Currency),
		))
	}
	row := components.Row("timeframe-controls", rowAutoWidths(len(btns)), btns...)
	row.Props["gap"] = "sm"
	return row
}

func buildModeControls(state ChartState, lang string) components.Component {
	abs := chartButton("chart-mode-abs",
		i18n.T(lang, "portfolio.chart.mode.abs"),
		state.Mode == "abs",
		evolutionURL(state.Timeframe, "abs", state.Currency),
	)
	pct := chartButton("chart-mode-pct",
		i18n.T(lang, "portfolio.chart.mode.pct"),
		state.Mode == "pct",
		evolutionURL(state.Timeframe, "pct", state.Currency),
	)
	row := components.Row("mode-controls", rowAutoWidths(2), abs, pct)
	row.Props["gap"] = "sm"
	return row
}

func buildCurrencyControls(state ChartState, currencies []string, lang string) components.Component {
	btns := make([]components.Component, 0, len(currencies))
	for _, c := range currencies {
		btns = append(btns, chartButton(
			"chart-currency-"+c,
			c,
			c == state.Currency,
			evolutionURL(state.Timeframe, state.Mode, c),
		))
	}
	row := components.Row("currency-controls", rowAutoWidths(len(btns)), btns...)
	row.Props["gap"] = "sm"
	return row
}

func chartButton(id, label string, selected bool, endpoint string) components.Component {
	variant, style := "secondary", "ghost"
	if selected {
		variant, style = "primary", "solid"
	}
	return components.Component{
		Type: "button",
		ID:   id,
		Props: map[string]any{
			"label":   label,
			"variant": variant,
			"style":   style,
			"size":    "sm",
		},
		Actions: []components.Action{
			{Trigger: "click", Type: "reload", Endpoint: endpoint, TargetID: "charts-section"},
		},
	}
}

func evolutionURL(timeframe, mode, currency string) string {
	q := url.Values{}
	q.Set("timeframe", timeframe)
	q.Set("mode", mode)
	if currency != "" {
		q.Set("currency", currency)
	}
	return "/actions/portfolio/evolution?" + q.Encode()
}

func rowAutoWidths(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "auto"
	}
	return out
}

func buildValueLineChart(points []EvolutionPoint, state ChartState, lang string) components.Component {
	data := make([]map[string]any, 0, len(points))
	anyCost := false
	for _, p := range points {
		if p.Currency != state.Currency {
			continue
		}
		row := map[string]any{"date": p.RecordedAt.Format("2006-01-02")}
		if state.Mode == "pct" {
			if p.TotalCost != nil && *p.TotalCost != 0 {
				anyCost = true
				row["value"] = (p.TotalValue - *p.TotalCost) / *p.TotalCost * 100
			} else {
				row["value"] = nil
			}
		} else {
			row["value"] = p.TotalValue
		}
		data = append(data, row)
	}

	valueFormat := "currency_compact"
	if state.Mode == "pct" {
		valueFormat = "percent_signed"
	}

	series := []components.Series{{
		Key:         "value",
		Label:       i18n.T(lang, "portfolio.chart.series.value"),
		Color:       "chart_1",
		ValueFormat: valueFormat,
	}}

	emptyMessage := ""
	switch {
	case state.Mode == "pct" && len(data) > 0 && !anyCost:
		data = data[:0]
		emptyMessage = i18n.T(lang, "portfolio.chart.no_cost_data")
	case len(data) < 2:
		data = data[:0]
		emptyMessage = i18n.T(lang, "portfolio.chart.not_enough_data")
	}

	return components.LineChart(
		"chart-value-over-time",
		"md",
		series,
		components.Axis{Key: "date", Format: "month_year"},
		components.Axis{Format: valueFormat},
		data,
		emptyMessage,
	)
}

var assetColors = []string{"chart_1", "chart_2", "chart_3", "chart_4", "chart_5"}

// BuildAssetValueOverTimeCard builds the per-asset multi-series card. Always
// absolute currency, regardless of state.Mode.
func BuildAssetValueOverTimeCard(points []EvolutionPoint, state ChartState, lang string) components.Component {
	title := components.Text("chart-asset-value-over-time-title", i18n.T(lang, "portfolio.chart.asset_value_over_time.title"), "md", "bold")
	chart := buildAssetLineChart(points, state, lang)
	content := components.ColumnWithGap("chart-asset-value-over-time-content", "md", title, chart)
	return components.Card("chart-asset-value-over-time-card", content)
}

func buildAssetLineChart(points []EvolutionPoint, state ChartState, lang string) components.Component {
	filtered := make([]EvolutionPoint, 0, len(points))
	for _, p := range points {
		if p.Currency == state.Currency {
			filtered = append(filtered, p)
		}
	}

	latestValues := map[string]float64{}
	seen := map[string]struct{}{}
	var tickers []string
	for _, p := range filtered {
		for _, a := range p.Assets {
			if _, ok := seen[a.Ticker]; !ok {
				seen[a.Ticker] = struct{}{}
				tickers = append(tickers, a.Ticker)
			}
		}
	}
	// Walk from the latest point backward; first appearance wins.
	for i := len(filtered) - 1; i >= 0; i-- {
		for _, a := range filtered[i].Assets {
			if _, have := latestValues[a.Ticker]; !have {
				latestValues[a.Ticker] = a.Value
			}
		}
	}
	sort.SliceStable(tickers, func(i, j int) bool {
		vi := latestValues[tickers[i]]
		vj := latestValues[tickers[j]]
		if vi == vj {
			return tickers[i] < tickers[j]
		}
		return vi > vj
	})

	series := make([]components.Series, 0, len(tickers))
	for i, t := range tickers {
		series = append(series, components.Series{
			Key:         t,
			Label:       t,
			Color:       assetColors[i%len(assetColors)],
			ValueFormat: "currency_compact",
		})
	}

	data := make([]map[string]any, 0, len(filtered))
	for _, p := range filtered {
		row := map[string]any{"date": p.RecordedAt.Format("2006-01-02")}
		present := map[string]float64{}
		for _, a := range p.Assets {
			present[a.Ticker] = a.Value
		}
		for _, t := range tickers {
			if v, ok := present[t]; ok {
				row[t] = v
			} else {
				row[t] = nil
			}
		}
		data = append(data, row)
	}

	emptyMessage := ""
	if len(data) < 2 || len(tickers) == 0 {
		data = data[:0]
		emptyMessage = i18n.T(lang, "portfolio.chart.not_enough_data")
	}

	return components.LineChart(
		"chart-asset-value-over-time",
		"md",
		series,
		components.Axis{Key: "date", Format: "month_year"},
		components.Axis{Format: "currency_compact"},
		data,
		emptyMessage,
	)
}
