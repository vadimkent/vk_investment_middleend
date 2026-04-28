package profile

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type passwordChanger interface {
	ChangePassword(ctx context.Context, authorization, currentPassword, newPassword string) error
}

type ChangePasswordHandler struct {
	changer passwordChanger
}

func NewChangePasswordHandler(changer passwordChanger) *ChangePasswordHandler {
	return &ChangePasswordHandler{changer: changer}
}

func (h *ChangePasswordHandler) Post(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	in, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}
	current, _ := in["current_password"].(string)
	newPw, _ := in["new_password"].(string)
	confirm, _ := in["confirm_password"].(string)

	// Middleend-side validation. No BE call on these paths.
	if current == "" || newPw == "" || confirm == "" {
		respondPasswordError(c, lang, "profile.password.error.missing_fields")
		return
	}
	if newPw != confirm {
		respondPasswordError(c, lang, "profile.password.error.do_not_match")
		return
	}

	if err := h.changer.ChangePassword(c.Request.Context(), auth, current, newPw); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			respondPasswordError(c, lang, passwordErrorKey(be.Code))
			return
		}
		respondBackendError(c, "could not change password")
		return
	}

	tree := BuildPasswordCard(lang, "")
	fb := components.Snackbar("feedback", i18n.T(lang, "profile.password.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: PasswordFormID, Tree: &tree, Feedback: &fb,
	})
}

func respondPasswordError(c *gin.Context, lang, key string) {
	tree := BuildPasswordCard(lang, i18n.T(lang, key))
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: PasswordFormID, Tree: &tree,
	})
}

func passwordErrorKey(code string) string {
	switch code {
	case "MISSING_FIELDS":
		return "profile.password.error.missing_fields"
	case "INVALID_CREDENTIALS":
		return "profile.password.error.invalid_credentials"
	case "INVALID_PASSWORD":
		return "profile.password.error.invalid_password"
	default:
		return "profile.password.error.invalid_credentials"
	}
}
