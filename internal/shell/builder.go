package shell

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// NavItem defines a navigation entry.
type NavItem struct {
	ID       string
	LabelKey string
	Icon     string
	Route    string
}

// DefaultNavItems returns the six navigation entries of the investment tracker.
func DefaultNavItems() []NavItem {
	return []NavItem{
		{ID: "portfolio", LabelKey: "nav.portfolio", Icon: "pie-chart", Route: "/portfolio"},
		{ID: "assets", LabelKey: "nav.assets", Icon: "coins", Route: "/assets"},
		{ID: "trades", LabelKey: "nav.trades", Icon: "arrow-swap", Route: "/trades"},
		{ID: "snapshots", LabelKey: "nav.snapshots", Icon: "camera", Route: "/snapshots"},
		{ID: "import", LabelKey: "nav.import", Icon: "upload", Route: "/import"},
		{ID: "analysis", LabelKey: "nav.analysis", Icon: "sparkles", Route: "/analysis"},
	}
}

// BuildShell builds the app shell component tree adapted per platform.
// Unknown or empty platforms fall back to "web".
func BuildShell(lang, platform string) components.Component {
	platform = normalizePlatform(platform)
	navType := navTypeForPlatform(platform)
	children := buildSlots(lang, platform, navType)

	return components.Component{
		Type:     "screen",
		ID:       "shell",
		Props:    map[string]any{"nav_type": navType},
		Children: children,
	}
}

func normalizePlatform(platform string) string {
	switch platform {
	case "web", "web_mobile", "android", "ios":
		return platform
	default:
		return "web"
	}
}

func navTypeForPlatform(platform string) string {
	if platform == "web" {
		return "sidebar"
	}
	return "bottombar"
}

func buildSlots(lang, platform, navType string) []components.Component {
	switch platform {
	case "web":
		return []components.Component{
			buildNavHeader(lang),
			buildNavMain(lang),
			buildNavFooter(lang),
			components.ContentSlot("content"),
		}
	case "web_mobile":
		return []components.Component{
			buildNavHeader(lang),
			components.ContentSlot("content"),
			buildBottomBar(lang),
		}
	default: // android, ios
		return []components.Component{
			components.ContentSlot("content"),
			buildBottomBar(lang),
		}
	}
}

func buildNavHeader(lang string) components.Component {
	return components.NavHeader("shell-header",
		components.Text("app-name", i18n.T(lang, "app.name"), "lg", "bold"),
	)
}

func buildNavMain(lang string) components.Component {
	items := DefaultNavItems()
	children := make([]components.Component, 0, len(items))
	for _, item := range items {
		children = append(children, components.NavItem(
			"nav-"+item.ID,
			i18n.T(lang, item.LabelKey),
			item.Icon,
			item.Route,
			components.Navigate(item.Route),
		))
	}
	return components.NavMain("shell-nav", children...)
}

func buildNavFooter(lang string) components.Component {
	themeToggle := components.IconToggle("theme-toggle", false,
		"sun", "moon",
		i18n.T(lang, "nav.theme_light"), i18n.T(lang, "nav.theme_dark"),
		components.Action{Trigger: "click", Type: "toggle_theme"},
		components.Action{Trigger: "click", Type: "toggle_theme"},
	)
	return components.NavFooter("shell-footer",
		themeToggle,
		components.Button("logout-btn", i18n.T(lang, "nav.logout"), components.Logout()),
	)
}

func buildBottomBar(lang string) components.Component {
	items := DefaultNavItems()
	children := make([]components.Component, 0, len(items))
	for _, item := range items {
		children = append(children, components.NavItem(
			"nav-"+item.ID,
			i18n.T(lang, item.LabelKey),
			item.Icon,
			item.Route,
			components.Navigate(item.Route),
		))
	}
	return components.BottomBar("shell-bottombar", children...)
}
