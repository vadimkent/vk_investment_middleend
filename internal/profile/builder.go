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
	EmailCardID    = "email-card"
	PasswordCardID = "password-card"
	DangerCardID   = "danger-card"
	ModalSlotID    = "profile-modal-slot"
	DeleteModalID  = "profile-delete-modal"
)

// BuildScreen assembles the full profile screen tree.
func BuildScreen(me *User, cfg *AppConfig, lang string) components.Component {
	col := components.Column("profile-column",
		BuildProfileCard(me, cfg, lang, ""),
		BuildEmailCard(me, lang, "", ""),
		BuildPasswordCard(lang, ""),
		BuildDangerCard(lang),
		components.Group(ModalSlotID),
	)
	return components.Screen(ScreenID, i18n.T(lang, "profile.title"), col)
}

// BuildProfileCard renders the Profile section using values from the User.
func BuildProfileCard(me *User, cfg *AppConfig, lang, errMessage string) components.Component {
	return buildProfileCardWith(strDeref(me.DisplayName), strDeref(me.Preferences.DefaultCurrency), cfg, lang, errMessage)
}

// buildProfileCardWith allows the update handler to re-emit with preserved
// (possibly invalid) inputs.
func buildProfileCardWith(displayName, currency string, cfg *AppConfig, lang, errMessage string) components.Component {
	children := []components.Component{
		components.Text("profile-section-title", i18n.T(lang, "profile.section.profile"), "lg", "bold"),
	}
	if errMessage != "" {
		children = append(children, components.TextStyled("profile-card-error", errMessage, "sm", "regular", "block", "error", "", ""))
	}
	currencyOptions := []components.SelectOption{{Value: "", Label: i18n.T(lang, "profile.default_currency_none")}}
	for _, code := range cfg.Currencies {
		currencyOptions = append(currencyOptions, components.SelectOption{Value: code, Label: code})
	}
	form := components.Form("profile-form",
		components.InputFull("input-display-name", "display_name", "text",
			i18n.T(lang, "profile.display_name"),
			i18n.T(lang, "profile.display_name_placeholder"),
			displayName, false, false, 100),
		components.SelectFull("input-default-currency", "default_currency",
			i18n.T(lang, "profile.default_currency"), "", currency,
			currencyOptions, false, false),
		components.ButtonFull("profile-save", i18n.T(lang, "profile.update.save"),
			"", "primary", "solid",
			components.Submit("/actions/profile/update", "POST", ProfileCardID)),
	)
	children = append(children, form)
	return components.Card(ProfileCardID, children...)
}

// BuildEmailCard renders the Email section. newEmail is the preserved input
// after a validation error; pass "" on the happy path.
func BuildEmailCard(me *User, lang, newEmail, errMessage string) components.Component {
	return buildEmailCardWith(me.Email, newEmail, lang, errMessage)
}

func buildEmailCardWith(currentEmail, newEmail, lang, errMessage string) components.Component {
	children := []components.Component{
		components.Text("email-section-title", i18n.T(lang, "profile.section.email"), "lg", "bold"),
		components.TextStyled("email-current",
			interpolate(i18n.T(lang, "profile.email.current"), map[string]string{"email": currentEmail}),
			"sm", "regular", "block", "muted", "", ""),
	}
	if errMessage != "" {
		children = append(children, components.TextStyled("email-card-error", errMessage, "sm", "regular", "block", "error", "", ""))
	}
	children = append(children, components.Form("email-form",
		components.InputFull("input-new-email", "new_email", "email",
			i18n.T(lang, "profile.email.new"), "", newEmail, true, false, 0),
		components.InputFull("input-current-password", "current_password", "password",
			i18n.T(lang, "profile.email.current_password"), "", "", true, false, 0),
		components.ButtonFull("email-save", i18n.T(lang, "profile.email.save"),
			"", "primary", "solid",
			components.Submit("/actions/profile/update_email", "POST", EmailCardID)),
	))
	return components.Card(EmailCardID, children...)
}

// BuildPasswordCard renders the Password section. All three inputs are always
// empty on render (success or error).
func BuildPasswordCard(lang, errMessage string) components.Component {
	children := []components.Component{
		components.Text("password-section-title", i18n.T(lang, "profile.section.password"), "lg", "bold"),
	}
	if errMessage != "" {
		children = append(children, components.TextStyled("password-card-error", errMessage, "sm", "regular", "block", "error", "", ""))
	}
	children = append(children, components.Form("password-form",
		components.InputFull("input-current-password", "current_password", "password",
			i18n.T(lang, "profile.password.current"), "", "", true, false, 0),
		components.InputFull("input-new-password", "new_password", "password",
			i18n.T(lang, "profile.password.new"), "", "", true, false, 128),
		components.InputFull("input-confirm-password", "confirm_password", "password",
			i18n.T(lang, "profile.password.confirm"), "", "", true, false, 128),
		components.ButtonFull("password-save", i18n.T(lang, "profile.password.save"),
			"", "primary", "solid",
			components.Submit("/actions/profile/change_password", "POST", PasswordCardID)),
	))
	return components.Card(PasswordCardID, children...)
}

// BuildDangerCard renders the Danger Zone section. The button opens the delete
// modal via Reload (GET + replace into the modal slot).
func BuildDangerCard(lang string) components.Component {
	return components.Card(DangerCardID,
		components.TextStyled("danger-title", i18n.T(lang, "profile.danger.title"), "lg", "bold", "block", "error", "", ""),
		components.Text("danger-body", i18n.T(lang, "profile.danger.body"), "sm", "regular"),
		components.ButtonFull("danger-delete-btn",
			i18n.T(lang, "profile.danger.delete_button"),
			"", "destructive", "solid",
			components.Reload("/actions/profile/delete_modal", ModalSlotID)),
	)
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
