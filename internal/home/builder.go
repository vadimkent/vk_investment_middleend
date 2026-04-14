package home

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildScreen builds the SDUI component tree for the home screen.
// All user-facing text uses i18n keys — never hardcoded strings.
func BuildScreen(lang, platform string) components.Component {
	return components.Screen("home", i18n.T(lang, "home.welcome_title"),
		components.Column("home-content",
			components.Text("welcome", i18n.T(lang, "home.welcome_title"), "xl", "bold"),
			components.Text("subtitle", i18n.T(lang, "home.subtitle"), "md", "normal"),
		),
	)
}
