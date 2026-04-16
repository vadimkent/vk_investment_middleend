package portfolio

import (
	"net/url"
	"sort"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// AllocationState holds the user's selections for the allocation section.
type AllocationState struct {
	GroupBy  string // "asset" | "type"
	Currency string
}

var allocationColors = []string{"chart_1", "chart_2", "chart_3", "chart_4", "chart_5"}

// BuildAllocationSection wraps controls + the pie chart card in a column
// serving as the reload target. Pure.
func BuildAllocationSection(positions []Position, state AllocationState, currencies []string, lang string) components.Component {
	controls := buildAllocationControls(state, currencies, lang)
	card := buildAllocationCard(positions, state, lang)
	return components.ColumnWithGap("allocation-section", "lg", controls, card)
}

func buildAllocationControls(state AllocationState, currencies []string, lang string) components.Component {
	gb := buildGroupByControls(state, lang)
	spacer := components.Column("allocation-controls-spacer")

	var children []components.Component
	var widths []string
	if len(currencies) > 1 {
		children = []components.Component{buildAllocationCurrencyControls(state, currencies), spacer, gb}
		widths = []string{"auto", "1fr", "auto"}
	} else {
		children = []components.Component{spacer, gb}
		widths = []string{"1fr", "auto"}
	}
	row := components.Row("allocation-controls-row", widths, children...)
	row.Props["gap"] = "lg"
	return row
}

func buildGroupByControls(state AllocationState, lang string) components.Component {
	asset := allocationButton("allocation-group-by-asset",
		i18n.T(lang, "portfolio.allocation.group_by.asset"),
		state.GroupBy == "asset",
		allocationURL("asset", state.Currency),
	)
	typ := allocationButton("allocation-group-by-type",
		i18n.T(lang, "portfolio.allocation.group_by.type"),
		state.GroupBy == "type",
		allocationURL("type", state.Currency),
	)
	row := components.Row("allocation-group-by-controls", []string{"auto", "auto"}, asset, typ)
	row.Props["gap"] = "sm"
	return row
}

func buildAllocationCurrencyControls(state AllocationState, currencies []string) components.Component {
	btns := make([]components.Component, 0, len(currencies))
	for _, c := range currencies {
		btns = append(btns, allocationButton(
			"chart-currency-"+c,
			c,
			c == state.Currency,
			allocationURL(state.GroupBy, c),
		))
	}
	widths := make([]string, len(btns))
	for i := range widths {
		widths[i] = "auto"
	}
	row := components.Row("currency-controls", widths, btns...)
	row.Props["gap"] = "sm"
	return row
}

func allocationButton(id, label string, selected bool, endpoint string) components.Component {
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
			{Trigger: "click", Type: "reload", Endpoint: endpoint, TargetID: "allocation-section", Loading: "section"},
		},
	}
}

func allocationURL(groupBy, currency string) string {
	q := url.Values{}
	q.Set("group_by", groupBy)
	if currency != "" {
		q.Set("currency", currency)
	}
	return "/actions/portfolio/allocation?" + q.Encode()
}

func buildAllocationCard(positions []Position, state AllocationState, lang string) components.Component {
	title := components.Text("chart-allocation-title", i18n.T(lang, "portfolio.allocation.title"), "md", "bold")
	slices := computeAllocationSlices(positions, state)
	emptyMessage := ""
	if len(slices) == 0 {
		emptyMessage = i18n.T(lang, "portfolio.allocation.empty")
	}
	chart := components.PieChart(
		"chart-allocation",
		"md",
		"donut",
		"currency_compact",
		slices,
		true,
		emptyMessage,
	)
	content := components.ColumnWithGap("chart-allocation-content", "md", title, chart)
	return components.Card("chart-allocation-card", content)
}

// computeAllocationSlices filters positions by currency + non-null current_value,
// groups by (asset|type), sums values, returns sorted DESC by value (ties by label ASC).
func computeAllocationSlices(positions []Position, state AllocationState) []components.Slice {
	type agg struct {
		key   string
		label string
		value float64
	}
	aggMap := map[string]*agg{}
	order := []string{}

	for _, p := range positions {
		if p.Currency != state.Currency || p.CurrentValue == nil {
			continue
		}
		var key, label string
		if state.GroupBy == "type" {
			key = p.AssetType
			label = p.AssetType
		} else {
			key = p.AssetID
			label = p.Ticker
		}
		if existing, ok := aggMap[key]; ok {
			existing.value += *p.CurrentValue
		} else {
			aggMap[key] = &agg{key: key, label: label, value: *p.CurrentValue}
			order = append(order, key)
		}
	}

	aggs := make([]*agg, 0, len(order))
	for _, k := range order {
		aggs = append(aggs, aggMap[k])
	}
	sort.SliceStable(aggs, func(i, j int) bool {
		if aggs[i].value == aggs[j].value {
			return aggs[i].label < aggs[j].label
		}
		return aggs[i].value > aggs[j].value
	})

	out := make([]components.Slice, 0, len(aggs))
	for i, a := range aggs {
		out = append(out, components.Slice{
			Key:   a.key,
			Label: a.label,
			Value: a.value,
			Color: allocationColors[i%len(allocationColors)],
		})
	}
	return out
}
