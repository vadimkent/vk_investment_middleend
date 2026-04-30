package login

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildScreen builds the standalone login screen component tree.
// The screen has no shell — it renders on its own, full-viewport.
func BuildScreen(lang string) components.Component {
	emailInput := components.InputAdvanced(components.InputOptions{
		ID: "login-email", Name: "email", InputType: "email",
		Label:           i18n.T(lang, "auth.email_label"),
		Placeholder:     i18n.T(lang, "auth.email_placeholder"),
		Required:        true,
		RequiredMessage: i18n.T(lang, "validation.required"),
	})

	passwordInput := components.InputAdvanced(components.InputOptions{
		ID: "login-password", Name: "password", InputType: "password",
		Label:           i18n.T(lang, "auth.password_label"),
		Placeholder:     i18n.T(lang, "auth.password_placeholder"),
		Required:        true,
		RequiredMessage: i18n.T(lang, "validation.required"),
	})

	submit := components.Button(
		"login-submit", i18n.T(lang, "auth.submit"),
		components.Submit("/actions/login", "POST", "login-form"),
	)

	// 1fr spacer pushes the submit button to the right.
	submitRow := components.Row("login-submit-row", []string{"1fr", "auto"},
		components.Column("login-submit-spacer"),
		submit,
	)

	form := components.Form("login-form",
		components.ColumnWithGap("login-form-stack", "lg",
			components.ColumnWithGap("login-fields", "sm",
				emailInput,
				passwordInput,
			),
			submitRow,
		),
	)

	registerLink := components.Component{
		Type: "button",
		ID:   "register-link",
		Props: map[string]any{
			"label": i18n.T(lang, "auth.register_link"),
			"style": "ghost",
		},
		Actions: []components.Action{components.Navigate("/register")},
	}

	registerRow := components.RowWithGap("register-row", []string{"1fr", "auto", "auto"}, "sm",
		components.Column("register-row-spacer"),
		components.Text("register-prompt", i18n.T(lang, "auth.no_account_prompt"), "sm", "normal"),
		registerLink,
	)
	registerRow.Props["align_items"] = "center"

	appName := components.Text("login-app-name", i18n.T(lang, "app.name"), "xl", "bold")
	title := components.Text("login-title", i18n.T(lang, "auth.login_title"), "lg", "normal")

	content := components.ColumnWithGap("login-content", "lg",
		appName,
		title,
		form,
		registerRow,
	)

	// Horizontal padding via a row with fixed-width side columns acting as
	// gutters. Vertical padding via Spacer siblings above and below.
	// Card width equals the row's total widths (40 + 360 + 40 = 440px).
	padded := components.Row("login-padded", []string{"40px", "360px", "40px"},
		components.Column("login-pad-left"),
		content,
		components.Column("login-pad-right"),
	)

	card := components.Card("login-card",
		components.Column("login-card-inner",
			components.Spacer("login-pad-top", "xl"),
			padded,
			components.Spacer("login-pad-bottom", "xl"),
		),
	)

	root := components.Column("login-root", card)
	root.Props["align_items"] = "center"
	root.Props["justify_items"] = "center"

	return components.Screen("login", i18n.T(lang, "auth.login_title"), root)
}
