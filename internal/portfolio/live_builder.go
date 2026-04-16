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

// BuildLiveDataSection builds the data portion of the portfolio screen that
// reacts to the live toggle: optional banner + summary + form + table.
// The header row (title + toggle) lives OUTSIDE this section so it doesn't
// flash on reload.
func BuildLiveDataSection(resp *PortfolioResponse, metrics SummaryMetrics, liveState LiveState, currencies []string, lang string, now time.Time) components.Component {
	children := []components.Component{}

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

// BuildPortfolioHeaderRow builds the title + live toggle row. Lives at the
// portfolio-root level, outside any reload target.
func BuildPortfolioHeaderRow(state LiveState, lang string) components.Component {
	title := components.Text("portfolio-title", i18n.T(lang, "portfolio.title"), "lg", "bold")
	spacer := components.Column("live-header-spacer")

	hideValues := components.IconToggle("hide-values-toggle", false,
		"eye", "eye-off",
		i18n.T(lang, "portfolio.hide_values.tooltip_inactive"),
		i18n.T(lang, "portfolio.hide_values.tooltip_active"),
		components.Action{Trigger: "click", Type: "toggle_sensitive"},
		components.Action{Trigger: "click", Type: "toggle_sensitive"},
	)

	toggle := components.IconToggle("live-toggle", state.Live,
		"radio", "radio",
		i18n.T(lang, "portfolio.live.toggle"), i18n.T(lang, "portfolio.live.toggle"),
		components.Reload("/actions/portfolio/live_data?live=true", "live-data-section"),
		components.Reload("/actions/portfolio/live_data?live=false", "live-data-section"),
	)

	return components.Row("live-header-row", []string{"auto", "1fr", "auto", "auto"}, title, spacer, hideValues, toggle)
}

func buildLiveBanner(resp *PortfolioResponse, lang string, now time.Time) components.Component {
	statusText := i18n.T(lang, "portfolio.live.status")
	if resp.PricesAsOf != nil {
		statusText = strings.Replace(statusText, "{time}", FormatRelativeTime(resp.PricesAsOf, now, lang), 1)
	}
	status := components.TextStyled("live-status", statusText, "sm", "normal", "", "positive", "", "")
	refresh := components.Component{
		Type: "button",
		ID:   "live-refresh",
		Props: map[string]any{
			"icon":    "refresh",
			"variant":   "secondary",
			"style":     "ghost",
			"size":      "sm",
		},
		Actions: []components.Action{
			{Trigger: "click", Type: "reload", Endpoint: "/actions/portfolio/live_data?live=true&refresh=true", TargetID: "live-data-section", Loading: "section"},
		},
	}
	spacer := components.Column("live-banner-spacer")
	row := components.Row("live-banner-row", []string{"auto", "auto", "1fr"}, status, refresh, spacer)
	row.Props["gap"] = "none"
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
