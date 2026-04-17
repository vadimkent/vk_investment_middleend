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
			buildNavHeader(lang, navType),
			buildNavMain(lang),
			buildNavFooter(lang),
			components.ContentSlot("content"),
		}
	case "web_mobile":
		return []components.Component{
			buildNavHeader(lang, navType),
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

// buildNavHeader emits the shell's top zone. On sidebar nav types it carries
// both an expanded app-name and a collapsed short variant, each gated by
// sidebar_visibility. On other nav types (bottombar, etc.) it emits only the
// full app name — sidebar_visibility is a no-op there.
func buildNavHeader(lang, navType string) components.Component {
	appName := components.Text("app-name", i18n.T(lang, "app.name"), "lg", "bold")
	if navType != "sidebar" {
		return components.NavHeader("shell-header", appName)
	}

	appName.Props["sidebar_visibility"] = "expanded"
	appNameShort := components.Text("app-name-short", i18n.T(lang, "app.name_short"), "lg", "bold")
	appNameShort.Props["sidebar_visibility"] = "collapsed"
	return components.NavHeader("shell-header", appName, appNameShort)
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

// buildNavFooter assembles the bottom zone of the sidebar. It assumes
// nav_type == "sidebar"; sidebar-toggle and the collapsed logout variant
// only make sense inside a collapsible sidebar. Do not call from non-sidebar
// platforms — buildSlots routes only the web case here.
func buildNavFooter(lang string) components.Component {
	sidebarToggle := components.IconToggle("sidebar-toggle", false,
		"panel-left-open", "panel-left-close",
		i18n.T(lang, "nav.sidebar_collapse"), i18n.T(lang, "nav.sidebar_expand"),
		components.ToggleSidebar(), components.ToggleSidebar(),
	)

	themeToggle := components.IconToggle("theme-toggle", false,
		"sun", "moon",
		i18n.T(lang, "nav.theme_light"), i18n.T(lang, "nav.theme_dark"),
		components.ToggleTheme(), components.ToggleTheme(),
	)

	logoutExpanded := components.Button("logout-btn", i18n.T(lang, "nav.logout"), components.Logout())
	logoutExpanded.Props["icon"] = "logout"
	logoutExpanded.Props["style"] = "ghost"
	logoutExpanded.Props["sidebar_visibility"] = "expanded"

	logoutCollapsed := components.Button("logout-btn-collapsed", "", components.Logout())
	logoutCollapsed.Props["icon"] = "logout"
	logoutCollapsed.Props["style"] = "ghost"
	logoutCollapsed.Props["sidebar_visibility"] = "collapsed"

	return components.NavFooter("shell-footer",
		sidebarToggle,
		themeToggle,
		logoutExpanded,
		logoutCollapsed,
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
