package register

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

const (
	ScreenID    = "register"
	CardID      = "register-card"
	FormID      = "register-form"
	BannerID    = "register-banner"
	EmailID     = "register-email"
	PasswordID  = "register-password"
	ConfirmID   = "register-confirm-password"
	SubmitID    = "register-submit"
	LoginLinkID = "register-login-link"
	TitleID     = "register-title"
)

// BuildScreen builds the standalone register screen component tree.
// errorMsg empty means no banner is rendered.
func BuildScreen(lang string, errorMsg string) components.Component {
	form := BuildForm(lang, "", errorMsg, false)

	loginLink := components.Component{
		Type: "button",
		ID:   LoginLinkID,
		Props: map[string]any{
			"label": i18n.T(lang, "auth.login_link"),
			"style": "ghost",
		},
		Actions: []components.Action{components.Navigate("/screens/login")},
	}

	loginRow := components.RowWithGap("register-login-row", []string{"1fr", "auto", "auto"}, "sm",
		components.Column("register-login-row-spacer"),
		components.Text("register-have-prompt", i18n.T(lang, "auth.have_account_prompt"), "sm", "normal"),
		loginLink,
	)
	loginRow.Props["align_items"] = "center"

	appName := components.Text("register-app-name", i18n.T(lang, "app.name"), "xl", "bold")
	title := components.Text(TitleID, i18n.T(lang, "auth.register_title"), "lg", "normal")

	content := components.ColumnWithGap("register-content", "lg",
		appName,
		title,
		form,
		loginRow,
	)

	padded := components.Row("register-padded", []string{"40px", "360px", "40px"},
		components.Column("register-pad-left"),
		content,
		components.Column("register-pad-right"),
	)

	card := components.Card(CardID,
		components.Column("register-card-inner",
			components.Spacer("register-pad-top", "xl"),
			padded,
			components.Spacer("register-pad-bottom", "xl"),
		),
	)

	root := components.Column("register-root", card)
	root.Props["align_items"] = "center"
	root.Props["justify_items"] = "center"

	return components.Screen(ScreenID, i18n.T(lang, "auth.register_title"), root)
}

// BuildForm rebuilds just the form subtree. Used by the action handler to
// produce a `replace` payload for register-form on validation/error outcomes.
//
//	prefillEmail   — value to put back into the email input (passwords always cleared)
//	errorMsg       — when non-empty, an error banner is rendered above the inputs
//	submitDisabled — when true, the submit button has disabled: true (used for REGISTRATION_DISABLED)
func BuildForm(lang, prefillEmail, errorMsg string, submitDisabled bool) components.Component {
	emailInput := components.InputAdvanced(components.InputOptions{
		ID: EmailID, Name: "email", InputType: "email",
		Label:           i18n.T(lang, "auth.email_label"),
		Placeholder:     i18n.T(lang, "auth.email_placeholder"),
		DefaultValue:    prefillEmail,
		Required:        true,
		RequiredMessage: i18n.T(lang, "validation.required"),
	})

	passwordInput := components.InputAdvanced(components.InputOptions{
		ID: PasswordID, Name: "password", InputType: "password",
		Label:            i18n.T(lang, "auth.password_label"),
		Placeholder:      i18n.T(lang, "auth.password_placeholder"),
		Required:         true,
		MinLength:        8,
		RequiredMessage:  i18n.T(lang, "validation.required"),
		MinLengthMessage: i18n.T(lang, "validation.min_length"),
	})

	confirmInput := components.InputAdvanced(components.InputOptions{
		ID: ConfirmID, Name: "confirm_password", InputType: "password",
		Label:             i18n.T(lang, "auth.confirm_password_label"),
		Placeholder:       i18n.T(lang, "auth.confirm_password_placeholder"),
		Required:          true,
		MatchField:        "password",
		RequiredMessage:   i18n.T(lang, "validation.required"),
		MatchFieldMessage: i18n.T(lang, "validation.passwords_must_match"),
	})

	submitProps := map[string]any{"label": i18n.T(lang, "auth.register_submit")}
	if submitDisabled {
		submitProps["disabled"] = true
	}
	submit := components.Component{
		Type:    "button",
		ID:      SubmitID,
		Props:   submitProps,
		Actions: []components.Action{components.Submit("/actions/register", "POST", FormID)},
	}

	submitRow := components.Row("register-submit-row", []string{"1fr", "auto"},
		components.Column("register-submit-spacer"),
		submit,
	)

	stack := components.ColumnWithGap("register-form-stack", "lg")
	if errorMsg != "" {
		stack.Children = append(stack.Children, components.Component{
			Type:  "banner",
			ID:    BannerID,
			Props: map[string]any{"variant": "error", "text": errorMsg},
		})
	}
	stack.Children = append(stack.Children,
		components.ColumnWithGap("register-fields", "md",
			emailInput,
			passwordInput,
			confirmInput,
		),
		submitRow,
	)

	return components.Form(FormID, stack)
}
