package register

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

func init() {
	_ = i18n.Load(filepath.Join("..", "..", "locales"))
}

func findByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if got := findByID(c.Children[i], id); got != nil {
			return got
		}
	}
	return nil
}

func TestBuildScreen_HasFormWithThreeInputs(t *testing.T) {
	tree := BuildScreen("en", "")

	assert.Equal(t, "screen", tree.Type)
	assert.Equal(t, "register", tree.ID)

	form := findByID(tree, "register-form")
	if assert.NotNil(t, form, "register-form must exist") {
		emailIn := findByID(*form, "register-email")
		passIn := findByID(*form, "register-password")
		confirmIn := findByID(*form, "register-confirm-password")
		assert.Equal(t, "email", emailIn.Props["name"])
		assert.Equal(t, true, emailIn.Props["required"])
		assert.Equal(t, "password", passIn.Props["name"])
		assert.Equal(t, true, passIn.Props["required"])
		assert.Equal(t, 8, passIn.Props["min_length"])
		assert.Equal(t, "confirm_password", confirmIn.Props["name"])
		assert.Equal(t, "password", confirmIn.Props["match_field"])
	}
}

func TestBuildScreen_NoBannerByDefault(t *testing.T) {
	tree := BuildScreen("en", "")
	banner := findByID(tree, "register-banner")
	assert.Nil(t, banner, "banner must be omitted when errorMsg is empty")
}

func TestBuildScreen_BannerWhenErrorMsgPresent(t *testing.T) {
	tree := BuildScreen("en", "Something failed")
	banner := findByID(tree, "register-banner")
	if assert.NotNil(t, banner, "banner must be present when errorMsg is non-empty") {
		assert.Equal(t, "Something failed", banner.Props["text"])
	}
}

func TestBuildScreen_LoginLinkNavigates(t *testing.T) {
	tree := BuildScreen("en", "")
	link := findByID(tree, "register-login-link")
	if assert.NotNil(t, link) {
		assert.Len(t, link.Actions, 1)
		assert.Equal(t, "navigate", link.Actions[0].Type)
		assert.Equal(t, "/screens/login", link.Actions[0].URL)
	}
}

func TestBuildScreen_NoShellSlots(t *testing.T) {
	tree := BuildScreen("en", "")
	for _, id := range []string{"nav_header", "nav_main", "nav_footer", "bottombar", "content_slot"} {
		assert.Nil(t, findByID(tree, id), "shell slot %s must not be present", id)
	}
}

func TestBuildScreen_Spanish(t *testing.T) {
	tree := BuildScreen("es", "")
	title := findByID(tree, "register-title")
	if assert.NotNil(t, title) {
		assert.Equal(t, "Crear cuenta", title.Props["content"])
	}
}

func TestBuildForm_PrefillEmailAndDisableSubmit(t *testing.T) {
	form := BuildForm("en", "user@example.com", "boom", true)
	email := findByID(form, "register-email")
	pass := findByID(form, "register-password")
	confirm := findByID(form, "register-confirm-password")
	submit := findByID(form, "register-submit")

	assert.Equal(t, "user@example.com", email.Props["default_value"])
	_, hasPassDefault := pass.Props["default_value"]
	_, hasConfirmDefault := confirm.Props["default_value"]
	assert.False(t, hasPassDefault, "password must be cleared on rebuild")
	assert.False(t, hasConfirmDefault, "confirm_password must be cleared on rebuild")
	assert.Equal(t, true, submit.Props["disabled"])
}
