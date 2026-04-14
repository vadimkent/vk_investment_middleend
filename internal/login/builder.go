package login

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildScreen builds the standalone login screen component tree.
// The screen has no shell — it renders on its own, full-viewport.
func BuildScreen(lang string) components.Component {
	emailInput := components.InputFull(
		"login-email", "email", "email",
		i18n.T(lang, "auth.email_label"),
		i18n.T(lang, "auth.email_placeholder"),
		"", true, false, 0,
	)

	passwordInput := components.InputFull(
		"login-password", "password", "password",
		i18n.T(lang, "auth.password_label"),
		i18n.T(lang, "auth.password_placeholder"),
		"", true, false, 0,
	)

	submit := components.Button(
		"login-submit", i18n.T(lang, "auth.submit"),
		components.Submit("/actions/login", "POST", "login-form"),
	)

	form := components.Form("login-form",
		components.ColumnWithGap("login-fields", "12px",
			emailInput,
			passwordInput,
			submit,
		),
	)

	registerRow := components.Row("register-row", []string{"auto", "auto"},
		components.Text("register-prompt", i18n.T(lang, "auth.no_account_prompt"), "sm", "normal"),
		components.ButtonFull(
			"register-link", i18n.T(lang, "auth.register_link"),
			"", "link", "solid",
			components.Navigate("/screens/register"),
		),
	)

	logo := components.Image("login-logo", "/logo.svg", i18n.T(lang, "app.name"))
	title := components.Text("login-title", i18n.T(lang, "auth.login_title"), "xl", "bold")

	card := components.Card("login-card",
		components.ColumnWithGap("login-content", "16px",
			logo,
			title,
			form,
			registerRow,
		),
	)

	root := components.Column("login-root", card)
	root.Props["align_items"] = "center"
	root.Props["justify_items"] = "center"

	return components.Screen("login", i18n.T(lang, "auth.login_title"), root)
}
