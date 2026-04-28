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
// Cards are constrained to 2/3 of the content width via a 2fr/1fr row with an
// empty spacer in the right column. The modal slot stays outside the row so the
// confirmation modal can overlay full-screen.
func BuildScreen(me *User, cfg *AppConfig, lang string) components.Component {
	cards := components.ColumnWithGap("profile-cards", "lg",
		BuildProfileCard(me, cfg, lang, ""),
		BuildEmailCard(me, lang, "", ""),
		BuildPasswordCard(lang, ""),
		BuildDangerCard(lang),
	)
	cardsRow := components.RowWithGap("profile-cards-row",
		[]string{"2fr", "1fr"}, "none",
		cards,
		components.Spacer("profile-cards-spacer", "none"),
	)
	col := components.ColumnWithGap("profile-column", "lg",
		cardsRow,
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
		components.InputFull("input-display-name", "display_name", "text",
			i18n.T(lang, "profile.display_name"),
			i18n.T(lang, "profile.display_name_placeholder"),
			displayName, false, false, 100),
		components.SelectFull("input-default-currency", "default_currency",
			i18n.T(lang, "profile.default_currency"), "", currency,
			currencyOptions, false, false),
	)
	saveBtn := components.ButtonFull("profile-save", i18n.T(lang, "profile.update.save"),
		"", "primary", "solid",
		components.Submit("/actions/profile/update", "POST", ProfileFormID))
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
		components.InputFull("input-new-email", "new_email", "email",
			i18n.T(lang, "profile.email.new"), "", newEmail, true, false, 0),
		components.InputFull("input-current-password", "current_password", "password",
			i18n.T(lang, "profile.email.current_password"), "", "", true, false, 0),
	)
	saveBtn := components.ButtonFull("email-save", i18n.T(lang, "profile.email.save"),
		"", "primary", "solid",
		components.Submit("/actions/profile/update_email", "POST", EmailFormID))
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
		components.InputFull("input-current-password", "current_password", "password",
			i18n.T(lang, "profile.password.current"), "", "", true, false, 0),
		components.InputFull("input-new-password", "new_password", "password",
			i18n.T(lang, "profile.password.new"), "", "", true, false, 128),
		components.InputFull("input-confirm-password", "confirm_password", "password",
			i18n.T(lang, "profile.password.confirm"), "", "", true, false, 128),
	)
	saveBtn := components.ButtonFull("password-save", i18n.T(lang, "profile.password.save"),
		"", "primary", "solid",
		components.Submit("/actions/profile/change_password", "POST", PasswordFormID))
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
		components.Reload("/actions/profile/delete_modal", ModalSlotID))
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
