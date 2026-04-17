package shell

import (
	"path/filepath"
	"testing"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Load locales from repo root for i18n-dependent assertions.
	_ = i18n.Load(filepath.Join("..", "..", "locales"))
}

func TestBuildShell_WebSidebar(t *testing.T) {
	shell := BuildShell("en", "web")

	assert.Equal(t, "screen", shell.Type)
	assert.Equal(t, "shell", shell.ID)
	assert.Equal(t, "sidebar", shell.Props["nav_type"])

	types := childTypes(shell)
	assert.Equal(t, []string{"nav_header", "nav_main", "nav_footer", "content_slot"}, types)
}

func TestBuildShell_WebMobileBottombar(t *testing.T) {
	shell := BuildShell("en", "web_mobile")

	assert.Equal(t, "bottombar", shell.Props["nav_type"])

	types := childTypes(shell)
	assert.Equal(t, []string{"nav_header", "content_slot", "bottombar"}, types)
}

func TestBuildShell_Android(t *testing.T) {
	shell := BuildShell("en", "android")

	assert.Equal(t, "bottombar", shell.Props["nav_type"])

	types := childTypes(shell)
	assert.Equal(t, []string{"content_slot", "bottombar"}, types)
}

func TestBuildShell_IOS(t *testing.T) {
	shell := BuildShell("en", "ios")

	assert.Equal(t, "bottombar", shell.Props["nav_type"])

	types := childTypes(shell)
	assert.Equal(t, []string{"content_slot", "bottombar"}, types)
}

func TestBuildShell_UnknownOrEmptyPlatformDefaultsToWeb(t *testing.T) {
	for _, p := range []string{"", "unknown", "desktop"} {
		shell := BuildShell("en", p)
		assert.Equal(t, "sidebar", shell.Props["nav_type"], "platform=%q", p)
	}
}

func TestBuildShell_AllNavItemsPresentWithNavigateAction(t *testing.T) {
	expected := []string{"portfolio", "assets", "trades", "snapshots", "import", "analysis"}
	routes := map[string]string{
		"portfolio": "/portfolio",
		"assets":    "/assets",
		"trades":    "/trades",
		"snapshots": "/snapshots",
		"import":    "/import",
		"analysis":  "/analysis",
	}

	shell := BuildShell("en", "web")
	navMain := findChild(shell, "nav_main")
	require.NotNil(t, navMain)

	got := make([]string, 0, len(navMain.Children))
	for _, item := range navMain.Children {
		require.Equal(t, "nav_item", item.Type)
		id := item.ID[len("nav-"):]
		got = append(got, id)

		require.Len(t, item.Actions, 1, "nav_item %s must have exactly one action", item.ID)
		action := item.Actions[0]
		assert.Equal(t, "navigate", action.Type)
		assert.Equal(t, routes[id], action.URL)
	}
	assert.Equal(t, expected, got)
}

func TestBuildShell_BottomBarHasAllSixItems(t *testing.T) {
	shell := BuildShell("en", "web_mobile")
	bb := findChild(shell, "bottombar")
	require.NotNil(t, bb)
	assert.Len(t, bb.Children, 6)
}

func TestBuildShell_LabelsAreTranslated(t *testing.T) {
	en := BuildShell("en", "web")
	es := BuildShell("es", "web")

	enLabels := navLabels(en)
	esLabels := navLabels(es)

	assert.Equal(t, "Portfolio", enLabels["portfolio"])
	assert.Equal(t, "Portafolio", esLabels["portfolio"])
	assert.Equal(t, "Analysis", enLabels["analysis"])
	assert.Equal(t, "Análisis", esLabels["analysis"])
}

func TestBuildShell_UnknownLanguageFallsBackToEnglish(t *testing.T) {
	shell := BuildShell("zz", "web")
	labels := navLabels(shell)
	assert.Equal(t, "Portfolio", labels["portfolio"])
}

func TestBuildShell_NavFooterHasLogoutOnWeb(t *testing.T) {
	shell := BuildShell("en", "web")
	footer := findChild(shell, "nav_footer")
	require.NotNil(t, footer)

	hasLogout := false
	for _, child := range footer.Children {
		for _, action := range child.Actions {
			if action.Type == "logout" {
				hasLogout = true
			}
		}
	}
	assert.True(t, hasLogout)
}

func TestBuildShell_ContentSlotAlwaysPresent(t *testing.T) {
	for _, p := range []string{"web", "web_mobile", "android", "ios"} {
		shell := BuildShell("en", p)
		assert.NotNil(t, findChild(shell, "content_slot"), "platform=%q", p)
	}
}

// helpers

func childTypes(c components.Component) []string {
	types := make([]string, 0, len(c.Children))
	for _, child := range c.Children {
		types = append(types, child.Type)
	}
	return types
}

func findChild(c components.Component, typ string) *components.Component {
	for i, child := range c.Children {
		if child.Type == typ {
			return &c.Children[i]
		}
	}
	return nil
}

func navLabels(shell components.Component) map[string]string {
	out := map[string]string{}
	container := findChild(shell, "nav_main")
	if container == nil {
		container = findChild(shell, "bottombar")
	}
	if container == nil {
		return out
	}
	for _, item := range container.Children {
		id := item.ID[len("nav-"):]
		if label, ok := item.Props["label"].(string); ok {
			out[id] = label
		}
	}
	return out
}

