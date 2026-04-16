package portfolio

import (
	"strings"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// LiveState holds the live toggle state.
type LiveState struct {
	Live    bool
	Refresh bool // only meaningful when Live=true
}

// BuildLiveDataSection builds the top portion of the portfolio screen that
// reacts to the live toggle: header row + optional banner + summary + form + table.
func BuildLiveDataSection(resp *PortfolioResponse, metrics SummaryMetrics, liveState LiveState, currencies []string, lang string, now time.Time) components.Component {
	children := []components.Component{}

	// Header: title + live toggle
	children = append(children, buildLiveHeaderRow(liveState, lang))

	// Banner (only in live mode)
	if resp.IsLive {
		children = append(children, buildLiveBanner(resp, lang, now))
		if len(resp.Warnings) > 0 {
			children = append(children, buildLiveWarnings(resp.Warnings, lang))
		}
	}

	// Summary
	children = append(children, buildSummaryRow(metrics, lang))

	// Include-closed form
	children = append(children, buildIncludeClosedForm(lang))

	// Positions table (with dots when live)
	children = append(children, BuildPositionsTable(resp.Positions, lang, now, resp.IsLive))

	return components.ColumnWithGap("live-data-section", "lg", children...)
}

func buildLiveHeaderRow(state LiveState, lang string) components.Component {
	title := components.Text("portfolio-title", i18n.T(lang, "portfolio.title"), "lg", "bold")
	spacer := components.Column("live-header-spacer")

	toggleVariant, toggleStyle := "secondary", "ghost"
	toggleURL := "/actions/portfolio/live_data?live=true"
	if state.Live {
		toggleVariant, toggleStyle = "primary", "solid"
		toggleURL = "/actions/portfolio/live_data?live=false"
	}

	toggle := components.Component{
		Type: "button",
		ID:   "live-toggle",
		Props: map[string]any{
			"label":   i18n.T(lang, "portfolio.live.toggle"),
			"variant": toggleVariant,
			"style":   toggleStyle,
			"size":    "sm",
		},
		Actions: []components.Action{
			{Trigger: "click", Type: "reload", Endpoint: toggleURL, TargetID: "live-data-section"},
		},
	}

	return components.Row("live-header-row", []string{"auto", "1fr", "auto"}, title, spacer, toggle)
}

func buildLiveBanner(resp *PortfolioResponse, lang string, now time.Time) components.Component {
	statusText := i18n.T(lang, "portfolio.live.status")
	if resp.PricesAsOf != nil {
		statusText = strings.Replace(statusText, "{time}", FormatRelativeTime(resp.PricesAsOf, now, lang), 1)
	}
	status := components.TextStyled("live-status", statusText, "sm", "normal", "", "primary", "", "")
	refresh := components.Component{
		Type: "button",
		ID:   "live-refresh",
		Props: map[string]any{
			"image_src": "refresh",
			"variant":   "secondary",
			"style":     "ghost",
			"size":      "sm",
		},
		Actions: []components.Action{
			{Trigger: "click", Type: "reload", Endpoint: "/actions/portfolio/live_data?live=true&refresh=true", TargetID: "live-data-section"},
		},
	}
	row := components.Row("live-banner-row", []string{"auto", "auto"}, status, refresh)
	row.Props["gap"] = "sm"
	row.Props["align_items"] = "center"
	return components.Card("live-banner", row)
}

func buildLiveWarnings(warnings []LiveWarning, lang string) components.Component {
	tickers := make([]string, 0, len(warnings))
	for _, w := range warnings {
		tickers = append(tickers, w.Ticker)
	}
	content := i18n.T(lang, "portfolio.live.warning_prefix") + " " + strings.Join(tickers, ", ")
	return components.TextStyled("live-warnings", content, "sm", "normal", "", "muted", "", "")
}
