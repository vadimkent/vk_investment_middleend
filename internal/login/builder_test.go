package login

import (
	"path/filepath"
	"testing"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = i18n.Load(filepath.Join("..", "..", "locales"))
}

func TestBuildScreen_RootIsLoginScreen(t *testing.T) {
	s := BuildScreen("en")
	assert.Equal(t, "screen", s.Type)
	assert.Equal(t, "login", s.ID)
}

func TestBuildScreen_ContainsCardWithLogoTitleForm(t *testing.T) {
	s := BuildScreen("en")

	card := findDescendantByType(s, "card")
	require.NotNil(t, card, "card should be present in the tree")

	assert.NotNil(t, findDescendantByID(*card, "login-logo"), "logo image missing")

	title := findDescendantByID(*card, "login-title")
	require.NotNil(t, title, "title missing")
	assert.Equal(t, "text", title.Type)
	assert.Equal(t, "Log in", title.Props["content"])

	form := findDescendantByType(*card, "form")
	require.NotNil(t, form, "form missing")
	assert.Equal(t, "login-form", form.ID)
}

func TestBuildScreen_EmailAndPasswordInputsRequired(t *testing.T) {
	s := BuildScreen("en")

	email := findDescendantByID(s, "login-email")
	require.NotNil(t, email)
	assert.Equal(t, "input", email.Type)
	assert.Equal(t, "email", email.Props["input_type"])
	assert.Equal(t, "email", email.Props["name"])
	assert.Equal(t, true, email.Props["required"])

	password := findDescendantByID(s, "login-password")
	require.NotNil(t, password)
	assert.Equal(t, "input", password.Type)
	assert.Equal(t, "password", password.Props["input_type"])
	assert.Equal(t, "password", password.Props["name"])
	assert.Equal(t, true, password.Props["required"])
}

func TestBuildScreen_SubmitButtonHasSubmitAction(t *testing.T) {
	s := BuildScreen("en")

	btn := findDescendantByID(s, "login-submit")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "click", a.Trigger)
	assert.Equal(t, "submit", a.Type)
	assert.Equal(t, "/actions/login", a.Endpoint)
	assert.Equal(t, "POST", a.Method)
	assert.Equal(t, "login-form", a.TargetID)
}

func TestBuildScreen_RegisterLinkNavigates(t *testing.T) {
	s := BuildScreen("en")

	btn := findDescendantByID(s, "register-link")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "click", a.Trigger)
	assert.Equal(t, "navigate", a.Type)
	assert.Equal(t, "/screens/register", a.URL)
	assert.Equal(t, "self", a.Target)
}

func TestBuildScreen_NoShellSlots(t *testing.T) {
	s := BuildScreen("en")
	for _, slot := range []string{"nav_header", "nav_main", "nav_footer", "bottombar", "content_slot"} {
		assert.Nil(t, findDescendantByType(s, slot), "shell slot %q should not appear", slot)
	}
}

func TestBuildScreen_LabelsTranslated(t *testing.T) {
	en := BuildScreen("en")
	es := BuildScreen("es")

	enTitle := findDescendantByID(en, "login-title")
	esTitle := findDescendantByID(es, "login-title")
	require.NotNil(t, enTitle)
	require.NotNil(t, esTitle)
	assert.Equal(t, "Log in", enTitle.Props["content"])
	assert.Equal(t, "Iniciar sesión", esTitle.Props["content"])
}

func TestBuildScreen_UnknownLanguageFallsBackToEnglish(t *testing.T) {
	s := BuildScreen("zz")
	title := findDescendantByID(s, "login-title")
	require.NotNil(t, title)
	assert.Equal(t, "Log in", title.Props["content"])
}

func TestBuildScreen_RootColumnCentersContent(t *testing.T) {
	s := BuildScreen("en")
	root := findDescendantByID(s, "login-root")
	require.NotNil(t, root)
	assert.Equal(t, "column", root.Type)
	assert.Equal(t, "center", root.Props["align_items"])
	assert.Equal(t, "center", root.Props["justify_items"])
}

// helpers

func findDescendantByType(c components.Component, typ string) *components.Component {
	if c.Type == typ {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByType(c.Children[i], typ); found != nil {
			return found
		}
	}
	return nil
}

func findDescendantByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByID(c.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}