func TestBuildShell_NavHeaderHasExpandedAndCollapsedAppName(t *testing.T) {
	shell := BuildShell("en", "web")
	header := findChild(shell, "nav_header")
	require.NotNil(t, header)
	require.Len(t, header.Children, 2, "nav_header should have expanded + collapsed app-name")

	expanded := header.Children[0]
	assert.Equal(t, "text", expanded.Type)
	assert.Equal(t, "app-name", expanded.ID)
	assert.Equal(t, "VK Investments", expanded.Props["content"])
	assert.Equal(t, "expanded", expanded.Props["sidebar_visibility"])

	collapsed := header.Children[1]
	assert.Equal(t, "text", collapsed.Type)
	assert.Equal(t, "app-name-short", collapsed.ID)
	assert.Equal(t, "VK", collapsed.Props["content"])
	assert.Equal(t, "collapsed", collapsed.Props["sidebar_visibility"])
}

func TestBuildShell_NavHeaderOnWebMobileIsSingleAppName(t *testing.T) {
	shell := BuildShell("en", "web_mobile")
	header := findChild(shell, "nav_header")
	require.NotNil(t, header)
	require.Len(t, header.Children, 1, "web_mobile nav_header must not emit the collapsed variant")

	only := header.Children[0]
	assert.Equal(t, "text", only.Type)
	assert.Equal(t, "app-name", only.ID)
	assert.Equal(t, "VK Investments", only.Props["content"])
	_, hasVisibility := only.Props["sidebar_visibility"]
	assert.False(t, hasVisibility, "bare app-name on non-sidebar nav should not set sidebar_visibility")
}

func TestBuildShell_AllNavItemsHaveNonEmptyIcon(t *testing.T) {
	// The shell spec requires every nav_item to have a non-empty icon so that
	// the sidebar can collapse to an icon-only view without blank cells.
	for _, platform := range []string{"web", "web_mobile", "android", "ios"} {
		shell := BuildShell("en", platform)
		container := findChild(shell, "nav_main")
		if container == nil {
			container = findChild(shell, "bottombar")
		}
		require.NotNil(t, container, "platform=%q has no nav_main or bottombar", platform)
		for _, item := range container.Children {
			if item.Type != "nav_item" {
				continue
			}
			icon, _ := item.Props["icon"].(string)
			assert.NotEmpty(t, icon, "platform=%q nav_item %s must have non-empty icon", platform, item.ID)
		}
	}
}

func TestBuildShell_NavFooterHasSidebarToggleFirst(t *testing.T) {
	shell := BuildShell("en", "web")
	footer := findChild(shell, "nav_footer")
	require.NotNil(t, footer)
	require.Len(t, footer.Children, 4, "nav_footer should have sidebar-toggle, theme-toggle, logout-btn, logout-btn-collapsed")

	toggle := footer.Children[0]
	assert.Equal(t, "icon_toggle", toggle.Type)
	assert.Equal(t, "sidebar-toggle", toggle.ID)
	assert.Equal(t, "panel-left-open", toggle.Props["icon_inactive"])
	assert.Equal(t, "panel-left-close", toggle.Props["icon_active"])
	assert.Equal(t, "Collapse sidebar", toggle.Props["tooltip_inactive"])
	assert.Equal(t, "Expand sidebar", toggle.Props["tooltip_active"])

	require.Len(t, toggle.Actions, 2)
	assert.Equal(t, "toggle_sidebar", toggle.Actions[0].Type)
	assert.Equal(t, "toggle_sidebar", toggle.Actions[1].Type)
}

func TestBuildShell_NavFooterLogoutSplitByVisibility(t *testing.T) {
	shell := BuildShell("en", "web")
	footer := findChild(shell, "nav_footer")
	require.NotNil(t, footer)

	var expanded, collapsed *components.Component
	for i, child := range footer.Children {
		if child.ID == "logout-btn" {
			expanded = &footer.Children[i]
		}
		if child.ID == "logout-btn-collapsed" {
			collapsed = &footer.Children[i]
		}
	}

	require.NotNil(t, expanded, "expanded logout button must exist")
	assert.Equal(t, "button", expanded.Type)
	assert.Equal(t, "Log out", expanded.Props["label"])
	assert.Equal(t, "logout", expanded.Props["icon"])
	assert.Equal(t, "ghost", expanded.Props["style"])
	assert.Equal(t, "expanded", expanded.Props["sidebar_visibility"])
	require.Len(t, expanded.Actions, 1)
	assert.Equal(t, "logout", expanded.Actions[0].Type)

	require.NotNil(t, collapsed, "collapsed logout button must exist")
	assert.Equal(t, "button", collapsed.Type)
	assert.Equal(t, "", collapsed.Props["label"])
	assert.Equal(t, "logout", collapsed.Props["icon"])
	assert.Equal(t, "ghost", collapsed.Props["style"])
	assert.Equal(t, "collapsed", collapsed.Props["sidebar_visibility"])
	require.Len(t, collapsed.Actions, 1)
	assert.Equal(t, "logout", collapsed.Actions[0].Type)
}
