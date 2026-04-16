package portfolio

import (
	"net/url"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// ChartState is the user-selected state for the Value Over Time chart.
type ChartState struct {
	Timeframe string // 1m, 3m, 6m, ytd, 1y, all
	Mode      string // abs, pct
	Currency  string // ISO code; "" if there are no currencies
}

var timeframes = []string{"1m", "3m", "6m", "ytd", "1y", "all"}

// BuildValueOverTimeCard renders the Value Over Time chart card. Pure —
// depends only on its inputs.
func BuildValueOverTimeCard(points []EvolutionPoint, state ChartState, currencies []string, lang string) components.Component {
	controls := buildChartControls(state, currencies, lang)
	chart := buildLineChart(points, state, lang)
	content := components.ColumnWithGap("chart-value-over-time-content", "md", controls, chart)
	return components.Card("chart-value-over-time-card", content)
}

func buildChartControls(state ChartState, currencies []string, lang string) components.Component {
	tf := buildTimeframeControls(state, lang)
	md := buildModeControls(state, lang)
	children := []components.Component{tf, md}
	if len(currencies) > 1 {
		children = append(children, buildCurrencyControls(state, currencies, lang))
	}
	row := components.Row("controls-row", rowAutoWidths(len(children)), children...)
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
		},
		Actions: []components.Action{
			{Trigger: "click", Type: "reload", Endpoint: endpoint, TargetID: "chart-value-over-time-card"},
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

func buildLineChart(points []EvolutionPoint, state ChartState, lang string) components.Component {
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
	yFormat := valueFormat

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
		components.Axis{Format: yFormat},
		data,
		emptyMessage,
	)
}
