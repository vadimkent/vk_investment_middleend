package profile

import (
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// Stable ids referenced by the screen tree and partial-update endpoints.
const (
	ScreenID       = "profile"
	ProfileCardID  = "profile-card"
	ProfileFormID  = "profile-form"
	EmailCardID    = "email-card"
	EmailFormID    = "email-form"
	PasswordCardID = "password-card"
	PasswordFormID = "password-form"
	DangerCardID   = "danger-card"
	ModalSlotID    = "profile-modal-slot"
	DeleteModalID  = "profile-delete-modal"
)

// BuildScreen assembles the full profile screen tree.
//
// Layout: Profile (full width), Email + Password side by side (50/50),
// Danger zone (full width). The modal slot stays outside the stack so the
// confirmation modal can overlay full-screen.
func BuildScreen(me *User, cfg *AppConfig, lang string) components.Component {
	credentialsRow := components.RowWithGap("profile-credentials-row",
		[]string{"1fr", "1fr"}, "md",
		BuildEmailCard(me, lang, "", ""),
		BuildPasswordCard(lang, ""),
	)
	credentialsRow.Props["align_items"] = "start"

	cards := components.ColumnWithGap("profile-cards", "md",
		BuildProfileCard(me, cfg, lang, ""),
		credentialsRow,
		BuildDangerCard(lang),
	)
	col := components.ColumnWithGap("profile-column", "lg",
		cards,
		components.Group(ModalSlotID),
	)
	return components.Screen(ScreenID, i18n.T(lang, "profile.title"), col)
}

// BuildProfileCard wraps the Profile form in a Card. Used by BuildScreen on
// initial render. Handlers re-emit only the Form via BuildProfileForm.
func BuildProfileCard(me *User, cfg *AppConfig, lang, errMessage string) components.Component {
	return components.Card(ProfileCardID, BuildProfileForm(me, cfg, lang, errMessage))
}

// BuildProfileForm returns the Profile form subtree (target_id = ProfileFormID).
// Card is the visual / positioning context (Form inside Card so the FE loading
// overlay anchors to the card).
func BuildProfileForm(me *User, cfg *AppConfig, lang, errMessage string) components.Component {
	return buildProfileFormWith(strDeref(me.DisplayName), strDeref(me.Preferences.DefaultCurrency), cfg, lang, errMessage)
}

// buildProfileFormWith allows the update handler to re-emit with preserved
// (possibly invalid) inputs.
func buildProfileFormWith(displayName, currency string, cfg *AppConfig, lang, errMessage string) components.Component {
	currencyOptions := []components.SelectOption{{Value: "", Label: i18n.T(lang, "profile.default_currency_none")}}
	for _, code := range cfg.Currencies {
		currencyOptions = append(currencyOptions, components.SelectOption{Value: code, Label: code})
	}
	fields := components.RowWithGap("profile-fields",
		[]string{"2fr", "1fr"}, "md",
		components.InputAdvanced(components.InputOptions{
			ID: "input-display-name", Name: "display_name", InputType: "text",
			Label:            i18n.T(lang, "profile.display_name"),
			Placeholder:      i18n.T(lang, "profile.display_name_placeholder"),
			DefaultValue:     displayName,
			MaxLength:        100,
			MaxLengthMessage: i18n.T(lang, "validation.max_length"),
		}),
		components.SelectFull("input-default-currency", "default_currency",
			i18n.T(lang, "profile.default_currency"), "", currency,
			currencyOptions, false, false),
	)
	saveBtn := components.ButtonFull("profile-save", i18n.T(lang, "profile.update.save"),
		"", "primary", "solid",
		components.Submit("/actions/profile/update", "POST", ProfileFormID))
	saveBtn.Props["size"] = "sm"
	actions := components.RowWithGap("profile-actions", []string{"1fr", "auto"}, "sm",
		components.Spacer("profile-actions-spacer", "none"),
		saveBtn,
	)
	formBody := components.ColumnWithGap("profile-form-body", "lg", fields, actions)
	content := cardContent("profile-card-content", lang,
		"profile-section-title", "profile.section.profile",
		"profile-card-error", errMessage,
		formBody,
	)
	return components.Form(ProfileFormID, content)
}

// BuildEmailCard wraps the Email form in a Card. Used by BuildScreen on initial
// render. Handlers re-emit only the Form via BuildEmailForm.
func BuildEmailCard(me *User, lang, newEmail, errMessage string) components.Component {
	return components.Card(EmailCardID, BuildEmailForm(me, lang, newEmail, errMessage))
}

// BuildEmailForm returns the Email form subtree (target_id = EmailFormID).
func BuildEmailForm(me *User, lang, newEmail, errMessage string) components.Component {
	return buildEmailFormWith(me.Email, newEmail, lang, errMessage)
}

func buildEmailFormWith(currentEmail, newEmail, lang, errMessage string) components.Component {
	header := components.ColumnWithGap("email-header", "xs",
		components.Text("email-section-title", i18n.T(lang, "profile.section.email"), "lg", "bold"),
		components.TextStyled("email-current",
			interpolate(i18n.T(lang, "profile.email.current"), map[string]string{"email": currentEmail}),
			"sm", "regular", "block", "muted", "", ""),
	)
	fields := components.RowWithGap("email-fields",
		[]string{"1fr", "1fr"}, "md",
		components.InputAdvanced(components.InputOptions{
			ID: "input-new-email", Name: "new_email", InputType: "email",
			Label:           i18n.T(lang, "profile.email.new"),
			DefaultValue:    newEmail,
			Required:        true,
			RequiredMessage: i18n.T(lang, "validation.required"),
		}),
		components.InputAdvanced(components.InputOptions{
			ID: "input-current-password", Name: "current_password", InputType: "password",
			Label:           i18n.T(lang, "profile.email.current_password"),
			Required:        true,
			RequiredMessage: i18n.T(lang, "validation.required"),
		}),
	)
	saveBtn := components.ButtonFull("email-save", i18n.T(lang, "profile.email.save"),
		"", "primary", "solid",
		components.Submit("/actions/profile/update_email", "POST", EmailFormID))
	saveBtn.Props["size"] = "sm"
	actions := components.RowWithGap("email-actions", []string{"1fr", "auto"}, "sm",
		components.Spacer("email-actions-spacer", "none"),
		saveBtn,
	)
	formBody := components.ColumnWithGap("email-form-body", "lg", fields, actions)
	body := []components.Component{header}
	if errMessage != "" {
		body = append(body, components.TextStyled("email-card-error", errMessage, "sm", "regular", "block", "error", "", ""))
	}
	body = append(body, formBody)
	content := components.ColumnWithGap("email-card-content", "md", body...)
	return components.Form(EmailFormID, content)
}

// BuildPasswordCard wraps the Password form in a Card. Used by BuildScreen on
// initial render. Handlers re-emit only the Form via BuildPasswordForm.
func BuildPasswordCard(lang, errMessage string) components.Component {
	return components.Card(PasswordCardID, BuildPasswordForm(lang, errMessage))
}

// BuildPasswordForm returns the Password form subtree (target_id = PasswordFormID).
// All three inputs are always empty on render (success or error).
func BuildPasswordForm(lang, errMessage string) components.Component {
	fields := components.RowWithGap("password-fields",
		[]string{"1fr", "1fr", "1fr"}, "md",
		components.InputAdvanced(components.InputOptions{
			ID: "input-current-password", Name: "current_password", InputType: "password",
			Label:           i18n.T(lang, "profile.password.current"),
			Required:        true,
			RequiredMessage: i18n.T(lang, "validation.required"),
		}),
		components.InputAdvanced(components.InputOptions{
			ID: "input-new-password", Name: "new_password", InputType: "password",
			Label:            i18n.T(lang, "profile.password.new"),
			Required:         true,
			MaxLength:        128,
			RequiredMessage:  i18n.T(lang, "validation.required"),
			MaxLengthMessage: i18n.T(lang, "validation.max_length"),
		}),
		components.InputAdvanced(components.InputOptions{
			ID: "input-confirm-password", Name: "confirm_password", InputType: "password",
			Label:            i18n.T(lang, "profile.password.confirm"),
			Required:         true,
			MaxLength:        128,
			RequiredMessage:  i18n.T(lang, "validation.required"),
			MaxLengthMessage: i18n.T(lang, "validation.max_length"),
		}),
	)
	saveBtn := components.ButtonFull("password-save", i18n.T(lang, "profile.password.save"),
		"", "primary", "solid",
		components.Submit("/actions/profile/change_password", "POST", PasswordFormID))
	saveBtn.Props["size"] = "sm"
	actions := components.RowWithGap("password-actions", []string{"1fr", "auto"}, "sm",
		components.Spacer("password-actions-spacer", "none"),
		saveBtn,
	)
	formBody := components.ColumnWithGap("password-form-body", "lg", fields, actions)
	content := cardContent("password-card-content", lang,
		"password-section-title", "profile.section.password",
		"password-card-error", errMessage,
		formBody,
	)
	return components.Form(PasswordFormID, content)
}

// BuildDangerCard renders the Danger Zone section. The button opens the delete
// modal via Reload (GET + replace into the modal slot).
func BuildDangerCard(lang string) components.Component {
	header := components.ColumnWithGap("danger-header", "xs",
		components.TextStyled("danger-title", i18n.T(lang, "profile.danger.title"), "lg", "bold", "block", "error", "", ""),
		components.Text("danger-body", i18n.T(lang, "profile.danger.body"), "sm", "regular"),
	)
	deleteBtn := components.ButtonFull("danger-delete-btn",
		i18n.T(lang, "profile.danger.delete_button"),
		"", "destructive", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/profile/delete_modal",
			TargetID: ModalSlotID,
			Loading:  "full",
		})
	deleteBtn.Props["size"] = "sm"
	actions := components.RowWithGap("danger-actions", []string{"1fr", "auto"}, "sm",
		components.Spacer("danger-actions-spacer", "none"),
		deleteBtn,
	)
	return components.Card(DangerCardID,
		components.ColumnWithGap("danger-card-content", "lg", header, actions),
	)
}

// cardContent wraps a card's children (title, optional error banner, body) in a
// vertically-spaced column. titleKey resolves via i18n; errMessage is rendered
// only when non-empty.
func cardContent(id, lang, titleID, titleKey, errID, errMessage string, body components.Component) components.Component {
	children := []components.Component{
		components.Text(titleID, i18n.T(lang, titleKey), "lg", "bold"),
	}
	if errMessage != "" {
		children = append(children, components.TextStyled(errID, errMessage, "sm", "regular", "block", "error", "", ""))
	}
	children = append(children, body)
	return components.ColumnWithGap(id, "md", children...)
}

func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func interpolate(tmpl string, vars map[string]string) string {
	for k, v := range vars {
		tmpl = strings.ReplaceAll(tmpl, "{"+k+"}", v)
	}
	return tmpl
}
